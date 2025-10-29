package memory

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hadi77ir/go-query/internal/cursor"
	"github.com/hadi77ir/go-query/query"
)

// FieldGetterFunc is a function that retrieves a field value from an object
// This allows custom field access logic for complex scenarios
// obj is the object to get the field from
// field is the field name to retrieve
// Returns the field value and any error
type FieldGetterFunc func(obj interface{}, field string) (interface{}, error)

// DataSourceFunc is a function that returns the data to query
// This allows the data to be dynamically fetched/updated between queries
type DataSourceFunc func() interface{}

// MemoryExecutorOptions extends ExecutorOptions with memory-specific options
type MemoryExecutorOptions struct {
	*query.ExecutorOptions

	// FieldGetter is an optional custom function to retrieve field values
	// If nil, the executor will use reflection (default behavior)
	// Use this for complex scenarios where reflection doesn't work well
	FieldGetter FieldGetterFunc
}

// MemoryExecutor executes queries on in-memory slices and maps
type MemoryExecutor struct {
	dataSource DataSourceFunc
	options    *MemoryExecutorOptions
}

// NewExecutor creates a new memory executor with static data
// For backwards compatibility - wraps the data in a function
// data must be a slice (e.g., []MyStruct{} or []map[string]interface{}{})
func NewExecutor(data interface{}, opts *query.ExecutorOptions) *MemoryExecutor {
	if opts == nil {
		opts = query.DefaultExecutorOptions()
	}
	return &MemoryExecutor{
		dataSource: func() interface{} { return data },
		options: &MemoryExecutorOptions{
			ExecutorOptions: opts,
			FieldGetter:     nil, // Use reflection by default
		},
	}
}

// NewExecutorWithOptions creates a new memory executor with extended options
// For backwards compatibility - wraps the data in a function
func NewExecutorWithOptions(data interface{}, opts *MemoryExecutorOptions) *MemoryExecutor {
	if opts == nil {
		opts = &MemoryExecutorOptions{
			ExecutorOptions: query.DefaultExecutorOptions(),
			FieldGetter:     nil,
		}
	}
	if opts.ExecutorOptions == nil {
		opts.ExecutorOptions = query.DefaultExecutorOptions()
	}
	return &MemoryExecutor{
		dataSource: func() interface{} { return data },
		options:    opts,
	}
}

// NewExecutorWithDataSource creates a new memory executor with a dynamic data source
// The dataSource function is called each time Execute runs, allowing for live data updates
// This enables querying data that changes between executions
func NewExecutorWithDataSource(dataSource DataSourceFunc, opts *query.ExecutorOptions) *MemoryExecutor {
	if opts == nil {
		opts = query.DefaultExecutorOptions()
	}
	return &MemoryExecutor{
		dataSource: dataSource,
		options: &MemoryExecutorOptions{
			ExecutorOptions: opts,
			FieldGetter:     nil,
		},
	}
}

// NewExecutorWithDataSourceAndOptions creates a new memory executor with both
// a dynamic data source and custom options
func NewExecutorWithDataSourceAndOptions(dataSource DataSourceFunc, opts *MemoryExecutorOptions) *MemoryExecutor {
	if opts == nil {
		opts = &MemoryExecutorOptions{
			ExecutorOptions: query.DefaultExecutorOptions(),
			FieldGetter:     nil,
		}
	}
	if opts.ExecutorOptions == nil {
		opts.ExecutorOptions = query.DefaultExecutorOptions()
	}
	return &MemoryExecutor{
		dataSource: dataSource,
		options:    opts,
	}
}

