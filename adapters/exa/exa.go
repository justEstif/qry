package exa

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	return "exa"
}

type mcpRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int        `json:"id"`
	Method  string     `json:"method"`
	Params  mcpParams  `json:"params"`
}

type mcpParams struct {
	Name      string  `json:"name"`
	Arguments mcpArgs `json:"arguments"`
}

type mcpArgs struct {
	Query                string `json:"query"`
	NumResults           int    `json:"numResults"`
	Type                 string `json:"type"`
	Livecrawl            string `json:"livecrawl"`
	ContextMaxCharacters int    `json:"contextMaxCharacters,omitempty"`
}

type mcpResponse struct {
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
}

var (
	reTitle = regexp.MustCompile(`(?m)^Title: (.+)$`)
	reURL   = regexp.MustCompile(`(?m)^URL: (.+)$`)
	reText  = regexp.MustCompile(`(?m)^Text: ([\s\S]+)`)
)

func parseResults(content string, num int) []result.Result {
	blocks := strings.Split(strings.TrimSpace(content), "\nTitle: ")
	for i := 1; i < len(blocks); i++ {
		blocks[i] = "Title: " + blocks[i]
	}
	results := make([]result.Result, 0, num)

	for _, b := range blocks {
		if len(results) >= num {
			break
		}
		titleM := reTitle.FindStringSubmatch(b)
		urlM := reURL.FindStringSubmatch(b)
		textM := reText.FindStringSubmatch(b)

		if urlM == nil {
			continue
		}

		title := ""
		if titleM != nil {
			title = strings.TrimSpace(titleM[1])
		}

		snippet := ""
		if textM != nil {
			snippet = strings.Join(strings.Fields(textM[1]), " ")
			if len(snippet) > 300 {
				snippet = snippet[:300]
			}
		}

		results = append(results, result.Result{
			Title:   title,
			URL:     strings.TrimSpace(urlM[1]),
			Snippet: snippet,
		})
	}
	return results
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	if query == "" {
		return nil, fmt.Errorf("invalid_query: query must not be empty")
	}

	searchType := config["type"]
	if searchType == "" {
		searchType = "auto"
	}
	livecrawl := config["livecrawl"]
	if livecrawl == "" {
		livecrawl = "fallback"
	}
	contextMaxChars := 2000
	if v := config["context_max_chars"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			contextMaxChars = n
		}
	}

	if num <= 0 {
		num = 8
	}

	mcpReq := mcpRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: mcpParams{
			Name: "web_search_exa",
			Arguments: mcpArgs{
				Query:                query,
				NumResults:           num,
				Type:                 searchType,
				Livecrawl:            livecrawl,
				ContextMaxCharacters: contextMaxChars,
			},
		},
	}

	body, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://mcp.exa.ai/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("unknown: failed to build HTTP request: %w", err)
	}
	httpReq.Header.Set("accept", "application/json, text/event-stream")
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unavailable: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate_limited: Exa MCP returned 429 Too Many Requests")
	default:
		return nil, fmt.Errorf("unavailable: Exa MCP returned unexpected status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var mcpResp mcpResponse
		if err := json.Unmarshal([]byte(line[6:]), &mcpResp); err != nil {
			continue
		}

		if len(mcpResp.Result.Content) == 0 {
			break
		}

		content := mcpResp.Result.Content[0].Text
		return parseResults(content, num), nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unknown: failed to read response: %w", err)
	}

	return nil, fmt.Errorf("unavailable: no results in Exa MCP response")
}
