---
# project-idea-cli-search-engine-drh6
title: Convert existing adapters to internal packages
status: completed
type: feature
priority: normal
created_at: 2026-04-18T00:45:11Z
updated_at: 2026-04-18T00:54:23Z
parent: project-idea-cli-search-engine-irmz
---

Move code from standalone adapter binaries into an adapters/ package and implement a standard Go interface.

## Summary of Changes\n\nRefactored all standalone adapters into internal Go packages using the new registry and  interface. Removed all executable  entrypoints.
