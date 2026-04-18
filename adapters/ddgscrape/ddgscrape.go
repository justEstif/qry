package ddgscrape

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
	"golang.org/x/net/html"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "ddg-scrape"
}

// --- randomized headers (from crush) ---

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
}

var acceptLanguages = []string{
	"en-US,en;q=0.9",
	"en-US,en;q=0.9,es;q=0.8",
	"en-GB,en;q=0.9,en-US;q=0.8",
	"en-US,en;q=0.5",
	"en-CA,en;q=0.9,en-US;q=0.8",
}

func setRandomizedHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", acceptLanguages[rand.IntN(len(acceptLanguages))])
	req.Header.Set("Accept-Encoding", "identity") // no compression
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
	if rand.IntN(2) == 0 {
		req.Header.Set("DNT", "1")
	}
}

// --- HTML parsing ---

type searchResult struct {
	Title   string
	Link    string
	Snippet string
}

func parseLiteResults(body string, maxResults int) []searchResult {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil
	}

	var results []searchResult
	var current *searchResult

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if len(results) >= maxResults {
			return
		}
		if n.Type == html.ElementNode {
			if n.Data == "a" && hasClass(n, "result-link") {
				if current != nil && current.Link != "" {
					results = append(results, *current)
				}
				if len(results) >= maxResults {
					return
				}
				current = &searchResult{Title: textContent(n)}
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						current.Link = cleanURL(attr.Val)
						break
					}
				}
			}
			if n.Data == "td" && hasClass(n, "result-snippet") && current != nil {
				current.Snippet = textContent(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if current != nil && current.Link != "" && len(results) < maxResults {
		results = append(results, *current)
	}
	return results
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func cleanURL(raw string) string {
	if strings.HasPrefix(raw, "//duckduckgo.com/l/?uddg=") {
		if _, after, ok := strings.Cut(raw, "uddg="); ok {
			encoded := after
			if i := strings.Index(encoded, "&"); i != -1 {
				encoded = encoded[:i]
			}
			if decoded, err := url.QueryUnescape(encoded); err == nil {
				return decoded
			}
		}
	}
	return raw
}

// --- rate limiting (from crush) ---

var lastSearch time.Time

func maybeDelay(ctx context.Context) error {
	gap := time.Duration(500+rand.IntN(1500)) * time.Millisecond
	elapsed := time.Since(lastSearch)
	if elapsed < gap {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(gap - elapsed):
		}
	}
	lastSearch = time.Now()
	return nil
}

// Search implements the Adapter interface
func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	if num <= 0 {
		num = 10
	}

	params := url.Values{}
	params.Set("q", query)
	if region := config["region"]; region != "" {
		params.Set("kl", region)
	}
	endpoint := "https://lite.duckduckgo.com/lite/?" + params.Encode()

	if err := maybeDelay(ctx); err != nil {
		return nil, err
	}

	transport := &http.Transport{
		ForceAttemptHTTP2: false,
	}
	client := &http.Client{
		Transport: transport,
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	setRandomizedHeaders(httpReq)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusAccepted {
		return nil, fmt.Errorf("rate_limited: DuckDuckGo returned %d — try again shortly", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unavailable: DuckDuckGo returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to read response body: %w", err)
	}

	parsed := parseLiteResults(string(body), num)
	if len(parsed) == 0 {
		return []result.Result{}, nil
	}

	results := make([]result.Result, len(parsed))
	for i, r := range parsed {
		results[i] = result.Result{Title: r.Title, URL: r.Link, Snippet: r.Snippet}
	}

	return results, nil
}
