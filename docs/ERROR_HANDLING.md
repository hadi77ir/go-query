# Error Handling Guide

This document covers the error handling system in go-query, including error types, usage patterns, and best practices.

## Error System Overview

The library uses a comprehensive error handling system with:
- **Public error variables** for easy matching with `errors.Is()`
- **Structured error types** for contextual information
- **Consistent error handling** across all executors

## Public Error Variables

All errors are defined in `query/errors.go`:

```go
var (
    ErrNoRecordsFound          // Query returns 0 results
    ErrInvalidFieldName        // Field name validation failed (SQL injection)
    ErrFieldNotAllowed         // Field not in AllowedFields whitelist
    ErrInvalidQuery            // Query structure invalid
    ErrInvalidCursor           // Cursor string decode failed
    ErrPageSizeExceeded        // Page size exceeds maximum
    ErrRegexNotSupported       // REGEX operator disabled
    ErrRandomOrderNotAllowed   // Random ordering disabled
    ErrExecutionFailed         // Database execution error
    ErrInvalidDestination      // Destination not pointer to slice
)
```

## Error Types

### FieldError

Wraps errors with field name information:

```go
type FieldError struct {
    Field string
    Err   error
}
```

**Helper Functions:**
```go
InvalidFieldNameError(field string) error   // For SQL injection attempts
FieldNotAllowedError(field string) error    // For AllowedFields violations
```

### ExecutionError

Wraps database execution errors with operation context:

```go
type ExecutionError struct {
    Operation string
    Err       error
}
```

**Helper Function:**
```go
NewExecutionError(operation string, err error) error
```

## Usage Patterns

### Pattern 1: Simple Error Check

```go
result, err := executor.Execute(ctx, query, "", &products)
if errors.Is(err, query.ErrNoRecordsFound) {
    return http.StatusNotFound
}
```

### Pattern 2: Switch Statement

```go
switch {
case errors.Is(err, query.ErrNoRecordsFound):
    return 404  // Not found
case errors.Is(err, query.ErrInvalidFieldName):
    return 400  // Bad request
case errors.Is(err, query.ErrFieldNotAllowed):
    return 403  // Forbidden
case errors.Is(err, query.ErrExecutionFailed):
    return 500  // Server error
default:
    return 500  // Unknown error
}
```

### Pattern 3: Extract Field Information

```go
var fieldErr *query.FieldError
if errors.As(err, &fieldErr) {
    log.Printf("Error with field: %s", fieldErr.Field)
    // Handle field-specific error
}
```

### Pattern 4: Extract Operation Information

```go
var execErr *query.ExecutionError
if errors.As(err, &execErr) {
    log.Printf("Operation failed: %s", execErr.Operation)
    // Handle operation-specific error
}
```

## HTTP Status Code Mapping

| Error | HTTP Status | Use Case |
|-------|-------------|----------|
| `ErrNoRecordsFound` | 404 | Empty query results |
| `ErrInvalidFieldName` | 400 | SQL injection attempt |
| `ErrFieldNotAllowed` | 403 | Field not in whitelist |
| `ErrInvalidCursor` | 400 | Invalid cursor string |
| `ErrRegexNotSupported` | 400 | REGEX disabled |
| `ErrRandomOrderNotAllowed` | 400 | Random disabled |
| `ErrInvalidDestination` | 500 | Programming error |
| `ErrExecutionFailed` | 500 | Database error |
| `ErrInvalidQuery` | 400 | Malformed query |

## Migration Notes

### Breaking Change: ErrNoRecordsFound

Empty results now return `ErrNoRecordsFound` instead of `nil` error.

**Before:**
```go
result, err := executor.Execute(ctx, query, "", &products)
if err != nil {
    return err  // Error
}
if len(products) == 0 {
    return "not found"  // Empty
}
```

**After:**
```go
result, err := executor.Execute(ctx, query, "", &products)
if errors.Is(err, query.ErrNoRecordsFound) {
    return http.StatusNotFound  // Empty (expected)
}
if err != nil {
    return err  // Other error
}
// Success with results
```

