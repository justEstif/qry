package parallel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/config"
	"github.com/justestif/qry/internal/result"
	"github.com/justestif/qry/internal/router"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Route the real adapter's HTTPS requests to a local HTTP server, checking the
// destination and headers before rewriting. Tests are deliberately not parallel.
func mockMCP(t *testing.T, sse bool, key string, call http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		var req struct {
			ID     any             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("invalid MCP request: %v", err)
		}
		switch req.Method {
		case "initialize":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Mcp-Session-Id", "test-session")
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%v,"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}}`, req.ID)
		case "notifications/initialized", "notifications/cancelled":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			if r.Header.Get("Mcp-Session-Id") != "test-session" {
				t.Error("missing negotiated session")
			}
			if r.Header.Get("MCP-Protocol-Version") != "2025-03-26" {
				t.Error("missing negotiated protocol version")
			}
			var params struct {
				Name      string `json:"name"`
				Arguments struct {
					Objective string   `json:"objective"`
					Queries   []string `json:"search_queries"`
				} `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				t.Error(err)
			}
			if params.Name != "web_search" || params.Arguments.Objective != "numpy latest release" || !reflect.DeepEqual(params.Arguments.Queries, []string{"numpy latest release"}) {
				t.Errorf("unexpected tool arguments: %+v", params)
			}
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			if sse {
				w.Header().Set("Content-Type", "text/event-stream")
			} else {
				w.Header().Set("Content-Type", "application/json")
			}
			call(w, r)
		default:
			t.Errorf("unexpected MCP method %q", req.Method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	destination, _ := url.Parse(server.URL)
	base := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != endpoint || r.Header.Get("User-Agent") != userAgent {
			t.Errorf("wrong destination or User-Agent: %s, %q", r.URL, r.Header.Get("User-Agent"))
		}
		wantAuth := ""
		if key != "" {
			wantAuth = "Bearer " + key
		}
		if r.Header.Get("Authorization") != wantAuth || r.Header.Get("X-Api-Key") != "" {
			t.Error("unexpected credentials")
		}
		if r.Method == http.MethodPost && (!strings.Contains(r.Header.Get("Accept"), "application/json") || !strings.Contains(r.Header.Get("Accept"), "text/event-stream")) {
			t.Error("missing Streamable HTTP Accept types")
		}
		r = r.Clone(r.Context())
		r.URL.Scheme, r.URL.Host = destination.Scheme, destination.Host
		r.Host = destination.Host
		return base.RoundTrip(r)
	})
	t.Cleanup(func() { http.DefaultTransport = base })
}

func respond(w http.ResponseWriter, r *http.Request, sse bool, payload string) {
	var req struct {
		ID any `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%v,"result":%s}`, req.ID, payload)
	if sse {
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", body)
	} else {
		fmt.Fprint(w, body)
	}
}

func TestRegisteredAdapterThroughRouter(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprintf("SSE=%v", sse), func(t *testing.T) {
			t.Setenv("PARALLEL_API_KEY", "must-not-be-read")
			mockMCP(t, sse, "", func(w http.ResponseWriter, r *http.Request) {
				respond(w, r, sse, `{"content":[],"structuredContent":{"results":[{"title":"NumPy releases","url":"https://numpy.org/news/","excerpts":["First\n excerpt", "Second excerpt"]},{"title":null,"url":"https://numpy.org/","excerpts":[]}]}}`)
			})
			cfg := &config.Config{Routing: config.Routing{Pool: []string{"parallel"}}, Defaults: config.Defaults{Num: 1}}
			cfg.ApplyDefaults()
			if err := cfg.Validate(); err != nil {
				t.Fatal(err)
			}
			got, err := router.New(cfg, "numpy latest release").Run(context.Background())
			want := []result.Result{{Title: "NumPy releases", URL: "https://numpy.org/news/", Snippet: "First excerpt Second excerpt"}}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, want)
			}
		})
	}
}

func TestResponseContracts(t *testing.T) {
	for _, tc := range []struct {
		name, payload, errorPrefix string
		count                      int
	}{
		{"text JSON", `{"content":[{"type":"text","text":"{\"results\":[{\"url\":\"https://numpy.org\",\"excerpts\":[]}]}"}]}`, "", 1},
		{"long Unicode excerpt", `{"content":[],"structuredContent":{"results":[{"url":"https://numpy.org","excerpts":["` + strings.Repeat("é", 400) + `"]}]}}`, "", 1},
		{"empty", `{"content":[],"structuredContent":{"results":[]}}`, "", 0},
		{"tool error", `{"isError":true,"content":[{"type":"text","text":"search temporarily unavailable"}]}`, "unavailable:", 0},
		{"malformed text", `{"content":[{"type":"text","text":"not JSON"}]}`, "unknown:", 0},
		{"missing results", `{"content":[],"structuredContent":{}}`, "unknown:", 0},
		{"null results", `{"content":[],"structuredContent":{"results":null}}`, "unknown:", 0},
		{"missing URL", `{"content":[],"structuredContent":{"results":[{"title":"broken","excerpts":[]}]}}`, "unknown:", 0},
		{"no content", `{"content":[]}`, "unknown:", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockMCP(t, false, "", func(w http.ResponseWriter, r *http.Request) { respond(w, r, false, tc.payload) })
			got, err := adapter.Get("parallel").Search(context.Background(), "numpy latest release", 0, nil)
			if tc.errorPrefix != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tc.errorPrefix) {
					t.Fatalf("got %v, want %s", err, tc.errorPrefix)
				}
			} else if err != nil || got == nil || len(got) != tc.count {
				t.Fatalf("got %#v, %v", got, err)
			}
			if tc.name == "long Unicode excerpt" && (!utf8.ValidString(got[0].Snippet) || utf8.RuneCountInString(got[0].Snippet) != 300) {
				t.Fatalf("snippet was not bounded at a Unicode boundary: %q", got[0].Snippet)
			}
		})
	}
}

func TestOptionalKeyAndHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		prefix string
	}{
		{429, "rate_limited:"}, {401, "auth_failed:"}, {403, "auth_failed:"},
		{400, "invalid_query:"}, {422, "invalid_query:"}, {503, "unavailable:"},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			calls := 0
			mockMCP(t, false, "configured-key", func(w http.ResponseWriter, r *http.Request) {
				calls++
				http.Error(w, "service error", tc.status)
			})
			_, err := adapter.Get("parallel").Search(context.Background(), "numpy latest release", 1, map[string]string{"api_key": "configured-key"})
			if err == nil || !strings.HasPrefix(err.Error(), tc.prefix) || calls != 1 {
				t.Fatalf("got %v after %d calls; want %s, no retries", err, calls, tc.prefix)
			}
		})
	}
}

func TestCancellationAndDeadline(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := context.Canceled
			if deadline {
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
				want = context.DeadlineExceeded
			}
			mockMCP(t, false, "", func(w http.ResponseWriter, r *http.Request) {
				if !deadline {
					cancel()
				}
				<-r.Context().Done()
			})
			_, err := adapter.Get("parallel").Search(ctx, "numpy latest release", 1, nil)
			if !errors.Is(err, want) {
				t.Fatalf("got %v; want %v", err, want)
			}
		})
	}
}

func TestEmptyQueryDoesNotConnect(t *testing.T) {
	base := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("empty query made a network request")
		return nil, errors.New("unexpected request")
	})
	defer func() { http.DefaultTransport = base }()
	_, err := adapter.Get("parallel").Search(context.Background(), "  \n", 1, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "invalid_query:") {
		t.Fatalf("got %v", err)
	}
}
