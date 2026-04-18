---
# project-idea-cli-search-engine-xdwd
title: Remove NPM adapter distribution
status: completed
type: task
priority: normal
created_at: 2026-04-18T00:58:31Z
updated_at: 2026-04-18T00:59:09Z
parent: project-idea-cli-search-engine-irmz
---

Delete npm package configurations for adapter binaries since they are now built-in. Update CI/CD release workflow to stop building and publishing adapter binaries to GitHub Releases and NPM.

## Summary of Changes\n\nDeleted all  directories. Updated  to stop building and packaging separate adapter binaries for release and NPM. Now it only publishes the  binary.
