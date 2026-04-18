---
# project-idea-cli-search-engine-irmz
title: Rearchitect adapters to be built-in
status: completed
type: epic
priority: normal
created_at: 2026-04-18T00:44:59Z
updated_at: 2026-04-18T00:59:15Z
---

Refactor qry to include adapters natively instead of relying on external binaries.

## Summary of Changes\n\nSuccessfully refactored qry to use built-in adapters, dropping the reliance on external binaries. This included defining a new  Go interface, converting all 9 adapter executables to packages, refactoring the config and router modules to use the internal registry, updating all docs/schemas, and cleaning up the build/release CI.
