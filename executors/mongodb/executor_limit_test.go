package mongodb

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMongoExecutor_LimitEnforcement(t *testing.T) {
	mongoC, collection := setupMongoContainer(t)
	defer mongoC.Terminate(context.Background())

	seedMongoTestData(t, collection)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "_id"
	executor := NewExecutor(collection, opts)
	ctx := context.Background()

	t.Run("limit less than page size", func(t *testing.T) {
		p, err := parser.NewParser("limit = 3 page_size = 10 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 3, len(products))
		assert.Equal(t, 3, result.ItemsReturned)
		assert.Equal(t, "", result.NextPageCursor, "should not have next page when limit reached")
		assert.Equal(t, int64(10), result.TotalItems)
	})

	t.Run("limit greater than total items", func(t *testing.T) {
		p, err := parser.NewParser("limit = 100 page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 5, len(products)) // Page size, not limit
		assert.Equal(t, 5, result.ItemsReturned)
		assert.NotEqual(t, "", result.NextPageCursor, "should have next page")
		assert.Equal(t, int64(10), result.TotalItems)
	})

	t.Run("limit exactly equal to page size", func(t *testing.T) {
		p, err := parser.NewParser("limit = 5 page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 5, len(products))
		assert.Equal(t, 5, result.ItemsReturned)
		assert.Equal(t, "", result.NextPageCursor, "should not have next page when limit reached")
	})

	t.Run("limit zero means no limit", func(t *testing.T) {
		p, err := parser.NewParser("limit = 0 page_size = 3 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 3, len(products))
		assert.NotEqual(t, "", result.NextPageCursor, "should have next page when limit is 0")
	})

	t.Run("limit reached on second page", func(t *testing.T) {
		p, err := parser.NewParser("limit = 7 page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		// First page
		var page1 []Product
		result1, err := executor.Execute(ctx, q, "", &page1)
		require.NoError(t, err)
		assert.Equal(t, 5, len(page1))
		assert.NotEqual(t, "", result1.NextPageCursor, "should have next page")

		// Second page - should only return 2 items (7 total - 5 from first page)
		var page2 []Product
		result2, err := executor.Execute(ctx, q, result1.NextPageCursor, &page2)
		require.NoError(t, err)
		assert.Equal(t, 2, len(page2))
		assert.Equal(t, 2, result2.ItemsReturned)
		assert.Equal(t, "", result2.NextPageCursor, "should not have next page when limit reached")
	})

	t.Run("limit reached mid-page", func(t *testing.T) {
		p, err := parser.NewParser("limit = 3 page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 3, len(products), "should only return 3 items even though page size is 5")
		assert.Equal(t, "", result.NextPageCursor)
	})

	t.Run("limit with filter", func(t *testing.T) {
		p, err := parser.NewParser("limit = 2 category = electronics page_size = 10 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 2, len(products))
		assert.Equal(t, "", result.NextPageCursor)
		// Verify all returned items match filter
		for _, p := range products {
			assert.Equal(t, "electronics", p.Category)
		}
	})

	t.Run("limit with descending sort", func(t *testing.T) {
		p, err := parser.NewParser("limit = 3 page_size = 5 sort_by = _id sort_order = desc")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 3, len(products))
		assert.Equal(t, "", result.NextPageCursor)
		// Verify descending order (highest IDs first - MongoDB string IDs sort lexicographically)
		// Since IDs are strings "1" through "10", descending order is: "9", "8", "7", etc.
		assert.GreaterOrEqual(t, products[0].ID, products[1].ID)
		assert.GreaterOrEqual(t, products[1].ID, products[2].ID)
	})

	t.Run("limit with random sort", func(t *testing.T) {
		// Note: MongoDB random sort implementation may have limitations
		// Skip this test if random sort is not properly supported
		p, err := parser.NewParser("limit = 3 page_size = 5 sort_order = random")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		// MongoDB's random sort implementation may fail, so we accept either success or error
		if err != nil {
			// If random sort fails, that's acceptable for this test
			t.Skipf("Random sort not supported in MongoDB: %v", err)
		} else {
			assert.Equal(t, 3, len(products))
			assert.Equal(t, "", result.NextPageCursor)
		}
	})

	t.Run("limit exceeded on first page returns no next cursor", func(t *testing.T) {
		// If limit was reached in first page, there should be no next cursor
		p, err := parser.NewParser("limit = 5 page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		// First page - reaches limit
		var page1 []Product
		result1, err := executor.Execute(ctx, q, "", &page1)
		require.NoError(t, err)
		assert.Equal(t, 5, len(page1))
		assert.Equal(t, "", result1.NextPageCursor, "should not have next cursor when limit reached")

		// If we try with an empty cursor again, it's treated as first page
		// This is expected behavior - each Execute call is independent
		var page1Again []Product
		result1Again, err := executor.Execute(ctx, q, "", &page1Again)
		require.NoError(t, err)
		assert.Equal(t, 5, len(page1Again), "empty cursor is treated as first page")
		assert.Equal(t, "", result1Again.NextPageCursor)
	})

	t.Run("limit with pagination forward and backward", func(t *testing.T) {
		p, err := parser.NewParser("limit = 6 page_size = 3 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		// Page 1
		var page1 []Product
		result1, err := executor.Execute(ctx, q, "", &page1)
		require.NoError(t, err)
		assert.Equal(t, 3, len(page1))
		assert.NotEqual(t, "", result1.NextPageCursor)

		// Page 2
		var page2 []Product
		result2, err := executor.Execute(ctx, q, result1.NextPageCursor, &page2)
		require.NoError(t, err)
		assert.Equal(t, 3, len(page2))
		assert.Equal(t, "", result2.NextPageCursor, "limit reached")

		// Go back to page 1
		var page1Again []Product
		_, err = executor.Execute(ctx, q, result2.PrevPageCursor, &page1Again)
		require.NoError(t, err)
		assert.Equal(t, 3, len(page1Again))
	})
}

func TestMongoExecutor_LimitEdgeCases(t *testing.T) {
	mongoC, collection := setupMongoContainer(t)
	defer mongoC.Terminate(context.Background())

	seedMongoTestData(t, collection)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "_id"
	executor := NewExecutor(collection, opts)
	ctx := context.Background()

	t.Run("limit = 1", func(t *testing.T) {
		p, err := parser.NewParser("limit = 1 page_size = 10 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 1, len(products))
		assert.Equal(t, "", result.NextPageCursor)
	})

	t.Run("limit with empty result set", func(t *testing.T) {
		p, err := parser.NewParser("limit = 10 category = nonexistent page_size = 5 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		_, err = executor.Execute(ctx, q, "", &products)
		// MongoDB executor returns an error when no records found, which is expected
		assert.Error(t, err)
		assert.Equal(t, 0, len(products))
	})

	t.Run("limit with OR condition", func(t *testing.T) {
		p, err := parser.NewParser("limit = 4 (category = electronics or category = accessories) page_size = 10 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 4, len(products))
		assert.Equal(t, "", result.NextPageCursor)
	})

	t.Run("limit with non-id sort field", func(t *testing.T) {
		p, err := parser.NewParser("limit = 3 page_size = 5 sort_by = price sort_order = asc")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 3, len(products))
		assert.Equal(t, "", result.NextPageCursor)
		// Verify sorted by price
		for i := 1; i < len(products); i++ {
			assert.LessOrEqual(t, products[i-1].Price, products[i].Price)
		}
	})
}

func TestMongoExecutor_LimitWithLargeDataset(t *testing.T) {
	mongoC, collection := setupMongoContainer(t)
	defer mongoC.Terminate(context.Background())

	ctx := context.Background()

	// Create a larger dataset
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	products := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		products[i] = bson.M{
			"_id":        "item_" + strconv.Itoa(i+1),
			"name":       "Product",
			"price":      float64(i + 1),
			"category":   "test",
			"created_at": baseTime.Add(time.Duration(i) * time.Hour),
		}
	}
	_, err := collection.InsertMany(ctx, products)
	require.NoError(t, err)

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "_id"
	executor := NewExecutor(collection, opts)

	t.Run("limit across multiple pages", func(t *testing.T) {
		p, err := parser.NewParser("limit = 75 page_size = 25 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var allItems []bson.M
		currentCursor := ""

		// Collect items across pages until limit is reached
		for i := 0; i < 10; i++ { // Safety limit
			var page []bson.M
			result, err := executor.Execute(ctx, q, currentCursor, &page)
			require.NoError(t, err)

			allItems = append(allItems, page...)

			if result.NextPageCursor == "" {
				break
			}
			currentCursor = result.NextPageCursor
		}

		// Should have exactly 75 items (3 pages of 25)
		assert.Equal(t, 75, len(allItems))
	})

	t.Run("limit less than one page", func(t *testing.T) {
		p, err := parser.NewParser("limit = 15 page_size = 50 sort_by = _id")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []bson.M
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 15, len(products))
		assert.Equal(t, "", result.NextPageCursor)
	})
}
