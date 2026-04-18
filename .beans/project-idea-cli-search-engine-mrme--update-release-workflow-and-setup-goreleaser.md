---
# project-idea-cli-search-engine-mrme
title: Update release workflow and setup GoReleaser
status: completed
type: task
priority: normal
created_at: 2026-04-18T01:15:06Z
updated_at: 2026-04-18T01:15:25Z
---

Update release workflow to use GoReleaser and setup homebrew tap

## Summary of Changes
- Replaced manual go build matrices in .github/workflows/release.yml with goreleaser action
- Added .goreleaser.yml to generate archives, checksums, and update homebrew tap
- Refactored NPM publishing job to depend on goreleaser release and use OIDC provenance
- Pointed the homebrew tap creation directly at justEstif/homebrew-tap
