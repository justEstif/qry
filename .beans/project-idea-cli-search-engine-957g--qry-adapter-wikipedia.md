---
# project-idea-cli-search-engine-957g
title: qry-adapter-wikipedia
status: completed
type: feature
priority: normal
created_at: 2026-03-09T12:02:43Z
updated_at: 2026-03-09T12:10:08Z
parent: project-idea-cli-search-engine-usbw
---

Wikipedia adapter using the MediaWiki Action API (action=query&list=search). Returns structured JSON natively, no scraping needed, no API key. Great for quick factual lookups.

## Summary of Changes\n\nCreated `adapters/qry-adapter-wikipedia/main.go` and `npm/qry-adapter-wikipedia/`. Uses MediaWiki Action API. Supports `language` config for non-English wikis.