## Implementation Details

### Error Wrapping

Errors wrap public errors with additional context:

```go
// Cursor errors wrapped with details
fmt.Errorf("%w: %v", query.ErrInvalidCursor, err)

// Execution errors wrapped with operation
query.NewExecutionError("execute query", err)

// Field errors wrapped with field name
query.InvalidFieldNameError(fieldName)
query.FieldNotAllowedError(fieldName)
```

### No Raw Errors

All `fmt.Errorf()` and `errors.New()` calls have been replaced with:
- Public error variables
- Helper functions that return wrapped public errors
- Execution/Field error types

## Testing Error Handling

### GORM Error Handling Tests

```bash
cd executors/gorm
go test -v -run TestGORMExecutor_ErrorHandling
```

Tests cover:
- ✅ ErrNoRecordsFound - empty result
- ✅ ErrInvalidFieldName - SQL injection attempt
- ✅ ErrFieldNotAllowed - field not in whitelist
- ✅ ErrRegexNotSupported - regex disabled
- ✅ ErrRandomOrderNotAllowed - random disabled
- ✅ ErrInvalidDestination - not a pointer to slice
- ✅ Error matching with switch
- ✅ Success case - no error

### Error Matching Best Practices

**✅ Do:**
```go
// Use errors.Is() for type checking
if errors.Is(err, query.ErrNoRecordsFound) {
    // Handle empty results
}

// Use errors.As() to extract structured info
var fieldErr *query.FieldError
if errors.As(err, &fieldErr) {
    log.Printf("Field: %s", fieldErr.Field)
}
```

**❌ Don't:**
```go
// Don't use string matching
if strings.Contains(err.Error(), "no records found") {
    // Fragile - breaks if error message changes
}

// Don't rely on error message text
if err.Error() == "invalid field name" {
    // Not type-safe
}
```

## Benefits

✅ **Type-Safe**: Use `errors.Is()` instead of string comparison  
✅ **Consistent**: Same errors across all executors  
✅ **Clear**: Well-named, self-documenting  
✅ **Contextual**: Field and operation information included  
✅ **HTTP-Friendly**: Clear mapping to HTTP status codes  
✅ **Testable**: Easy to test error scenarios  
✅ **Robust**: Works even if error messages change  

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    
    "github.com/hadi77ir/go-query/v2/executors/gorm"
    "github.com/hadi77ir/go-query/v2/parser"
    "github.com/hadi77ir/go-query/v2/query"
)

func handleQuery(w http.ResponseWriter, r *http.Request) {
    queryStr := r.URL.Query().Get("q")
    
    // Parse query using cache (recommended for performance)
    cache := parser.NewParserCache(100)
    q, err := cache.Parse(queryStr)
    if err != nil {
        http.Error(w, "Invalid query syntax", http.StatusBadRequest)
        return
    }
    
    // Execute query
    executor := getExecutor()
    var results []Product
    result, err := executor.Execute(r.Context(), q, "", &results)
    
    // Handle errors
    if err != nil {
        switch {
        case errors.Is(err, query.ErrNoRecordsFound):
            http.Error(w, "No products found", http.StatusNotFound)
            return
        case errors.Is(err, query.ErrInvalidFieldName):
            http.Error(w, "Invalid field name", http.StatusBadRequest)
            return
        case errors.Is(err, query.ErrFieldNotAllowed):
            http.Error(w, "Field not allowed", http.StatusForbidden)
            return
        case errors.Is(err, query.ErrExecutionFailed):
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        default:
            http.Error(w, "Unknown error", http.StatusInternalServerError)
            return
        }
    }
    
    // Success - return results
    json.NewEncoder(w).Encode(result)
}
```

---

**Status**: ✅ Complete  
**All Executors**: ✅ Using public error variables  
**Tests**: ✅ Comprehensive coverage  

**Generated**: AI (Claude Sonnet 4.5 and Cursor Auto)
**License**: Apache 2.0

