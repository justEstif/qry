// qry-adapter-stackoverflow searches via the Stack Exchange API.
//
// Optional config:
//   key    — Stack Exchange API key (higher quota)
//   site   — Stack Exchange site (default: stackoverflow)
//   tagged — semicolon-separated tag filter
package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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

type seResponse struct {
	Items          []seItem `json:"items"`
	Backoff        int      `json:"backoff"`
	QuotaRemaining int     `json:"quota_remaining"`
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

	count := req.Num
	if count <= 0 || count > 25 {
		count = 10
	}

	site := req.Config["site"]
	if site == "" {
		site = "stackoverflow"
	}

	params := url.Values{}
	params.Set("order", "desc")
	params.Set("sort", "relevance")
	params.Set("q", req.Query)
	params.Set("site", site)
	params.Set("pagesize", strconv.Itoa(count))

	if v := req.Config["key"]; v != "" {
		params.Set("key", v)
	}
	if v := req.Config["tagged"]; v != "" {
		params.Set("tagged", v)
	}

	endpoint := "https://api.stackexchange.com/2.3/search/excerpts?" + params.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusBadRequest:
		writeError("invalid_query", fmt.Sprintf("Stack Exchange API returned 400 for query: %q", req.Query))
		os.Exit(1)
	case http.StatusBadGateway:
		writeError("unavailable", "Stack Exchange API returned 502 Bad Gateway")
		os.Exit(1)
	default:
		writeError("unavailable", fmt.Sprintf("Stack Exchange API returned unexpected status %d", resp.StatusCode))
		os.Exit(1)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			writeError("unknown", "failed to decompress gzip response: "+err.Error())
			os.Exit(1)
		}
		defer gz.Close()
		reader = gz
	}

	var seResp seResponse
	if err := json.NewDecoder(reader).Decode(&seResp); err != nil {
		writeError("unknown", "failed to parse Stack Exchange API response: "+err.Error())
		os.Exit(1)
	}

	if seResp.Backoff > 0 {
		writeError("rate_limited", fmt.Sprintf("Stack Exchange API requested backoff of %d seconds", seResp.Backoff))
		os.Exit(1)
	}

	if seResp.QuotaRemaining <= 0 {
		writeError("rate_limited", "Stack Exchange API quota exhausted")
		os.Exit(1)
	}

	results := make([]Result, 0, len(seResp.Items))
	for _, item := range seResp.Items {
		results = append(results, Result{
			Title:   stripHTML(item.Title),
			URL:     fmt.Sprintf("https://stackoverflow.com/q/%d", item.QuestionID),
			Snippet: stripHTML(item.Excerpt),
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		writeError("unknown", "failed to encode results: "+err.Error())
		os.Exit(1)
	}
}
