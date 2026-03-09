---
# project-idea-cli-search-engine-1tt1
title: qry-adapter-github
status: completed
type: feature
priority: normal
created_at: 2026-03-09T12:02:42Z
updated_at: 2026-03-09T12:10:08Z
parent: project-idea-cli-search-engine-usbw
---

GitHub search adapter using the GitHub REST/GraphQL API. Search repos, code, issues, discussions. Extremely useful for developer agents. Uses GITHUB_TOKEN for auth (optional but recommended for rate limits).

## Summary of Changes\n\nCreated `adapters/qry-adapter-github/main.go` and `npm/qry-adapter-github/`. Supports `repositories` (default), `code`, and `issues` search types. Optional `token` for auth.
