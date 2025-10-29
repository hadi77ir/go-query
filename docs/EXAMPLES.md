# Examples Guide

Real-world usage examples for go-query.

## Table of Contents

1. [E-Commerce Search](#e-commerce-search)
2. [User Management](#user-management)
3. [Content Search](#content-search)
4. [API Endpoints](#api-endpoints)
5. [Testing](#testing)

## E-Commerce Search

Product search with multiple criteria:

```go
import (
    "context"
    "github.com/hadi77ir/go-query/executors/mongodb"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
)

// Setup
cache := parser.NewParserCache(100)
executor := mongodb.NewExecutor(collection, query.DefaultExecutorOptions())

// Complex product search
queryStr := `
  wireless mouse
  price >= 10 price <= 50
  (brand IN [logitech, microsoft, razer] or featured = true)
  rating >= 4
`

q, _ := cache.Parse(queryStr)
var products []Product
result, _ := executor.Execute(ctx, q, &products)
```

### With Pagination

```go
queryStr := `
  page_size = 20
  sort_by = price
  sort_order = desc
  wireless mouse
  price >= 10 price <= 50
  (brand IN [logitech, microsoft] or featured = true)
`

q, _ := cache.Parse(queryStr)
var products []Product
result, _ := executor.Execute(ctx, q, &products)

// Navigate to next page
if result.NextPageCursor != "" {
    q.Cursor = result.NextPageCursor
    var nextPage []Product
    executor.Execute(ctx, q, &nextPage)
}
```

## User Management

### Basic User Search

```go
// Find users by name and status
queryStr := `john doe status = active role IN [admin, moderator]`

cache := parser.NewParserCache(100)
q, _ := cache.Parse(queryStr)

var users []User
result, _ := executor.Execute(ctx, q, &users)
```

### Search by Email

```go
// Configure to search email field
opts := &query.ExecutorOptions{
    DefaultSearchField: "email",
}
executor := mongodb.NewExecutor(collection, opts)

// Find active Gmail users
queryStr := `@gmail.com status = active`
q, _ := cache.Parse(queryStr)

var users []User
executor.Execute(ctx, q, &users)
```

### Advanced User Filtering

```go
queryStr := `
  (role = admin or role = moderator) 
  and active = true 
  and last_login >= 2024-01-01
  and email CONTAINS "@company.com"
`

q, _ := cache.Parse(queryStr)
var admins []User
executor.Execute(ctx, q, &admins)
```

## Content Search

### Blog Post Search

```go
queryStr := `
  "machine learning" "neural networks"
  published = true
  created_at >= 2024-01-01
  category IN [tech, ai, science]
`

q, _ := cache.Parse(queryStr)
var posts []Post
result, _ := executor.Execute(ctx, q, &posts)
```

### Content Filtering with Tags

```go
queryStr := `
  page_size = 10
  sort_by = published_at
  sort_order = desc
  (tags CONTAINS "tutorial" or tags CONTAINS "guide")
  and published = true
  and featured = true
`

q, _ := cache.Parse(queryStr)
var content []Content
executor.Execute(ctx, q, &content)
```

## API Endpoints

### HTTP Handler Example

```go
package main

import (
    "encoding/json"
    "net/http"
    
    "github.com/hadi77ir/go-query/executors/memory"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
)

var parserCache = parser.NewParserCache(100) // Shared cache

func searchProducts(w http.ResponseWriter, r *http.Request) {
    // Get filter from query parameter
    filterStr := r.URL.Query().Get("filter")
    if filterStr == "" {
        filterStr = "in_stock = true" // Default filter
    }
    
    // Parse using cache
    q, err := parserCache.Parse(filterStr)
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
```

### REST API Example

```go
// GET /api/products?filter=price < 100&page_size=20
func ProductsHandler(w http.ResponseWriter, r *http.Request) {
    filter := r.URL.Query().Get("filter")
    pageSize := r.URL.Query().Get("page_size")
    
    // Build query string
    queryStr := filter
    if pageSize != "" {
        queryStr = "page_size = " + pageSize + " " + queryStr
    }
    
    // Parse and execute
    q, err := parserCache.Parse(queryStr)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    var products []Product
    result, err := executor.Execute(r.Context(), q, &products)
    // ... handle response
}
```

## Testing

### Unit Testing with Memory Executor

```go
func TestProductSearch(t *testing.T) {
    // Mock data
    testProducts := []Product{
        {Name: "Test Product", Price: 99.99, InStock: true},
        {Name: "Another Product", Price: 149.99, InStock: false},
    }
    
    executor := memory.NewExecutor(testProducts, query.DefaultExecutorOptions())
    cache := parser.NewParserCache(100)
    
    // Test your queries
    q, _ := cache.Parse("price < 100 and in_stock = true")
    
    var results []Product
    result, err := executor.Execute(context.Background(), q, &results)
    require.NoError(t, err)
    assert.Equal(t, 1, len(results))
    assert.Equal(t, "Test Product", results[0].Name)
}
```

### Integration Testing

```go
func TestUserSearch(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    executor := gorm.NewExecutor(db, &User{}, query.DefaultExecutorOptions())
    cache := parser.NewParserCache(100)
    
    // Insert test data
    seedTestData(t, db)
    
    // Test query
    q, _ := cache.Parse("status = active and role = admin")
    var users []User
    result, err := executor.Execute(ctx, q, &users)
    
    require.NoError(t, err)
    assert.Greater(t, len(users), 0)
    for _, user := range users {
        assert.Equal(t, "active", user.Status)
        assert.Equal(t, "admin", user.Role)
    }
}
```

## Advanced Examples

### Date Range Queries

```go
queryStr := `
  created_at >= 2024-01-01
  and created_at <= 2024-12-31
  and status IN [published, approved]
`

q, _ := cache.Parse(queryStr)
var items []Item
executor.Execute(ctx, q, &items)
```

### Complex Nested Queries

```go
queryStr := `
  (
    (category = electronics and price < 500) 
    or 
    (category = accessories and featured = true)
  )
  and 
  (rating >= 4 or reviews > 100)
  and 
  status = active
`

q, _ := cache.Parse(queryStr)
var products []Product
executor.Execute(ctx, q, &products)
```

### Mixed Search Types

```go
queryStr := `
  "wireless" "mouse"
  price >= 10 price <= 50
  brand IN [logitech, microsoft]
  rating >= 4
  (featured = true or reviews > 50)
`

q, _ := cache.Parse(queryStr)
var products []Product
executor.Execute(ctx, q, &products)
```

## See Also

- [Query Syntax Guide](QUERY_SYNTAX.md) - Complete syntax reference
- [Configuration Guide](CONFIGURATION.md) - Configuration options
- [Performance Guide](PERFORMANCE.md) - Optimization tips

