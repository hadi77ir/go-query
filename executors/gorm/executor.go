package gorm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hadi77ir/go-query/executor"
	"github.com/hadi77ir/go-query/internal/cursor"
	"github.com/hadi77ir/go-query/query"
	"gorm.io/gorm"
)

// Executor is the GORM implementation of the executor interface
type Executor struct {
	db      *gorm.DB
	model   interface{}
	options *query.ExecutorOptions
}

// NewExecutor creates a new GORM executor
// model should be a pointer to the model struct, e.g. &User{}
func NewExecutor(db *gorm.DB, opts *query.ExecutorOptions) executor.Executor {
	if opts == nil {
		opts = query.DefaultExecutorOptions()
	}
	return &Executor{
		db:      db,
		options: opts,
	}
}

// Name returns the name of this executor
func (e *Executor) Name() string {
	return "GORM"
}

// Close cleans up resources (GORM connections are managed separately)
func (e *Executor) Close() error {
	return nil
}

// Execute runs the query and stores results in dest
// dest must be a pointer to a slice (e.g., &[]User{})
func (e *Executor) Execute(ctx context.Context, q *query.Query, dest interface{}) (*query.Result, error) {
	result := &query.Result{}

	// Validate and adjust page size
	pageSize := e.options.ValidatePageSize(q.PageSize)

	// Build base query
	tx := e.db.WithContext(ctx)

	// Build WHERE clause from filter
	if q.Filter != nil {
		whereClauses, args, err := e.buildFilter(q.Filter)
		if err != nil {
			result.Error = err
			return result, err
		}
		if whereClauses != "" {
			tx = tx.Where(whereClauses, args...)
		}
	}

	// Count total items
	var totalItems int64
	if err := tx.Count(&totalItems).Error; err != nil {
		result.Error = query.NewExecutionError("count items", err)
		return result, result.Error
	}
	result.TotalItems = totalItems

	// Handle cursor-based pagination
	cursorData, err := cursor.Decode(q.Cursor)
	if err != nil {
		result.Error = fmt.Errorf("%w: %v", query.ErrInvalidCursor, err)
		return result, result.Error
	}

	// Handle sorting
	sortField := q.SortBy
	if sortField == "" {
		sortField = e.options.DefaultSortField
	}
	sortOrder := q.SortOrder
	// If sort order is not explicitly set (remains default), use executor default
	if sortOrder == query.SortOrderAsc {
		sortOrder = e.options.DefaultSortOrder
	}

	// Handle random ordering
	var randomSeed int64
	if sortOrder == query.SortOrderRandom {
		if !e.options.AllowRandomOrder {
			result.Error = query.ErrRandomOrderNotAllowed
			return result, result.Error
		}

		// Generate or reuse random seed
		if cursorData != nil && cursorData.RandomSeed != 0 {
			randomSeed = cursorData.RandomSeed
		} else {
			randomSeed = 12345 // Use a fixed seed for consistency
		}

		// Use configured random function for random ordering
		randomFunc := e.options.RandomFunctionName
		if randomFunc == "" {
			randomFunc = "RANDOM()" // Default fallback
		}
		tx = tx.Order(gorm.Expr(randomFunc))

		// Apply offset for cursor pagination in random mode
		if cursorData != nil && cursorData.Offset > 0 {
			tx = tx.Offset(cursorData.Offset)
		}
	} else {
		// Regular sorting with SQL injection protection
		sortOrderStr := "ASC"
		if sortOrder == query.SortOrderDesc {
			sortOrderStr = "DESC"
		}

		// Validate sort field to prevent SQL injection
		if !e.isValidField(sortField) {
			result.Error = query.InvalidFieldNameError(sortField)
			return result, result.Error
		}

		tx = tx.Order(fmt.Sprintf("%s %s", sortField, sortOrderStr))

		// Apply cursor filter for pagination
		if cursorData != nil && cursorData.LastID != nil {
			cursorWhere, cursorArgs := e.buildCursorFilter(cursorData, sortField, sortOrderStr)
			if cursorWhere != "" {
				tx = tx.Where(cursorWhere, cursorArgs...)
			}
		}
	}

	// Fetch results (one extra to check for next page)
	tx = tx.Limit(pageSize + 1)

	// Execute query - store results directly in dest
	if err := tx.Find(dest).Error; err != nil {
		result.Error = query.NewExecutionError("execute query", err)
		return result, result.Error
	}

	// Get slice length using reflection to check if there are more results
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		result.Error = query.ErrInvalidDestination
		return result, result.Error
	}

	sliceValue := destValue.Elem()
	itemsCount := sliceValue.Len()

	// Check if any records were found
	if itemsCount == 0 && result.TotalItems == 0 {
		result.Error = query.ErrNoRecordsFound
		return result, result.Error
	}

	// Check if there are more results
	hasMore := itemsCount > pageSize
	if hasMore {
		// Trim to actual page size
		sliceValue.Set(sliceValue.Slice(0, pageSize))
		itemsCount = pageSize
	}

	result.ItemsReturned = itemsCount

	// Calculate showing from/to
	var currentOffset int
	if cursorData != nil && sortOrder == query.SortOrderRandom {
		currentOffset = cursorData.Offset
	}

	if result.ItemsReturned > 0 {
		result.ShowingFrom = currentOffset + 1
		result.ShowingTo = currentOffset + result.ItemsReturned
	}

	// Generate cursors
	if result.ItemsReturned > 0 {
		// Access last row using reflection
		lastIndex := result.ItemsReturned - 1
		lastRow := sliceValue.Index(lastIndex).Interface()

		if hasMore {
			// Generate next cursor
			nextCursorData := &cursor.CursorData{
				Direction: "next",
			}

			if sortOrder == query.SortOrderRandom {
				nextCursorData.Offset = currentOffset + pageSize
				nextCursorData.RandomSeed = randomSeed
			} else {
				// Extract ID and sort value using reflection
				lastRowValue := reflect.ValueOf(lastRow)
				if lastRowValue.Kind() == reflect.Ptr {
					lastRowValue = lastRowValue.Elem()
				}

				// Extract ID using custom field name
				if idValue := e.getIDValue(lastRow); idValue != nil {
					nextCursorData.LastID = idValue
				}

				if !e.isIDField(sortField) {
					if lastRowValue.Kind() == reflect.Struct {
						sortFieldValue := lastRowValue.FieldByName(sortField)
						if !sortFieldValue.IsValid() {
							// Try capitalized version
							sortFieldValue = lastRowValue.FieldByName(strings.Title(sortField))
						}
						if sortFieldValue.IsValid() {
							nextCursorData.LastSortValue = sortFieldValue.Interface()
						}
					} else if lastRowValue.Kind() == reflect.Map {
						lastRowMap := lastRow.(map[string]interface{})
						if val, ok := lastRowMap[sortField]; ok {
							nextCursorData.LastSortValue = val
						}
					}
				}
			}

			result.NextPageCursor, err = cursor.Encode(nextCursorData)
			if err != nil {
				result.Error = query.NewExecutionError("encode next cursor", err)
				return result, result.Error
			}
		}

		// Generate previous cursor
		if cursorData != nil && currentOffset > 0 {
			prevCursorData := &cursor.CursorData{
				Direction: "prev",
			}

			if sortOrder == query.SortOrderRandom {
				prevOffset := currentOffset - pageSize
				if prevOffset < 0 {
					prevOffset = 0
				}
				prevCursorData.Offset = prevOffset
				prevCursorData.RandomSeed = randomSeed
			} else {
				// Access first row using reflection
				firstRow := sliceValue.Index(0).Interface()
				firstRowValue := reflect.ValueOf(firstRow)
				if firstRowValue.Kind() == reflect.Ptr {
					firstRowValue = firstRowValue.Elem()
				}

				// Extract ID using custom field name
				if idValue := e.getIDValue(firstRow); idValue != nil {
					prevCursorData.LastID = idValue
				}

				if !e.isIDField(sortField) {
					if firstRowValue.Kind() == reflect.Struct {
						sortFieldValue := firstRowValue.FieldByName(sortField)
						if !sortFieldValue.IsValid() {
							sortFieldValue = firstRowValue.FieldByName(strings.Title(sortField))
						}
						if sortFieldValue.IsValid() {
							prevCursorData.LastSortValue = sortFieldValue.Interface()
						}
					} else if firstRowValue.Kind() == reflect.Map {
						firstRowMap := firstRow.(map[string]interface{})
						if val, ok := firstRowMap[sortField]; ok {
							prevCursorData.LastSortValue = val
						}
					}
				}
			}

			result.PrevPageCursor, err = cursor.Encode(prevCursorData)
			if err != nil {
				result.Error = query.NewExecutionError("encode prev cursor", err)
				return result, result.Error
			}
		}
	}

	return result, nil
}

