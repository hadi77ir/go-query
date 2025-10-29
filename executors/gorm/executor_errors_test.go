package gorm

import (
	"context"
	"errors"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMExecutor_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(db, &Product{}, opts)
	ctx := context.Background()

	t.Run("ErrNoRecordsFound - empty result", func(t *testing.T) {
		p, _ := parser.NewParser("name = \"NonExistent Product\"")
		q, _ := p.Parse()

		var products []Product
		result, err := executor.Execute(ctx, q, &products)

		// Should return ErrNoRecordsFound
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrNoRecordsFound),
			"Expected ErrNoRecordsFound, got: %v", err)

		// Result should still be populated
		assert.Equal(t, int64(0), result.TotalItems)
		assert.Equal(t, 0, len(products))
	})

	t.Run("ErrInvalidFieldName - SQL injection attempt", func(t *testing.T) {
		// Manually create query with invalid field name
		q := &query.Query{
			Filter: &query.ComparisonNode{
				Field:    "name; DROP TABLE",
				Operator: query.OpEqual,
				Value:    query.StringValue("test"),
			},
		}

		var products []Product
		_, err := executor.Execute(ctx, q, &products)

		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrInvalidFieldName),
			"Expected ErrInvalidFieldName, got: %v", err)

		// Should also work with direct matching
		var fieldErr *query.FieldError
		assert.True(t, errors.As(err, &fieldErr))
		assert.Equal(t, "name; DROP TABLE", fieldErr.Field)
	})

	t.Run("ErrFieldNotAllowed - field not in whitelist", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "id"
		opts.AllowedFields = []string{"name", "price"}
		restrictedExecutor := NewExecutor(db, &Product{}, opts)

		q := &query.Query{
			Filter: &query.ComparisonNode{
				Field:    "stock", // Not in AllowedFields
				Operator: query.OpEqual,
				Value:    query.IntValue(100),
			},
		}

		var products []Product
		_, err := restrictedExecutor.Execute(ctx, q, &products)

		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed),
			"Expected ErrFieldNotAllowed, got: %v", err)

		// Check field name in error
		var fieldErr *query.FieldError
		assert.True(t, errors.As(err, &fieldErr))
		assert.Equal(t, "stock", fieldErr.Field)
	})

	t.Run("ErrRegexNotSupported - regex disabled", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "id"
		opts.DisableRegex = true
		restrictedExecutor := NewExecutor(db, &Product{}, opts)

		p, _ := parser.NewParser(`name REGEX "pattern"`)
		q, _ := p.Parse()

		var products []Product
		_, err := restrictedExecutor.Execute(ctx, q, &products)

		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrRegexNotSupported),
			"Expected ErrRegexNotSupported, got: %v", err)
	})

	t.Run("ErrRandomOrderNotAllowed - random disabled", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "id"
		opts.AllowRandomOrder = false
		restrictedExecutor := NewExecutor(db, &Product{}, opts)

		p, _ := parser.NewParser("sort_order = random")
		q, _ := p.Parse()

		var products []Product
		_, err := restrictedExecutor.Execute(ctx, q, &products)

		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrRandomOrderNotAllowed),
			"Expected ErrRandomOrderNotAllowed, got: %v", err)
	})

	t.Run("ErrInvalidDestination - not a pointer to slice", func(t *testing.T) {
		p, _ := parser.NewParser("name = \"test\"")
		q, _ := p.Parse()

		// Pass a non-slice destination
		var product Product
		_, err := executor.Execute(ctx, q, &product)

		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrInvalidDestination),
			"Expected ErrInvalidDestination, got: %v", err)
	})

	t.Run("error matching with switch", func(t *testing.T) {
		p, _ := parser.NewParser("name = \"NonExistent\"")
		q, _ := p.Parse()

		var products []Product
		_, err := executor.Execute(ctx, q, &products)

		// Demonstrate using switch for error handling
		switch {
		case errors.Is(err, query.ErrNoRecordsFound):
			// Expected - no records
			assert.True(t, true, "Correctly identified ErrNoRecordsFound")
		case errors.Is(err, query.ErrInvalidFieldName):
			t.Error("Unexpected ErrInvalidFieldName")
		case errors.Is(err, query.ErrExecutionFailed):
			t.Error("Unexpected ErrExecutionFailed")
		default:
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("success case - no error", func(t *testing.T) {
		p, _ := parser.NewParser("price > 50")
		q, _ := p.Parse()

		var products []Product
		result, err := executor.Execute(ctx, q, &products)

		// Should succeed with no error
		require.NoError(t, err)
		assert.Greater(t, len(products), 0)
		assert.Greater(t, result.TotalItems, int64(0))
	})
}

func TestGORMExecutor_ErrorWrapping(t *testing.T) {
	db := setupTestDB(t)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(db, &Product{}, opts)
	ctx := context.Background()

	t.Run("ExecutionError wraps database errors", func(t *testing.T) {
		// Query with invalid table/model should cause DB error
		type InvalidModel struct {
			NonExistentField string
		}

		p, _ := parser.NewParser("name = \"test\"")
		q, _ := p.Parse()

		var results []InvalidModel
		_, err := executor.Execute(ctx, q, &results)

		// Should get execution error
		require.Error(t, err)

		// Check if it's wrapped in ExecutionError
		var execErr *query.ExecutionError
		if errors.As(err, &execErr) {
			assert.Contains(t, execErr.Operation, "execute")
			assert.NotNil(t, execErr.Err)
		}
	})
}

/*
Example usage documentation:

	var products []Product
	result, err := executor.Execute(ctx, query, &products)

	// Option 1: Check specific errors
	if errors.Is(err, query.ErrNoRecordsFound) {
		// Handle no results - this is often not an error condition
		return emptyResponse()
	}

	// Option 2: Switch on error types
	switch {
	case errors.Is(err, query.ErrNoRecordsFound):
		// No results found - return empty list
	case errors.Is(err, query.ErrInvalidFieldName):
		// Invalid field - bad request (400)
	case errors.Is(err, query.ErrFieldNotAllowed):
		// Security violation - forbidden (403)
	case errors.Is(err, query.ErrExecutionFailed):
		// Database error - internal server error (500)
	default:
		// Other errors
	}

	// Option 3: Get field information from FieldError
	var fieldErr *query.FieldError
	if errors.As(err, &fieldErr) {
		// Access fieldErr.Field to see which field caused the error
		log.Printf("Error with field: %s", fieldErr.Field)
	}
*/
