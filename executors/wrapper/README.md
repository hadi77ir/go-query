# Wrapper Executor

A wrapper executor for go-query that imposes additional field restrictions on top of another executor's allowed fields list. This enables layered security by requiring fields to be in BOTH the wrapper's allowed list AND the inner executor's allowed list.

## Features

- ✅ **Layered Security**: Impose additional field restrictions on top of existing executor restrictions
- ✅ **Field Intersection**: Fields must be allowed by BOTH wrapper and inner executor
- ✅ **Transparent Wrapping**: Works with any executor that implements the Executor interface
- ✅ **Recursive Validation**: Validates all fields in complex filter expressions (AND/OR)
- ✅ **Sort Field Validation**: Also validates sort field names

## Installation

```bash
go get github.com/hadi77ir/go-query/executors/wrapper
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/hadi77ir/go-query/executors/memory"
    "github.com/hadi77ir/go-query/executors/wrapper"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
)

type User struct {
    ID       int
    Name     string
    Email    string
    Password string
    SSN      string
    Balance  float64
}

func main() {
    users := []User{
        {ID: 1, Name: "Alice", Email: "alice@example.com", Password: "secret1", Balance: 100.0},
        {ID: 2, Name: "Bob", Email: "bob@example.com", Password: "secret2", Balance: 200.0},
    }

    // Inner executor allows: name, email, id, balance
    opts := query.DefaultExecutorOptions()
    opts.AllowedFields = []string{"name", "email", "id", "balance"}
    innerExecutor := memory.NewExecutor(users, opts)
    
    // Wrapper executor further restricts to: name, email only
    // Now only name and email can be queried (intersection)
    wrapperExecutor := wrapper.NewExecutor(innerExecutor, []string{"name", "email"})

    ctx := context.Background()
    
    // This works - name is in both lists
    p, _ := parser.NewParser("name = Alice")
    q, _ := p.Parse()
    var result []User
    wrapperExecutor.Execute(ctx, q, "", &result)
    fmt.Printf("Found %d users\n", len(result))
    
    // This fails - id is not in wrapper's allowed list
    p2, _ := parser.NewParser("id = 1")
    q2, _ := p2.Parse()
    _, err := wrapperExecutor.Execute(ctx, q2, &result)
    // err will be: field 'id': field not allowed
}
```

## Use Cases

### Multi-Tenant Security

Restrict fields based on user roles or tenant permissions:

```go
// Base executor allows admin fields
adminOpts := query.DefaultExecutorOptions()
adminOpts.AllowedFields = []string{"name", "email", "id", "balance", "password", "ssn"}
adminExecutor := memory.NewExecutor(users, adminOpts)

// Regular user wrapper only allows safe fields
userExecutor := wrapper.NewExecutor(adminExecutor, []string{"name", "email", "id"})

// Admin can access all fields, regular users only safe fields
```

### API Versioning

Different API versions expose different field sets:

```go
// V2 allows more fields
v2Opts := query.DefaultExecutorOptions()
v2Opts.AllowedFields = []string{"name", "email", "id", "balance", "metadata"}
v2Executor := memory.NewExecutor(users, v2Opts)

// V1 wrapper restricts to legacy fields
v1Executor := wrapper.NewExecutor(v2Executor, []string{"name", "email", "id"})
```

### Layered Permissions

Chain multiple wrappers for fine-grained control:

```go
// Base executor
baseExecutor := memory.NewExecutor(users, query.DefaultExecutorOptions())

// First wrapper: allow public fields
publicExecutor := wrapper.NewExecutor(baseExecutor, []string{"name", "email", "id", "balance"})

// Second wrapper: further restrict for anonymous users
anonymousExecutor := wrapper.NewExecutor(publicExecutor, []string{"name", "email"})
```

## Behavior

### Empty Allowed Fields

If the wrapper's allowed fields list is empty, it doesn't impose additional restrictions. The inner executor's restrictions still apply:

```go
// No restriction from wrapper
wrapperExecutor := wrapper.NewExecutor(innerExecutor, []string{})

// All fields allowed by inner executor can be used
```

### Field Intersection

A field is allowed only if it exists in BOTH:
1. Wrapper's allowed fields list (empty means no restriction from wrapper)
2. Inner executor's allowed fields list (empty means no restriction from inner)

```go
// Inner allows: name, email, id
// Wrapper allows: name, email, balance
// Result: Only "name" and "email" are allowed (intersection)
wrapperExecutor := wrapper.NewExecutor(innerExecutor, []string{"name", "email", "balance"})
```

### Complex Filters

All fields in complex filter expressions (AND/OR) are validated:

```go
// This will fail because "balance" is not in wrapper's list
p, _ := parser.NewParser("name = Alice AND balance > 50")
q, _ := p.Parse()
_, err := wrapperExecutor.Execute(ctx, q, "", &result)
// Error: field 'balance': field not allowed
```

### Sort Fields

Sort fields are also validated:

```go
q.SortBy = "balance"
_, err := wrapperExecutor.Execute(ctx, q, "", &result)
// Error: field 'balance': field not allowed
```

## API Reference

### NewExecutor

Creates a new wrapper executor.

```go
func NewExecutor(innerExecutor executor.Executor, allowedFields []string) *WrapperExecutor
```

**Parameters:**
- `innerExecutor`: The executor to wrap
- `allowedFields`: List of fields allowed by this wrapper (empty slice means no additional restriction)

**Returns:**
- `*WrapperExecutor`: A new wrapper executor instance

### Execute

Executes a query with field restrictions.

```go
func (e *WrapperExecutor) Execute(ctx context.Context, q *query.Query, cursor string, dest interface{}) (*query.Result, error)
```

**Parameters:**
- `ctx`: Context for the operation
- `q`: The parsed query
- `cursor`: Optional cursor string for pagination (empty string for first page)
- `dest`: Pointer to slice where results will be stored

**Returns:**
- `*query.Result`: Query results with pagination info
- `error`: Error if execution fails or fields are not allowed

Validates all fields in the query against the wrapper's allowed fields before delegating to the inner executor.

### Count

Returns the total number of items that would be returned by the given query without applying pagination.

```go
func (e *WrapperExecutor) Count(ctx context.Context, q *query.Query) (int64, error)
```

**Parameters:**
- `ctx`: Context for the operation
- `q`: The parsed query

**Returns:**
- `int64`: Total count of matching items
- `error`: Error if count fails or fields are not allowed

The Count method also validates fields against the wrapper's restrictions before delegating to the inner executor.

### Close

Cleans up resources and closes the inner executor.

```go
func (e *WrapperExecutor) Close() error
```

### Name

Returns the executor name: "wrapper"

```go
func (e *WrapperExecutor) Name() string
```

## Error Handling

The wrapper executor returns standard go-query errors:

- `query.FieldNotAllowedError`: When a field is not in the wrapper's allowed fields list
- Errors from inner executor: Passed through as-is

```go
_, err := wrapperExecutor.Execute(ctx, q, "", &result)
if err != nil {
    var fieldErr *query.FieldError
    if errors.As(err, &fieldErr) {
        fmt.Printf("Field '%s' is not allowed\n", fieldErr.Field)
    }
}
```

