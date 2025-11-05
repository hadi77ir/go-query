# Configuration Guide

Complete guide to configuring go-query executors and options.

## Table of Contents

1. [Executor Options](#executor-options)
2. [Default Search Field](#default-search-field)
3. [Parser Cache](#parser-cache)
4. [Field Restrictions](#field-restrictions)
5. [Value Converter](#value-converter)
6. [Database-Specific Settings](#database-specific-settings)

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
    ValueConverter:     nil,       // Value converter function (see Value Converter section)
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
// - ValueConverter: nil (no conversion)
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

## Value Converter

The `ValueConverter` function allows you to convert query values to their underlying representation before query execution. This is particularly useful for converting enum strings (e.g., `"usbc"`, `"bluetooth"`) to their numeric representations (e.g., `2`, `3`) that are stored in the database.

### Basic Usage

```go
opts := &query.ExecutorOptions{
    ValueConverter: func(field string, value interface{}) (interface{}, error) {
        // Convert enum strings to integers for "features" field
        if field == "features" {
            if str, ok := value.(string); ok {
                switch str {
                case "usbc":
                    return 2, nil
                case "bluetooth":
                    return 3, nil
                case "wifi":
                    return 4, nil
                }
            }
        }
        // No conversion for other fields
        return value, nil
    },
}

executor := memory.NewExecutor(data, opts)
```

### Use Cases

**1. Enum String to Integer Conversion**

Convert user-friendly enum strings to database integers:

```go
// Query: features = "usbc"
// Converts to: features = 2
// Database stores: features = 2

opts.ValueConverter = func(field string, value interface{}) (interface{}, error) {
    if field == "features" {
        enumMap := map[string]int{
            "usbc":      2,
            "bluetooth": 3,
            "wifi":      4,
        }
        if str, ok := value.(string); ok {
            if intVal, exists := enumMap[str]; exists {
                return intVal, nil
            }
        }
    }
    return value, nil
}
```

**2. Array Field with Enum Conversion**

The converter works with array operations like `IN` and `CONTAINS`:

```go
// Query: features IN ["usbc", "bluetooth"]
// Converts to: features IN [2, 3]

// Query: features CONTAINS "usbc"
// Converts to: features CONTAINS 2
// Works with array fields: checks if array contains the converted value
```

**3. Multiple Field Conversions**

Handle different conversion logic for different fields:

```go
opts.ValueConverter = func(field string, value interface{}) (interface{}, error) {
    switch field {
    case "features":
        // Convert feature enum strings to integers
        if str, ok := value.(string); ok {
            switch str {
            case "usbc": return 2, nil
            case "bluetooth": return 3, nil
            case "wifi": return 4, nil
            }
        }
    case "status":
        // Convert status strings to integers
        if str, ok := value.(string); ok {
            switch str {
            case "active": return 1, nil
            case "inactive": return 0, nil
            }
        }
    case "priority":
        // Convert priority strings to integers
        if str, ok := value.(string); ok {
            switch str {
            case "low": return 1, nil
            case "medium": return 2, nil
            case "high": return 3, nil
            }
        }
    }
    // No conversion for other fields
    return value, nil
}
```

### Array CONTAINS Support

When using `CONTAINS` with array fields, the converter automatically handles conversion:

```go
type Product struct {
    ID       int
    Name     string
    Features []int  // Stored as integers: [2, 3, 4]
}

// With converter configured
// Query: features CONTAINS "usbc"
// 1. Converts "usbc" → 2
// 2. Checks if Features array contains 2
// 3. Returns matching products

opts.ValueConverter = func(field string, value interface{}) (interface{}, error) {
    if field == "features" {
        if str, ok := value.(string); ok {
            switch str {
            case "usbc": return 2, nil
            case "bluetooth": return 3, nil
            case "wifi": return 4, nil
            }
        }
    }
    return value, nil
}
```

### Error Handling

If conversion fails, return an error:

```go
opts.ValueConverter = func(field string, value interface{}) (interface{}, error) {
    if field == "features" {
        if str, ok := value.(string); ok {
            switch str {
            case "usbc": return 2, nil
            case "bluetooth": return 3, nil
            default:
                return nil, fmt.Errorf("unknown feature: %s", str)
            }
        }
    }
    return value, nil
}
```

### Complete Example

```go
import (
    "github.com/hadi77ir/go-query/query"
    "github.com/hadi77ir/go-query/executors/memory"
)

type Product struct {
    ID       int
    Name     string
    Features []int  // Stored as: 2=usbc, 3=bluetooth, 4=wifi
}

data := []Product{
    {ID: 1, Name: "Laptop", Features: []int{2, 3}},    // usbc, bluetooth
    {ID: 2, Name: "Phone", Features: []int{3, 4}},     // bluetooth, wifi
    {ID: 3, Name: "Tablet", Features: []int{2, 4}},    // usbc, wifi
}

opts := &query.ExecutorOptions{
    ValueConverter: func(field string, value interface{}) (interface{}, error) {
        if field == "features" {
            if str, ok := value.(string); ok {
                enumMap := map[string]int{
                    "usbc":      2,
                    "bluetooth": 3,
                    "wifi":      4,
                }
                if intVal, exists := enumMap[str]; exists {
                    return intVal, nil
                }
            }
        }
        return value, nil
    },
}

executor := memory.NewExecutor(data, opts)

// Query with enum strings - automatically converted
// features CONTAINS "usbc" → features CONTAINS 2
// features CONTAINS "bluetooth" → features CONTAINS 3
// features IN ["usbc", "wifi"] → features IN [2, 4]
```

### Notes

- **Field-Specific**: The converter is called with the field name, allowing different conversion logic per field
- **All Operators**: Works with all comparison operators (`=`, `!=`, `>`, `<`, `IN`, `CONTAINS`, etc.)
- **Array Operations**: Automatically converts values in `IN` arrays and `CONTAINS` operations
- **Optional**: If `ValueConverter` is `nil`, no conversion is performed (default behavior)
- **Backward Compatible**: Existing queries work without a converter configured

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

