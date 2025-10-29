package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	ID       int
	Name     string
	Email    string
	Password string // Sensitive field that should be restricted
	SSN      string // Sensitive field that should be restricted
}

func TestMemoryExecutor_AllowedFields(t *testing.T) {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", Password: "secret1", SSN: "111-11-1111"},
		{ID: 2, Name: "Bob", Email: "bob@example.com", Password: "secret2", SSN: "222-22-2222"},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", Password: "secret3", SSN: "333-33-3333"},
	}

	t.Run("unrestricted access - all fields allowed", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		// Empty AllowedFields means no restriction
		executor := NewExecutor(users, opts)

		p, _ := parser.NewParser("password = secret1")
		q, _ := p.Parse()

		var results []User
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "Alice", results[0].Name)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("restricted access - only allowed fields", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		// Use lowercase field names (query will use lowercase)
		// Reflection will find the actual struct fields case-insensitively
		opts.AllowedFields = []string{"id", "name", "email"}
		executor := NewExecutor(users, opts)

		// Allowed field query should work (lowercase matches lowercase in allowed list)
		p, _ := parser.NewParser("name = Bob")
		q, _ := p.Parse()

		var results []User
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "Bob", results[0].Name)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("restricted access - block sensitive field", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		// Lowercase allowed fields to test case-insensitive matching in query
		opts.AllowedFields = []string{"id", "name", "email"}
		executor := NewExecutor(users, opts)

		// Try to query password (not in allowed list) - uses lowercase
		// Will match "Password" field via case-insensitive reflection
		// but "password" is not in allowed list
		p, _ := parser.NewParser("password = secret1")
		q, _ := p.Parse()

		var results []User
		_, err := executor.Execute(context.Background(), q, &results)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed))
		// Verify field name is in error
		var fieldErr *query.FieldError
		if errors.As(err, &fieldErr) {
			assert.Equal(t, "password", fieldErr.Field)
		}
	})

	t.Run("restricted access - block SSN field", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.AllowedFields = []string{"id", "name", "email"}
		executor := NewExecutor(users, opts)

		// Try to query SSN (not in allowed list) - uses lowercase
		p, _ := parser.NewParser(`ssn = "111-11-1111"`)
		q, _ := p.Parse()

		var results []User
		_, err := executor.Execute(context.Background(), q, &results)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed))
		// Verify field name is in error
		var fieldErr *query.FieldError
		if errors.As(err, &fieldErr) {
			assert.Equal(t, "ssn", fieldErr.Field)
		}
	})

	t.Run("multiple fields in query", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.AllowedFields = []string{"id", "name", "email"}
		executor := NewExecutor(users, opts)

		// Mix of allowed and disallowed fields
		p, _ := parser.NewParser("name = Alice and password = secret1")
		q, _ := p.Parse()

		var results []User
		_, err := executor.Execute(context.Background(), q, &results)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed))
		// Verify field name is in error
		var fieldErr *query.FieldError
		if errors.As(err, &fieldErr) {
			assert.Equal(t, "password", fieldErr.Field)
		}
	})

	t.Run("case sensitive field names", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		// Note: field names are case-sensitive in the allowed list
		opts.AllowedFields = []string{"ID", "Name", "Email"} // Capital case
		executor := NewExecutor(users, opts)

		// Query with lowercase (should fail - case sensitive in allowed list)
		p, _ := parser.NewParser("name = Alice")
		q, _ := p.Parse()

		var results []User
		_, err := executor.Execute(context.Background(), q, &results)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed))
		// Verify field name is in error
		var fieldErr *query.FieldError
		if errors.As(err, &fieldErr) {
			assert.Equal(t, "name", fieldErr.Field)
		}
	})

	t.Run("sorting with restricted fields", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.AllowedFields = []string{"id", "name", "email"}
		executor := NewExecutor(users, opts)

		// Sort by allowed field (lowercase matches via reflection, but security check first)
		p, _ := parser.NewParser("sort_by = name sort_order = asc")
		q, _ := p.Parse()

		var results []User
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, 3, len(results))
		assert.Equal(t, "Alice", results[0].Name)
		assert.Equal(t, int64(3), result.TotalItems)
	})
}

func TestMemoryExecutor_MapWithAllowedFields(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "public": "visible", "private": "hidden1"},
		{"id": 2, "public": "visible2", "private": "hidden2"},
	}

	t.Run("unrestricted map access", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		executor := NewExecutor(data, opts)

		p, _ := parser.NewParser("private = hidden1")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("restricted map access", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.AllowedFields = []string{"id", "public"}
		executor := NewExecutor(data, opts)

		// Allowed field
		p, _ := parser.NewParser("public = visible")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("block private map field", func(t *testing.T) {
		opts := query.DefaultExecutorOptions()
		opts.AllowedFields = []string{"id", "public"}
		executor := NewExecutor(data, opts)

		// Try to access private field
		p, _ := parser.NewParser("private = hidden1")
		q, _ := p.Parse()

		var results []map[string]interface{}
		_, err := executor.Execute(context.Background(), q, &results)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed))
	})
}
