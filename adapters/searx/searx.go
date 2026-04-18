package searx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "searx"
}

type searxResponse struct {
	Results []searxResult `json:"results"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	instance := config["instance"]
	if instance == "" {
		instance = "https://search.sapti.me"
	}
	instance = strings.TrimRight(instance, "/")

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")

	if v := config["engines"]; v != "" {
		params.Set("engines", v)
	}
	if v := config["language"]; v != "" {
		params.Set("language", v)
	}
	if v := config["time_range"]; v != "" {
		params.Set("time_range", v)
	}
	if v := config["safesearch"]; v != "" {
		params.Set("safesearch", v)
	}

	endpoint := instance + "/search?" + params.Encode()

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "qry-adapter-searx/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("rate_limited: SearXNG returned 429 Too Many Requests")
		case http.StatusBadRequest:
			return nil, fmt.Errorf("invalid_query: SearXNG returned 400: %s", string(body))
		default:
			return nil, fmt.Errorf("unavailable: SearXNG returned status %d: %s", resp.StatusCode, string(body))
		}
	}

	var searxResp searxResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxResp); err != nil {
		return nil, fmt.Errorf("unknown: failed to parse SearXNG response: %w", err)
	}

	if num <= 0 {
		num = 10
	}

	results := make([]result.Result, 0, len(searxResp.Results))
	for _, r := range searxResp.Results {
		if len(results) >= num {
			break
		}
		results = append(results, result.Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}

	return results, nil
}
