package braveapi

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "brave-api"
}

type braveResponse struct {
	Web struct {
		Results []braveResult `json:"results"`
	} `json:"web"`
}

type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	apiKey := config["api_key"]
	if apiKey == "" {
		return nil, fmt.Errorf("auth_failed: api_key is required but not set in adapter config")
	}
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	count := num
	if count <= 0 || count > 20 {
		count = 20
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("count", strconv.Itoa(count))

	if v := config["country"]; v != "" {
		params.Set("country", v)
	}
	if v := config["search_lang"]; v != "" {
		params.Set("search_lang", v)
	}
	if v := config["freshness"]; v != "" {
		params.Set("freshness", v)
	}

	endpoint := "https://api.search.brave.com/res/v1/web/search?" + params.Encode()

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Accept-Encoding", "gzip")
	httpReq.Header.Set("X-Subscription-Token", apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("auth_failed: Brave API returned %d — check your api_key", resp.StatusCode)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate_limited: Brave API returned 429 Too Many Requests")
	case http.StatusBadRequest:
		return nil, fmt.Errorf("invalid_query: Brave API returned 400 Bad Request for query: %q", query)
	default:
		return nil, fmt.Errorf("unavailable: Brave API returned unexpected status %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("unknown: failed to decompress gzip response: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	var braveResp braveResponse
	if err := json.NewDecoder(reader).Decode(&braveResp); err != nil {
		return nil, fmt.Errorf("unknown: failed to parse Brave API response: %w", err)
	}

	results := make([]result.Result, 0, len(braveResp.Web.Results))
	for _, r := range braveResp.Web.Results {
		results = append(results, result.Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}

	return results, nil
}
