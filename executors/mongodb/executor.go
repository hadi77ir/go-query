package mongodb

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hadi77ir/go-query/executor"
	"github.com/hadi77ir/go-query/internal/cursor"
	"github.com/hadi77ir/go-query/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Executor is the MongoDB implementation of the executor interface
type Executor struct {
	collection *mongo.Collection
	options    *query.ExecutorOptions
}

// NewExecutor creates a new MongoDB executor
func NewExecutor(collection *mongo.Collection, opts *query.ExecutorOptions) executor.Executor {
	if opts == nil {
		opts = query.DefaultExecutorOptions()
	}
	return &Executor{
		collection: collection,
		options:    opts,
	}
}

// Name returns the name of this executor
func (e *Executor) Name() string {
	return "MongoDB"
}

// Close cleans up resources (MongoDB connections are managed separately)
func (e *Executor) Close() error {
	return nil
}

// Execute runs the query and stores results in dest
// dest must be a pointer to a slice (e.g., &[]MyStruct{} or &[]bson.M{})
func (e *Executor) Execute(ctx context.Context, q *query.Query, dest interface{}) (*query.Result, error) {
	result := &query.Result{}

	// Validate and adjust page size
	pageSize := e.options.ValidatePageSize(q.PageSize)

	// Build MongoDB filter
	filter := bson.M{}
	if q.Filter != nil {
		var err error
		filter, err = e.buildFilter(q.Filter)
		if err != nil {
			result.Error = err
			return result, err
		}
	}

	// Handle cursor-based pagination
	cursorData, err := cursor.Decode(q.Cursor)
	if err != nil {
		result.Error = fmt.Errorf("%w: %v", query.ErrInvalidCursor, err)
		return result, result.Error
	}

	// Count total items
	totalItems, err := e.collection.CountDocuments(ctx, filter)
	if err != nil {
		result.Error = query.NewExecutionError("count documents", err)
		return result, result.Error
	}
	result.TotalItems = totalItems

	// Build find options
	findOpts := options.Find()
	findOpts.SetLimit(int64(pageSize + 1)) // Fetch one extra to check if there's a next page

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
			randomSeed = time.Now().UnixNano()
		}

		// For random ordering, we add a random sort key based on seed
		// We hash each document's _id with the seed to get a consistent random order
		findOpts.SetSort(bson.D{
			{Key: "$expr", Value: bson.M{
				"$mod": bson.A{
					bson.M{"$toLong": "$_id"},
					999999,
				},
			}},
		})

		// Apply offset for cursor pagination in random mode
		if cursorData != nil && cursorData.Offset > 0 {
			findOpts.SetSkip(int64(cursorData.Offset))
		}
	} else {
		// Regular sorting
		sortOrderInt := 1
		if sortOrder == query.SortOrderDesc {
			sortOrderInt = -1
		}
		findOpts.SetSort(bson.D{{Key: sortField, Value: sortOrderInt}})

		// Apply cursor filter for pagination
		if cursorData != nil && cursorData.LastID != nil {
			cursorFilter, err := e.buildCursorFilter(cursorData, sortField, sortOrderInt)
			if err != nil {
				result.Error = err
				return result, result.Error
			}
			// Combine with existing filter
			filter = bson.M{"$and": bson.A{filter, cursorFilter}}
		}
	}

	// Execute query
	mongoCursor, err := e.collection.Find(ctx, filter, findOpts)
	if err != nil {
		result.Error = query.NewExecutionError("execute query", err)
		return result, result.Error
	}
	defer mongoCursor.Close(ctx)

	// Fetch results into dest
	if err := mongoCursor.All(ctx, dest); err != nil {
		result.Error = query.NewExecutionError("fetch results", err)
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

	// Generate cursors - need to get last document
	if result.ItemsReturned > 0 {
		// Get the last item from the slice for cursor generation
		lastIndex := result.ItemsReturned - 1
		lastItem := sliceValue.Index(lastIndex).Interface()

		var lastDoc bson.M
		// Convert to bson.M for field access
		if doc, ok := lastItem.(bson.M); ok {
			lastDoc = doc
		} else {
			// If not bson.M, marshal and unmarshal to get map representation
			data, _ := bson.Marshal(lastItem)
			bson.Unmarshal(data, &lastDoc)
		}

		if hasMore {
			// Generate next cursor
			nextCursorData := &cursor.CursorData{
				Direction: "next",
			}

			if sortOrder == query.SortOrderRandom {
				nextCursorData.Offset = currentOffset + pageSize
				nextCursorData.RandomSeed = randomSeed
			} else {
				idFieldName := e.getIDFieldName()
				nextCursorData.LastID = lastDoc[idFieldName]
				if !e.isIDField(sortField) {
					nextCursorData.LastSortValue = lastDoc[sortField]
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
				// Get first document
				firstItem := sliceValue.Index(0).Interface()
				var firstDoc bson.M
				if doc, ok := firstItem.(bson.M); ok {
					firstDoc = doc
				} else {
					data, _ := bson.Marshal(firstItem)
					bson.Unmarshal(data, &firstDoc)
				}

				idFieldName := e.getIDFieldName()
				prevCursorData.LastID = firstDoc[idFieldName]
				if !e.isIDField(sortField) {
					prevCursorData.LastSortValue = firstDoc[sortField]
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

// buildFilter converts the AST filter into a MongoDB filter
func (e *Executor) buildFilter(node query.Node) (bson.M, error) {
	switch n := node.(type) {
	case *query.BinaryOpNode:
		left, err := e.buildFilter(n.Left)
		if err != nil {
			return nil, err
		}
		right, err := e.buildFilter(n.Right)
		if err != nil {
			return nil, err
		}

		if n.Operator == query.BinaryOpAnd {
			return bson.M{"$and": bson.A{left, right}}, nil
		} else if n.Operator == query.BinaryOpOr {
			return bson.M{"$or": bson.A{left, right}}, nil
		}
		return nil, query.ErrInvalidQuery

	case *query.ComparisonNode:
		// Handle default search field
		field := n.Field
		if field == "__DEFAULT_SEARCH__" {
			field = e.options.DefaultSearchField
		}
		switch n.Operator {
		case query.OpEqual:
			value := e.convertValue(n.Value)
			return bson.M{field: value}, nil
		case query.OpNotEqual:
			value := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$ne": value}}, nil
		case query.OpGreaterThan:
			value := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$gt": value}}, nil
		case query.OpGreaterThanOrEqual:
			value := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$gte": value}}, nil
		case query.OpLessThan:
			value := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$lt": value}}, nil
		case query.OpLessThanOrEqual:
			value := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$lte": value}}, nil
		case query.OpLike:
			// Convert SQL LIKE to MongoDB regex
			pattern := e.likeToRegex(n.Value)
			return bson.M{field: bson.M{"$regex": pattern, "$options": ""}}, nil
		case query.OpNotLike:
			pattern := e.likeToRegex(n.Value)
			return bson.M{field: bson.M{"$not": bson.M{"$regex": pattern, "$options": ""}}}, nil
		case query.OpContains:
			str := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$regex": fmt.Sprintf("%v", str), "$options": ""}}, nil
		case query.OpIContains:
			str := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$regex": fmt.Sprintf("%v", str), "$options": "i"}}, nil
		case query.OpStartsWith:
			str := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$regex": fmt.Sprintf("^%v", str), "$options": ""}}, nil
		case query.OpEndsWith:
			str := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$regex": fmt.Sprintf("%v$", str), "$options": ""}}, nil
		case query.OpRegex:
			// Check if regex is disabled
			if e.options.DisableRegex {
				return nil, query.ErrRegexNotSupported
			}
			str := e.convertValue(n.Value)
			return bson.M{field: bson.M{"$regex": fmt.Sprintf("%v", str), "$options": ""}}, nil
		case query.OpIn:
			arr := e.convertArrayValue(n.Value)
			return bson.M{field: bson.M{"$in": arr}}, nil
		case query.OpNotIn:
			arr := e.convertArrayValue(n.Value)
			return bson.M{field: bson.M{"$nin": arr}}, nil
		default:
			return nil, query.ErrInvalidQuery
		}

	default:
		return nil, query.ErrInvalidQuery
	}
}

// convertValue converts query values to MongoDB-compatible values
func (e *Executor) convertValue(val interface{}) interface{} {
	switch v := val.(type) {
	case query.StringValue:
		// Try to convert to ObjectID if it looks like one
		str := string(v)
		if oid, err := primitive.ObjectIDFromHex(str); err == nil {
			return oid
		}
		return str
	case query.IntValue:
		return int64(v)
	case query.FloatValue:
		return float64(v)
	case query.BoolValue:
		return bool(v)
	case query.DateTimeValue:
		return time.Time(v)
	default:
		return val
	}
}

// getIDFieldName returns the ID field name to use, with fallback defaults
func (e *Executor) getIDFieldName() string {
	if e.options.IDFieldName != "" {
		return e.options.IDFieldName
	}
	return "_id" // Default for MongoDB
}

// isIDField checks if a field name is the ID field
func (e *Executor) isIDField(fieldName string) bool {
	idFieldName := e.getIDFieldName()
	return fieldName == idFieldName
}

// buildCursorFilter builds a filter for cursor-based pagination
func (e *Executor) buildCursorFilter(cursorData *cursor.CursorData, sortField string, sortOrder int) (bson.M, error) {
	if cursorData.LastID == nil {
		return bson.M{}, nil
	}

	// Convert string IDs to ObjectID if needed
	lastID := cursorData.LastID
	if idStr, ok := lastID.(string); ok {
		if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
			lastID = oid
		}
	}

	// Build cursor filter based on sort direction
	if cursorData.Direction == "prev" {
		sortOrder = -sortOrder // Reverse direction for previous page
	}

	idFieldName := e.getIDFieldName()

	if e.isIDField(sortField) {
		if sortOrder > 0 {
			return bson.M{idFieldName: bson.M{"$gt": lastID}}, nil
		}
		return bson.M{idFieldName: bson.M{"$lt": lastID}}, nil
	}

	// For non-id sort fields, we need to handle ties with id
	if sortOrder > 0 {
		return bson.M{
			"$or": bson.A{
				bson.M{sortField: bson.M{"$gt": cursorData.LastSortValue}},
				bson.M{
					sortField:   cursorData.LastSortValue,
					idFieldName: bson.M{"$gt": lastID},
				},
			},
		}, nil
	}

	return bson.M{
		"$or": bson.A{
			bson.M{sortField: bson.M{"$lt": cursorData.LastSortValue}},
			bson.M{
				sortField:   cursorData.LastSortValue,
				idFieldName: bson.M{"$lt": lastID},
			},
		},
	}, nil
}

// hashID generates a hash for random ordering
func hashID(id interface{}, seed int64) int64 {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%v:%d", id, seed)))
	sum := h.Sum(nil)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

// likeToRegex converts SQL LIKE pattern to MongoDB regex
func (e *Executor) likeToRegex(value interface{}) string {
	str := fmt.Sprintf("%v", e.convertValue(value))
	// Escape regex special characters except % and _
	str = escapeRegex(str)
	// Convert SQL wildcards to regex
	str = strings.ReplaceAll(str, "%", ".*")
	str = strings.ReplaceAll(str, "_", ".")
	return "^" + str + "$"
}

// escapeRegex escapes regex special characters
func escapeRegex(s string) string {
	special := []string{".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|", "\\"}
	result := s
	for _, char := range special {
		if char == "*" || char == "?" {
			continue // These are handled by LIKE conversion
		}
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// convertArrayValue converts an array value to a slice for MongoDB
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
