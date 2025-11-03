package wrapper

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/v2/executors/memory"
	"github.com/hadi77ir/go-query/v2/parser"
	"github.com/hadi77ir/go-query/v2/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Product struct {
	ID       int
	Name     string
	Category string
	Price    float64
}

func getTestProducts() []Product {
	return []Product{
		{ID: 1, Name: "iPhone", Category: "electronics", Price: 999.99},
		{ID: 2, Name: "MacBook", Category: "electronics", Price: 1299.99},
		{ID: 3, Name: "Headphones", Category: "electronics", Price: 49.99},
		{ID: 4, Name: "Book 1", Category: "books", Price: 19.99},
		{ID: 5, Name: "Book 2", Category: "books", Price: 24.99},
	}
}

func TestWrapperExecutor_Count(t *testing.T) {
	data := getTestProducts()
	innerExecutor := memory.NewExecutor(data, query.DefaultExecutorOptions())
	wrapperExecutor := NewExecutor(innerExecutor, []string{"category", "price"})
	ctx := context.Background()

	t.Run("count all items", func(t *testing.T) {
		p, _ := parser.NewParser("")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("count with allowed field filter", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count with price filter", func(t *testing.T) {
		p, _ := parser.NewParser("price > 50")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Greater(t, count, int64(0))
	})

	t.Run("count rejects disallowed field", func(t *testing.T) {
		p, _ := parser.NewParser("name = iPhone")
		q, _ := p.Parse()

		_, err := wrapperExecutor.Count(ctx, q)
		require.Error(t, err)
		assert.ErrorIs(t, err, query.ErrFieldNotAllowed)
	})

	t.Run("count ignores pagination", func(t *testing.T) {
		p, _ := parser.NewParser("page_size = 2")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("count matches execute total items", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics")
		q, _ := p.Parse()

		var products []Product
		result, err := wrapperExecutor.Execute(ctx, q, "", &products)
		require.NoError(t, err)

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)

		assert.Equal(t, result.TotalItems, count)
	})

	t.Run("count with wrapper having no restrictions", func(t *testing.T) {
		// Wrapper with empty allowed fields means no restrictions from wrapper
		permissiveWrapper := NewExecutor(innerExecutor, []string{})

		p, _ := parser.NewParser("category = electronics")
		q, _ := p.Parse()

		count, err := permissiveWrapper.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count delegates to inner executor", func(t *testing.T) {
		// Count should work the same as Execute in terms of filtering
		testCases := []struct {
			query     string
			expected  int64
			shouldErr bool
		}{
			{"category = electronics", 3, false},
			{"price > 100", 2, false},
			{"name = iPhone", 0, true}, // name not in wrapper's allowed list
		}

		for _, tc := range testCases {
			t.Run(tc.query, func(t *testing.T) {
				p, _ := parser.NewParser(tc.query)
				q, _ := p.Parse()

				count, err := wrapperExecutor.Count(ctx, q)
				if tc.shouldErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expected, count)
				}
			})
		}
	})
}

func TestWrapperExecutor_Count_EdgeCases(t *testing.T) {
	data := getTestProducts()
	innerExecutor := memory.NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	t.Run("count with complex query and wrapper restrictions", func(t *testing.T) {
		// Wrapper allows category and price
		wrapperExecutor := NewExecutor(innerExecutor, []string{"category", "price"})

		p, _ := parser.NewParser("category = electronics AND price > 500")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count) // iPhone and MacBook
	})

	t.Run("count with OR query and wrapper restrictions", func(t *testing.T) {
		wrapperExecutor := NewExecutor(innerExecutor, []string{"category"})

		p, _ := parser.NewParser("category = electronics OR category = books")
		q, _ := p.Parse()

		count, err := wrapperExecutor.Count(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}