// getIDFieldName returns the ID field name to use, with fallback defaults
func (e *Executor) getIDFieldName() string {
	if e.options.IDFieldName != "" {
		return e.options.IDFieldName
	}
	return "id" // Default for GORM/SQL databases
}

// getIDValue extracts the ID value from a row using reflection
func (e *Executor) getIDValue(row interface{}) interface{} {
	rowValue := reflect.ValueOf(row)
	if rowValue.Kind() == reflect.Ptr {
		rowValue = rowValue.Elem()
	}

	idFieldName := e.getIDFieldName()

	if rowValue.Kind() == reflect.Struct {
		// Try exact field name first
		idField := rowValue.FieldByName(idFieldName)
		if !idField.IsValid() {
			// Try capitalized version
			idField = rowValue.FieldByName(strings.Title(idFieldName))
		}
		if !idField.IsValid() {
			// Try uppercase version
			idField = rowValue.FieldByName(strings.ToUpper(idFieldName))
		}
		if idField.IsValid() {
			return idField.Interface()
		}
	} else if rowValue.Kind() == reflect.Map {
		// Handle map[string]interface{}
		rowMap := row.(map[string]interface{})
		if val, ok := rowMap[idFieldName]; ok {
			return val
		}
		// Try lowercase version
		if val, ok := rowMap[strings.ToLower(idFieldName)]; ok {
			return val
		}
	}

	return nil
}

