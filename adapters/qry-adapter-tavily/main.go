// qry-adapter-tavily searches via the Tavily Search API.
//
// Required config:
//   api_key  — Tavily API key
//
// Optional config:
//   search_depth     — "basic" or "advanced" (default: "basic")
//   include_domains  — comma-separated domain allowlist
//   exclude_domains  — comma-separated domain blocklist
//   max_results      — number of results to return (default: from req.Num)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// --- qry adapter protocol types ---

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

// --- Tavily API types ---

type tavilyRequest struct {
	Query          string   `json:"query"`
	SearchDepth    string   `json:"search_depth,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
}

type tavilyResponse struct {
	Results []tavilyResult `json:"results"`
}

type tavilyResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// --- error helpers ---

func writeError(code, message string) {
	json.NewEncoder(os.Stderr).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

func main() {
	// 1. Read and parse the qry request from stdin
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError("unknown", "failed to parse request: "+err.Error())
		os.Exit(1)
	}

	// 2. Validate required config
	apiKey := req.Config["api_key"]
	if apiKey == "" {
		writeError("auth_failed", "api_key is required but not set in adapter config")
		os.Exit(1)
	}

	if req.Query == "" {
		writeError("invalid_query", "query must not be empty")
		os.Exit(1)
	}

	// 3. Build Tavily request body
	maxResults := req.Num
	if maxResults <= 0 || maxResults > 20 {
		maxResults = 20
	}
	if v := req.Config["max_results"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 20 {
			maxResults = n
		}
	}

	searchDepth := "basic"
	if v := req.Config["search_depth"]; v == "basic" || v == "advanced" {
		searchDepth = v
	}

	tavilyReq := tavilyRequest{
		Query:       req.Query,
		SearchDepth: searchDepth,
		MaxResults:  maxResults,
	}

	if v := req.Config["include_domains"]; v != "" {
		tavilyReq.IncludeDomains = splitDomains(v)
	}
	if v := req.Config["exclude_domains"]; v != "" {
		tavilyReq.ExcludeDomains = splitDomains(v)
	}

	body, err := json.Marshal(tavilyReq)
	if err != nil {
		writeError("unknown", "failed to marshal request body: "+err.Error())
		os.Exit(1)
	}

	// 4. Execute the HTTP request
	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequest("POST", "https://api.tavily.com/search", bytes.NewReader(body))
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	// 5. Handle HTTP error codes
	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized, http.StatusForbidden:
		writeError("auth_failed", fmt.Sprintf("Tavily API returned %d — check your api_key", resp.StatusCode))
		os.Exit(1)
	case http.StatusTooManyRequests:
		writeError("rate_limited", "Tavily API returned 429 Too Many Requests")
		os.Exit(1)
	case http.StatusBadRequest:
		writeError("invalid_query", fmt.Sprintf("Tavily API returned 400 Bad Request for query: %q", req.Query))
		os.Exit(1)
	default:
		writeError("unavailable", fmt.Sprintf("Tavily API returned unexpected status %d", resp.StatusCode))
		os.Exit(1)
	}

	// 6. Parse the Tavily response
	var tavilyResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		writeError("unknown", "failed to parse Tavily API response: "+err.Error())
		os.Exit(1)
	}

	// 7. Map to qry result format
	results := make([]Result, 0, len(tavilyResp.Results))
	for _, r := range tavilyResp.Results {
		results = append(results, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}

	// 8. Write results to stdout
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		writeError("unknown", "failed to encode results: "+err.Error())
		os.Exit(1)
	}
}

func splitDomains(s string) []string {
	parts := strings.Split(s, ",")
	domains := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			domains = append(domains, p)
		}
	}
	return domains
}
