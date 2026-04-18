---
# project-idea-cli-search-engine-d742
title: Update config loader to drop external binary paths
status: completed
type: feature
priority: normal
created_at: 2026-04-18T00:45:06Z
updated_at: 2026-04-18T00:55:00Z
parent: project-idea-cli-search-engine-irmz
---

Modify internal/config to map configured adapters to internal implementations rather than resolving file paths to separate binaries.

## Summary of Changes\n\nRemoved external binary mapping and validation from the `config` package.
