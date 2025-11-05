# Query Syntax Guide

Complete reference for the go-query query language syntax.

## Table of Contents

1. [Google-Style Bare Search](#google-style-bare-search)
2. [Complex Parentheses](#complex-parentheses)
3. [String Matching](#string-matching)
4. [Array Operations](#array-operations)
5. [Query Options](#query-options)
6. [Real-World Examples](#real-world-examples)

## Google-Style Bare Search

Just type words - they'll be searched automatically!

```go
// Bare words are automatically searched in the default field
"hello world"                    // Searches for "hello" AND "world" in default field
`wireless mouse`                 // Searches name for "wireless" AND "mouse"
`"hello world"`                  // Exact phrase search

// Mix with field-specific queries
`name LIKE "John%" hello`        // name starts with "John" AND contains "hello"
`wireless price < 100`           // Contains "wireless" AND price < 100
```

### How It Works

- Bare words (without field names) are automatically searched in the `DefaultSearchField` (default: `"name"`)
- Multiple bare words are AND'ed together
- Phrases in quotes are treated as exact matches
- Mix bare words with field-specific queries freely

## Complex Parentheses

Full support for nested expressions:

```go
// Simple grouping
`(status = active and age > 18) or premium = true`

// Deeply nested
`((a = 1 and b = 2) or (c = 3 and d = 4)) and e = 5`

// Complex filters
`(category IN [electronics, computers] and price < 500) or featured = true`
```

### Operator Precedence

1. Parentheses `()` - Highest precedence
2. `AND` - Evaluated before OR
3. `OR` - Lowest precedence

## String Matching

Powerful string matching operators:

```go
// SQL-style LIKE with wildcards
name LIKE "%John%"              // Contains "John"
email NOT LIKE "%spam%"         // Doesn't contain "spam"
name LIKE "John%"              // Starts with "John"
name LIKE "%John"              // Ends with "John"

// Wildcards
%                              // Matches any sequence of characters
_                              // Matches any single character

// Substring matching
description CONTAINS "error"    // Case-sensitive substring
title ICONTAINS "hello"         // Case-insensitive substring

// Prefix/suffix
path STARTS_WITH "/api"         // Prefix match
filename ENDS_WITH ".pdf"       // Suffix match

// Regular expressions
pattern REGEX "^[A-Z][0-9]+"    // Regular expression (if supported)
```

## Array Operations

Filter using arrays:

```go
// In array
status IN [active, pending, approved]
role NOT IN [guest, banned]
id IN [1, 2, 3, 5, 8]
category IN ["electronics", "computers"]

// Works with strings, numbers, and mixed types
tags IN ["tag1", "tag2", "tag3"]
priority IN [1, 2, 3]
```

## Query Options

Query options control pagination, sorting, cursors, and result limits:

```go
// Pagination
"page_size = 20 status = active"

// Sorting
"sort_by = created_at sort_order = desc status = active"

// Limit total results (different from page size)
"limit = 50 page_size = 20 status = active"

// Combined
"page_size = 25 sort_by = price sort_order = asc category = electronics limit = 100"

// Random ordering
"sort_order = random category = electronics"
```

**Note**: Query options can be placed **anywhere** in the query string:

```go
// Options at the beginning (traditional)
"page_size = 20 sort_by = name status = active"

// Options in the middle
"status = active page_size = 20 name = test"

// Options at the end
"status = active and price < 100 sort_by = price sort_order = desc"

// Options mixed with AND
"status = active and page_size = 20 and name = test"

// Limit can be used with quoted numbers (like page_size)
"limit = 50 status = active"
"limit = \"50\" status = active"  // Quoted numbers also work
```

See [Query Options](FEATURES.md#query-options) in FEATURES.md for complete documentation.

## Real-World Examples

### E-Commerce Search

```go
// Product search with multiple criteria
query := `
  wireless mouse
  price >= 10 price <= 50
  (brand IN [logitech, microsoft, razer] or featured = true)
  rating >= 4
`
```

### User Management

```go
// Find users by name and status
query := `john doe status = active role IN [admin, moderator]`

// With custom search field
opts := &query.ExecutorOptions{
    DefaultSearchField: "email",  // Search email instead
}
query := `@gmail.com status = active`  // Finds active Gmail users
```

### Content Search

```go
// Blog post search
query := `
  "machine learning" "neural networks"
  published = true
  created_at >= 2024-01-01
  category IN [tech, ai, science]
`
```

### Advanced Filtering

```go
// Complex nested query
query := `
  page_size = 20 
  sort_by = price 
  sort_order = desc
  (category = electronics and name CONTAINS "wireless") 
  or (featured = true and price < 100)
`

// Date ranges with arrays
query := `
  created_at >= 2020-01-01 
  and status IN [published, approved] 
  and author_id IN [1, 5, 10]
`

// Using limit to cap total results
query := `
  limit = 50
  page_size = 10
  category = electronics
  and price >= 50
  and price <= 500
`
// Returns max 50 items total, 10 per page
```

## Operator Reference

### Comparison Operators
- `=` - Equal
- `!=` - Not equal
- `>` - Greater than
- `>=` - Greater than or equal
- `<` - Less than
- `<=` - Less than or equal

### String Operators
- `LIKE` - SQL-style pattern matching (`%` and `_` wildcards)
- `NOT LIKE` - Negated LIKE
- `CONTAINS` - Case-sensitive substring match
- `ICONTAINS` - Case-insensitive substring match
- `STARTS_WITH` - Prefix match
- `ENDS_WITH` - Suffix match
- `REGEX` - Regular expression (database-dependent)

### Array Operators
- `IN` - Value is in array
- `NOT IN` - Value is not in array

### Logical Operators
- `AND` - Logical AND (higher precedence)
- `OR` - Logical OR (lower precedence)
- `()` - Parentheses for grouping

## See Also

- [Configuration Guide](CONFIGURATION.md) - Configure default search field and other options
- [Examples](EXAMPLES.md) - More real-world examples
- [Features Guide](FEATURES.md) - Advanced features and capabilities

