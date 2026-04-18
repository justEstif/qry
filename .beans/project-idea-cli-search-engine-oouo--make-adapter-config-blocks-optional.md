---
# project-idea-cli-search-engine-oouo
title: Make adapter config blocks optional
status: completed
type: task
priority: normal
created_at: 2026-04-18T01:34:29Z
updated_at: 2026-04-18T01:47:51Z
---

Remove the strict requirement for [adapters.<name>] blocks in config.toml. Adapters listed in routing.pool or routing.fallback should be automatically instantiated with default settings if they are registered built-in adapters.

## Summary of Changes
- Removed strict requirement for [adapters.<name>] blocks in config.toml
- Refactored config struct and info payload to omit the 'fallback' slice
- Simplified routing system to use a single 'pool' array
- Updated local test environment and README config examples
