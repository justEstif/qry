# Parallel Search adapter

The built-in `parallel` adapter uses [Parallel Search MCP](https://docs.parallel.ai/integrations/mcp/search-mcp)
over Streamable HTTP at `https://search.parallel.ai/mcp`. Anonymous searches use
Fast mode and require no account or API key.

Select it for one invocation:

```bash
qry --adapter parallel --num 5 "latest numpy release"
```

To use it in your configured routing pool, add `"parallel"` to the existing
`routing.pool` list. For example, this pool tries Parallel first, then DuckDuckGo
if Parallel fails:

```toml
[routing]
mode = "first"
pool = ["parallel", "ddg-scrape"]
```

No adapter config block is needed. Existing routing settings are unchanged until
you select Parallel or edit your pool. The adapter sends the query as both the
search objective and a keyword query, then maps result titles, URLs and excerpts
to qry's `title`, `url` and `snippet` fields, with snippets capped at 300 characters. `--num` caps the returned list locally;
the service controls how many results are available and may return fewer.

Anonymous access has lower rate limits. An optional Parallel API key enables
higher limits:

```toml
[adapters.parallel.config]
api_key = "${PARALLEL_API_KEY}"
```

The key is used only when configured in this block. The adapter bounds each search,
including MCP initialization, to 30 seconds and respects cancellation from its
caller. Redirects stay on the original HTTPS origin; cross-origin redirects and
HTTPS-to-HTTP downgrades fail without contacting the redirect destination.
Service, protocol and tool failures return errors so qry can try the next
adapter in a `first` pool or report a warning in `merge` mode. Page fetching is
outside qry's search-result interface.
