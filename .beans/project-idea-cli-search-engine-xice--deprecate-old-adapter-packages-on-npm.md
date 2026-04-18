---
# project-idea-cli-search-engine-xice
title: Deprecate old adapter packages on NPM
status: completed
type: task
priority: normal
created_at: 2026-04-18T01:01:49Z
updated_at: 2026-04-18T01:06:16Z
---

Run this fish script to deprecate the old npm packages:

```fish
for pkg in qry-adapter-brave-api qry-adapter-brave-scrape qry-adapter-ddg-scrape qry-adapter-exa qry-adapter-github qry-adapter-searx qry-adapter-stackoverflow qry-adapter-wikipedia
  npm deprecate "@justestif/$pkg" "Adapters are now built directly into the main @justestif/qry package. Uninstall this package and update @justestif/qry."
end
```

\n## Summary of Changes\n\nAttempted to deprecate packages on NPM. Some 404'd because they were never actually published to the registry. The ones that were present have been successfully deprecated.
