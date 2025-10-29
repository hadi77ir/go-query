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

// TestGORMExecutor_SQLInjectionProtection tests protection against SQL injection
// via field names and values
func TestGORMExecutor_SQLInjectionProtection(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(db, &Product{}, opts)
	ctx := context.Background()

	t.Run("field values are parameterized - safe from injection", func(t *testing.T) {
		// Values are always parameterized with ?, so they're safe
		tests := []struct {
			name  string
			query string
		}{
			{
				name:  "single quote in value",
				query: `name = "Test'; DROP TABLE products; --"`,
			},
			{
				name:  "double quote in value",
				query: `name = "Test\"; DROP TABLE products; --"`,
			},
			{
				name:  "semicolon in value",
				query: `name = "Test; DELETE FROM products"`,
			},
			{
				name:  "comment in value",
				query: `name = "Test -- comment"`,
			},
			{
				name:  "union injection attempt in value",
				query: `name = "' UNION SELECT * FROM users --"`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p, err := parser.NewParser(tt.query)
				require.NoError(t, err)
				q, err := p.Parse()
				require.NoError(t, err)

				var products []Product
				// Should not error - values are parameterized
				// Will return ErrNoRecordsFound since no products match
				result, err := executor.Execute(ctx, q, &products)
				require.Error(t, err)
				assert.True(t, errors.Is(err, query.ErrNoRecordsFound))
				// Should return 0 results (no product with that exact name)
				assert.Equal(t, 0, len(products))
				assert.Equal(t, int64(0), result.TotalItems)
			})
		}
	})

	t.Run("field names with SQL injection attempts are rejected", func(t *testing.T) {
		// Field names cannot be parameterized, so they're validated with regex
		tests := []struct {
			name      string
			fieldName string
		}{
			{
				name:      "field with single quote",
				fieldName: "name'; DROP TABLE products; --",
			},
			{
				name:      "field with semicolon",
				fieldName: "name; DELETE FROM products",
			},
			{
				name:      "field with space",
				fieldName: "name OR 1=1",
			},
			{
				name:      "field with dash",
				fieldName: "name--comment",
			},
			{
				name:      "field with special chars",
				fieldName: "name@#$%",
			},
			{
				name:      "union injection in field",
				fieldName: "name UNION SELECT",
			},
			{
				name:      "parentheses in field",
				fieldName: "name()",
			},
			{
				name:      "asterisk in field",
				fieldName: "name*",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Manually construct a query with invalid field name
				// (parser might reject some of these, so we test at executor level)
				q := &query.Query{
					Filter: &query.ComparisonNode{
						Field:    tt.fieldName,
						Operator: query.OpEqual,
						Value:    query.StringValue("test"),
					},
				}

				var products []Product
				_, err := executor.Execute(ctx, q, &products)
				// Should return error due to invalid field name
				require.Error(t, err)
				assert.True(t, errors.Is(err, query.ErrInvalidFieldName))
				// Verify field name is in error
				var fieldErr *query.FieldError
				if errors.As(err, &fieldErr) {
					assert.Equal(t, tt.fieldName, fieldErr.Field)
				}
			})
		}
	})

	t.Run("valid field names are allowed", func(t *testing.T) {
		tests := []struct {
			name      string
			fieldName string
		}{
			{
				name:      "simple field",
				fieldName: "name",
			},
			{
				name:      "field with underscore",
				fieldName: "created_at",
			},
			{
				name:      "field starting with underscore",
				fieldName: "_id",
			},
			{
				name:      "field with numbers",
				fieldName: "field123",
			},
			{
				name:      "mixed case",
				fieldName: "FirstName",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				q := &query.Query{
					Filter: &query.ComparisonNode{
						Field:    tt.fieldName,
						Operator: query.OpEqual,
						Value:    query.StringValue("test"),
					},
				}

				var products []Product
				_, err := executor.Execute(ctx, q, &products)
				// Should not error due to field name validation
				// (might error due to "no such column" but not "invalid field name")
				if err != nil {
					assert.False(t, errors.Is(err, query.ErrInvalidFieldName),
						"Field '%s' should pass validation", tt.fieldName)
				}
			})
		}
	})

	t.Run("field name validation regex details", func(t *testing.T) {
		// Test field validation by trying to use them in queries
		// Valid fields should not error with "invalid field name"
		validFields := []string{
			"name",
			"_id",
			"field_name",
			"Field123",
			"CamelCase",
			"snake_case_123",
		}

		for _, field := range validFields {
			q := &query.Query{
				Filter: &query.ComparisonNode{
					Field:    field,
					Operator: query.OpEqual,
					Value:    query.StringValue("test"),
				},
			}
			var products []Product
			_, err := executor.Execute(ctx, q, &products)
			if err != nil {
				assert.False(t, errors.Is(err, query.ErrInvalidFieldName),
					"Field '%s' should pass validation (may fail for other reasons)", field)
			}
		}

		invalidFields := []string{
			"name; DROP TABLE", // semicolon
			"name OR 1=1",      // space
			"name'",            // single quote
			"name\"",           // double quote
			"name--",           // double dash
			"name/*comment*/",  // comment
			"name()",           // parentheses
			"name.field",       // dot (for now - could support in future)
			"123field",         // starts with number
			"name@example",     // special char
			"",                 // empty
			"name\nDROP",       // newline
			"name\x00",         // null byte
		}

		for _, field := range invalidFields {
			q := &query.Query{
				Filter: &query.ComparisonNode{
					Field:    field,
					Operator: query.OpEqual,
					Value:    query.StringValue("test"),
				},
			}
			var products []Product
			_, err := executor.Execute(ctx, q, &products)
			require.Error(t, err, "Field '%s' should be rejected", field)
			assert.True(t, errors.Is(err, query.ErrInvalidFieldName),
				"Field '%s' should fail with invalid field name error", field)
			// Verify field name is in error
			var fieldErr *query.FieldError
			if errors.As(err, &fieldErr) {
				assert.Equal(t, field, fieldErr.Field)
			}
		}
	})

	t.Run("complex injection scenarios", func(t *testing.T) {
		// Real-world SQL injection attempts
		injectionAttempts := []struct {
			name  string
			query string
		}{
			{
				name:  "classic SQL injection",
				query: `name = "' OR '1'='1"`,
			},
			{
				name:  "time-based blind injection",
				query: `name = "'; WAITFOR DELAY '00:00:05'--"`,
			},
			{
				name:  "stacked queries",
				query: `name = "'; DROP TABLE products; --"`,
			},
			{
				name:  "union-based injection",
				query: `name = "' UNION SELECT password FROM users--"`,
			},
		}

		for _, tt := range injectionAttempts {
			t.Run(tt.name, func(t *testing.T) {
				p, err := parser.NewParser(tt.query)
				require.NoError(t, err)
				q, err := p.Parse()
				require.NoError(t, err)

				var products []Product
				result, err := executor.Execute(ctx, q, &products)

				// Should execute safely (values are parameterized)
				// Will return ErrNoRecordsFound since injection attempt is treated as literal string
				require.Error(t, err)
				assert.True(t, errors.Is(err, query.ErrNoRecordsFound))

				// Should return 0 results (injection attempt treated as literal string)
				assert.Equal(t, 0, len(products))
				assert.Equal(t, int64(0), result.TotalItems)

				// Most importantly: database should still exist and work
				var allProducts []Product
				db.Find(&allProducts)
				assert.Greater(t, len(allProducts), 0, "Database should still contain products")
			})
		}
	})
}

