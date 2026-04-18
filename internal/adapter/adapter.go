package adapter

import (
	"context"

	"github.com/justestif/qry/internal/result"
)

// Adapter defines the interface for a search provider.
type Adapter interface {
	// Name returns the identifier of the adapter.
	Name() string

	// Search executes a query against the provider.
	Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error)
}
