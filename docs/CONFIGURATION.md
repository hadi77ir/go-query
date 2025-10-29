# Configuration Guide

Complete guide to configuring go-query executors and options.

## Table of Contents

1. [Executor Options](#executor-options)
2. [Default Search Field](#default-search-field)
3. [Parser Cache](#parser-cache)
4. [Field Restrictions](#field-restrictions)
5. [Database-Specific Settings](#database-specific-settings)

## Executor Options

All executors accept `ExecutorOptions` for configuration:

```go
opts := &query.ExecutorOptions{
    MaxPageSize:        100,       // Maximum allowed page size
    DefaultPageSize:    10,        // Default page size when not specified
    DefaultSortField:   "_id",     // Default field to sort by
    DefaultSortOrder:   query.SortOrderAsc,  // Default sort order
    AllowRandomOrder:   true,     // Allow random ordering
    DefaultSearchField: "name",    // Field for bare search terms
    AllowedFields:      nil,       // Whitelist of allowed fields (nil = all allowed)
    DisableRegex:       false,     // Disable REGEX operator
    RandomFunctionName: "RANDOM()", // SQL random function (GORM only)
    IDFieldName:        "",        // Custom ID field name for cursors
}
```

### Using Default Options

```go
// Use defaults (recommended for most cases)
opts := query.DefaultExecutorOptions()
executor := mongodb.NewExecutor(collection, opts)

// Defaults:
// - MaxPageSize: 100
// - DefaultPageSize: 10
// - DefaultSortField: "_id"
// - DefaultSortOrder: SortOrderAsc
// - AllowRandomOrder: true
// - DefaultSearchField: "name"
// - AllowedFields: nil (all fields allowed)
// - DisableRegex: false
// - RandomFunctionName: "RANDOM()"
// - IDFieldName: "" (executor-specific default)
```

## Default Search Field

Configure which field bare search terms query:

```go
opts := &query.ExecutorOptions{
    DefaultSearchField: "name",  // Bare terms search this field
}

executor := mongodb.NewExecutor(collection, opts)
```

Now queries like `"hello world"` will search the `name` field using CONTAINS.

### Examples

```go
// Default: searches "name" field
opts := query.DefaultExecutorOptions()
// Query: "wireless mouse" searches name field

// Custom: search "email" field
opts.DefaultSearchField = "email"
// Query: "@gmail.com" searches email field

// Custom: search "title" field
opts.DefaultSearchField = "title"
// Query: "javascript tutorial" searches title field
```

## Parser Cache

**Recommended for production**: Use `ParserCache` to cache parsed queries for maximum performance.

```go
import "github.com/hadi77ir/go-query/parser"

// Create cache (recommended: 50-100 entries)
cache := parser.NewParserCache(100)

// Parse queries - cache automatically handles hits/misses
q, err := cache.Parse("name = test AND age > 18")
```

### Cache Configuration

```go
// Cache 100 queries (recommended for production)
cache := parser.NewParserCache(100)

// Cache 50 queries (good for smaller applications)
cache := parser.NewParserCache(50)

// Disable caching (parse directly every time)
cache := parser.NewParserCache(0)
```

### Cache Benefits

- **Cache Hit**: Returns instantly (microseconds)
- **Cache Miss**: Parses normally, then caches for future use
- **Thread-Safe**: Multiple goroutines can use the same cache
- **Smart Eviction**: Keeps frequently used and recently added queries

See [Parser Cache](FEATURES.md#parser-cache) in FEATURES.md for complete documentation.

## Field Restrictions

Restrict which fields can be queried for security:

```go
opts := &query.ExecutorOptions{
    AllowedFields: []string{"id", "name", "email", "status"},
}

executor := mongodb.NewExecutor(collection, opts)

// These queries work:
// "name = test" ✅
// "email = user@example.com" ✅
// "status = active" ✅

// This query fails:
// "password = secret" ❌ Error: field not allowed
```

### Empty List = All Fields Allowed

```go
opts := &query.ExecutorOptions{
    AllowedFields: []string{},  // Empty = all fields allowed
}
// or
opts := &query.ExecutorOptions{
    AllowedFields: nil,  // nil = all fields allowed
}
```

## Database-Specific Settings

### GORM: Random Function Name

Different databases use different random functions:

```go
opts := query.DefaultExecutorOptions()

// MySQL
opts.RandomFunctionName = "RAND()"

// PostgreSQL, SQLite (default)
opts.RandomFunctionName = "RANDOM()"

executor := gorm.NewExecutor(db, &Product{}, opts)
```

### Custom ID Field Name

Configure custom ID field names for cursor pagination:

```go
opts := query.DefaultExecutorOptions()

// GORM: Use "product_id" instead of "id"
opts.IDFieldName = "product_id"

// MongoDB: Use "custom_id" instead of "_id"
opts.IDFieldName = "custom_id"

executor := gorm.NewExecutor(db, &Product{}, opts)
```

**Note**: The ID field name should match the actual database column/document field name.

## Complete Configuration Example

```go
import (
    "github.com/hadi77ir/go-query/query"
    "github.com/hadi77ir/go-query/executors/mongodb"
)

// Create custom options
opts := &query.ExecutorOptions{
    // Pagination
    MaxPageSize:     200,
    DefaultPageSize: 20,
    
    // Sorting
    DefaultSortField:   "created_at",
    DefaultSortOrder:   query.SortOrderDesc,
    AllowRandomOrder:   true,
    
    // Search
    DefaultSearchField: "title",
    
    // Security
    AllowedFields: []string{
        "id", "title", "content", "status", 
        "created_at", "updated_at",
    },
    
    // Features
    DisableRegex: false,
}

executor := mongodb.NewExecutor(collection, opts)
```

## See Also

- [Features Guide](FEATURES.md) - Advanced features and capabilities
- [Security Guide](SECURITY.md) - Security best practices
- [Performance Guide](PERFORMANCE.md) - Optimization tips

