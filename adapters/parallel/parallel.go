// Package parallel provides web search through the anonymous Parallel Search MCP.
package parallel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	endpoint  = "https://search.parallel.ai/mcp"
	userAgent = "qry/parallel (+https://github.com/justEstif/qry)"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string { return "parallel" }

// requestTransport adds project identification to every MCP request, including
// initialization and session cleanup. Credentials are opt-in, never read from env.
type requestTransport struct {
	ctx    context.Context
	base   http.RoundTripper
	apiKey string
	status atomic.Int32
}

func (t *requestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// The SDK detaches its session context for background requests and cleanup.
	// Keep every request and its response body within this search's lifetime.
	ctx, cancel := context.WithCancel(req.Context())
	stop := context.AfterFunc(t.ctx, cancel)
	if t.ctx.Err() != nil {
		cancel()
	}
	finish := func() { stop(); cancel() }
	req = req.Clone(ctx)
	req.Header.Set("User-Agent", userAgent)
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		finish()
		return resp, err
	}
	resp.Body = &contextBody{ReadCloser: resp.Body, finish: finish}
	// Successful cleanup/notification responses must not hide the failed call.
	if req.Method == http.MethodPost && resp.StatusCode >= 400 {
		t.status.Store(int32(resp.StatusCode))
	}
	return resp, err
}

type contextBody struct {
	io.ReadCloser
	finish func()
}

func (b *contextBody) Close() error {
	err := b.ReadCloser.Close()
	b.finish()
	return err
}

func (t *requestTransport) failure(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("timeout: Parallel MCP request stopped: %w", ctx.Err())
	}
	code := "unavailable"
	switch t.status.Load() {
	case http.StatusTooManyRequests:
		code = "rate_limited"
	case http.StatusUnauthorized, http.StatusForbidden:
		code = "auth_failed"
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		code = "invalid_query"
	}
	return fmt.Errorf("%s: Parallel MCP request failed: %w", code, err)
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}
	if num <= 0 {
		num = 10
	}

	// Bound the whole operation as well as respecting any caller deadline.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	transport := &requestTransport{ctx: ctx, base: http.DefaultTransport, apiKey: config["api_key"]}
	client := mcp.NewClient(&mcp.Implementation{Name: "qry", Version: "dev"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: transport, Timeout: 30 * time.Second},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-03-26"})
	if err != nil {
		return nil, transport.failure(ctx, err)
	}
	defer session.Close()

	response, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "web_search",
		Arguments: map[string]any{
			"objective":      query,
			"search_queries": []string{query},
		},
	})
	if err != nil {
		return nil, transport.failure(ctx, err)
	}
	if response.IsError {
		var messages []string
		for _, content := range response.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				messages = append(messages, text.Text)
			}
		}
		return nil, fmt.Errorf("unavailable: Parallel MCP search failed: %s", strings.Join(messages, "; "))
	}

	// Prefer the advertised structured output. Older MCP responses may instead
	// contain the same search object as JSON in a text content block.
	if response.StructuredContent != nil {
		data, err := json.Marshal(response.StructuredContent)
		if err != nil {
			return nil, fmt.Errorf("unknown: invalid Parallel search response: %w", err)
		}
		return parseResults(data, num)
	}
	for _, content := range response.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			return parseResults([]byte(text.Text), num)
		}
	}
	return nil, fmt.Errorf("unknown: missing Parallel search results")
}

func parseResults(data []byte, num int) ([]result.Result, error) {
	var response struct {
		Results *[]struct {
			Title    string   `json:"title"`
			URL      string   `json:"url"`
			Excerpts []string `json:"excerpts"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("unknown: invalid Parallel search response: %w", err)
	}
	if response.Results == nil {
		return nil, errors.New("unknown: missing Parallel search results")
	}
	results := make([]result.Result, 0)
	for _, r := range *response.Results {
		if len(results) >= num {
			break
		}
		if r.URL == "" {
			return nil, errors.New("unknown: Parallel search result has no URL")
		}
		snippet := []rune(strings.Join(strings.Fields(strings.Join(r.Excerpts, "\n")), " "))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		results = append(results, result.Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: string(snippet),
		})
	}
	return results, nil
}
