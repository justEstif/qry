---
# project-idea-cli-search-engine-21c2
title: Restore missing built-in adapters
status: completed
type: bug
priority: normal
created_at: 2026-04-18T02:09:20Z
updated_at: 2026-04-18T02:13:39Z
---

Commit 4615ae2 deleted the standalone adapter binaries but failed to commit the new internal Go package implementations. Need to extract the old code from git history, refactor to the new Adapter interface, and register them.

## Summary of Changes
- Extracted old adapter source from git history before 4615ae2
- Converted ddg-scrape to internal adapter structure and registered it
- Added missing golang.org/x/net/html dependency
- Recompiled and verified ddg-scrape adapter loads and executes successfully

## Summary of Changes
- Refactored all 9 missing adapters (braveapi, bravescrape, ddgscrape, exa, github, searx, stackoverflow, wikipedia) to implement the internal `Adapter` interface
- Registered all adapters in `main.go` via anonymous imports
- Built and verified adapters are working correctly internally
