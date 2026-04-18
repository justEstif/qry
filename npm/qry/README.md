# qry

**[justestif.github.io/qry](https://justestif.github.io/qry/)**

A terminal-native, agent-first web search CLI. Routes queries through swappable built-in adapters and always outputs JSON.

```bash
qry "what is the latest version of numpy"
```

```json
[
  {
    "title": "NumPy 2.0 Release Notes",
    "url": "https://numpy.org/doc/stable/release/2.0.0-notes.html",
    "snippet": "NumPy 2.0.0 is the first major release since 2006..."
  }
]
```

## Setup & Configuration

See the [main repository](https://github.com/justEstif/qry) for full documentation on:
- How to configure `qry` (`~/.config/qry/config.toml`)
- Available search adapters (Brave, DuckDuckGo, Exa, GitHub, Wikipedia, etc.)
- Using the `qry` agent skill
- "First" vs "Merge" routing modes