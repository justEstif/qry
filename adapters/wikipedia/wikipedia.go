package wikipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "wikipedia"
}

type wikiResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			PageID  int    `json:"pageid"`
		} `json:"search"`
	} `json:"query"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return htmlTagRe.ReplaceAllString(s, "")
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	lang := config["language"]
	if lang == "" {
		lang = "en"
	}

	count := num
	if count <= 0 || count > 50 {
		count = 10
	}

	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", query)
	params.Set("srlimit", strconv.Itoa(count))
	params.Set("format", "json")
	params.Set("utf8", "1")

	endpoint := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?%s", lang, params.Encode())

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("User-Agent", "qry-adapter-wikipedia/1.0 (https://github.com/justestif/qry)")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unavailable: Wikipedia API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to read response: %w", err)
	}

	var wikiResp wikiResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return nil, fmt.Errorf("unknown: failed to parse Wikipedia API response: %w", err)
	}

	results := make([]result.Result, 0, len(wikiResp.Query.Search))
	for _, r := range wikiResp.Query.Search {
		wikiURL := fmt.Sprintf("https://%s.wikipedia.org/wiki/%s", lang, strings.ReplaceAll(r.Title, " ", "_"))
		results = append(results, result.Result{
			Title:   r.Title,
			URL:     wikiURL,
			Snippet: stripHTML(r.Snippet),
		})
	}

	return results, nil
}
