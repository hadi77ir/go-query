package mongodb

import (
	"testing"
	"time"

	"github.com/hadi77ir/go-query/internal/cursor"
	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestExecutor_BuildFilter(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}

	tests := []struct {
		name     string
		input    string
		expected bson.M
	}{
		{
			name:  "simple equality",
			input: "user_id = 123",
			expected: bson.M{
				"user_id": int64(123),
			},
		},
		{
			name:  "string equality",
			input: `name = "John"`,
			expected: bson.M{
				"name": "John",
			},
		},
		{
			name:  "greater than",
			input: "age > 18",
			expected: bson.M{
				"age": bson.M{"$gt": int64(18)},
			},
		},
		{
			name:  "less than or equal",
			input: "score <= 100",
			expected: bson.M{
				"score": bson.M{"$lte": int64(100)},
			},
		},
		{
			name:  "not equal",
			input: "status != deleted",
			expected: bson.M{
				"status": bson.M{"$ne": "deleted"},
			},
		},
		{
			name:  "AND operation",
			input: "age > 18 and status = active",
			expected: bson.M{
				"$and": bson.A{
					bson.M{"age": bson.M{"$gt": int64(18)}},
					bson.M{"status": "active"},
				},
			},
		},
		{
			name:  "OR operation",
			input: "type = admin or type = moderator",
			expected: bson.M{
				"$or": bson.A{
					bson.M{"type": "admin"},
					bson.M{"type": "moderator"},
				},
			},
		},
		{
			name:  "complex expression",
			input: "(age > 18 and status = active) or premium = true",
			expected: bson.M{
				"$or": bson.A{
					bson.M{
						"$and": bson.A{
							bson.M{"age": bson.M{"$gt": int64(18)}},
							bson.M{"status": "active"},
						},
					},
					bson.M{"premium": true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parser.NewParser(tt.input)
			require.NoError(t, err)

			q, err := p.Parse()
			require.NoError(t, err)
			require.NotNil(t, q.Filter)

			filter, err := executor.buildFilter(q.Filter)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, filter)
		})
	}
}

func TestExecutor_ConvertValue(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}

	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "string value",
			value:    query.StringValue("test"),
			expected: "test",
		},
		{
			name:     "int value",
			value:    query.IntValue(123),
			expected: int64(123),
		},
		{
			name:     "float value",
			value:    query.FloatValue(3.14),
			expected: float64(3.14),
		},
		{
			name:     "bool value true",
			value:    query.BoolValue(true),
			expected: true,
		},
		{
			name:     "bool value false",
			value:    query.BoolValue(false),
			expected: false,
		},
		{
			name:     "datetime value",
			value:    query.DateTimeValue(time.Date(2020, 1, 3, 4, 15, 0, 0, time.UTC)),
			expected: time.Date(2020, 1, 3, 4, 15, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.convertValue("test_field", tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutor_ConvertValue_ObjectID(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}

	// Valid ObjectID
	oid := primitive.NewObjectID()
	value := query.StringValue(oid.Hex())
	result, err := executor.convertValue("_id", value)
	require.NoError(t, err)

	resultOID, ok := result.(primitive.ObjectID)
	require.True(t, ok)
	assert.Equal(t, oid, resultOID)

	// Invalid ObjectID (should remain string)
	value = query.StringValue("not-an-objectid")
	result, err = executor.convertValue("_id", value)
	require.NoError(t, err)
	assert.Equal(t, "not-an-objectid", result)
}

func TestExecutorOptions_ValidatePageSize(t *testing.T) {
	opts := &query.ExecutorOptions{
		MaxPageSize:     100,
		DefaultPageSize: 10,
	}

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "valid size",
			input:    50,
			expected: 50,
		},
		{
			name:     "zero size (use default)",
			input:    0,
			expected: 10,
		},
		{
			name:     "negative size (use default)",
			input:    -5,
			expected: 10,
		},
		{
			name:     "exceeds max (use max)",
			input:    200,
			expected: 100,
		},
		{
			name:     "exactly max",
			input:    100,
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := opts.ValidatePageSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutor_BuildCursorFilter(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}

	tests := []struct {
		name       string
		cursorData *cursor.CursorData
		sortField  string
		sortOrder  int
		expected   bson.M
	}{
		{
			name: "simple next page with _id",
			cursorData: &cursor.CursorData{
				LastID:    "507f1f77bcf86cd799439011",
				Direction: "next",
			},
			sortField: "_id",
			sortOrder: 1,
			expected: bson.M{
				"_id": bson.M{"$gt": mustParseObjectID("507f1f77bcf86cd799439011")},
			},
		},
		{
			name: "previous page with _id",
			cursorData: &cursor.CursorData{
				LastID:    "507f1f77bcf86cd799439011",
				Direction: "prev",
			},
			sortField: "_id",
			sortOrder: 1,
			expected: bson.M{
				"_id": bson.M{"$lt": mustParseObjectID("507f1f77bcf86cd799439011")},
			},
		},
		{
			name: "next page with custom sort field",
			cursorData: &cursor.CursorData{
				LastID:        "507f1f77bcf86cd799439011",
				LastSortValue: "2020-01-03",
				Direction:     "next",
			},
			sortField: "created_at",
			sortOrder: 1,
			expected: bson.M{
				"$or": bson.A{
					bson.M{"created_at": bson.M{"$gt": "2020-01-03"}},
					bson.M{
						"created_at": "2020-01-03",
						"_id":        bson.M{"$gt": mustParseObjectID("507f1f77bcf86cd799439011")},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.buildCursorFilter(tt.cursorData, tt.sortField, tt.sortOrder)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutor_Name(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}
	assert.Equal(t, "MongoDB", executor.Name())
}

func TestExecutor_Close(t *testing.T) {
	executor := &Executor{
		options: query.DefaultExecutorOptions(),
	}
	err := executor.Close()
	assert.NoError(t, err)
}

// Helper function to parse ObjectID
func mustParseObjectID(hex string) primitive.ObjectID {
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return oid
}

// Integration test structure (requires actual MongoDB connection)
// These tests are commented out but show how to test with a real MongoDB instance

/*
func TestExecutor_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Connect to MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	// Use test database and collection
	collection := client.Database("test_goquery").Collection("users")

	// Clean up before test
	_, err = collection.DeleteMany(ctx, bson.M{})
	require.NoError(t, err)

	// Insert test data
	testData := []interface{}{
		bson.M{"_id": primitive.NewObjectID(), "name": "Alice", "age": 25, "status": "active"},
		bson.M{"_id": primitive.NewObjectID(), "name": "Bob", "age": 30, "status": "active"},
		bson.M{"_id": primitive.NewObjectID(), "name": "Charlie", "age": 35, "status": "inactive"},
		bson.M{"_id": primitive.NewObjectID(), "name": "Dave", "age": 20, "status": "active"},
	}
	_, err = collection.InsertMany(ctx, testData)
	require.NoError(t, err)

	// Create executor
	executor := NewExecutor(collection, query.DefaultExecutorOptions())

	// Test simple query
	t.Run("simple query", func(t *testing.T) {
		p, err := parser.NewParser("status = active")
		require.NoError(t, err)

		q, err := p.Parse()
		require.NoError(t, err)

		result, err := executor.Execute(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.TotalItems)
		assert.Len(t, result.Data, 3)
	})

	// Test with pagination
	t.Run("pagination", func(t *testing.T) {
		p, err := parser.NewParser("page_size = 2 status = active")
		require.NoError(t, err)

		q, err := p.Parse()
		require.NoError(t, err)

		result, err := executor.Execute(ctx, q)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.TotalItems)
		assert.Len(t, result.Data, 2)
		assert.NotEmpty(t, result.NextPageCursor)
	})

	// Clean up after test
	_, err = collection.DeleteMany(ctx, bson.M{})
	require.NoError(t, err)
}
*/