// Execute runs the query on the in-memory data
func (e *MemoryExecutor) Execute(ctx context.Context, q *query.Query, dest interface{}) (*query.Result, error) {
	// Validate destination
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr || destVal.Elem().Kind() != reflect.Slice {
		return nil, query.ErrInvalidDestination
	}

	// Get source data from the data source function
	data := e.dataSource()
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() == reflect.Ptr {
		dataVal = dataVal.Elem()
	}
	if dataVal.Kind() != reflect.Slice {
		return nil, query.ErrInvalidQuery
	}

	// Filter data
	filtered := []reflect.Value{}
	for i := 0; i < dataVal.Len(); i++ {
		item := dataVal.Index(i)
		if q.Filter == nil {
			filtered = append(filtered, item)
		} else {
			match, err := e.evaluateFilter(q.Filter, item)
			if err != nil {
				// If error is already an ExecutionError, preserve it
				var execErr *query.ExecutionError
				if errors.As(err, &execErr) {
					return nil, err
				}
				return nil, query.NewExecutionError("evaluate filter", err)
			}
			if match {
				filtered = append(filtered, item)
			}
		}
	}

	totalItems := int64(len(filtered))

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

	// Handle random order
	if sortOrder == query.SortOrderRandom {
		if !e.options.AllowRandomOrder {
			return nil, query.ErrRandomOrderNotAllowed
		}
		// Use seed from cursor or generate new one
		seed := int64(42) // Default seed
		if q.Cursor != "" {
			cursorData, err := cursor.Decode(q.Cursor)
			if err == nil && cursorData != nil && cursorData.RandomSeed != 0 {
				seed = cursorData.RandomSeed
			}
		}
		e.shuffleWithSeed(filtered, seed)
	} else {
		// Regular sorting
		e.sortData(filtered, sortField, sortOrder)
	}

	// Apply pagination
	pageSize := e.options.ValidatePageSize(q.PageSize)

	startIdx := 0
	if q.Cursor != "" {
		cursorData, err := cursor.Decode(q.Cursor)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", query.ErrInvalidCursor, err)
		}
		if cursorData != nil {
			startIdx = cursorData.Offset
			if cursorData.Direction == "prev" {
				startIdx -= pageSize
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}
	}

	endIdx := startIdx + pageSize
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	// Get page of results
	pageData := filtered[startIdx:endIdx]

	// Convert to destination type
	destSlice := destVal.Elem()
	destSlice.Set(reflect.MakeSlice(destSlice.Type(), 0, len(pageData)))
	for _, item := range pageData {
		// Convert if needed
		converted := e.convertItem(item, destSlice.Type().Elem())
		destSlice.Set(reflect.Append(destSlice, converted))
	}

	// Generate cursors
	var nextCursor, prevCursor string
	if endIdx < len(filtered) {
		nextCursorData := &cursor.CursorData{
			Offset:    endIdx,
			Direction: "next",
		}
		if sortOrder == query.SortOrderRandom {
			nextCursorData.RandomSeed = 42 // Use consistent seed
		}
		nextCursor, _ = cursor.Encode(nextCursorData)
	}

	if startIdx > 0 {
		prevCursorData := &cursor.CursorData{
			Offset:    startIdx,
			Direction: "prev",
		}
		if sortOrder == query.SortOrderRandom {
			prevCursorData.RandomSeed = 42
		}
		prevCursor, _ = cursor.Encode(prevCursorData)
	}

	return &query.Result{
		NextPageCursor: nextCursor,
		PrevPageCursor: prevCursor,
		TotalItems:     totalItems,
		ShowingFrom:    startIdx + 1,
		ShowingTo:      endIdx,
		ItemsReturned:  len(pageData),
	}, nil
}

// evaluateFilter evaluates a filter node against an item
func (e *MemoryExecutor) evaluateFilter(node query.Node, item reflect.Value) (bool, error) {
	switch n := node.(type) {
	case *query.ComparisonNode:
		return e.evaluateComparison(n, item)
	case *query.BinaryOpNode:
		leftMatch, err := e.evaluateFilter(n.Left, item)
		if err != nil {
			return false, err
		}
		rightMatch, err := e.evaluateFilter(n.Right, item)
		if err != nil {
			return false, err
		}
		if n.Operator == query.BinaryOpAnd {
			return leftMatch && rightMatch, nil
		}
		return leftMatch || rightMatch, nil
	default:
		return false, query.ErrInvalidQuery
	}
}

