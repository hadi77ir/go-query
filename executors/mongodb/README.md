# MongoDB Executor

MongoDB implementation for go-query.

## Installation

```bash
go get github.com/hadi77ir/go-query/executors/mongodb
```

## Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/hadi77ir/go-query/executors/mongodb"
    "github.com/hadi77ir/go-query/parser"
    "github.com/hadi77ir/go-query/query"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    // Connect to MongoDB
    ctx := context.Background()
    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)
    
    collection := client.Database("mydb").Collection("users")
    
    // Create executor with options
    opts := &query.ExecutorOptions{
        MaxPageSize:      100,
        DefaultPageSize:  10,
        DefaultSortField: "_id",
        DefaultSortOrder: query.SortOrderAsc,
        AllowRandomOrder: true,
        IDFieldName:      "_id", // Default for MongoDB (can be customized)
    }
    exec := mongodb.NewExecutor(collection, opts)
    
    // Parse and execute query
    // Use ParserCache for better performance (recommended)
    cache := parser.NewParserCache(100)
    q, _ := cache.Parse("status = active and age >= 18")
    
    var users []User
    result, err := exec.Execute(ctx, q, "", &users)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use results
    for _, user := range users {
        // Process user
        fmt.Println(user.Name)
    }
    
    // Get count separately (optional)
    count, err := exec.Count(ctx, q)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total matching users: %d\n", count)
}
```

## Supported Operators

- Comparison: `=`, `!=`, `>`, `>=`, `<`, `<=`
- String matching: `LIKE`, `NOT LIKE`, `CONTAINS`, `ICONTAINS`, `STARTS_WITH`, `ENDS_WITH`, `REGEX`
- Array matching: `IN`, `NOT IN`
- Logical: `AND`, `OR`

## Notes

- Custom ID field: By default, the executor uses `"_id"` as the ID field name for cursor pagination. You can configure a custom ID field name:
  ```go
  opts := query.DefaultExecutorOptions()
  opts.IDFieldName = "custom_id" // Use custom ID field name
  executor := mongodb.NewExecutor(collection, opts)
  ```
  The ID field name should match the actual MongoDB document field name.

