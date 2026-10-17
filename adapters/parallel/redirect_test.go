package parallel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/justestif/qry/internal/adapter"
)

// Exercise the registered adapter and real http.Client redirect handling. Only
// the final network transport maps the production origin onto a local fixture.
func TestRedirectsThroughSearch(t *testing.T) {
	for _, key := range []string{"", "fake-configured-key"} {
		for _, phase := range []string{"initialize", "tools/call", "cleanup"} {
			for _, status := range []int{301, 302, 303, 307, 308} {
				for _, destination := range []string{"different host", "subdomain", "different port", "downgrade", "userinfo"} {
					t.Run(fmt.Sprintf("auth=%v/%s/%d/%s", key != "", phase, status, destination), func(t *testing.T) {
						runRedirectSearch(t, key, phase, status, destination)
					})
				}
			}
		}
		for _, phase := range []string{"initialize", "tools/call", "cleanup"} {
			for _, destination := range []string{"relative", "same origin", "explicit HTTPS port", "host case", "redirect loop"} {
				t.Run(fmt.Sprintf("auth=%v/%s/%s", key != "", phase, destination), func(t *testing.T) {
					runRedirectSearch(t, key, phase, http.StatusTemporaryRedirect, destination)
				})
			}
		}
	}
}

func runRedirectSearch(t *testing.T, key, phase string, status int, destination string) {
	t.Helper()
	t.Setenv("PARALLEL_API_KEY", "fake-environment-key-must-not-be-read")
	var alternateRequests, alternateCredentials, redirectedTransports, redirectedRequests, phaseRequests atomic.Int32
	alternate := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		alternateRequests.Add(1)
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Api-Key") != "" {
			alternateCredentials.Add(1)
		}
		http.Error(w, "unexpected redirect destination", http.StatusBadGateway)
	}))
	defer alternate.Close()
	plain := httptest.NewServer(alternate.Config.Handler)
	defer plain.Close()
	location := map[string]string{
		"different host":      alternate.URL + "/mcp",
		"subdomain":           "https://sub.search.parallel.ai/mcp",
		"different port":      "https://search.parallel.ai:444/mcp",
		"downgrade":           "http://search.parallel.ai/mcp",
		"userinfo":            "https://fake-user:fake-password@search.parallel.ai/redirected",
		"relative":            "/redirected",
		"same origin":         "https://search.parallel.ai/redirected",
		"explicit HTTPS port": "https://search.parallel.ai:443/redirected",
		"host case":           "https://SEARCH.PARALLEL.AI/redirected",
		"redirect loop":       "/mcp",
	}[destination]
	allowed := destination == "relative" || destination == "same origin" || destination == "explicit HTTPS port" || destination == "host case"
	fixture := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantAuth := ""
		if key != "" {
			wantAuth = "Bearer " + key
		}
		if r.Header.Get("Authorization") != wantAuth || r.Header.Get("X-Api-Key") != "" || r.Header.Get("User-Agent") != userAgent {
			t.Error("unexpected credentials or project identification at original origin")
		}
		method := "cleanup"
		var message struct {
			Method string `json:"method"`
			ID     any    `json:"id"`
		}
		if r.Method != http.MethodDelete {
			if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
				t.Errorf("invalid request: %v", err)
			}
			method = message.Method
		}
		if method == phase && (r.URL.Path != "/redirected" || destination == "redirect loop") {
			phaseRequests.Add(1)
			w.Header().Set("Location", location)
			w.WriteHeader(status)
			return
		}
		if r.URL.Path == "/redirected" {
			redirectedRequests.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "redirect-fixture")
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%v,"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}}`, message.ID)
		case "notifications/initialized", "notifications/cancelled":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%v,"result":{"content":[],"structuredContent":{"results":[{"title":"Fixture result","url":"https://example.com/result","excerpts":["Local fixture output"]}]}}}`, message.ID)
		case "cleanup":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected MCP method %q", method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer fixture.Close()
	originalURL, _ := url.Parse(fixture.URL)
	alternateURL, _ := url.Parse(alternate.URL)
	plainURL, _ := url.Parse(plain.URL)
	base := http.DefaultTransport
	// This transport is test-only and trusts only httptest's fixture certificate.
	localTransport := fixture.Client().Transport.(*http.Transport).Clone()
	defer localTransport.CloseIdleConnections()
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		original := r.URL.Scheme == "https" && strings.EqualFold(r.URL.Hostname(), "search.parallel.ai") && (r.URL.Port() == "" || r.URL.Port() == "443") && r.URL.User == nil
		dest := originalURL
		if !original {
			redirectedTransports.Add(1)
			dest = alternateURL
			if r.URL.Scheme == "http" {
				dest = plainURL
			}
		}
		r = r.Clone(r.Context())
		u := *r.URL
		r.URL = &u
		r.URL.Scheme, r.URL.Host, r.URL.User, r.Host = dest.Scheme, dest.Host, nil, dest.Host
		return localTransport.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = base }()
	got, err := adapter.Get("parallel").Search(context.Background(), "local fixture query", 1, map[string]string{"api_key": key})
	if phaseRequests.Load() == 0 {
		t.Fatal("search never reached redirect response")
	}
	if alternateRequests.Load() != 0 || alternateCredentials.Load() != 0 || redirectedTransports.Load() != 0 {
		t.Fatalf("redirect escaped original origin: transport=%d requests=%d credential-bearing=%d", redirectedTransports.Load(), alternateRequests.Load(), alternateCredentials.Load())
	}
	if allowed || phase == "cleanup" {
		// A rejected cleanup redirect does not invalidate results already returned.
		if err != nil || len(got) != 1 || got[0].URL != "https://example.com/result" || got[0].Snippet != "Local fixture output" {
			t.Fatalf("unexpected downstream results: %#v, %v", got, err)
		}
	} else if err == nil || !strings.HasPrefix(err.Error(), "unavailable:") || len(got) != 0 {
		t.Fatalf("redirect must fail search: %#v, %v", got, err)
	}
	if allowed && redirectedRequests.Load() != 1 {
		t.Fatalf("same-origin HTTPS redirect received %d requests, want 1", redirectedRequests.Load())
	}
	if destination == "redirect loop" && phaseRequests.Load() != 10 {
		t.Fatalf("redirect loop made %d requests, want standard limit of 10", phaseRequests.Load())
	}
}
