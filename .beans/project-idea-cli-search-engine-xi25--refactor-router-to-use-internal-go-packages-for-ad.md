---
# project-idea-cli-search-engine-xi25
title: Refactor router to use internal Go packages for adapters
status: completed
type: feature
priority: normal
created_at: 2026-04-18T00:45:01Z
updated_at: 2026-04-18T00:57:05Z
parent: project-idea-cli-search-engine-irmz
---

Update the router component in internal/router to invoke Go functions from adapters/ instead of executing external binaries.

## Summary of Changes\n\nRefactored the `invokeAdapter` function to fetch implementations from the internal registry and invoke them natively rather than spawning subprocesses.

## Summary of Changes\n\nFixed compiler errors from refactoring adapters. Updated  and  to use the new  interface and removed references to the  field which was dropped.
