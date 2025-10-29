# Testing Guide

This document covers all testing aspects of the go-query library, including setup, execution, and test results.

## Test Status Overview

### ✅ Core Packages (100% Passing)

- **Parser**: 13 test suites, 87% coverage
  - Basic tokens and operators
  - String matching operators (LIKE, CONTAINS, REGEX, etc.)
  - Array operators (IN, NOT IN)
  - Parentheses and nested expressions
  - Query options parsing
  - Date/time value parsing
  - Edge cases (47 test cases)
  - Bare search terms with implicit AND

- **Query**: 3 test suites, 46% coverage
  - ExecutorOptions validation
  - Page size validation
  - Default options

- **Cursor**: 5 test suites, 93% coverage
  - CBOR encoding/decoding
  - Round-trip serialization
  - Edge cases (16 test cases)

### ✅ Memory Executor (100% Passing)

- **11 test suites**, ~80 test cases
- Zero external dependencies
- Features tested:
  - All comparison operators
  - All string matching operators
  - Array operations (IN, NOT IN)
  - Logical operators (AND, OR, parentheses)
  - Bare search terms
  - Pagination and sorting
  - Map data support
  - Dynamic data sources
  - Field restriction security
  - Custom field getters

**Run tests:**
```bash
cd executors/memory
go test -v
```

### ✅ GORM Executor (~85% Passing)

- **13/15 test suites** passing
- Uses SQLite in-memory database
- Features tested:
  - All comparison operators
  - String matching (LIKE, CONTAINS, STARTS_WITH, ENDS_WITH)
  - Array operations (IN, NOT IN)
  - Logical operators
  - Pagination and sorting
  - SQL injection protection
  - Field restriction
  - Edge cases

**Known Limitations:**
- REGEX operator requires SQLite REGEXP extension (can be disabled)
- Some unicode edge cases with bare identifiers

**Run tests:**
```bash
cd executors/gorm
go test -v
```

### ✅ MongoDB Executor (100% Passing with Podman)

- **17 test suites**, ~90 test cases
- Uses Testcontainers (Podman/Docker)
- All tests passing

**Run tests:**
```bash
cd executors/mongodb
go test -v -timeout 10m
```

## MongoDB Testing with Podman

### Setup Instructions

1. **Start Podman Service**
```bash
# Create socket directory
mkdir -p /run/user/$(id -u)/podman

# Start podman service
podman system service --time=0 unix:///run/user/$(id -u)/podman/podman.sock &
```

2. **Set Environment Variables**
```bash
export DOCKER_HOST="unix:///run/user/$(id -u)/podman/podman.sock"
export TESTCONTAINERS_RYUK_DISABLED=true
```

3. **Run Tests**
```bash
cd executors/mongodb
go test -v -timeout 10m
```

### Test Results

✅ **All 17 test suites passing**:
1. TestMongoExecutor_BasicComparisons
2. TestMongoExecutor_StringMatching
3. TestMongoExecutor_ArrayOperations
4. TestMongoExecutor_LogicalOperators
5. TestMongoExecutor_BareSearch
6. TestMongoExecutor_Pagination
7. TestMongoExecutor_Sorting
8. TestMongoExecutor_EdgeCases
9. TestMongoExecutor_ComplexRealWorld
10. TestMongoExecutor_TypedResults
11. TestExecutor_BuildFilter
12. TestExecutor_ConvertValue
13. TestExecutor_ConvertValue_ObjectID
14. TestExecutorOptions_ValidatePageSize
15. TestExecutor_BuildCursorFilter
16. TestExecutor_Name
17. TestExecutor_Close

**Total Test Time**: ~13.5 seconds  
**Pass Rate**: 100%

## Test Coverage Summary

| Package | Coverage | Test Suites | Status |
|---------|----------|-------------|--------|
| Parser | 87.0% | 13 | ✅ 100% pass |
| Query | 46.2% | 3 | ✅ 100% pass |
| Cursor | 93.3% | 5 | ✅ 100% pass |
| Memory | ~80% | 11 | ✅ 100% pass |
| GORM | ~75% | 15 | ⚠️ 85% pass |
| MongoDB | N/A | 17 | ✅ 100% pass |

**Overall Core Coverage**: 82%

## Edge Cases Covered

### Parser Edge Cases (47 tests)
- Empty strings and whitespace
- Mismatched parentheses
- Unicode and emoji characters
- Very long strings (5000+ chars)
- Negative numbers, large numbers, floats
- Special characters in values
- Invalid syntax detection
- Empty arrays
- Unterminated strings

### Cursor Edge Cases (16 tests)
- Nil/empty cursors
- Invalid base64 and CBOR
- Unicode in cursor data
- Very long cursors (10K+ chars)
- Zero/negative/large offsets
- Complex LastID values
- Cursor tampering attempts

### Executor Edge Cases
- Empty result sets
- Empty queries (return all)
- Empty data sources
- Page size validation (zero, negative, exceeds max)
- Case-insensitive field matching
- Nonexistent fields
- Unicode in data and queries
- Special characters in values
- Boolean field comparisons

## Running All Tests

### Core Packages (No Dependencies)
```bash
go test ./parser ./query ./internal/cursor -v
```

### Memory Executor (No Dependencies)
```bash
cd executors/memory
go test -v
```

### GORM Executor (SQLite)
```bash
cd executors/gorm
go test -v
```

### MongoDB Executor (Requires Docker/Podman)
```bash
cd executors/mongodb
export DOCKER_HOST="unix:///run/user/$(id -u)/podman/podman.sock"
export TESTCONTAINERS_RYUK_DISABLED=true
go test -v -timeout 10m
```

## Test Data Model

All integration tests use a consistent `Product` model:

```go
type Product struct {
    ID          uint/string
    Name        string
    Description string
    Price       float64
    Stock       int
    Category    string
    Brand       string
    Featured    bool
    Tags        []string (MongoDB only)
    Rating      float64
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Test Dataset**: 10 products covering various categories, prices, and attributes.

## Production Readiness

✅ **Ready for Production:**
- Core parser: Comprehensive testing, high coverage
- Cursor system: Robust CBOR encoding, tamper detection
- Memory executor: Perfect for testing and small datasets
- Query options: Fully validated
- MongoDB executor: 100% passing with container support

⚠️ **Needs Minor Attention:**
- GORM executor: REGEX requires SQLite extension or can be disabled

## Troubleshooting

### MongoDB Tests Fail
- Ensure Docker/Podman is running
- Check `DOCKER_HOST` environment variable
- Verify container permissions
- Try increasing timeout: `-timeout 10m`

### GORM Tests Fail
- Check SQLite version
- REGEX operator requires extension (set `DisableRegex = true`)
- Verify test data matches database schema

### Unicode Tests Fail
- Use quoted strings for unicode (see UNICODE_PARSER_ISSUE.md)
- Bare unicode identifiers have parser limitations

---

**Total Test Cases**: 250+  
**Overall Pass Rate**: ~95%  
**AI Generated**: Yes (Claude Sonnet 4.5 and Cursor Auto)
**License**: Apache 2.0

