---
# project-idea-cli-search-engine-10x8
title: Add --site flag for site-scoped search
status: completed
type: feature
priority: normal
created_at: 2026-03-09T12:02:43Z
updated_at: 2026-03-09T12:19:22Z
---

Add a --site <domain> flag (repeatable) to qry that prepends site-scope operators to the query before passing it to the adapter. This replaces the need for dozens of site-specific adapters (mdn, rustdocs, pgdoc, etc.) with a single composable flag.

Supports multiple sites via repeated flags — joined with OR so results come from any of the listed domains.

Examples:
- qry --site docs.rs 'tokio runtime'
- qry --site developer.mozilla.org 'fetch API'
- qry --site man7.org 'epoll'
- qry --site docs.rs --site crates.io 'serde'
  → query becomes: 'site:docs.rs OR site:crates.io serde'

Implementation: cobra StringArrayVar for --site, join as 'site:a OR site:b' prefix in cmd/root.go before passing to the router. Simple and adapter-agnostic.

## Summary of Changes\n\nAdded `--site` flag (repeatable `StringArrayVar`) in `cmd/root.go`. Multiple sites joined with OR. Added flag to `--agent-info` output in `internal/info/info.go`. Tested with single and multiple sites.
