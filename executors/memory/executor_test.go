package memory

import (
	"context"
	"testing"
	"time"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structures
type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	Stock       int
	Category    string
	Brand       string
	Featured    bool
	Rating      float64
	CreatedAt   time.Time
}

func getTestData() []Product {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return []Product{
		{ID: 1, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Price: 29.99, Stock: 100, Category: "electronics", Brand: "Logitech", Featured: true, Rating: 4.5, CreatedAt: baseTime},
		{ID: 2, Name: "Mechanical Keyboard", Description: "RGB mechanical keyboard", Price: 89.99, Stock: 50, Category: "electronics", Brand: "Corsair", Featured: false, Rating: 4.8, CreatedAt: baseTime.Add(24 * time.Hour)},
		{ID: 3, Name: "USB Cable", Description: "High speed USB-C cable", Price: 9.99, Stock: 200, Category: "accessories", Brand: "Anker", Featured: false, Rating: 4.2, CreatedAt: baseTime.Add(48 * time.Hour)},
		{ID: 4, Name: "Wireless Headphones", Description: "Noise cancelling headphones", Price: 199.99, Stock: 30, Category: "electronics", Brand: "Sony", Featured: true, Rating: 4.9, CreatedAt: baseTime.Add(72 * time.Hour)},
		{ID: 5, Name: "Gaming Mouse Pad", Description: "Large extended mouse pad", Price: 19.99, Stock: 75, Category: "accessories", Brand: "Razer", Featured: false, Rating: 4.3, CreatedAt: baseTime.Add(96 * time.Hour)},
		{ID: 6, Name: "Wireless Charger", Description: "Fast wireless charging pad", Price: 39.99, Stock: 60, Category: "accessories", Brand: "Anker", Featured: true, Rating: 4.6, CreatedAt: baseTime.Add(120 * time.Hour)},
		{ID: 7, Name: "USB Hub", Description: "7-port USB 3.0 hub", Price: 24.99, Stock: 90, Category: "accessories", Brand: "Anker", Featured: false, Rating: 4.4, CreatedAt: baseTime.Add(144 * time.Hour)},
		{ID: 8, Name: "Bluetooth Speaker", Description: "Portable bluetooth speaker", Price: 49.99, Stock: 45, Category: "electronics", Brand: "JBL", Featured: true, Rating: 4.7, CreatedAt: baseTime.Add(168 * time.Hour)},
		{ID: 9, Name: "Webcam HD", Description: "1080p HD webcam", Price: 69.99, Stock: 35, Category: "electronics", Brand: "Logitech", Featured: false, Rating: 4.5, CreatedAt: baseTime.Add(192 * time.Hour)},
		{ID: 10, Name: "Monitor Stand", Description: "Adjustable monitor stand", Price: 34.99, Stock: 55, Category: "accessories", Brand: "AmazonBasics", Featured: false, Rating: 4.1, CreatedAt: baseTime.Add(216 * time.Hour)},
	}
}

func TestMemoryExecutor_BasicComparisons(t *testing.T) {
	data := getTestData()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(data, opts)
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
		checkFirst    func(*testing.T, []Product)
	}{
		{
			name:          "equals",
			query:         "brand = Logitech",
			expectedCount: 2,
			checkFirst: func(t *testing.T, products []Product) {
				assert.Equal(t, "Logitech", products[0].Brand)
			},
		},
		{
			name:          "not equals",
			query:         "category != electronics",
			expectedCount: 5,
		},
		{
			name:          "greater than",
			query:         "price > 50",
			expectedCount: 3, // Keyboard(89.99), Headphones(199.99), Webcam(69.99)
		},
		{
			name:          "greater than or equal",
			query:         "price >= 49.99",
			expectedCount: 4, // Speaker(49.99), Webcam(69.99), Keyboard(89.99), Headphones(199.99)
		},
		{
			name:          "less than",
			query:         "price < 30",
			expectedCount: 4,
		},
		{
			name:          "less than or equal",
			query:         "stock <= 50",
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parser.NewParser(tt.query)
			require.NoError(t, err)
			q, err := p.Parse()
			require.NoError(t, err)

			var products []Product
			result, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(products))
			assert.Equal(t, int64(tt.expectedCount), result.TotalItems)

			if tt.checkFirst != nil && len(products) > 0 {
				tt.checkFirst(t, products)
			}
		})
	}
}

func TestMemoryExecutor_StringMatching(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "LIKE with wildcard",
			query:         `name LIKE "Wireless%"`,
			expectedCount: 3,
		},
		{
			name:          "LIKE with middle wildcard",
			query:         `name LIKE "%Mouse%"`,
			expectedCount: 2,
		},
		{
			name:          "NOT LIKE",
			query:         `name NOT LIKE "%Wireless%"`,
			expectedCount: 7,
		},
		{
			name:          "CONTAINS",
			query:         `description CONTAINS "USB"`,
			expectedCount: 2,
		},
		{
			name:          "ICONTAINS case insensitive",
			query:         `description ICONTAINS "usb"`,
			expectedCount: 2,
		},
		{
			name:          "STARTS_WITH",
			query:         `name STARTS_WITH "Wireless"`,
			expectedCount: 3,
		},
		{
			name:          "ENDS_WITH",
			query:         `name ENDS_WITH "Mouse"`,
			expectedCount: 1,
		},
		{
			name:          "REGEX",
			query:         `name REGEX "^[A-Z].*Mouse$"`,
			expectedCount: 1, // Only "Wireless Mouse" matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := parser.NewParser(tt.query)
			q, _ := p.Parse()

			var products []Product
			result, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(products), "Query: %s", tt.query)
			assert.Equal(t, int64(tt.expectedCount), result.TotalItems)
		})
	}
}

