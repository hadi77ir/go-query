package executor

import (
	"context"

	"github.com/hadi77ir/go-query/v2/query"
)

// Executor is the interface that all database-specific executors must implement
type Executor interface {
	// Execute runs the query and stores results in dest (must be a pointer to a slice)
	// cursor is optional and used for pagination (can be empty string for first page)
	// Example: var users []User; executor.Execute(ctx, q, "", &users)
	Execute(ctx context.Context, q *query.Query, cursor string, dest interface{}) (*query.Result, error)

	// Count returns the total number of items that would be returned by the given query
	// This does not apply pagination - it counts all matching items
	// Example: count, err := executor.Count(ctx, q)
	Count(ctx context.Context, q *query.Query) (int64, error)

	// Name returns the name of this executor
	Name() string

	// Close cleans up any resources used by the executor
	Close() error
}
