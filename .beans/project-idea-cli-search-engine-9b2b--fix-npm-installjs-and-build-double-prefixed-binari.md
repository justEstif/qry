---
# project-idea-cli-search-engine-9b2b
title: 'Fix npm install.js and build: double-prefixed binaries + ENOENT on redirect'
status: completed
type: bug
priority: normal
created_at: 2026-03-09T13:54:14Z
updated_at: 2026-03-09T13:54:19Z
---

Found during npm package testing:

## Bugs
- [x] install.js: fs.unlinkSync fails with ENOENT on GitHub redirect (file not yet created)
- [x] install.js: tar extraction doesn't handle ./ prefix in archives
- [x] release workflow: adapter binaries double-prefixed (qry-adapter-qry-adapter-*)
- [x] package.json versions committed as 4.5.0 instead of placeholder

## Fixes
- Added tryUnlink helper to gracefully handle missing files
- Changed tar extraction to extract all files (not just named binary)
- Removed qry-adapter- prefix from build step (dirs already have it)
- Reset all package.json versions to 0.0.0
