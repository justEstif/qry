---
# project-idea-cli-search-engine-bkh8
title: qry-adapter-searx
status: completed
type: feature
priority: normal
created_at: 2026-03-09T12:02:42Z
updated_at: 2026-03-09T12:10:08Z
parent: project-idea-cli-search-engine-usbw
---

SearXNG adapter. Privacy-friendly meta-engine, self-hostable, many public instances. No API key needed. SearXNG has a JSON API (/search?format=json) making this straightforward to implement.

## Summary of Changes\n\nCreated `adapters/qry-adapter-searx/main.go` and `npm/qry-adapter-searx/` (package.json, bin.js, install.js). Queries SearXNG `/search?format=json` endpoint. Supports `instance`, `engines`, `language`, `time_range`, `safesearch` config keys.
