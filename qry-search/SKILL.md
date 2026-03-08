---
name: qry-search
description: Web search using qry, a terminal-native agent-first search CLI that routes queries through pluggable adapters and always outputs JSON. Use this skill whenever you need to search the web for documentation, APIs, error messages, package info, changelogs, or any current information. Triggers on "search the web", "look up", "find docs for", "what's the latest version of", "check if X exists", or whenever current information is needed. Prefer this over ddgr or other search tools when qry is available.
---

# qry Search Skill

`qry` routes search queries through pluggable adapter binaries and always outputs JSON.

## Check availability

```bash
qry --agent-info   # prints tool description + your current config as JSON
```

If `qry` is not found, install it and at least one adapter:

```bash
# Install qry
mise use -g go:github.com/justestif/qry@latest && mise reshim

# Install an adapter (no API key required)
mise use -g go:github.com/justestif/qry/adapters/ddg-scrape@latest && mise reshim
```

See the [qry README](https://github.com/justestif/qry) for all available adapters and config.

## Core usage

```bash
# Basic search
qry "your query here"

# Limit results (default from config, usually 5–10)
qry --num 5 "your query"

# Force a specific adapter (bypass routing)
qry --adapter ddg-scrape "your query"

# Merge results from all pool adapters
qry --mode merge "your query"
```

## Output format

**`first` mode** (default) — array of results:
```json
[
  { "title": "...", "url": "https://...", "snippet": "..." }
]
```

**`merge` mode** — object with `results` (and optional `warnings`):
```json
{
  "results": [{ "title": "...", "url": "https://...", "snippet": "..." }],
  "warnings": ["brave-api failed: rate_limited — results may be incomplete"]
}
```

## Tips

- **Be specific** — `"python requests post json body example"` beats `"python http"`
- **Version lookups** — `"numpy latest release site:pypi.org"`
- **Error messages** — wrap in quotes: `qry '"ModuleNotFoundError: No module named X"'`
- **Docs** — `"site:docs.python.org pathlib"` gives cleaner results
- **After searching** — fetch the most relevant URL for full content: `curl -s <url> | cat`
- **Partial failures in merge mode** are non-fatal — results from successful adapters are returned alongside warnings
