package wrapper

import (
	"context"
	"fmt"

	"github.com/hadi77ir/go-query/v2/executor"
	"github.com/hadi77ir/go-query/v2/query"
)

// WrapperExecutor wraps another executor and imposes additional field restrictions
// Fields must be in BOTH the wrapper's allowed fields list AND the inner executor's allowed fields list
type WrapperExecutor struct {
	innerExecutor executor.Executor
	allowedFields []string
}

// NewExecutor creates a new wrapper executor that wraps another executor
// and restricts fields to the intersection of wrapper's allowedFields and inner executor's allowed fields
//
// Parameters:
//   - innerExecutor: The executor to wrap
//   - allowedFields: List of fields allowed by this wrapper (empty slice means no additional restriction from wrapper)
func NewExecutor(innerExecutor executor.Executor, allowedFields []string) *WrapperExecutor {
	return &WrapperExecutor{
		innerExecutor: innerExecutor,
		allowedFields: allowedFields,
	}
}

// Name returns the name of this executor
func (e *WrapperExecutor) Name() string {
	return "wrapper"
}

// Close cleans up resources (also closes inner executor)
func (e *WrapperExecutor) Close() error {
	if e.innerExecutor != nil {
		return e.innerExecutor.Close()
	}
	return nil
}

// Execute runs the query and stores results in dest
// It validates all fields in the query against the wrapper's allowed fields list
// before delegating to the inner executor
func (e *WrapperExecutor) Execute(ctx context.Context, q *query.Query, cursor string, dest interface{}) (*query.Result, error) {
	// Validate all fields in the query
	if err := e.validateQueryFields(q); err != nil {
		return nil, err
	}

	// Delegate to inner executor (which will also validate its own allowed fields)
	return e.innerExecutor.Execute(ctx, q, cursor, dest)
}

// Count returns the total number of items that would be returned by the given query
// It validates all fields in the query against the wrapper's allowed fields list
// before delegating to the inner executor
func (e *WrapperExecutor) Count(ctx context.Context, q *query.Query) (int64, error) {
	// Validate all fields in the query
	if err := e.validateQueryFields(q); err != nil {
		return 0, err
	}

	// Delegate to inner executor
	return e.innerExecutor.Count(ctx, q)
}

// validateQueryFields traverses the query AST and validates all field references
// against the wrapper's allowed fields list
func (e *WrapperExecutor) validateQueryFields(q *query.Query) error {
	// Validate sort field
	if q.SortBy != "" {
		if !e.isFieldAllowed(q.SortBy) {
			return query.FieldNotAllowedError(q.SortBy)
		}
	}

	// Validate fields in filter
	if q.Filter != nil {
		if err := e.validateFilterFields(q.Filter); err != nil {
			return err
		}
	}

	return nil
}

// validateFilterFields recursively validates all fields in a filter node
func (e *WrapperExecutor) validateFilterFields(node query.Node) error {
	switch n := node.(type) {
	case *query.BinaryOpNode:
		// Recursively validate left and right subtrees
		if err := e.validateFilterFields(n.Left); err != nil {
			return err
		}
		if err := e.validateFilterFields(n.Right); err != nil {
			return err
		}
		return nil

	case *query.ComparisonNode:
		// Check if field is allowed
		field := n.Field
		// Handle default search field - we need to check what it resolves to
		// But we can't resolve it here without executor options context
		// The inner executor will handle this, but we still need to validate
		// the field name as-is if it's not the special placeholder
		if field != "__DEFAULT_SEARCH__" {
			if !e.isFieldAllowed(field) {
				return query.FieldNotAllowedError(field)
			}
		}
		// For __DEFAULT_SEARCH__, we let the inner executor resolve it
		// but the inner executor will validate it against its own allowed fields
		return nil

	default:
		return fmt.Errorf("%w: unknown node type", query.ErrInvalidQuery)
	}
}

// isFieldAllowed checks if a field is in the wrapper's allowed fields list
// Returns true if allowedFields is empty (no restriction) or field is in the list
func (e *WrapperExecutor) isFieldAllowed(field string) bool {
	// Empty list means no additional restriction from wrapper
	// The inner executor will still enforce its own restrictions
	if len(e.allowedFields) == 0 {
		return true
	}

	// Check if field is in allowed list
	for _, allowed := range e.allowedFields {
		if allowed == field {
			return true
		}
	}
	return false
}
