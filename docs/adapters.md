# qry — Adapter Contract

An adapter is a Go package that implements the `Adapter` interface. This document defines
what an adapter is, how `qry` invokes it, what it must do, and how to build one.

For the full JSON schemas referenced here, see [schema.md](./schema.md).

---

## What is an adapter?

An adapter is a Go package that implements the `Adapter` interface:

```go
type Adapter interface {
	Search(ctx context.Context, query string, num int, cfg map[string]string) ([]Result, error)
}
```

That's the entire contract. `qry` knows nothing about how the adapter works internally.
The adapter owns its backend, its credentials, and its logic. `qry` owns the orchestration.

---

## Lifecycle

```
qry invokes adapter.Search()
        │
        ▼
adapter executes search
        │
   ┌────┴────┐
success    failure
   │            │
   ▼            ▼
return []Result return error
```

`qry` enforces a **timeout** per adapter using `context.WithTimeout`. If the adapter does not
return within the timeout, `context` cancels and `qry` treats it as a `timeout` failure.

---

## Protocol

### Invocation

`qry` invokes the adapter by calling its `Search` method:

```go
results, err := adapter.Search(ctx, req.Query, req.Num, req.Config)
```

### Output — success

Return a slice of `Result` objects and a `nil` error:

```go
[]Result{
  {
    Title:   "NumPy 2.0 Release Notes",
    URL:     "https://numpy.org/doc/stable/release/2.0.0-notes.html",
    Snippet: "NumPy 2.0.0 is the first major release since 2006...",
  },
}
```

- An empty slice `[]` is valid — it means the search returned no results, not that something failed
- Results should be ordered by relevance (most relevant first)
- Return at most `num` results

### Output — failure

Return a non-nil error:

```go
return nil, fmt.Errorf("rate_limited: 429 Too Many Requests from Brave API")
```

- `qry` will handle fallback routing based on the error code

---

## Building a Real Adapter

### Step 1 — Implement the Interface

```go
package googleapi

import "context"

type Adapter struct{}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]Result, error) {
    // ...
}
```

### Step 2 — Extract config

Your adapter's config block from `config.toml` is passed verbatim as the `config` map.
Pull what you need:

```go
apiKey := config["api_key"]
if apiKey == "" {
    return nil, fmt.Errorf("auth_failed: api_key is required but not set in config")
}
```

### Step 3 — Execute the search

Call your backend. Handle errors by mapping them to standard error codes:

```go
results, err := callGoogleAPI(ctx, apiKey, query, num)
if err != nil {
    if isRateLimit(err) {
        return nil, fmt.Errorf("rate_limited: %v", err)
    }
    return nil, fmt.Errorf("unknown: %v", err)
}
```

### Step 4 — Return results

```go
return results, nil
```

---

## Checklist for adapter authors

- [ ] Implements `Adapter` interface
- [ ] Validates required config fields and returns an error if missing
- [ ] Respects `ctx` context cancellation / timeout
- [ ] Returns a slice of `Result` on success
- [ ] Returns an error on failure with appropriate error code string
- [ ] Tested with mock data before wiring to real backend
