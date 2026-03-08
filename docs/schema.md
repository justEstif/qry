# qry — Schema Reference

All data structures used by `qry` and its adapters. This is the source of truth for the
config file format, the adapter communication protocol, and all JSON shapes.

---

## 1. Config File — `~/.config/qry/config.toml`

The config file controls the core runtime behavior and registers adapters.

```toml
# Global defaults — applied to all adapters unless overridden per-adapter
[defaults]
  num     = 10       # Number of results to return
  timeout = "5s"     # How long to wait for an adapter before treating it as failed

# Routing controls how qry selects and combines adapters
[routing]
  # mode controls how qry uses the pool:
  #   "first"  — try pool adapters in order, stop at first success. use fallback on failure.
  #   "merge"  — query all pool adapters concurrently, deduplicate and combine results.
  #              on partial failure, returns results from successful adapters + warnings.
  mode = "first"

  # pool is the set of adapters actively used for queries.
  # in "first" mode: tried in order, first success wins.
  # in "merge" mode: all queried concurrently, results combined.
  pool = ["brave-api", "google-api"]

  # fallback is only used in "first" mode.
  # tried in order if all pool adapters fail.
  # ignored in "merge" mode.
  fallback = ["ddg-scrape"]

# Adapter declarations — one [adapters.<name>] block per adapter
[adapters.brave-api]
  bin     = "/usr/local/bin/qry-adapter-brave-api"  # Path to the adapter binary
  timeout = "5s"                                     # Overrides [defaults].timeout
  num     = 10                                       # Overrides [defaults].num

  # Adapter-specific config — passed through to the adapter via the request envelope
  [adapters.brave-api.config]
    api_key = "YOUR_BRAVE_API_KEY"

[adapters.google-api]
  bin     = "/usr/local/bin/qry-adapter-google-api"
  timeout = "8s"

  [adapters.google-api.config]
    api_key = "YOUR_GOOGLE_API_KEY"
    cx      = "YOUR_CUSTOM_SEARCH_ENGINE_ID"

[adapters.ddg-scrape]
  bin = "/usr/local/bin/qry-adapter-ddg-scrape"
  # No config block — this adapter needs no credentials
```

### ENV variable interpolation

Adapter config values support `${VAR}` syntax. When qry loads the config, it expands
these references from the environment — so secrets never need to live in the file itself:

```toml
[adapters.brave-api.config]
  api_key = "${BRAVE_API_KEY}"
```

`qry --agent-info` shows the **template string** (`"${BRAVE_API_KEY}"`), not the resolved
value — so agents can see which env vars are required without exposing secrets.

### Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `defaults.num` | int | No | Default result count (default: 10) |
| `defaults.timeout` | duration | No | Default adapter timeout (default: `"5s"`) |
| `routing.mode` | string | Yes | `"first"` or `"merge"` — controls how pool adapters are used |
| `routing.pool` | []string | Yes | Adapters actively used for queries |
| `routing.fallback` | []string | No | Adapters tried in order if all pool adapters fail (`"first"` mode only) |
| `adapters.<name>.bin` | string | Yes | Absolute path to the adapter binary |
| `adapters.<name>.timeout` | duration | No | Per-adapter timeout, overrides `defaults.timeout` |
| `adapters.<name>.num` | int | No | Per-adapter result count, overrides `defaults.num` |
| `adapters.<name>.config` | map | No | Arbitrary adapter-specific config (API keys, etc.) |

### Duration format

Durations follow Go's `time.ParseDuration` format: `"5s"`, `"500ms"`, `"1m30s"`.

---

## 2. Adapter Request — stdin

When `qry` invokes an adapter binary, it writes a single JSON object to the adapter's stdin.
The adapter must read this, execute the search, and write results to stdout.

> In `"merge"` mode, `qry` invokes multiple adapters concurrently — each receives the same
> request envelope independently.

```json
{
  "query":  "what is the latest version of numpy",
  "num":    10,
  "config": {
    "api_key": "YOUR_KEY"
  }
}
```

### Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `query` | string | Yes | The raw search query string |
| `num` | int | Yes | Maximum number of results to return |
| `config` | object | No | The contents of `adapters.<name>.config` from config.toml, passed through verbatim |

---

## 3. Adapter Response — stdout

On success the adapter writes a JSON array to stdout. Each element is a search result.

```json
[
  {
    "title":   "NumPy 2.0 Release Notes",
    "url":     "https://numpy.org/doc/stable/release/2.0.0-notes.html",
    "snippet": "NumPy 2.0.0 is the first major release since 2006..."
  },
  {
    "title":   "NumPy on PyPI",
    "url":     "https://pypi.org/project/numpy/",
    "snippet": "Fundamental package for array computing in Python."
  }
]
```

### Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | Yes | Page title of the result |
| `url` | string | Yes | Full URL of the result |
| `snippet` | string | Yes | Short excerpt or description from the result |

### Rules

- The array may be empty (`[]`) if the search returned no results — this is **not** an error
- Results should be ordered by relevance (most relevant first)
- `num` is a maximum — returning fewer results is acceptable
- The adapter must write **only** the JSON array to stdout — no extra text, no logging

---

## 4. Adapter Error — stderr + exit code

When an adapter fails it must:
1. Exit with a **non-zero exit code**
2. Write a JSON error object to **stderr**

```json
{
  "error":   "rate_limited",
  "message": "429 Too Many Requests from Brave API"
}
```

### Field Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `error` | string | Yes | Machine-readable error code (see table below) |
| `message` | string | Yes | Human-readable description of what went wrong |

### Standard Error Codes

| Code | Meaning | qry behavior |
|---|---|---|
| `rate_limited` | Upstream returned 429 or equivalent | Try next fallback adapter |
| `auth_failed` | API key missing, invalid, or expired | Try next fallback adapter |
| `no_results` | Query returned zero results (use `[]` on stdout instead) | N/A — use empty array |
| `timeout` | Adapter exceeded its own internal timeout | Try next fallback adapter |
| `unavailable` | Upstream service unreachable or returned 5xx | Try next fallback adapter |
| `invalid_query` | Query was malformed or rejected by upstream | **Do not** retry — propagate error |
| `unknown` | Any other error | Try next fallback adapter |

### Rules

- Adapters **must not** write anything to stdout on error
- Adapters **must not** write anything to stderr on success
- `qry` treats any non-zero exit as failure, regardless of stderr content
- If stderr is not valid JSON, `qry` will still handle the failure but cannot surface a clean error message

---

## 5. qry Final Output — stdout to caller

What `qry` writes to its own stdout. This is what agents and humans receive.

### "first" mode — success

`qry` passes the adapter response through without modification:

```json
[
  {
    "title":   "NumPy 2.0 Release Notes",
    "url":     "https://numpy.org/doc/stable/release/2.0.0-notes.html",
    "snippet": "NumPy 2.0.0 is the first major release since 2006..."
  }
]
```

### "merge" mode — full success

All pool adapters succeeded. Results are deduplicated by URL and combined:

```json
{
  "results": [
    {
      "title":   "NumPy 2.0 Release Notes",
      "url":     "https://numpy.org/doc/stable/release/2.0.0-notes.html",
      "snippet": "NumPy 2.0.0 is the first major release since 2006..."
    }
  ]
}
```

### "merge" mode — partial failure (MVP behavior)

One or more pool adapters failed but at least one succeeded. Results from successful adapters
are returned alongside warnings. `qry` exits 0 — partial results are not a failure.

```json
{
  "results": [
    {
      "title":   "NumPy 2.0 Release Notes",
      "url":     "https://numpy.org/doc/stable/release/2.0.0-notes.html",
      "snippet": "NumPy 2.0.0 is the first major release since 2006..."
    }
  ],
  "warnings": [
    "brave-api failed: rate_limited — results may be incomplete"
  ]
}
```

> **Future:** a `merge_require_all = true` config option could make partial failure a hard error.

### Deduplication rules ("merge" mode)

- Results are deduplicated by exact URL match
- Where two adapters return the same URL, the result from the higher-priority pool adapter is kept
- Pool order in config determines priority

### On total failure

**"first" mode** — all pool adapters and all fallback adapters failed:

```json
{
  "error": "all_adapters_failed",
  "message": "All adapters failed. Last error: rate_limited — 429 Too Many Requests",
  "attempts": [
    { "adapter": "brave-api",  "error": "rate_limited", "message": "429 Too Many Requests" },
    { "adapter": "google-api", "error": "auth_failed",  "message": "Invalid API key" },
    { "adapter": "ddg-scrape", "error": "unavailable",  "message": "Connection refused" }
  ]
}
```

**"merge" mode** — all pool adapters failed (no results at all):

```json
{
  "error": "all_adapters_failed",
  "message": "All pool adapters failed. No results returned.",
  "attempts": [
    { "adapter": "brave-api",  "error": "rate_limited", "message": "429 Too Many Requests" },
    { "adapter": "google-api", "error": "auth_failed",  "message": "Invalid API key" }
  ]
}
```

In both cases `qry` exits non-zero and writes the error to stderr.