// evaluateComparison evaluates a comparison against an item
func (e *MemoryExecutor) evaluateComparison(n *query.ComparisonNode, item reflect.Value) (bool, error) {
	// Get field name
	field := n.Field
	if field == "__DEFAULT_SEARCH__" {
		field = e.options.DefaultSearchField
	}

	// Get field value
	fieldValue, err := e.getFieldValue(item, field)
	if err != nil {
		// Check if it's a security error (not in allowed list) or custom field getter error
		if e.options.FieldGetter != nil || errors.Is(err, query.ErrFieldNotAllowed) {
			// Security violations and custom field getter errors should propagate
			return false, err
		}
		// Field not found - no match but not an error
		return false, nil
	}

	// Evaluate operator
	switch n.Operator {
	case query.OpEqual:
		return e.compareEqual(fieldValue, n.Value), nil
	case query.OpNotEqual:
		return !e.compareEqual(fieldValue, n.Value), nil
	case query.OpGreaterThan:
		return e.compareGreater(fieldValue, n.Value, false), nil
	case query.OpGreaterThanOrEqual:
		return e.compareGreater(fieldValue, n.Value, true), nil
	case query.OpLessThan:
		return e.compareLess(fieldValue, n.Value, false), nil
	case query.OpLessThanOrEqual:
		return e.compareLess(fieldValue, n.Value, true), nil
	case query.OpLike:
		return e.evaluateLike(fieldValue, n.Value), nil
	case query.OpNotLike:
		return !e.evaluateLike(fieldValue, n.Value), nil
	case query.OpContains:
		return e.evaluateContains(fieldValue, n.Value, true), nil
	case query.OpIContains:
		return e.evaluateContains(fieldValue, n.Value, false), nil
	case query.OpStartsWith:
		return e.evaluateStartsWith(fieldValue, n.Value), nil
	case query.OpEndsWith:
		return e.evaluateEndsWith(fieldValue, n.Value), nil
	case query.OpRegex:
		// Check if regex is disabled
		if e.options.ExecutorOptions.DisableRegex {
			return false, query.ErrRegexNotSupported
		}
		return e.evaluateRegex(fieldValue, n.Value), nil
	case query.OpIn:
		return e.evaluateIn(fieldValue, n.Value), nil
	case query.OpNotIn:
		return !e.evaluateIn(fieldValue, n.Value), nil
	default:
		return false, query.ErrInvalidQuery
	}
}

// getFieldValue gets a field value from an item (struct or map)
func (e *MemoryExecutor) getFieldValue(item reflect.Value, fieldName string) (interface{}, error) {
	// Check if field is allowed (security check)
	if !e.options.ExecutorOptions.IsFieldAllowed(fieldName) {
		return nil, query.FieldNotAllowedError(fieldName)
	}

	// Use custom field getter if provided
	if e.options.FieldGetter != nil {
		// Get the actual interface value
		var itemInterface interface{}
		if item.Kind() == reflect.Ptr {
			itemInterface = item.Interface()
		} else {
			// Make it addressable so we can get its address
			if item.CanAddr() {
				itemInterface = item.Addr().Interface()
			} else {
				itemInterface = item.Interface()
			}
		}
		val, err := e.options.FieldGetter(itemInterface, fieldName)
		// Always propagate errors from custom field getters
		if err != nil {
			// Wrap error with appropriate operation name
			return nil, query.NewExecutionError("custom getter", err)
		}
		return val, nil
	}

	// Default: Use reflection
	// Dereference pointer
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
	}

	switch item.Kind() {
	case reflect.Struct:
		// Try to find field by name (case-insensitive)
		typ := item.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			// Check field name or json/bson tag
			if strings.EqualFold(field.Name, fieldName) {
				return item.Field(i).Interface(), nil
			}
			// Check tags
			if tag := field.Tag.Get("json"); tag != "" && strings.EqualFold(strings.Split(tag, ",")[0], fieldName) {
				return item.Field(i).Interface(), nil
			}
			if tag := field.Tag.Get("bson"); tag != "" && strings.EqualFold(strings.Split(tag, ",")[0], fieldName) {
				return item.Field(i).Interface(), nil
			}
		}
		return nil, query.ErrInvalidQuery

	case reflect.Map:
		// Try exact match first
		val := item.MapIndex(reflect.ValueOf(fieldName))
		if val.IsValid() {
			return val.Interface(), nil
		}
		// Try case-insensitive
		iter := item.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() == reflect.String && strings.EqualFold(key.String(), fieldName) {
				return iter.Value().Interface(), nil
			}
		}
		return nil, query.ErrInvalidQuery

	default:
		return nil, query.ErrInvalidQuery
	}
}

// Comparison helpers
func (e *MemoryExecutor) compareEqual(a, b interface{}) bool {
	aFloat, aOk := e.toFloat64(a)
	bFloat, bOk := e.toFloat64(b)
	if aOk && bOk {
		return aFloat == bFloat
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (e *MemoryExecutor) compareGreater(a, b interface{}, orEqual bool) bool {
	aFloat, aOk := e.toFloat64(a)
	bFloat, bOk := e.toFloat64(b)
	if aOk && bOk {
		if orEqual {
			return aFloat >= bFloat
		}
		return aFloat > bFloat
	}
	// String comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if orEqual {
		return aStr >= bStr
	}
	return aStr > bStr
}

func (e *MemoryExecutor) compareLess(a, b interface{}, orEqual bool) bool {
	aFloat, aOk := e.toFloat64(a)
	bFloat, bOk := e.toFloat64(b)
	if aOk && bOk {
		if orEqual {
			return aFloat <= bFloat
		}
		return aFloat < bFloat
	}
	// String comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if orEqual {
		return aStr <= bStr
	}
	return aStr < bStr
}

