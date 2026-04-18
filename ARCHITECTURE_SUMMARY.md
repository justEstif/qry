# qry Architecture Summary

`qry` is a CLI-based search hub designed to provide a stable query interface while delegating actual web searches to built-in adapter packages.

## Core Components

1. **CLI Layer**: Built with `cobra` and `viper`, it parses flags, accepts the query, and overrides configuration settings for a single invocation.
2. **Config Loader**: Reads `~/.config/qry/config.toml` on every invocation. It maps adapter configurations and merges CLI flags with TOML values.
3. **Router**: Orchestrates adapter invocations. It selects adapters based on the routing mode and handles their execution via the unified `Adapter` interface.
4. **Adapters**: Built-in Go packages that perform the actual searches. Each adapter registers itself on initialization.

## Routing Modes

- **"first" Mode**: Adapters are tried sequentially in pool order. The first successful result is returned. Fast and ideal for quick lookups.
- **"merge" Mode**: All pool adapters are invoked concurrently. Results are aggregated, deduplicated, and returned along with warnings for any failed adapters. Useful for broader coverage.

## Adapter Interface

Adapters are built-in Go packages that implement a simple interface:

```go
type Adapter interface {
	Name() string
	Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error)
}
```

## Key Design Decisions

- **Built-in Adapters**: Compiling adapters directly into `qry` simplifies distribution via a single static binary.
- **Stateless Configuration**: Read fresh per invocation, avoiding the need for a daemon.
- **JSON Only Output**: Designed for composition with other tools and agents.
- **Partial Failure Handling**: In "merge" mode, partial results are returned rather than failing the entire query.