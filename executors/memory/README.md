# Memory Executor

An in-memory executor for go-query that allows you to filter and query Go slices and maps without needing a database.

## Features

- ✅ **Zero Dependencies**: No database required
- ✅ **Works with Structs**: Query slices of any struct type
- ✅ **Works with Maps**: Query slices of `map[string]interface{}`
- ✅ **All Operators Supported**: Same powerful query language as database executors
- ✅ **Pagination**: Full cursor-based pagination support
- ✅ **Sorting**: Sort by any field, ascending or descending
- ✅ **Case-Insensitive Fields**: Automatically matches field names
- ✅ **Tag Support**: Respects `json` and `bson` struct tags
- ✅ **Perfect for Testing**: Test your queries without a database

## Installation

```bash
go get github.com/hadi77ir/go-query/executors/memory
```

## Quick Start

### With Structs

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/hadi77ir/go-query/executors/memory"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
)

type Product struct {
    ID       int
    Name     string
    Price    float64
    Category string
    Featured bool
}

func main() {
    // Your in-memory data
    products := []Product{
        {ID: 1, Name: "Wireless Mouse", Price: 29.99, Category: "electronics", Featured: true},
        {ID: 2, Name: "USB Cable", Price: 9.99, Category: "accessories", Featured: false},
        {ID: 3, Name: "Keyboard", Price: 89.99, Category: "electronics", Featured: true},
    }

    // Create executor
    executor := memory.NewExecutor(products, query.DefaultExecutorOptions())

    // Use ParserCache for better performance (recommended)
    cache := parser.NewParserCache(100)
    q, _ := cache.Parse("category = electronics and price < 50")

    var results []Product
    result, _ := executor.Execute(context.Background(), q, &results)

    fmt.Printf("Found %d products\n", result.TotalItems)
    for _, product := range results {
        fmt.Printf("- %s: $%.2f\n", product.Name, product.Price)
    }
}
```

### With Maps

```go
data := []map[string]interface{}{
    {"id": 1, "name": "Product A", "price": 10.0, "active": true},
    {"id": 2, "name": "Product B", "price": 20.0, "active": false},
    {"id": 3, "name": "Product C", "price": 30.0, "active": true},
}

executor := memory.NewExecutor(data, query.DefaultExecutorOptions())

// Use ParserCache for better performance (recommended)
cache := parser.NewParserCache(100)
q, _ := cache.Parse("active = true and price > 15")

var results []map[string]interface{}
executor.Execute(context.Background(), q, &results)
```

## Use Cases

### 1. Testing

Test your query logic without spinning up a database:

```go
func TestProductSearch(t *testing.T) {
    // Mock data
    testProducts := []Product{
        {Name: "Test Product", Price: 99.99},
    }
    
    executor := memory.NewExecutor(testProducts, query.DefaultExecutorOptions())
    
    // Test your queries
    // Use ParserCache for better performance (recommended)
    cache := parser.NewParserCache(100)
    q, _ := cache.Parse("price < 100")
    
    var results []Product
    _, err := executor.Execute(context.Background(), q, &results)
    require.NoError(t, err)
    assert.Equal(t, 1, len(results))
}
```

### 2. In-Memory Filtering

Filter slices with complex logic:

```go
users := loadUsersFromFile()
executor := memory.NewExecutor(users, query.DefaultExecutorOptions())

// Complex filtering
// Use ParserCache for better performance (recommended)
cache := parser.NewParserCache(100)
q, _ := cache.Parse(`
    (role = admin or role = moderator) 
    and active = true 
    and last_login >= 2024-01-01
`)

var activeAdmins []User
executor.Execute(ctx, q, &activeAdmins)
```

### 3. API Response Filtering

Filter API results before returning:

```go
func GetProducts(w http.ResponseWriter, r *http.Request) {
    allProducts := fetchAllProducts()
    
    // User-provided filter
    filterQuery := r.URL.Query().Get("filter")
    
    // Use ParserCache for better performance (recommended)
    cache := parser.NewParserCache(100)
    q, _ := cache.Parse(filterQuery)
    
    executor := memory.NewExecutor(allProducts, query.DefaultExecutorOptions())
    
    var filtered []Product
    result, _ := executor.Execute(r.Context(), q, &filtered)
    
    json.NewEncoder(w).Write(result)
}
```

## Query Examples

All the same operators work as with database executors:

```go
// Comparisons
"price > 50"
"stock <= 10"
"category != accessories"

// String matching
"name LIKE \"Wireless%\""
"description CONTAINS \"fast\""
"email ENDS_WITH \"@example.com\""
"code REGEX \"^[A-Z]{3}[0-9]{3}$\""

// Arrays
"category IN [electronics, computers]"
"status NOT IN [deleted, archived]"

// Logical
"(featured = true and price < 100) or rating >= 4.5"

// Bare search
"wireless mouse"  // Searches default field

