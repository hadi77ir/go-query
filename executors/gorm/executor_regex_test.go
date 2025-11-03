package gorm

import (
	"context"
	"errors"
	"testing"

	"github.com/hadi77ir/go-query/v2/parser"
	"github.com/hadi77ir/go-query/v2/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMExecutor_RegexDisabled(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	// Create executor with regex disabled (common for SQLite)
	opts := query.DefaultExecutorOptions()
	opts.DisableRegex = true
	opts.DefaultSortField = "id" // SQL uses 'id' not '_id'
	executor := NewExecutor(db.Model(&Product{}), opts)
	ctx := context.Background()

	t.Run("regex returns clear error", func(t *testing.T) {
		p, err := parser.NewParser(`name REGEX "^[A-Z].*"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		_, err = executor.Execute(ctx, q, "", &products)
		require.Error(t, err)
		assert.True(t, errors.Is(err, query.ErrRegexNotSupported))
	})

	t.Run("other string operators work fine", func(t *testing.T) {
		tests := []struct {
			name  string
			query string
		}{
			{"LIKE", `name LIKE "Wire%"`},
			{"CONTAINS", `name CONTAINS "Wire"`},
			{"STARTS_WITH", `name STARTS_WITH "Wire"`},
			{"ENDS_WITH", `name ENDS_WITH "Mouse"`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p, err := parser.NewParser(tt.query)
				require.NoError(t, err)
				q, err := p.Parse()
				require.NoError(t, err)

				var products []Product
				_, err = executor.Execute(ctx, q, "", &products)
				require.NoError(t, err, "Operator %s should work even with regex disabled", tt.name)
			})
		}
	})

	t.Run("regex enabled by default", func(t *testing.T) {
		// Default options have regex enabled
		defaultOpts := query.DefaultExecutorOptions()
		assert.False(t, defaultOpts.DisableRegex)

		// Note: This test doesn't actually execute a regex query because
		// SQLite doesn't support REGEXP by default. We're just verifying
		// that the default is "enabled" (not disabled)
	})
}