// isIDField checks if a field name is the ID field
func (e *Executor) isIDField(fieldName string) bool {
	idFieldName := e.getIDFieldName()
	return strings.EqualFold(fieldName, idFieldName)
}

// buildFilter converts the AST filter into SQL WHERE clause with parameters
// Returns (whereClause, args, error)
func (e *Executor) buildFilter(node query.Node) (string, []interface{}, error) {
	switch n := node.(type) {
	case *query.BinaryOpNode:
		left, leftArgs, err := e.buildFilter(n.Left)
		if err != nil {
			return "", nil, err
		}
		right, rightArgs, err := e.buildFilter(n.Right)
		if err != nil {
			return "", nil, err
		}

		args := append(leftArgs, rightArgs...)

		if n.Operator == query.BinaryOpAnd {
			return fmt.Sprintf("(%s) AND (%s)", left, right), args, nil
		} else if n.Operator == query.BinaryOpOr {
			return fmt.Sprintf("(%s) OR (%s)", left, right), args, nil
		}
		return "", nil, query.ErrInvalidQuery

	case *query.ComparisonNode:
		// Handle default search field
		field := n.Field
		if field == "__DEFAULT_SEARCH__" {
			field = e.options.DefaultSearchField
		}

		// Check if field is in allowed list (security)
		if !e.options.IsFieldAllowed(field) {
			return "", nil, query.FieldNotAllowedError(field)
		}

		// Validate field name to prevent SQL injection
		if !e.isValidField(field) {
			return "", nil, query.InvalidFieldNameError(field)
		}

		switch n.Operator {
		case query.OpEqual:
			return fmt.Sprintf("%s = ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpNotEqual:
			return fmt.Sprintf("%s != ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpGreaterThan:
			return fmt.Sprintf("%s > ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpGreaterThanOrEqual:
			return fmt.Sprintf("%s >= ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpLessThan:
			return fmt.Sprintf("%s < ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpLessThanOrEqual:
			return fmt.Sprintf("%s <= ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpLike:
			return fmt.Sprintf("%s LIKE ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpNotLike:
			return fmt.Sprintf("%s NOT LIKE ?", field), []interface{}{e.convertValue(n.Value)}, nil
		case query.OpContains:
			str := e.convertValue(n.Value)
			return fmt.Sprintf("%s LIKE ?", field), []interface{}{fmt.Sprintf("%%%v%%", str)}, nil
		case query.OpIContains:
			str := e.convertValue(n.Value)
			return fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", field), []interface{}{fmt.Sprintf("%%%v%%", str)}, nil
		case query.OpStartsWith:
			str := e.convertValue(n.Value)
			return fmt.Sprintf("%s LIKE ?", field), []interface{}{fmt.Sprintf("%v%%", str)}, nil
		case query.OpEndsWith:
			str := e.convertValue(n.Value)
			return fmt.Sprintf("%s LIKE ?", field), []interface{}{fmt.Sprintf("%%%v", str)}, nil
		case query.OpRegex:
			// Check if regex is disabled
			if e.options.DisableRegex {
				return "", nil, query.ErrRegexNotSupported
			}
			// Note: Regex support varies by database
			// PostgreSQL: ~, MySQL: REGEXP, SQLite: REGEXP (needs extension)
			str := e.convertValue(n.Value)
			return fmt.Sprintf("%s REGEXP ?", field), []interface{}{str}, nil
		case query.OpIn:
			arr := e.convertArrayValue(n.Value)
			if len(arr) == 0 {
				return "1 = 0", []interface{}{}, nil // Empty IN clause
			}
			placeholders := make([]string, len(arr))
			for i := range arr {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", ")), arr, nil
		case query.OpNotIn:
			arr := e.convertArrayValue(n.Value)
			if len(arr) == 0 {
				return "1 = 1", []interface{}{}, nil // Empty NOT IN clause
			}
			placeholders := make([]string, len(arr))
			for i := range arr {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(placeholders, ", ")), arr, nil
		default:
			return "", nil, query.ErrInvalidQuery
		}

	default:
		return "", nil, query.ErrInvalidQuery
	}
}

// isValidField validates field names to prevent SQL injection
// Only allows alphanumeric characters and underscores, must start with letter or underscore
func (e *Executor) isValidField(field string) bool {
	if len(field) == 0 {
		return false
	}

	// First character must be a letter or underscore
	first := field[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Remaining characters must be alphanumeric or underscore
	for i := 1; i < len(field); i++ {
		c := field[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// convertValue converts query values to appropriate types
func (e *Executor) convertValue(val interface{}) interface{} {
	switch v := val.(type) {
	case query.StringValue:
		return string(v)
	case query.IntValue:
		return int64(v)
	case query.FloatValue:
		return float64(v)
	case query.BoolValue:
		return bool(v)
	case query.DateTimeValue:
		return v
	default:
		return val
	}
}

// convertArrayValue converts an array value to a slice
func (e *Executor) convertArrayValue(val interface{}) []interface{} {
	if arrVal, ok := val.(query.ArrayValue); ok {
		result := make([]interface{}, len(arrVal))
		for i, v := range arrVal {
			result[i] = e.convertValue(v)
		}
		return result
	}
	return []interface{}{e.convertValue(val)}
}

// buildCursorFilter builds a WHERE clause for cursor-based pagination
func (e *Executor) buildCursorFilter(cursorData *cursor.CursorData, sortField string, sortOrder string) (string, []interface{}) {
	if cursorData.LastID == nil {
		return "", []interface{}{}
	}

	// Validate field to prevent SQL injection
	if !e.isValidField(sortField) {
		return "", []interface{}{}
	}

	// Reverse direction for previous page
	if cursorData.Direction == "prev" {
		if sortOrder == "ASC" {
			sortOrder = "DESC"
		} else {
			sortOrder = "ASC"
		}
	}

	idFieldName := e.getIDFieldName()

	if e.isIDField(sortField) {
		// Validate ID field to prevent SQL injection
		if !e.isValidField(idFieldName) {
			return "", []interface{}{}
		}
		if sortOrder == "ASC" {
			return fmt.Sprintf("%s > ?", idFieldName), []interface{}{cursorData.LastID}
		}
		return fmt.Sprintf("%s < ?", idFieldName), []interface{}{cursorData.LastID}
	}

	// For non-id sort fields, handle ties with id
	// Validate both fields to prevent SQL injection
	if !e.isValidField(sortField) || !e.isValidField(idFieldName) {
		return "", []interface{}{}
	}

	if sortOrder == "ASC" {
		return fmt.Sprintf("(%s > ? OR (%s = ? AND %s > ?))", sortField, sortField, idFieldName),
			[]interface{}{cursorData.LastSortValue, cursorData.LastSortValue, cursorData.LastID}
	}

	return fmt.Sprintf("(%s < ? OR (%s = ? AND %s < ?))", sortField, sortField, idFieldName),
		[]interface{}{cursorData.LastSortValue, cursorData.LastSortValue, cursorData.LastID}
}
