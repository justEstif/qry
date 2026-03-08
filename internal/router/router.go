package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/justestif/qry/internal/config"
	"github.com/justestif/qry/internal/result"
)

// Router orchestrates adapter invocation based on routing mode.
type Router struct {
	cfg   *config.Config
	query string
}

// New creates a new Router.
func New(cfg *config.Config, query string) *Router {
	return &Router{cfg: cfg, query: query}
}

// Run executes the query using the configured routing mode.
// Returns results for "first" mode or merge output for "merge" mode.
func (r *Router) Run(ctx context.Context) (any, error) {
	switch r.cfg.Routing.Mode {
	case "first":
		return r.runFirst(ctx)
	case "merge":
		return r.runMerge(ctx)
	default:
		return nil, fmt.Errorf("unknown routing mode: %q", r.cfg.Routing.Mode)
	}
}

// runFirst tries pool adapters sequentially, then fallback adapters.
// Returns on first success.
func (r *Router) runFirst(ctx context.Context) (result.FirstOutput, error) {
	chain := append(r.cfg.Routing.Pool, r.cfg.Routing.Fallback...)
	attempts := make([]result.Attempt, 0, len(chain))

	for _, name := range chain {
		adapter, err := r.cfg.ResolvedAdapter(name)
		if err != nil {
			attempts = append(attempts, result.Attempt{Adapter: name, Error: "unknown", Message: err.Error()})
			continue
		}

		results, attempt := invokeAdapter(ctx, name, adapter, r.query)
		if attempt != nil {
			attempts = append(attempts, *attempt)
			continue
		}

		return results, nil
	}

	return nil, &allAdaptersFailedError{
		Mode:     "first",
		Attempts: attempts,
	}
}

// runMerge invokes all pool adapters concurrently and combines results.
// Partial failure is acceptable — returns results from successful adapters with warnings.
func (r *Router) runMerge(ctx context.Context) (result.MergeOutput, error) {
	type adapterResult struct {
		name    string
		results []result.Result
		attempt *result.Attempt
	}

	ch := make(chan adapterResult, len(r.cfg.Routing.Pool))
	var wg sync.WaitGroup

	for _, name := range r.cfg.Routing.Pool {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			adapter, err := r.cfg.ResolvedAdapter(name)
			if err != nil {
				ch <- adapterResult{name: name, attempt: &result.Attempt{
					Adapter: name, Error: "unknown", Message: err.Error(),
				}}
				return
			}
			results, attempt := invokeAdapter(ctx, name, adapter, r.query)
			ch <- adapterResult{name: name, results: results, attempt: attempt}
		}(name)
	}

	wg.Wait()
	close(ch)

	var combined []result.Result
	var warnings []string
	var attempts []result.Attempt

	for ar := range ch {
		if ar.attempt != nil {
			attempts = append(attempts, *ar.attempt)
			warnings = append(warnings, fmt.Sprintf("%s failed: %s — %s", ar.name, ar.attempt.Error, ar.attempt.Message))
		} else {
			combined = append(combined, ar.results...)
		}
	}

	if len(combined) == 0 && len(attempts) > 0 {
		return result.MergeOutput{}, &allAdaptersFailedError{
			Mode:     "merge",
			Attempts: attempts,
		}
	}

	return result.MergeOutput{
		Results:  result.Deduplicate(combined),
		Warnings: warnings,
	}, nil
}

// FailureReporter is implemented by errors that carry a structured failure payload.
// cmd callers use errors.As(err, new(FailureReporter)) instead of type-asserting
// the concrete error type, keeping the router's internals encapsulated.
type FailureReporter interface {
	error
	FailureOutput() result.FailureOutput
}

// allAdaptersFailedError is returned when every adapter in the chain has failed.
type allAdaptersFailedError struct {
	Mode     string
	Attempts []result.Attempt
}

func (e *allAdaptersFailedError) Error() string {
	if len(e.Attempts) == 0 {
		return "all adapters failed"
	}
	last := e.Attempts[len(e.Attempts)-1]
	return fmt.Sprintf("all adapters failed. last error: %s — %s", last.Error, last.Message)
}

// FailureOutput converts the error into the structured stderr payload.
func (e *allAdaptersFailedError) FailureOutput() result.FailureOutput {
	return result.FailureOutput{
		Error:    "all_adapters_failed",
		Message:  e.Error(),
		Attempts: e.Attempts,
	}
}
