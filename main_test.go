package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/justestif/qry/internal/result"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type cliTransport func(*http.Request) (*http.Response, error)

func (f cliTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Run the actual entry point in a subprocess to isolate Cobra/Viper globals and
// os.Exit. Only the test process redirects requests to the fixture server.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("QRY_TEST_HELPER") != "1" {
		return
	}
	if destination := os.Getenv("QRY_TEST_MCP_ENDPOINT"); destination != "" {
		u, _ := url.Parse(destination)
		base := http.DefaultTransport
		http.DefaultTransport = cliTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://search.parallel.ai/mcp" || r.Header.Get("User-Agent") != "qry/parallel (+https://github.com/justEstif/qry)" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Api-Key") != "" {
				return nil, fmt.Errorf("unexpected Parallel destination, identification or credentials")
			}
			r = r.Clone(r.Context())
			r.URL.Scheme, r.URL.Host, r.Host = u.Scheme, u.Host, u.Host
			return base.RoundTrip(r)
		})
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"qry"}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

func TestParallelCLI(t *testing.T) {
	var searches atomic.Int32
	var fail atomic.Bool
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "web_search", InputSchema: map[string]any{"type": "object"}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		searches.Add(1)
		if fail.Load() {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "service unavailable"}}}, nil
		}
		var args struct {
			Objective string   `json:"objective"`
			Queries   []string `json:"search_queries"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil || args.Objective != "numpy latest release" || len(args.Queries) != 1 || args.Queries[0] != args.Objective {
			t.Errorf("unexpected CLI search arguments: %s", req.Params.Arguments)
		}
		return &mcp.CallToolResult{StructuredContent: map[string]any{"results": []map[string]any{
			{"title": "NumPy releases", "url": "https://numpy.org/news/", "excerpts": []string{"Release information"}},
			{"title": "NumPy", "url": "https://numpy.org/", "excerpts": []string{}},
		}}}, nil
	})
	endpoint := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{JSONResponse: true}))
	defer endpoint.Close()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	original := "[defaults]\nnum = 7\n[routing]\nmode = 'first'\npool = ['mock']\n"
	if err := os.WriteFile(cfgFile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) []byte {
		t.Helper()
		proc := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--", "--config", cfgFile}, args...)...)
		proc.Env = append(os.Environ(), "QRY_TEST_HELPER=1", "QRY_TEST_MCP_ENDPOINT="+endpoint.URL, "PARALLEL_API_KEY=must-not-be-read")
		out, err := proc.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI failed: %v\n%s", err, out)
		}
		return out
	}
	out := run("--adapter", "parallel", "--num", "1", "numpy latest release")
	var results []result.Result
	if err := json.Unmarshal(out, &results); err != nil || len(results) != 1 || results[0].URL != "https://numpy.org/news/" || results[0].Snippet != "Release information" || searches.Load() != 1 {
		t.Fatalf("unexpected CLI output: %s (searches=%d, parse=%v)", out, searches.Load(), err)
	}

	// Agent discovery lists Parallel without adding it to the user's saved pool.
	out = run("--agent-info")
	var info struct {
		Config struct {
			Routing struct {
				Pool []string `json:"pool"`
			} `json:"routing"`
			Adapters map[string]struct {
				Available bool `json:"available"`
			} `json:"adapters"`
		} `json:"config"`
	}
	if err := json.Unmarshal(out, &info); err != nil || !info.Config.Adapters["parallel"].Available || strings.Join(info.Config.Routing.Pool, ",") != "mock" || searches.Load() != 1 {
		t.Fatalf("unexpected discovery or routing change: %s", out)
	}

	// Without explicit selection the original mock pool is still used.
	run("numpy latest release")
	if searches.Load() != 1 {
		t.Fatal("existing routing unexpectedly invoked Parallel")
	}
	fail.Store(true)
	fallbackCfg := filepath.Join(dir, "fallback.toml")
	if err := os.WriteFile(fallbackCfg, []byte("[routing]\nmode = 'first'\npool = ['parallel', 'mock']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out = run("--config", fallbackCfg, "numpy latest release")
	if err := json.Unmarshal(out, &results); err != nil || len(results) == 0 || results[0].URL != "https://example.com/result-1" || searches.Load() != 2 {
		t.Fatalf("failed Parallel search did not fall back: %s", out)
	}
	if current, _ := os.ReadFile(cfgFile); string(current) != original {
		t.Fatal("CLI changed saved config")
	}
}
