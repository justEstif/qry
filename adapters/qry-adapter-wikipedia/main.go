// qry-adapter-wikipedia searches via the MediaWiki Action API.
//
// Optional config:
//   language — Wikipedia language code (default: "en")
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Query  string            `json:"query"`
	Num    int               `json:"num"`
	Config map[string]string `json:"config"`
}

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
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

func writeError(code, message string) {
	json.NewEncoder(os.Stderr).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

func main() {
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError("unknown", "failed to parse request: "+err.Error())
		os.Exit(1)
	}

	if req.Query == "" {
		writeError("invalid_query", "query must not be empty")
		os.Exit(1)
	}

	lang := req.Config["language"]
	if lang == "" {
		lang = "en"
	}

	count := req.Num
	if count <= 0 || count > 50 {
		count = 10
	}

	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", req.Query)
	params.Set("srlimit", strconv.Itoa(count))
	params.Set("format", "json")
	params.Set("utf8", "1")

	endpoint := fmt.Sprintf("https://%s.wikipedia.org/w/api.php?%s", lang, params.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("User-Agent", "qry-adapter-wikipedia/1.0 (https://github.com/justestif/qry)")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError("unavailable", fmt.Sprintf("Wikipedia API returned status %d", resp.StatusCode))
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError("unknown", "failed to read response: "+err.Error())
		os.Exit(1)
	}

	var wikiResp wikiResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		writeError("unknown", "failed to parse Wikipedia API response: "+err.Error())
		os.Exit(1)
	}

	results := make([]Result, 0, len(wikiResp.Query.Search))
	for _, r := range wikiResp.Query.Search {
		wikiURL := fmt.Sprintf("https://%s.wikipedia.org/wiki/%s", lang, strings.ReplaceAll(r.Title, " ", "_"))
		results = append(results, Result{
			Title:   r.Title,
			URL:     wikiURL,
			Snippet: stripHTML(r.Snippet),
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		writeError("unknown", "failed to encode results: "+err.Error())
		os.Exit(1)
	}
}
