---
# project-idea-cli-search-engine-3jos
title: 'Fix npm publish workflow: unpublish bad 4.5.0 and harden version stamping'
status: completed
type: bug
priority: normal
created_at: 2026-03-09T13:43:20Z
updated_at: 2026-03-09T13:43:34Z
---

Version 4.5.0 was erroneously published to npm, blocking v0.5.0 release. Need to unpublish it and harden the workflow.

## TODO
- [x] Add version sanity check to workflow
- [x] Add explicit --tag latest to npm publish
- [x] Document manual step: unpublish 4.5.0 and re-tag

## Summary of Changes

Fixed `.github/workflows/release.yml`:
1. Added semver validation check before publishing
2. Added `--tag latest` to `npm publish` to force correct tagging even if a higher version exists

## Manual Steps Required
Before re-running the v0.5.0 release:
```bash
npm unpublish @justestif/qry@4.5.0
npm dist-tag add @justestif/qry@0.4.5 latest
```
Then re-trigger the workflow or delete+re-push the v0.5.0 tag.
