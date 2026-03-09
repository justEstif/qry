// qry-adapter-searx searches via a SearXNG instance.
//
// Required config:
//   instance — base URL of the SearXNG instance (default: https://search.sapti.me)
//
// Optional config:
//   engines    — comma-separated engine list (e.g. "google,bing")
//   language   — language code (e.g. "en")
//   time_range — day | week | month | year
//   safesearch — 0 | 1 | 2
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

type searxResponse struct {
	Results []searxResult `json:"results"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
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

	instance := req.Config["instance"]
	if instance == "" {
		instance = "https://search.sapti.me"
	}
	instance = strings.TrimRight(instance, "/")

	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("format", "json")

	if v := req.Config["engines"]; v != "" {
		params.Set("engines", v)
	}
	if v := req.Config["language"]; v != "" {
		params.Set("language", v)
	}
	if v := req.Config["time_range"]; v != "" {
		params.Set("time_range", v)
	}
	if v := req.Config["safesearch"]; v != "" {
		params.Set("safesearch", v)
	}

	endpoint := instance + "/search?" + params.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "qry-adapter-searx/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			writeError("rate_limited", fmt.Sprintf("SearXNG returned 429 Too Many Requests"))
		case http.StatusBadRequest:
			writeError("invalid_query", fmt.Sprintf("SearXNG returned 400: %s", string(body)))
		default:
			writeError("unavailable", fmt.Sprintf("SearXNG returned status %d: %s", resp.StatusCode, string(body)))
		}
		os.Exit(1)
	}

	var searxResp searxResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxResp); err != nil {
		writeError("unknown", "failed to parse SearXNG response: "+err.Error())
		os.Exit(1)
	}

	num := req.Num
	if num <= 0 {
		num = 10
	}

	results := make([]Result, 0, len(searxResp.Results))
	for _, r := range searxResp.Results {
		if len(results) >= num {
			break
		}
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		writeError("unknown", "failed to encode results: "+err.Error())
		os.Exit(1)
	}
}