// With pagination
"page_size = 20 sort_by = price featured = true"
```

## Features

### Case-Insensitive Field Names

The executor automatically matches field names case-insensitively:

```go
type Product struct {
    ProductName string
}

// All of these work:
"ProductName = test"
"productname = test"
"PRODUCTNAME = test"
```

### Struct Tag Support

Respects `json` and `bson` tags:

```go
type Product struct {
    ID   int    `json:"product_id" bson:"_id"`
    Name string `json:"product_name"`
}

// Can query using tag names:
"product_id = 123"
"product_name LIKE \"Wireless%\""
```

### Pagination

Full cursor-based pagination:

```go
// First page
// Use ParserCache for better performance (recommended)
cache := parser.NewParserCache(100)
q, _ := cache.Parse("page_size = 10 sort_by = price")

var page1 []Product
result1, _ := executor.Execute(ctx, q, &page1)

// Next page
q.Cursor = result1.NextPageCursor
var page2 []Product
result2, _ := executor.Execute(ctx, q, &page2)

// Previous page
q.Cursor = result2.PrevPageCursor
var prevPage []Product
executor.Execute(ctx, q, &prevPage)
```

### Sorting

Sort by any field:

```go
"sort_by = price sort_order = asc"   // Ascending
"sort_by = created_at sort_order = desc"  // Descending
```

## Performance

The memory executor:
- **Filters** in O(n) time where n is the number of items
- **Sorts** in O(n log n) time
- **No allocations** for the filtering pass
- **Minimal copying** - only matched items are copied to results

For large datasets (>10,000 items), consider using a database executor instead.

## Limitations

1. **No Indexes**: Unlike databases, there's no index optimization
2. **Memory Usage**: All data must fit in memory
3. **Limited Aggregation**: No COUNT, SUM, AVG, etc.
4. **No Joins**: Can only query a single slice at a time

## Comparison with Database Executors

| Feature | Memory | MongoDB | GORM |
|---------|--------|---------|------|
| Setup | ✅ None | ⚠️ MongoDB instance | ⚠️ Database connection |
| Performance (small datasets) | ✅ Fast | ⚠️ Network overhead | ⚠️ Network overhead |
| Performance (large datasets) | ❌ Slow | ✅ Indexed | ✅ Indexed |
| Testing | ✅ Perfect | ⚠️ Requires testcontainers | ⚠️ Requires DB |
| Type Safety | ✅ Full | ✅ Full | ✅ Full |
| All Operators | ✅ Yes | ✅ Yes | ✅ Yes |

## Example: Complete Application

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/hadi77ir/go-query/executors/memory"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
)

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Category    string  `json:"category"`
    InStock     bool    `json:"in_stock"`
}

var products = []Product{
    {ID: 1, Name: "Laptop", Price: 999.99, Category: "electronics", InStock: true},
    {ID: 2, Name: "Mouse", Price: 29.99, Category: "accessories", InStock: true},
    {ID: 3, Name: "Keyboard", Price: 79.99, Category: "accessories", InStock: false},
    {ID: 4, Name: "Monitor", Price: 299.99, Category: "electronics", InStock: true},
}

func main() {
    http.HandleFunc("/products", searchProducts)
    http.ListenAndServe(":8080", nil)
}

func searchProducts(w http.ResponseWriter, r *http.Request) {
    // Get filter from query parameter
    // e.g., /products?filter=in_stock = true and price < 100
    filterStr := r.URL.Query().Get("filter")
    if filterStr == "" {
        filterStr = "in_stock = true" // Default filter
    }
    
    // Parse query using cache (recommended for performance)
    cache := parser.NewParserCache(100)
    q, err := cache.Parse(filterStr)
    if err != nil {
        http.Error(w, "Invalid filter syntax", http.StatusBadRequest)
        return
    }
    
    // Execute on in-memory data
    executor := memory.NewExecutor(products, query.DefaultExecutorOptions())
    
    var results []Product
    result, err := executor.Execute(r.Context(), q, &results)
    if err != nil {
        http.Error(w, "Query execution failed", http.StatusInternalServerError)
        return
    }
    
    // Return results
    response := map[string]interface{}{
        "total":   result.TotalItems,
        "showing": map[string]int{
            "from": result.ShowingFrom,
            "to":   result.ShowingTo,
        },
        "data": results,
        "next_cursor": result.NextPageCursor,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Try it:
// curl "http://localhost:8080/products?filter=price < 100"
// curl "http://localhost:8080/products?filter=category = electronics and in_stock = true"
```

## Testing

```bash
go test -v
```

The test suite includes:
- ✅ All comparison operators
- ✅ All string matching operators
- ✅ Array operations
- ✅ Logical operators
- ✅ Bare search terms
- ✅ Pagination (forward and backward)
- ✅ Sorting
- ✅ Map data
- ✅ Edge cases

## License

Apache 2.0 (same as parent project)

---

**Perfect for**: Testing, prototyping, small datasets, API filtering, configuration filtering, local development

