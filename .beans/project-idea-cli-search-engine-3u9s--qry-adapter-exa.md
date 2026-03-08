---
# project-idea-cli-search-engine-3u9s
title: qry-adapter-exa
status: completed
type: task
priority: normal
created_at: 2026-03-08T03:28:58Z
updated_at: 2026-03-08T03:30:14Z
parent: project-idea-cli-search-engine-usbw
---

Adapter for Exa AI search via the public MCP endpoint used by opencode. No API key required. POST to https://mcp.exa.ai/mcp with JSON-RPC 2.0, parses SSE response. Output: adapters/exa/main.go

## Summary of Changes\n\nadapters/exa/main.go:\n- Discovered via opencode's websearch.ts — hits https://mcp.exa.ai/mcp, no API key needed\n- JSON-RPC 2.0 POST, method tools/call, tool web_search_exa\n- Parses SSE response (data: lines), extracts first content block\n- Splits text on newline+Title: boundaries to get per-result blocks\n- Extracts Title, URL, Text via regexp per block\n- Collapses whitespace in snippet, truncates to 300 chars\n- Maps 429 → rate_limited\n- Optional config: type (auto/fast/deep), livecrawl (fallback/preferred), context_max_chars
