---
# project-idea-cli-search-engine-ygej
title: Apply design review fixes
status: completed
type: task
priority: normal
created_at: 2026-03-08T15:45:27Z
updated_at: 2026-03-08T15:46:30Z
---

Three improvements from design review:
- [x] Fix --timeout flag (registered but never applied)
- [x] Consolidate defaults into Config.ApplyDefaults()
- [x] Decouple cmd from AllAdaptersFailedError concrete type via interface

## Summary of Changes\n\n- **config.go**: Added `ApplyDefaults()` — centralizes all default values (mode, num, timeout) in one place, removes scattered hardcoded fallbacks from the cmd layer.\n- **router.go**: Renamed `AllAdaptersFailedError` → `allAdaptersFailedError` (unexported). Added exported `FailureReporter` interface so cmd can use `errors.As` without importing the concrete type.\n- **cmd/root.go**: Fixed `--timeout` flag (now parsed and applied to `cfg.Defaults.Timeout`). Replaced hardcoded defaults with `cfg.ApplyDefaults()`. Replaced type assertion with `errors.As(err, &failed)`.
