package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryExecutor_MapsComprehensive demonstrates all map functionality
func TestMemoryExecutor_MapsComprehensive(t *testing.T) {
	t.Run("basic map querying", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "Product A", "price": 99.99, "active": true},
			{"id": 2, "name": "Product B", "price": 149.99, "active": false},
			{"id": 3, "name": "Product C", "price": 79.99, "active": true},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "id"
		executor := NewExecutor(data, opts)

		p, err := parser.NewParser("active = true and price < 100")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var results []map[string]interface{}
		result, err := executor.Execute(context.Background(), q, &results)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.TotalItems)
		assert.Equal(t, 2, len(results))
	})

	t.Run("maps with string operations", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "email": "john@example.com", "status": "active"},
			{"id": 2, "email": "jane@test.com", "status": "inactive"},
			{"id": 3, "email": "bob@example.com", "status": "active"},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		p, _ := parser.NewParser(`email ENDS_WITH "@example.com" and status = active`)
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(2), result.TotalItems)
	})

	t.Run("maps with numeric comparisons", func(t *testing.T) {
		data := []map[string]interface{}{
			{"product": "A", "stock": 50, "price": 99.99},
			{"product": "B", "stock": 5, "price": 49.99},
			{"product": "C", "stock": 100, "price": 149.99},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		// Find products with low stock
		p, _ := parser.NewParser("stock < 10")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(1), result.TotalItems)
		assert.Equal(t, "B", results[0]["product"])
	})

	t.Run("maps with IN operator", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "category": "electronics", "name": "Laptop"},
			{"id": 2, "category": "clothing", "name": "Shirt"},
			{"id": 3, "category": "electronics", "name": "Phone"},
			{"id": 4, "category": "books", "name": "Novel"},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("category IN [electronics, books]")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(3), result.TotalItems)
	})

	t.Run("maps with pagination", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "value": 10},
			{"id": 2, "value": 20},
			{"id": 3, "value": 30},
			{"id": 4, "value": 40},
			{"id": 5, "value": 50},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		// First page
		p, _ := parser.NewParser("page_size = 2 sort_by = id")
		q, _ := p.Parse()

		var page1 []map[string]interface{}
		result1, _ := executor.Execute(context.Background(), q, &page1)
		assert.Equal(t, 2, len(page1))
		assert.Equal(t, int64(5), result1.TotalItems)
		assert.NotEmpty(t, result1.NextPageCursor)

		// Second page
		q.Cursor = result1.NextPageCursor
		var page2 []map[string]interface{}
		_, _ = executor.Execute(context.Background(), q, &page2)
		assert.Equal(t, 2, len(page2))
	})

	t.Run("maps with dynamic data source", func(t *testing.T) {
		// Mutable map data
		inventory := []map[string]interface{}{
			{"sku": "ABC123", "quantity": 10, "location": "A1"},
			{"sku": "DEF456", "quantity": 0, "location": "B2"},
		}

		executor := NewExecutorWithDataSource(func() interface{} {
			return inventory
		}, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("quantity > 0")
		q, _ := p.Parse()

		// Query 1 - one item in stock
		var results1 []map[string]interface{}
		result1, _ := executor.Execute(context.Background(), q, &results1)
		assert.Equal(t, int64(1), result1.TotalItems)

		// Update inventory
		inventory[1]["quantity"] = 20

		// Query 2 - two items in stock
		var results2 []map[string]interface{}
		result2, _ := executor.Execute(context.Background(), q, &results2)
		assert.Equal(t, int64(2), result2.TotalItems)
	})

	t.Run("maps with complex nested values", func(t *testing.T) {
		data := []map[string]interface{}{
			{
				"id":       1,
				"user":     "john",
				"settings": map[string]interface{}{"theme": "dark", "notifications": true},
				"tags":     []string{"admin", "verified"},
			},
			{
				"id":       2,
				"user":     "jane",
				"settings": map[string]interface{}{"theme": "light", "notifications": false},
				"tags":     []string{"user"},
			},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("user = john")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(1), result.TotalItems)
		assert.Equal(t, "john", results[0]["user"])
	})

	t.Run("maps with case-insensitive field names", func(t *testing.T) {
		data := []map[string]interface{}{
			{"ProductName": "Item A", "ProductPrice": 100},
			{"ProductName": "Item B", "ProductPrice": 200},
		}

		executor := NewExecutor(data, query.DefaultExecutorOptions())

		// Query using different case
		p, _ := parser.NewParser("productname = \"Item A\"")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(1), result.TotalItems)
	})

	t.Run("maps with bare search", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "Wireless Mouse", "category": "electronics"},
			{"id": 2, "name": "Wired Keyboard", "category": "electronics"},
			{"id": 3, "name": "USB Cable", "category": "accessories"},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSearchField = "name"
		executor := NewExecutor(data, opts)

		// Bare search term
		p, _ := parser.NewParser("Wireless")
		q, _ := p.Parse()

		var results []map[string]interface{}
		result, _ := executor.Execute(context.Background(), q, &results)
		assert.Equal(t, int64(1), result.TotalItems)
		assert.Equal(t, "Wireless Mouse", results[0]["name"])
	})
}

// Example demonstrating real-world map usage
func ExampleMemoryExecutor_mapsUsage() {
	// Simulated JSON API response data
	apiData := []map[string]interface{}{
		{
			"id":          1,
			"username":    "john_doe",
			"email":       "john@example.com",
			"role":        "admin",
			"active":      true,
			"login_count": 150,
		},
		{
			"id":          2,
			"username":    "jane_smith",
			"email":       "jane@example.com",
			"role":        "user",
			"active":      true,
			"login_count": 45,
		},
		{
			"id":          3,
			"username":    "bob_inactive",
			"email":       "bob@example.com",
			"role":        "user",
			"active":      false,
			"login_count": 2,
		},
	}

	executor := NewExecutor(apiData, query.DefaultExecutorOptions())

	// Query for active users with significant activity
	p, _ := parser.NewParser("active = true and login_count > 50")
	q, _ := p.Parse()

	var results []map[string]interface{}
	result, _ := executor.Execute(context.Background(), q, &results)

	fmt.Printf("Found %d active users\n", result.TotalItems)
	for _, user := range results {
		fmt.Printf("- %s (%s)\n", user["username"], user["role"])
	}
	// Output:
	// Found 1 active users
	// - john_doe (admin)
}
