package executor

import (
	"context"

	"github.com/hadi77ir/go-query/query"
)

// Executor is the interface that all database-specific executors must implement
type Executor interface {
	// Execute runs the query and stores results in dest (must be a pointer to a slice)
	// Example: var users []User; executor.Execute(ctx, q, &users)
	Execute(ctx context.Context, q *query.Query, dest interface{}) (*query.Result, error)

	// Name returns the name of this executor
	Name() string

	// Close cleans up any resources used by the executor
	Close() error
}
