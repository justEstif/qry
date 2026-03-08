// qry-adapter-exa searches via Exa AI's public MCP endpoint.
// No API key required.
//
// The Exa MCP endpoint (https://mcp.exa.ai/mcp) and JSON-RPC protocol were
// discovered by reading anomalyco/opencode's websearch tool
// (https://github.com/anomalyco/opencode), which is MIT licensed.
// This adapter is an independent implementation.
//
// Optional config:
//   type                — "auto" (default) | "fast" | "deep"
//   livecrawl           — "fallback" (default) | "preferred"
//   context_max_chars   — max chars of page content per result (default: 2000)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"net/http"
	"strconv"
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

// --- Exa MCP JSON-RPC types ---

type mcpRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int        `json:"id"`
	Method  string     `json:"method"`
	Params  mcpParams  `json:"params"`
}

type mcpParams struct {
	Name      string      `json:"name"`
	Arguments mcpArgs     `json:"arguments"`
}

type mcpArgs struct {
	Query              string `json:"query"`
	NumResults         int    `json:"numResults"`
	Type               string `json:"type"`
	Livecrawl          string `json:"livecrawl"`
	ContextMaxCharacters int  `json:"contextMaxCharacters,omitempty"`
}

type mcpResponse struct {
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
}

// --- parsing ---

var (
	reTitle = regexp.MustCompile(`(?m)^Title: (.+)$`)
	reURL   = regexp.MustCompile(`(?m)^URL: (.+)$`)
	reText  = regexp.MustCompile(`(?m)^Text: ([\s\S]+)`)
)

func parseResults(content string, num int) []Result {
	// Split on blank line before "Title: " — each result block starts with Title:
	blocks := strings.Split(strings.TrimSpace(content), "\nTitle: ")
	// Re-add the prefix to all but the first block
	for i := 1; i < len(blocks); i++ {
		blocks[i] = "Title: " + blocks[i]
	}
	results := make([]Result, 0, num)

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
			// Collapse whitespace for a clean snippet
			snippet = strings.Join(strings.Fields(textM[1]), " ")
			if len(snippet) > 300 {
				snippet = snippet[:300]
			}
		}

		results = append(results, Result{
			Title:   title,
			URL:     strings.TrimSpace(urlM[1]),
			Snippet: snippet,
		})
	}

	return results
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

	if req.Query == "" {
		writeError("invalid_query", "query must not be empty")
		os.Exit(1)
	}

	// 2. Read optional config
	searchType := req.Config["type"]
	if searchType == "" {
		searchType = "auto"
	}
	livecrawl := req.Config["livecrawl"]
	if livecrawl == "" {
		livecrawl = "fallback"
	}
	contextMaxChars := 2000
	if v := req.Config["context_max_chars"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			contextMaxChars = n
		}
	}

	num := req.Num
	if num <= 0 {
		num = 8
	}

	// 3. Build JSON-RPC request
	mcpReq := mcpRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: mcpParams{
			Name: "web_search_exa",
			Arguments: mcpArgs{
				Query:                req.Query,
				NumResults:           num,
				Type:                 searchType,
				Livecrawl:            livecrawl,
				ContextMaxCharacters: contextMaxChars,
			},
		},
	}

	body, err := json.Marshal(mcpReq)
	if err != nil {
		writeError("unknown", "failed to marshal request: "+err.Error())
		os.Exit(1)
	}

	// 4. POST to Exa MCP endpoint
	httpReq, err := http.NewRequest("POST", "https://mcp.exa.ai/mcp", bytes.NewReader(body))
	if err != nil {
		writeError("unknown", "failed to build HTTP request: "+err.Error())
		os.Exit(1)
	}
	httpReq.Header.Set("accept", "application/json, text/event-stream")
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError("unavailable", "HTTP request failed: "+err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusTooManyRequests:
		writeError("rate_limited", "Exa MCP returned 429 Too Many Requests")
		os.Exit(1)
	default:
		writeError("unavailable", fmt.Sprintf("Exa MCP returned unexpected status %d", resp.StatusCode))
		os.Exit(1)
	}

	// 5. Parse SSE response — find the data: line with the result
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large responses
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
		results := parseResults(content, num)

		if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
			writeError("unknown", "failed to encode results: "+err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := scanner.Err(); err != nil {
		writeError("unknown", "failed to read response: "+err.Error())
		os.Exit(1)
	}

	writeError("unavailable", "no results in Exa MCP response")
	os.Exit(1)
}