func (e *MemoryExecutor) toFloat64(v interface{}) (float64, bool) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	case reflect.String:
		// Try to parse string as float
		f, err := strconv.ParseFloat(val.String(), 64)
		if err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func (e *MemoryExecutor) evaluateLike(fieldVal, pattern interface{}) bool {
	str := fmt.Sprintf("%v", fieldVal)
	patternStr := fmt.Sprintf("%v", pattern)

	// Convert SQL LIKE to regex
	regexPattern := "^" + regexp.QuoteMeta(patternStr)
	regexPattern = strings.ReplaceAll(regexPattern, "%", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "_", ".")
	regexPattern += "$"

	matched, _ := regexp.MatchString(regexPattern, str)
	return matched
}

func (e *MemoryExecutor) evaluateContains(fieldVal, substr interface{}, caseSensitive bool) bool {
	str := fmt.Sprintf("%v", fieldVal)
	subStr := fmt.Sprintf("%v", substr)

	if !caseSensitive {
		str = strings.ToLower(str)
		subStr = strings.ToLower(subStr)
	}

	return strings.Contains(str, subStr)
}

func (e *MemoryExecutor) evaluateStartsWith(fieldVal, prefix interface{}) bool {
	str := fmt.Sprintf("%v", fieldVal)
	prefixStr := fmt.Sprintf("%v", prefix)
	return strings.HasPrefix(str, prefixStr)
}

func (e *MemoryExecutor) evaluateEndsWith(fieldVal, suffix interface{}) bool {
	str := fmt.Sprintf("%v", fieldVal)
	suffixStr := fmt.Sprintf("%v", suffix)
	return strings.HasSuffix(str, suffixStr)
}

func (e *MemoryExecutor) evaluateRegex(fieldVal, pattern interface{}) bool {
	str := fmt.Sprintf("%v", fieldVal)
	patternStr := fmt.Sprintf("%v", pattern)
	matched, _ := regexp.MatchString(patternStr, str)
	return matched
}

func (e *MemoryExecutor) evaluateIn(fieldVal, arrayVal interface{}) bool {
	// Convert array to slice
	arr := reflect.ValueOf(arrayVal)
	if arr.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < arr.Len(); i++ {
		if e.compareEqual(fieldVal, arr.Index(i).Interface()) {
			return true
		}
	}
	return false
}

// sortData sorts a slice of reflect.Values
func (e *MemoryExecutor) sortData(data []reflect.Value, sortField string, sortOrder query.SortOrder) {
	sort.Slice(data, func(i, j int) bool {
		valI, errI := e.getFieldValue(data[i], sortField)
		valJ, errJ := e.getFieldValue(data[j], sortField)

		if errI != nil || errJ != nil {
			return false
		}

		less := e.compareLess(valI, valJ, false)
		if sortOrder == query.SortOrderDesc {
			return !less
		}
		return less
	})
}

// shuffleWithSeed shuffles data with a seed for reproducibility
func (e *MemoryExecutor) shuffleWithSeed(data []reflect.Value, seed int64) {
	// Simple deterministic shuffle using seed
	if seed == 0 {
		seed = 42
	}

	// Fisher-Yates shuffle with seeded random
	for i := len(data) - 1; i > 0; i-- {
		// Simple LCG random number generator
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		j := int(seed) % (i + 1)
		data[i], data[j] = data[j], data[i]
	}
}

// convertItem converts an item to the destination type
func (e *MemoryExecutor) convertItem(item reflect.Value, destType reflect.Type) reflect.Value {
	// If types match, return as-is
	if item.Type() == destType {
		return item
	}

	// If source is ptr and dest is not, dereference
	if item.Kind() == reflect.Ptr && destType.Kind() != reflect.Ptr {
		item = item.Elem()
		if item.Type() == destType {
			return item
		}
	}

	// If dest is ptr and source is not, take address
	if item.Kind() != reflect.Ptr && destType.Kind() == reflect.Ptr {
		if item.CanAddr() {
			addr := item.Addr()
			if addr.Type() == destType {
				return addr
			}
		}
	}

	// Try to convert
	if item.Type().ConvertibleTo(destType) {
		return item.Convert(destType)
	}

	// As last resort, return as-is and let it fail
	return item
}

// Name returns the executor name
func (e *MemoryExecutor) Name() string {
	return "memory"
}

// Close does nothing for memory executor
func (e *MemoryExecutor) Close() error {
	return nil
}
