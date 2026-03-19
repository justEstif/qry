# qry

**[justestif.github.io/qry](https://justestif.github.io/qry/)**

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

**npm (recommended):**

```bash
npm install -g @justestif/qry
```

**mise:**

```bash
mise cache clear
mise use -g go:github.com/justestif/qry@latest
mise reshim
```

## Adapters

Adapters are separate binaries that do the actual searching. Install the ones you want:

| Adapter                         | Source                          | Key required |
| ------------------------------- | ------------------------------- | ------------ |
| `qry-adapter-brave-api`         | Brave Search API                | ✓            |
| `qry-adapter-brave-scrape`      | Brave Search (scraping)         | ✗            |
| `qry-adapter-ddg-scrape`        | DuckDuckGo Lite (scraping)      | ✗            |
| `qry-adapter-exa`               | Exa AI (via MCP)                | ✗            |
| `qry-adapter-github`            | GitHub Search API               | ✗ (optional) |
| `qry-adapter-searx`             | SearXNG (self-hostable)         | ✗            |
| `qry-adapter-stackoverflow`     | Stack Exchange API              | ✗ (optional) |
| `qry-adapter-wikipedia`         | Wikipedia / MediaWiki API       | ✗            |

**npm:**

```bash
npm install -g @justestif/qry-adapter-ddg-scrape
```

**mise:**

```bash
mise use -g go:github.com/justestif/qry/adapters/qry-adapter-ddg-scrape@latest
mise reshim
```

## Configure

Create `~/.config/qry/config.toml`:

Use `${VAR}` syntax in adapter config values — qry expands them from the environment
at runtime so secrets never live in the file:

```toml
[adapters.brave-api.config]
  api_key = "${BRAVE_API_KEY}"
```

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

## Agent usage

An **agent skill** is available for one-line install into any supported agent:

```bash
npx skills add justestif/qry -g -y
```

Browse skills at [skills.sh](https://skills.sh).

---

Run `qry --agent-info` (or `-A`) to get a JSON description of the tool and your current
configuration — useful for agents to orient themselves before making search calls:

```bash
qry --agent-info
```

The output includes the tool description, available flags, routing mode explanations,
and each configured adapter with its binary path and availability status. Adapter config
maps show `${VAR}` template strings rather than resolved values, so secrets are never exposed.

## Routing modes

- **`first`** — tries adapters in order, returns on first success. Fast, good for most use cases.
- **`merge`** — queries all adapters concurrently, deduplicates by URL, returns combined results.

## More

See [`docs/`](./docs) for full documentation:

- [`docs/architecture.md`](./docs/architecture.md) — how qry works internally
- [`docs/schema.md`](./docs/schema.md) — config and JSON schemas
- [`docs/adapters.md`](./docs/adapters.md) — how to build your own adapter
