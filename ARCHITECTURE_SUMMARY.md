# qry Architecture Summary

`qry` is a CLI-based search hub designed to provide a stable query interface while delegating actual web searches to pluggable external binaries called adapters.

## Core Components

1. **CLI Layer**: Built with `cobra` and `viper`, it parses flags, accepts the query, and overrides configuration settings for a single invocation.
2. **Config Loader**: Reads `~/.config/qry/config.toml` on every invocation (no daemon or caching). It resolves adapter paths and merges CLI flags with config values.
3. **Router**: Orchestrates adapter invocations. It selects adapters based on the routing mode and handles their execution as subprocesses.
4. **Adapters**: External, standalone executables that perform the actual searches. They communicate with the `qry` core via stdin/stdout using JSON.

## Routing Modes

- **"first" Mode**: Adapters are tried sequentially (pool order, then fallback). The first successful result is returned. Fast and ideal for quick lookups.
- **"merge" Mode**: All pool adapters are invoked concurrently. Results are aggregated, deduplicated, and returned along with warnings for any failed adapters. Useful for broader coverage.

## Adapter Contract

Adapters are entirely decoupled from `qry`. They can be written in any language as long as they adhere to the protocol:

- **Input**: `qry` writes a JSON request (query, number of results, and config) to the adapter's stdin.
- **Output (Success)**: The adapter writes a JSON array of results to stdout and exits with code 0.
- **Output (Failure)**: The adapter writes a JSON error object to stderr and exits with a non-zero code.
- **Lifecycle**: Adapters are stateless, invoked as subprocesses, and subject to a timeout managed by `qry`.

## Key Design Decisions

- **Subprocess Adapters**: Provides language-agnostic extensibility.
- **Stateless Configuration**: Read fresh per invocation, avoiding the need for a daemon.
- **JSON Only Output**: Designed for composition with other tools and agents.
- **Partial Failure Handling**: In "merge" mode, partial results are returned rather than failing the entire query.