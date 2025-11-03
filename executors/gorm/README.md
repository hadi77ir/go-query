# GORM Executor

GORM implementation for go-query. Supports PostgreSQL, MySQL, SQLite, SQL Server and more.

## Installation

```bash
go get github.com/hadi77ir/go-query/executors/gorm
```

You'll also need to install your database driver:

```bash
# PostgreSQL
go get gorm.io/driver/postgres

# MySQL
go get gorm.io/driver/mysql

# SQLite
go get gorm.io/driver/sqlite

# SQL Server
go get gorm.io/driver/sqlserver
```

## Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/hadi77ir/go-query/executors/gorm"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
    gormpkg "gorm.io/gorm"
    "gorm.io/driver/postgres"
)

type User struct {
    ID     uint   `gorm:"primarykey"`
    Name   string
    Email  string
    Age    int
    Status string
}

func main() {
    // Connect to database
    dsn := "host=localhost user=postgres password=postgres dbname=mydb port=5432"
    db, err := gormpkg.Open(postgres.Open(dsn), &gormpkg.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create executor
    opts := &query.ExecutorOptions{
        MaxPageSize:      100,
        DefaultPageSize:  10,
        DefaultSortField: "id",
        DefaultSortOrder: "asc",
        AllowRandomOrder: true,
    }
    exec := gorm.NewExecutor(db, &User{}, opts)
    
    // Parse and execute query
    // Use ParserCache for better performance (recommended)
    cache := parser.NewParserCache(100)
    q, _ := cache.Parse("status = active and age >= 18")
    
    var users []User
    result, err := exec.Execute(context.Background(), q, "", &users)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use results
    for _, user := range users {
        // Process user
        fmt.Println(user.Name)
    }
    
    // Get count separately (optional)
    count, err := exec.Count(context.Background(), q)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total matching users: %d\n", count)
}
```

## SQL Injection Protection

This executor uses parameterized queries throughout and validates all field names to prevent SQL injection attacks. Never concatenate user input into query strings - always use the query parser.

## Supported Operators

- Comparison: `=`, `!=`, `>`, `>=`, `<`, `<=`
- String matching: `LIKE`, `NOT LIKE`, `CONTAINS`, `ICONTAINS`, `STARTS_WITH`, `ENDS_WITH`, `REGEX`
- Array matching: `IN`, `NOT IN`
- Logical: `AND`, `OR`

## Notes

- `REGEX` operator support varies by database
- Random ordering: Configure `RandomFunctionName` in executor options for database-specific syntax:
  - PostgreSQL, SQLite: `"RANDOM()"` (default)
  - MySQL: `"RAND()"`
  ```go
  opts := query.DefaultExecutorOptions()
  opts.RandomFunctionName = "RAND()" // For MySQL
  executor := gorm.NewExecutor(db, &Product{}, opts)
  ```
- Custom ID field: By default, the executor uses `"id"` as the ID field name for cursor pagination. You can configure a custom ID field name:
  ```go
  opts := query.DefaultExecutorOptions()
  opts.IDFieldName = "product_id" // Use custom ID field name
  executor := gorm.NewExecutor(db, &Product{}, opts)
  ```
  **Note:** The ID field name should match the actual database column name (not the Go struct field name). For structs, the executor will attempt to find the field using reflection (trying exact match, Title case, and uppercase variations).
- Field names are validated to contain only alphanumeric characters and underscores

