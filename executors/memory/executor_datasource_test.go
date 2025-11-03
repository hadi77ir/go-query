package memory

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryExecutor_DataSource tests the dynamic data source functionality
func TestMemoryExecutor_DataSource(t *testing.T) {
	t.Run("data source is called on each execution", func(t *testing.T) {
		// Simulated data that changes
		products := []Product{
			{Name: "Laptop", Price: 999.99, Category: "electronics", Stock: 5},
			{Name: "Mouse", Price: 29.99, Category: "accessories", Stock: 50},
		}

		// Create executor with data source function
		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "Name"
		executor := NewExecutorWithDataSource(func() interface{} {
			return products // Data fetched fresh each time
		}, opts)

		ctx := context.Background()

		// First query - should find Mouse
		p, err := parser.NewParser("price < 100")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var results1 []Product
		result1, err := executor.Execute(ctx, q, "", &results1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result1.TotalItems)
		assert.Equal(t, "Mouse", results1[0].Name)

		// Update the data - add a new cheap product
		products = append(products, Product{Name: "Keyboard", Price: 79.99, Category: "accessories", Stock: 30})

		// Second query - should now find both Mouse and Keyboard
		var results2 []Product
		result2, err := executor.Execute(ctx, q, "", &results2)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result2.TotalItems)

		names := []string{results2[0].Name, results2[1].Name}
		assert.Contains(t, names, "Mouse")
		assert.Contains(t, names, "Keyboard")
	})

	t.Run("backwards compatible with NewExecutor", func(t *testing.T) {
		// Original API should still work
		products := []Product{
			{Name: "Laptop", Price: 999.99, Category: "electronics", Stock: 5},
			{Name: "Mouse", Price: 29.99, Category: "accessories", Stock: 50},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "Name"
		executor := NewExecutor(products, opts)

		ctx := context.Background()
		p, err := parser.NewParser("price < 100")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var results []Product
		result, err := executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
		assert.Equal(t, "Mouse", results[0].Name)
	})

	t.Run("data source with map", func(t *testing.T) {
		// Use maps instead of structs
		data := []map[string]interface{}{
			{"name": "Laptop", "price": 999.99},
			{"name": "Mouse", "price": 29.99},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "name"

		executor := NewExecutorWithDataSource(func() interface{} {
			return data
		}, opts)

		ctx := context.Background()
		p, err := parser.NewParser("price < 100")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var results []map[string]interface{}
		result, err := executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)

		// Add more data
		data = append(data, map[string]interface{}{"name": "Keyboard", "price": 79.99})

		// Query again - should see new data
		results = []map[string]interface{}{}
		result, err = executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.TotalItems)
	})

	t.Run("data source with complex filters", func(t *testing.T) {
		counter := 0
		dataSource := func() interface{} {
			counter++
			// Simulate data changing over time
			if counter == 1 {
				return []Product{
					{Name: "Product A", Price: 50, Category: "cat1", Stock: 10},
					{Name: "Product B", Price: 150, Category: "cat2", Stock: 5},
				}
			}
			return []Product{
				{Name: "Product A", Price: 50, Category: "cat1", Stock: 10},
				{Name: "Product B", Price: 150, Category: "cat2", Stock: 5},
				{Name: "Product C", Price: 75, Category: "cat1", Stock: 20},
				{Name: "Product D", Price: 200, Category: "cat2", Stock: 3},
			}
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "Name"
		executor := NewExecutorWithDataSource(dataSource, opts)

		ctx := context.Background()
		p, err := parser.NewParser("price < 100 and category = cat1")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		// First execution - 1 result
		var results1 []Product
		result1, err := executor.Execute(ctx, q, "", &results1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result1.TotalItems)

		// Second execution - 2 results (data source returns more data)
		var results2 []Product
		result2, err := executor.Execute(ctx, q, "", &results2)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result2.TotalItems)
	})

	t.Run("data source with custom options", func(t *testing.T) {
		products := []Product{
			{Name: "Laptop", Price: 999.99, Category: "electronics", Stock: 5},
			{Name: "Mouse", Price: 29.99, Category: "accessories", Stock: 50},
		}

		opts := &MemoryExecutorOptions{
			ExecutorOptions: query.DefaultExecutorOptions(),
			FieldGetter: func(obj interface{}, field string) (interface{}, error) {
				// Custom field getter
				p := obj.(*Product)
				switch field {
				case "Name":
					return p.Name, nil
				case "Price":
					return p.Price, nil
				default:
					return nil, nil
				}
			},
		}
		opts.ExecutorOptions.DefaultSortField = "Name"

		executor := NewExecutorWithDataSourceAndOptions(func() interface{} {
			return products
		}, opts)

		ctx := context.Background()
		p, err := parser.NewParser("Price < 100")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var results []Product
		result, err := executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.TotalItems)
	})
}

// TestMemoryExecutor_LiveDataUpdates tests real-world scenarios with changing data
func TestMemoryExecutor_LiveDataUpdates(t *testing.T) {
	t.Run("simulate cache or database that updates", func(t *testing.T) {
		// Simulate a cache that gets updated
		type Cache struct {
			data []Product
		}

		cache := &Cache{
			data: []Product{
				{Name: "Item1", Price: 100, Stock: 10},
				{Name: "Item2", Price: 200, Stock: 5},
			},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "Name"

		// Executor always queries current cache state
		executor := NewExecutorWithDataSource(func() interface{} {
			return cache.data
		}, opts)

		ctx := context.Background()
		p, err := parser.NewParser("Stock > 0")
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		// Query 1 - 2 items in stock
		var results1 []Product
		result1, err := executor.Execute(ctx, q, "", &results1)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result1.TotalItems)

		// Update cache - one item goes out of stock
		cache.data[1].Stock = 0

		// Query 2 - only 1 item in stock now
		var results2 []Product
		result2, err := executor.Execute(ctx, q, "", &results2)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result2.TotalItems)
		assert.Equal(t, "Item1", results2[0].Name)

		// Add new item to cache
		cache.data = append(cache.data, Product{Name: "Item3", Price: 150, Stock: 7})

		// Query 3 - 2 items in stock now
		var results3 []Product
		result3, err := executor.Execute(ctx, q, "", &results3)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result3.TotalItems)
	})
}
