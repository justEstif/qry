---
# project-idea-cli-search-engine-tm0p
title: Add --agent-info flag for agent-readable self-description
status: completed
type: feature
priority: normal
created_at: 2026-03-08T15:09:45Z
updated_at: 2026-03-08T15:25:59Z
---

Add two tightly coupled features:

1. **`--agent-info` flag** — outputs structured JSON describing the tool, routing modes, and resolved config so agents can orient before querying
2. **ENV var interpolation in config** — support `${VAR}` in adapter config values so secrets never live in the file; agent-info shows the template strings, not resolved values

## Tasks

- [x] Add `internal/info/info.go` — `AgentInfo` struct + `Build()` function
- [x] Add ENV var expansion in `internal/config/config.go` — `ExpandEnv()` called after viper unmarshal, only on `Adapters[*].Config` map values
- [x] Update `cmd/root.go` — add `--agent-info` / `-A` flag, change `cobra.ExactArgs(1)` to `cobra.RangeArgs(0,1)`, branch in RunE
- [x] Update `docs/schema.md` — document `${ENV_VAR}` syntax in adapter config section
- [x] Update `README.md` — mention `--agent-info` flag and env var support in Configure section

## Summary of Changes

- **`internal/info/info.go`** (new) — `AgentInfo` struct hierarchy and `Build()` function. Captures tool metadata, routing mode descriptions, and per-adapter availability. Shows raw config template strings, not resolved env values.
- **`internal/config/config.go`** — Added `ExpandEnv()` method that expands `${VAR}` in adapter config map values using `os.ExpandEnv`. Only touches `Adapters[*].Config` — paths and timeouts are untouched.
- **`cmd/root.go`** — Added `--agent-info` / `-A` flag. Changed `cobra.ExactArgs(1)` → `cobra.RangeArgs(0,1)`. Snapshots raw adapter configs before `ExpandEnv()` so `--agent-info` shows templates. `ExpandEnv()` is called on both paths (search + agent-info).
- **`docs/schema.md`** — Documented `${VAR}` interpolation syntax under the config section.
- **`README.md`** — Added Agent usage section with `--agent-info` example; added env var note to Configure section.
