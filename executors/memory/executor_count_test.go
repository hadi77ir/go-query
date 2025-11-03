package memory

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/v2/parser"
	"github.com/hadi77ir/go-query/v2/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestProductsForCount() []Product {
	return []Product{
		{ID: 1, Name: "iPhone", Category: "electronics", Price: 999.99},
		{ID: 2, Name: "MacBook", Category: "electronics", Price: 1299.99},
		{ID: 3, Name: "Headphones", Category: "electronics", Price: 49.99},
		{ID: 4, Name: "Book 1", Category: "books", Price: 19.99},
		{ID: 5, Name: "Book 2", Category: "books", Price: 24.99},
		{ID: 6, Name: "T-Shirt", Category: "clothing", Price: 29.99},
		{ID: 7, Name: "Jeans", Category: "clothing", Price: 59.99},
		{ID: 8, Name: "Shoes", Category: "clothing", Price: 79.99},
		{ID: 9, Name: "Watch", Category: "accessories", Price: 199.99},
		{ID: 10, Name: "Belt", Category: "accessories", Price: 39.99},
	}
}

func TestMemoryExecutor_Count(t *testing.T) {
	data := getTestProductsForCount()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
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
		assert.Equal(t, int64(3), count)
	})

	t.Run("count with multiple filters", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics AND price > 500")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count) // iPhone and MacBook
	})

	t.Run("count with OR filter", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics OR category = books")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count) // 3 electronics + 2 books
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
		p, _ := parser.NewParser(`name LIKE "%phone%"`)
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0)) // Should match "iPhone"
	})

	t.Run("count with IN filter", func(t *testing.T) {
		p, _ := parser.NewParser("category IN [electronics, books]")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("count with NOT IN filter", func(t *testing.T) {
		p, _ := parser.NewParser("category NOT IN [electronics]")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(7), count) // All except 3 electronics
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
		p, _ := parser.NewParser("name CONTAINS phone")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0)) // Should match "iPhone"
	})

	t.Run("count with starts with filter", func(t *testing.T) {
		p, _ := parser.NewParser("name STARTS_WITH Book")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("count with ends with filter", func(t *testing.T) {
		p, _ := parser.NewParser("name ENDS_WITH Phone")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count) // Should match "iPhone"
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

	t.Run("count with dynamic data source", func(t *testing.T) {
		data := []Product{
			{ID: 1, Name: "Item 1", Category: "A", Price: 10},
			{ID: 2, Name: "Item 2", Category: "A", Price: 20},
			{ID: 3, Name: "Item 3", Category: "B", Price: 30},
		}

		// Create executor with dynamic data source
		executor := NewExecutorWithDataSource(func() interface{} {
			return data
		}, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("category = A")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Modify data
		data = append(data, Product{ID: 4, Name: "Item 4", Category: "A", Price: 40})

		// Count should reflect new data
		count, err = executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count with empty result", func(t *testing.T) {
		emptyData := []Product{}
		executor := NewExecutor(emptyData, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("")
		q, _ := p.Parse()

		count, err := executor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
