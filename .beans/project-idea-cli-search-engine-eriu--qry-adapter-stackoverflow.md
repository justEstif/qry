---
# project-idea-cli-search-engine-eriu
title: qry-adapter-stackoverflow
status: completed
type: feature
priority: normal
created_at: 2026-03-09T12:02:42Z
updated_at: 2026-03-09T12:10:08Z
parent: project-idea-cli-search-engine-usbw
---

Stack Exchange API adapter. Search Stack Overflow Q&A. The API is well-documented, free (with throttling), returns structured JSON. No key required (optional for higher quota).

## Summary of Changes\n\nCreated `adapters/qry-adapter-stackoverflow/main.go` and `npm/qry-adapter-stackoverflow/`. Uses Stack Exchange `/2.3/search/excerpts` API with gzip decompression and HTML tag stripping. Supports `key`, `site`, `tagged` config.
