# qry

A terminal-native, agent-first web search CLI. Routes queries through swappable adapter binaries and always outputs JSON.

```bash
qry "what is the latest version of numpy"
```

```json
[
  {
    "title": "NumPy 2.0 Release Notes",
    "url": "https://numpy.org/doc/stable/release/2.0.0-notes.html",
    "snippet": "NumPy 2.0.0 is the first major release since 2006..."
  }
]
```

## Install

```bash
mise use -g go:github.com/justestif/qry@latest
mise reshim
```

## Adapters

Adapters are separate binaries that do the actual searching. Install the ones you want:

| Adapter | Source | Key required |
|---|---|---|
| `qry-adapter-brave-api` | Brave Search API | ✓ |
| `qry-adapter-brave-scrape` | Brave Search (scraping) | ✗ |
| `qry-adapter-ddg-scrape` | DuckDuckGo Lite (scraping) | ✗ |
| `qry-adapter-exa` | Exa AI (via MCP) | ✗ |

```bash
mise use -g go:github.com/justestif/qry/adapters/ddg-scrape@latest
mise reshim
```

## Configure

Create `~/.config/qry/config.toml`:

```toml
[defaults]
  num     = 10
  timeout = "5s"

[routing]
  mode     = "first"
  pool     = ["ddg-scrape"]
  fallback = ["brave-scrape"]

[adapters.ddg-scrape]
  bin = "~/.local/share/mise/shims/qry-adapter-ddg-scrape"

[adapters.brave-scrape]
  bin = "~/.local/share/mise/shims/qry-adapter-brave-scrape"
```

## Routing modes

- **`first`** — tries adapters in order, returns on first success. Fast, good for most use cases.
- **`merge`** — queries all adapters concurrently, deduplicates by URL, returns combined results.

## More

See [`docs/`](./docs) for full documentation:

- [`docs/architecture.md`](./docs/architecture.md) — how qry works internally
- [`docs/schema.md`](./docs/schema.md) — config and JSON schemas
- [`docs/adapters.md`](./docs/adapters.md) — how to build your own adapter
