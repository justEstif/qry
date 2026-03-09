// qry-adapter-github searches via the GitHub Search API.
//
// Optional config:
//   token — GitHub personal access token (for higher rate limits)
//   type  — search type: repositories (default), code, issues
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	searchType := req.Config["type"]
	if searchType == "" {
		searchType = "repositories"
	}

	count := req.Num
	if count <= 0 || count > 100 {
		count = 10
	}

	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("per_page", strconv.Itoa(count))

	endpoint := fmt.Sprintf("https://api.github.com/search/%s?%s", searchType, params.Encode())

	client := &http.Client{Timeout: 15 * time.Second}
	httpReq, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("User-Agent", "qry-adapter-github")

	if token := req.Config["token"]; token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized:
		writeError("auth_failed", "GitHub API returned 401 — check your token")
		os.Exit(1)
	case http.StatusForbidden:
		writeError("rate_limited", "GitHub API returned 403 — rate limited or forbidden")
		os.Exit(1)
	case http.StatusUnprocessableEntity:
		writeError("invalid_query", fmt.Sprintf("GitHub API returned 422 for query: %q", req.Query))
		os.Exit(1)
	default:
		writeError("unavailable", fmt.Sprintf("GitHub API returned unexpected status %d", resp.StatusCode))
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError("unknown", "failed to read response body: "+err.Error())
		os.Exit(1)
	}

	var results []Result

	switch searchType {
	case "repositories":
		var data struct {
			Items []struct {
				FullName    string `json:"full_name"`
				HTMLURL     string `json:"html_url"`
				Description string `json:"description"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			writeError("unknown", "failed to parse GitHub API response: "+err.Error())
			os.Exit(1)
		}
		for _, item := range data.Items {
			results = append(results, Result{
				Title:   item.FullName,
				URL:     item.HTMLURL,
				Snippet: item.Description,
			})
		}

	case "code":
		var data struct {
			Items []struct {
				Name    string `json:"name"`
				HTMLURL string `json:"html_url"`
				Repo    struct {
					FullName    string `json:"full_name"`
					Description string `json:"description"`
				} `json:"repository"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			writeError("unknown", "failed to parse GitHub API response: "+err.Error())
			os.Exit(1)
		}
		for _, item := range data.Items {
			results = append(results, Result{
				Title:   item.Repo.FullName + "/" + item.Name,
				URL:     item.HTMLURL,
				Snippet: item.Repo.Description,
			})
		}

	case "issues":
		var data struct {
			Items []struct {
				Title   string `json:"title"`
				HTMLURL string `json:"html_url"`
				Body    string `json:"body"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			writeError("unknown", "failed to parse GitHub API response: "+err.Error())
			os.Exit(1)
		}
		for _, item := range data.Items {
			snippet := item.Body
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			results = append(results, Result{
				Title:   item.Title,
				URL:     item.HTMLURL,
				Snippet: snippet,
			})
		}
	}

	if results == nil {
		results = []Result{}
	}

	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		writeError("unknown", "failed to encode results: "+err.Error())
		os.Exit(1)
	}
}
