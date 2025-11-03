package gorm

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/v2/parser"
	"github.com/hadi77ir/go-query/v2/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGORMExecutor_Count(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(db.Model(&Product{}), opts)
	ctx := context.Background()

	t.Run("count all items", func(t *testing.T) {
		p, _ := parser.NewParser("")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	t.Run("count with filter", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count) // Mouse, Keyboard, Headphones, Speaker, Webcam
	})

	t.Run("count with multiple filters", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics AND price > 500")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count) // No electronics items > 500 in test data
	})

	t.Run("count with OR filter", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics OR category = accessories")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count) // 5 electronics + 4 accessories + 1 other = all 10 items (accessories overlap)
	})

	t.Run("count with no matches", func(t *testing.T) {
		p, _ := parser.NewParser("category = nonexistent")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("count with complex filter", func(t *testing.T) {
		p, _ := parser.NewParser("price >= 10 AND price <= 50")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("count ignores pagination", func(t *testing.T) {
		p, _ := parser.NewParser("page_size = 3")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		// Should still count all items, not just 3
		assert.Equal(t, int64(10), count)
	})

	t.Run("count ignores cursor", func(t *testing.T) {
		p, _ := parser.NewParser("")
		q, _ := p.Parse()

		// Execute to get a cursor
		var products []Product
		result, _ := executor.Execute(ctx, q, "", &products)

		// Count should be the same regardless of cursor
		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count)

		// Count should match TotalItems from Execute
		assert.Equal(t, result.TotalItems, count)
	})

	t.Run("count with LIKE filter", func(t *testing.T) {
		p, _ := parser.NewParser(`name LIKE "%mouse%"`)
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0)) // Should match "Wireless Mouse"
	})

	t.Run("count with IN filter", func(t *testing.T) {
		p, _ := parser.NewParser("category IN [electronics, accessories]")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count) // All items are either electronics or accessories
	})

	t.Run("count with NOT IN filter", func(t *testing.T) {
		p, _ := parser.NewParser("category NOT IN [electronics]")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count) // All except 5 electronics (4 accessories + 1 other)
	})

	t.Run("count with greater than filter", func(t *testing.T) {
		p, _ := parser.NewParser("price > 100")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("count with less than filter", func(t *testing.T) {
		p, _ := parser.NewParser("price < 50")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("count with contains filter", func(t *testing.T) {
		p, _ := parser.NewParser("name CONTAINS mouse")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0)) // Should match "Wireless Mouse"
	})

	t.Run("count matches execute total items", func(t *testing.T) {
		testCases := []string{
			"",
			"category = electronics",
			"price > 50",
			"name CONTAINS i",
		}

		for _, queryStr := range testCases {
			t.Run(queryStr, func(t *testing.T) {
				p, _ := parser.NewParser(queryStr)
				q, _ := p.Parse()

				var products []Product
				result, err := executor.Execute(ctx, q, "", &products)
				require.NoError(t, err)

				count, err := executor.Count(ctx, q)
				require.NoError(t, err)

				assert.Equal(t, result.TotalItems, count, "Count should match Execute TotalItems")
			})
		}
	})
}
