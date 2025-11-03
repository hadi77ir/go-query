# Performance Guide

Optimization tips and best practices for maximum performance with go-query.

## Table of Contents

1. [Parser Cache](#parser-cache)
2. [Database Indexing](#database-indexing)
3. [Query Optimization](#query-optimization)
4. [Executor Configuration](#executor-configuration)
5. [Memory Usage](#memory-usage)

## Parser Cache

**Most Important**: Always use `ParserCache` in production. This is the single biggest performance improvement.

### Why Cache?

- **Parsing is expensive**: Complex queries can take milliseconds to parse
- **Repeated queries**: Production environments often repeat the same queries
- **Cache hits**: Return in microseconds vs milliseconds

### Recommended Setup

```go
// Create cache once at application startup
var parserCache = parser.NewParserCache(100) // Shared across requests

// Use in handlers/endpoints
func handleRequest(w http.ResponseWriter, r *http.Request) {
    queryStr := r.URL.Query().Get("q")
    q, err := parserCache.Parse(queryStr) // Fast!
    // ...
}
```

### Cache Sizing

- **Small apps** (< 1000 req/min): 50 entries
- **Medium apps** (1000-10000 req/min): 100 entries
- **Large apps** (> 10000 req/min): 200-500 entries

Monitor cache stats:

```go
stats := cache.GetStats()
fmt.Printf("Cache size: %d, Total accesses: %d\n", 
    stats.Size, stats.TotalAccess)
```

## Database Indexing

Proper indexing dramatically improves query performance.

### MongoDB Indexing

```javascript
// Text search index
db.products.createIndex({ name: "text" })

// Compound indexes for common queries
db.products.createIndex({ category: 1, price: 1 })
db.products.createIndex({ status: 1, created_at: -1 })

// Single field indexes
db.users.createIndex({ email: 1 })
db.orders.createIndex({ user_id: 1 })
```

### PostgreSQL Indexing

```sql
-- Text search index
CREATE INDEX idx_products_name ON products USING gin(to_tsvector('english', name));

-- Composite indexes
CREATE INDEX idx_products_category_price ON products(category, price);
CREATE INDEX idx_users_status_created ON users(status, created_at DESC);

-- Single field indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_user_id ON orders(user_id);
```

### MySQL Indexing

```sql
-- Full-text search
CREATE FULLTEXT INDEX idx_products_name ON products(name);

-- Composite indexes
CREATE INDEX idx_products_category_price ON products(category, price);
CREATE INDEX idx_users_status_created ON users(status, created_at DESC);
```

### Index Strategy

1. **Index fields used in WHERE clauses**
2. **Index fields used for sorting** (`sort_by`)
3. **Create compound indexes** for common query patterns
4. **Monitor query performance** and add indexes as needed

## Query Optimization

### Prefer Specific Fields Over Bare Search

```go
// Slower: Bare search (may not use index efficiently)
"wireless mouse"

// Faster: Specific field (can use index)
"name CONTAINS wireless and name CONTAINS mouse"
```

### Use IN Instead of Multiple ORs

```go
// Slower: Multiple OR conditions
"(category = electronics or category = computers or category = accessories)"

// Faster: Single IN operation
"category IN [electronics, computers, accessories]"
```

### Limit Results Early

```go
// Always specify page_size
"page_size = 20 status = active"

// Instead of fetching all and filtering client-side
```

### Avoid Complex Regex When Possible

```go
// Slower: Regex may not use index
"email REGEX \"^[a-z]+@[a-z]+\\.com$\""

// Faster: Use LIKE or STARTS_WITH/ENDS_WITH
"email LIKE \"%@%.com\""
```

## Executor Configuration

### Page Size Limits

Set appropriate limits to prevent memory issues:

```go
opts := &query.ExecutorOptions{
    MaxPageSize:     100,  // Prevent large result sets
    DefaultPageSize: 20,   // Reasonable default
}
```

### Field Restrictions

Restricting fields improves security and can help with performance:

```go
opts := &query.ExecutorOptions{
    AllowedFields: []string{"id", "name", "price", "status"},
    // Only these fields can be queried
}
```

## Memory Usage

### Large Result Sets

For large datasets, use cursor pagination:

```go
// Good: Cursor-based pagination
"page_size = 20 sort_by = created_at"

// Avoid: Offset-based (doesn't scale)
// Don't use large offsets
```

### Memory Executor

The memory executor loads all data into memory:

- ✅ **Good for**: Small datasets (< 10,000 items), testing
- ❌ **Avoid for**: Large datasets (> 50,000 items)

For large datasets, use database executors (MongoDB/GORM) which can use indexes.

## Performance Monitoring

### Cache Statistics

```go
stats := cache.GetStats()
fmt.Printf("Cache size: %d\n", stats.Size)
fmt.Printf("Total accesses: %d\n", stats.TotalAccess)
```

### Query Timing

```go
start := time.Now()
q, err := cache.Parse(queryStr)
parseTime := time.Since(start)

start = time.Now()
result, err := executor.Execute(ctx, q, "", &results)
execTime := time.Since(start)

fmt.Printf("Parse: %v, Execute: %v\n", parseTime, execTime)
```

## Best Practices Summary

1. ✅ **Always use ParserCache** in production
2. ✅ **Index frequently queried fields**
3. ✅ **Use specific field queries** when possible
4. ✅ **Set reasonable page size limits**
5. ✅ **Use cursor pagination** for large datasets
6. ✅ **Prefer IN over multiple ORs**
7. ✅ **Avoid complex regex** when simpler operators work
8. ✅ **Monitor cache effectiveness** with stats

## See Also

- [Configuration Guide](CONFIGURATION.md) - Executor configuration
- [Query Syntax Guide](QUERY_SYNTAX.md) - Query language reference
- [Examples](EXAMPLES.md) - Usage examples

