package stackoverflow

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "stackoverflow"
}

type seResponse struct {
	Items          []seItem `json:"items"`
	Backoff        int      `json:"backoff"`
	QuotaRemaining int      `json:"quota_remaining"`
	HasMore        bool     `json:"has_more"`
}

type seItem struct {
	Title      string `json:"title"`
	QuestionID int    `json:"question_id"`
	Excerpt    string `json:"excerpt"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return htmlTagRe.ReplaceAllString(s, "")
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	count := num
	if count <= 0 || count > 25 {
		count = 10
	}

	site := config["site"]
	if site == "" {
		site = "stackoverflow"
	}

	params := url.Values{}
	params.Set("order", "desc")
	params.Set("sort", "relevance")
	params.Set("q", query)
	params.Set("site", site)
	params.Set("pagesize", strconv.Itoa(count))

	if v := config["key"]; v != "" {
		params.Set("key", v)
	}
	if v := config["tagged"]; v != "" {
		params.Set("tagged", v)
	}

	endpoint := "https://api.stackexchange.com/2.3/search/excerpts?" + params.Encode()

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		return nil, fmt.Errorf("invalid_query: Stack Exchange API returned 400 for query: %q", query)
	case http.StatusBadGateway:
		return nil, fmt.Errorf("unavailable: Stack Exchange API returned 502 Bad Gateway")
	default:
		return nil, fmt.Errorf("unavailable: Stack Exchange API returned unexpected status %d", resp.StatusCode)
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

	var seResp seResponse
	if err := json.NewDecoder(reader).Decode(&seResp); err != nil {
		return nil, fmt.Errorf("unknown: failed to parse Stack Exchange API response: %w", err)
	}

	if seResp.Backoff > 0 {
		return nil, fmt.Errorf("rate_limited: Stack Exchange API requested backoff of %d seconds", seResp.Backoff)
	}

	if seResp.QuotaRemaining <= 0 {
		return nil, fmt.Errorf("rate_limited: Stack Exchange API quota exhausted")
	}

	results := make([]result.Result, 0, len(seResp.Items))
	for _, item := range seResp.Items {
		results = append(results, result.Result{
			Title:   stripHTML(item.Title),
			URL:     fmt.Sprintf("https://stackoverflow.com/q/%d", item.QuestionID),
			Snippet: stripHTML(item.Excerpt),
		})
	}

	return results, nil
}
