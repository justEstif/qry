package github

import (
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
	return "github"
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	searchType := config["type"]
	if searchType == "" {
		searchType = "repositories"
	}

	count := num
	if count <= 0 || count > 100 {
		count = 10
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("per_page", strconv.Itoa(count))

	endpoint := fmt.Sprintf("https://api.github.com/search/%s?%s", searchType, params.Encode())

	client := &http.Client{}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("User-Agent", "qry-adapter-github")

	if token := config["token"]; token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("auth_failed: GitHub API returned 401 — check your token")
	case http.StatusForbidden:
		return nil, fmt.Errorf("rate_limited: GitHub API returned 403 — rate limited or forbidden")
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("invalid_query: GitHub API returned 422 for query: %q", query)
	default:
		return nil, fmt.Errorf("unavailable: GitHub API returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to read response body: %w", err)
	}

	var results []result.Result

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
			return nil, fmt.Errorf("unknown: failed to parse GitHub API response: %w", err)
		}
		for _, item := range data.Items {
			results = append(results, result.Result{
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
			return nil, fmt.Errorf("unknown: failed to parse GitHub API response: %w", err)
		}
		for _, item := range data.Items {
			results = append(results, result.Result{
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
			return nil, fmt.Errorf("unknown: failed to parse GitHub API response: %w", err)
		}
		for _, item := range data.Items {
			snippet := item.Body
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			results = append(results, result.Result{
				Title:   item.Title,
				URL:     item.HTMLURL,
				Snippet: snippet,
			})
		}
	}

	if results == nil {
		results = []result.Result{}
	}
	return results, nil
}