// TestGORMExecutor_AllowedFieldsAdditionalSecurity tests that AllowedFields
// provides an additional layer of security on top of field name validation
func TestGORMExecutor_AllowedFieldsAdditionalSecurity(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	// Even if a field name passes regex validation, it can be blocked by AllowedFields
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	opts.AllowedFields = []string{"name", "price", "category"} // Limited set
	executor := NewExecutor(db, &Product{}, opts)
	ctx := context.Background()

	t.Run("allowed field works", func(t *testing.T) {
		q := &query.Query{
			Filter: &query.ComparisonNode{
				Field:    "name",
				Operator: query.OpEqual,
				Value:    query.StringValue("test"),
			},
		}

		var products []Product
		_, err := executor.Execute(ctx, q, &products)
		// Empty result will return ErrNoRecordsFound
		if err != nil && !errors.Is(err, query.ErrNoRecordsFound) {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("valid but not allowed field is blocked", func(t *testing.T) {
		q := &query.Query{
			Filter: &query.ComparisonNode{
				Field:    "stock", // Valid field name, but not in AllowedFields
				Operator: query.OpEqual,
				Value:    query.IntValue(100),
			},
		}

		var products []Product
		_, err := executor.Execute(ctx, q, &products)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrFieldNotAllowed), "Expected ErrFieldNotAllowed")
	})
}