func TestMemoryExecutor_ArrayOperations(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "IN with strings",
			query:         `brand IN [Logitech, Sony, JBL]`,
			expectedCount: 4,
		},
		{
			name:          "NOT IN",
			query:         `category NOT IN [electronics]`,
			expectedCount: 5,
		},
		{
			name:          "IN with numbers",
			query:         `id IN [1, 3, 5, 7, 9]`,
			expectedCount: 5,
		},
		{
			name:          "IN with single value",
			query:         `brand IN [Anker]`,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := parser.NewParser(tt.query)
			q, _ := p.Parse()

			var products []Product
			_, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(products))
		})
	}
}

func TestMemoryExecutor_LogicalOperators(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "AND simple",
			query:         "category = electronics and featured = true",
			expectedCount: 3,
		},
		{
			name:          "OR simple",
			query:         "brand = Logitech or brand = Sony",
			expectedCount: 3,
		},
		{
			name:          "complex with parentheses",
			query:         "(category = electronics and price < 100) or featured = true",
			expectedCount: 6, // 4 electronics <100 + 2 additional featured (Headphones, Charger)
		},
		{
			name:          "nested parentheses",
			query:         "((brand = Anker or brand = Logitech) and price < 50) or rating >= 4.8",
			expectedCount: 6, // 4 (Anker/Logitech <50) + 2 (rating >=4.8: Keyboard, Headphones)
		},
		{
			name:          "multiple AND",
			query:         "category = electronics and price > 50 and featured = true",
			expectedCount: 1, // Only Headphones (electronics, 199.99, featured)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := parser.NewParser(tt.query)
			q, _ := p.Parse()

			var products []Product
			_, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(products), "Query: %s", tt.query)
		})
	}
}

func TestMemoryExecutor_BareSearch(t *testing.T) {
	data := getTestData()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSearchField = "name"
	executor := NewExecutor(data, opts)
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "single bare term",
			query:         "Wireless",
			expectedCount: 3,
		},
		{
			name:          "multiple bare terms",
			query:         "Wireless Mouse",
			expectedCount: 1,
		},
		{
			name:          "bare with field filter",
			query:         "Wireless price < 100",
			expectedCount: 2, // Wireless Mouse(29.99), Wireless Charger(39.99)
		},
		{
			name:          "quoted bare term",
			query:         `"USB"`,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := parser.NewParser(tt.query)
			q, _ := p.Parse()

			var products []Product
			_, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(products), "Query: %s", tt.query)
		})
	}
}

func TestMemoryExecutor_Pagination(t *testing.T) {
	data := getTestData()
	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(data, opts)
	ctx := context.Background()

	t.Run("first page", func(t *testing.T) {
		p, _ := parser.NewParser("page_size = 3 sort_by = id")
		q, _ := p.Parse()

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)

		assert.Equal(t, 3, len(products))
		assert.Equal(t, int64(10), result.TotalItems)
		assert.Equal(t, 1, result.ShowingFrom)
		assert.Equal(t, 3, result.ShowingTo)
		assert.NotEmpty(t, result.NextPageCursor)
		assert.Empty(t, result.PrevPageCursor)
		assert.Equal(t, 1, products[0].ID)
	})

	t.Run("second page using cursor", func(t *testing.T) {
		// Get first page
		p, _ := parser.NewParser("page_size = 3 sort_by = id")
		q, _ := p.Parse()
		var products1 []Product
		result1, _ := executor.Execute(ctx, q, "", &products1)

		// Get second page
		var products2 []Product
		result2, err := executor.Execute(ctx, q, result1.NextPageCursor, &products2)
		require.NoError(t, err)

		assert.Equal(t, 3, len(products2))
		assert.Equal(t, 4, result2.ShowingFrom)
		assert.Equal(t, 6, result2.ShowingTo)
		assert.NotEmpty(t, result2.NextPageCursor)
		assert.NotEmpty(t, result2.PrevPageCursor)
		assert.Equal(t, 4, products2[0].ID)
	})

	t.Run("previous page", func(t *testing.T) {
		// Get to page 2
		p, _ := parser.NewParser("page_size = 3 sort_by = id")
		q, _ := p.Parse()
		var products []Product
		result, _ := executor.Execute(ctx, q, "", &products)
		result, _ = executor.Execute(ctx, q, result.NextPageCursor, &products)

		// Go back to page 1
		products = []Product{}
		prevResult, err := executor.Execute(ctx, q, result.PrevPageCursor, &products)
		require.NoError(t, err)

		assert.Equal(t, 3, len(products))
		assert.Equal(t, 1, prevResult.ShowingFrom)
		assert.Equal(t, 1, products[0].ID)
	})
}

