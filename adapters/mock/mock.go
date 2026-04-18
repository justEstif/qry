package mock

import (
	"context"
	"fmt"
	"github.com/justestif/qry/internal/adapter"
	"github.com/justestif/qry/internal/result"
)

type Adapter struct{}

func init() {
	adapter.Register(&Adapter{})
}

func (a *Adapter) Name() string {
	return "mock"
}

func (a *Adapter) Search(ctx context.Context, query string, num int, config map[string]string) ([]result.Result, error) {
	results := []result.Result{
		{
			Title:   fmt.Sprintf("Mock Result 1 for: %s", query),
			URL:     "https://example.com/result-1",
			Snippet: "This is a mock search result for testing qry.",
		},
		{
			Title:   fmt.Sprintf("Mock Result 2 for: %s", query),
			URL:     "https://example.com/result-2",
			Snippet: "Another mock result to validate the adapter contract.",
		},
	}

	if num > 0 && num < len(results) {
		results = results[:num]
	}

	return results, nil
}
