package router

import (
	"context"

	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/config"
	"github.com/justestif/qry/internal/result"
)

// invokeAdapter runs a single adapter by fetching it from the registry
// and calling its Search method.
func invokeAdapter(ctx context.Context, name string, adapterCfg config.Adapter, query string) ([]result.Result, *result.Attempt) {
	adp := adapter.Get(name)
	if adp == nil {
		return nil, &result.Attempt{
			Adapter: name,
			Error:   "unknown",
			Message: "adapter not found in registry",
		}
	}

	results, err := adp.Search(ctx, query, adapterCfg.Num, adapterCfg.Config)
	if err != nil {
		return nil, &result.Attempt{
			Adapter: name,
			Error:   "search_failed",
			Message: err.Error(),
		}
	}

	return results, nil
}