func TestMemoryExecutor_Sorting(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	tests := []struct {
		name       string
		query      string
		checkOrder func(*testing.T, []Product)
	}{
		{
			name:  "sort by price ascending",
			query: "sort_by = price sort_order = asc",
			checkOrder: func(t *testing.T, products []Product) {
				assert.LessOrEqual(t, products[0].Price, products[1].Price)
				assert.Equal(t, 9.99, products[0].Price)
			},
		},
		{
			name:  "sort by price descending",
			query: "sort_by = price sort_order = desc",
			checkOrder: func(t *testing.T, products []Product) {
				assert.GreaterOrEqual(t, products[0].Price, products[1].Price)
				assert.Equal(t, 199.99, products[0].Price)
			},
		},
		{
			name:  "sort by name",
			query: "sort_by = name sort_order = asc",
			checkOrder: func(t *testing.T, products []Product) {
				assert.LessOrEqual(t, products[0].Name, products[1].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := parser.NewParser(tt.query)
			q, _ := p.Parse()

			var products []Product
			_, err := executor.Execute(ctx, q, "", &products)
			require.NoError(t, err)

			if tt.checkOrder != nil && len(products) >= 2 {
				tt.checkOrder(t, products)
			}
		})
	}
}

func TestMemoryExecutor_MapData(t *testing.T) {
	// Test with map data instead of structs
	data := []map[string]interface{}{
		{"id": 1, "name": "Product A", "price": 10.0, "category": "electronics"},
		{"id": 2, "name": "Product B", "price": 20.0, "category": "accessories"},
		{"id": 3, "name": "Product C", "price": 30.0, "category": "electronics"},
	}

	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	t.Run("filter maps", func(t *testing.T) {
		p, _ := parser.NewParser("category = electronics")
		q, _ := p.Parse()

		var results []map[string]interface{}
		_, err := executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})

	t.Run("sort maps", func(t *testing.T) {
		p, _ := parser.NewParser("sort_by = price sort_order = desc")
		q, _ := p.Parse()

		var results []map[string]interface{}
		_, err := executor.Execute(ctx, q, "", &results)
		require.NoError(t, err)
		assert.Equal(t, 30.0, results[0]["price"])
	})
}

func TestMemoryExecutor_EdgeCases(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	t.Run("empty results", func(t *testing.T) {
		p, _ := parser.NewParser("price > 1000")
		q, _ := p.Parse()

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 0, len(products))
		assert.Equal(t, int64(0), result.TotalItems)
	})

	t.Run("empty query returns all", func(t *testing.T) {
		p, _ := parser.NewParser("")
		q, _ := p.Parse()

		var products []Product
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 10, len(products))
		assert.Equal(t, int64(10), result.TotalItems)
	})

	t.Run("empty data source", func(t *testing.T) {
		emptyData := []Product{}
		emptyExecutor := NewExecutor(emptyData, query.DefaultExecutorOptions())

		p, _ := parser.NewParser("brand = Logitech")
		q, _ := p.Parse()

		var products []Product
		result, err := emptyExecutor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 0, len(products))
		assert.Equal(t, int64(0), result.TotalItems)
	})

	t.Run("case insensitive field names", func(t *testing.T) {
		p, _ := parser.NewParser("BRAND = Logitech")
		q, _ := p.Parse()

		var products []Product
		_, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 2, len(products))
	})

	t.Run("nonexistent field", func(t *testing.T) {
		p, _ := parser.NewParser("nonexistent_field = value")
		q, _ := p.Parse()

		var products []Product
		_, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 0, len(products))
	})
}

func TestMemoryExecutor_ComplexQueries(t *testing.T) {
	data := getTestData()
	executor := NewExecutor(data, query.DefaultExecutorOptions())
	ctx := context.Background()

	t.Run("e-commerce search", func(t *testing.T) {
		query := `page_size = 5 
				  (category = electronics and price < 100) 
				  or (featured = true and rating >= 4.5)
				  sort_by = price`

		p, _ := parser.NewParser(query)
		q, _ := p.Parse()

		var products []Product
		_, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Greater(t, len(products), 0)
		assert.LessOrEqual(t, len(products), 5)

		// Verify sorting
		if len(products) >= 2 {
			assert.LessOrEqual(t, products[0].Price, products[1].Price)
		}
	})

	t.Run("inventory filter", func(t *testing.T) {
		query := `stock < 50 
				  and category IN [electronics, accessories]
				  and featured = false`

		p, _ := parser.NewParser(query)
		q, _ := p.Parse()

		var products []Product
		_, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)

		for _, product := range products {
			assert.Less(t, product.Stock, 50)
			assert.Contains(t, []string{"electronics", "accessories"}, product.Category)
			assert.False(t, product.Featured)
		}
	})
}
