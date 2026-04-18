package bravescrape

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "brave-scrape"
}

var (
	reBlock   = regexp.MustCompile(`class="snippet[^"]*svelte-jmfu5f[^"]*" data-pos="\d+" data-type="web"`)
	reURL     = regexp.MustCompile(`<a href="(https?://[^"]+)"`)
	reTitle   = regexp.MustCompile(`class="title search-snippet-title[^"]*" title="([^"]+)"`)
	reSnippet = regexp.MustCompile(`class="content desktop-default-regular[^"]*">(.*?)</div>`)
	reTag     = regexp.MustCompile(`<[^>]+>`)
)

func stripTags(s string) string {
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

func splitBlocks(page string) []string {
	indices := reBlock.FindAllStringIndex(page, -1)
	if len(indices) == 0 {
		return nil
	}
	blocks := make([]string, 0, len(indices))
	for i, loc := range indices {
		end := len(page)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		blocks = append(blocks, page[loc[0]:end])
	}
	return blocks
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	safe := config["safe"]
	if safe == "" {
		safe = "moderate"
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("source", "web")
	params.Set("safe", safe)
	endpoint := "https://search.brave.com/search?" + params.Encode()

	transport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	client := &http.Client{Transport: transport}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml")
	httpReq.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate_limited: Brave Search returned 429 Too Many Requests")
	default:
		return nil, fmt.Errorf("unavailable: Brave Search returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to read response body: %w", err)
	}
	page := string(body)

	blocks := splitBlocks(page)
	if len(blocks) == 0 {
		if strings.Contains(page, "captcha") || strings.Contains(page, "rate-limit") {
			return nil, fmt.Errorf("rate_limited: Brave Search served a challenge page")
		}
		return nil, fmt.Errorf("unavailable: no results found or page structure changed")
	}

	if num <= 0 {
		num = len(blocks)
	}

	results := make([]result.Result, 0, num)
	for _, b := range blocks {
		if len(results) >= num {
			break
		}

		urlM := reURL.FindStringSubmatch(b)
		if urlM == nil {
			continue
		}
		u := urlM[1]

		title := ""
		if m := reTitle.FindStringSubmatch(b); m != nil {
			title = html.UnescapeString(m[1])
		}

		snippet := ""
		if m := reSnippet.FindStringSubmatch(b); m != nil {
			snippet = stripTags(m[1])
		}

		results = append(results, result.Result{
			Title:   title,
			URL:     u,
			Snippet: snippet,
		})
	}

	return results, nil
}
