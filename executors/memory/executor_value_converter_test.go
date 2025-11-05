package memory

import (
	"context"
	"testing"

	"github.com/hadi77ir/go-query/parser"
	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProductWithFeatures represents a product with enum features
type ProductWithFeatures struct {
	ID       int
	Name     string
	Features []int // Stored as integers: 2 = usbc, 3 = bluetooth, 4 = wifi
}

func getTestDataWithFeatures() []ProductWithFeatures {
	return []ProductWithFeatures{
		{ID: 1, Name: "Laptop", Features: []int{2, 3}},      // usbc, bluetooth
		{ID: 2, Name: "Phone", Features: []int{3, 4}},       // bluetooth, wifi
		{ID: 3, Name: "Tablet", Features: []int{2, 4}},      // usbc, wifi
		{ID: 4, Name: "Mouse", Features: []int{3}},          // bluetooth
		{ID: 5, Name: "Keyboard", Features: []int{2, 3, 4}}, // usbc, bluetooth, wifi
		{ID: 6, Name: "Monitor", Features: []int{2}},        // usbc
	}
}

// TestMemoryExecutor_ValueConverter tests the ValueConverter functionality
func TestMemoryExecutor_ValueConverter(t *testing.T) {
	data := getTestDataWithFeatures()

	// Create a converter that converts enum strings to integers
	converter := func(field string, value interface{}) (interface{}, error) {
		if field == "features" {
			if str, ok := value.(string); ok {
				switch str {
				case "usbc":
					return 2, nil
				case "bluetooth":
					return 3, nil
				case "wifi":
					return 4, nil
				}
			}
		}
		return value, nil // No conversion, use original value
	}

	opts := query.DefaultExecutorOptions()
	opts.ValueConverter = converter
	opts.DefaultSortField = "id"
	executor := NewExecutor(data, opts)
	ctx := context.Background()

	t.Run("single enum value conversion with equals", func(t *testing.T) {
		// Query: features = "usbc" should convert to features = 2
		// But since features is an array, we need to use CONTAINS
		p, err := parser.NewParser(`features CONTAINS "usbc"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// Should find: Laptop(1), Tablet(3), Keyboard(5), Monitor(6) = 4 items
		assert.Equal(t, 4, len(products))
		assert.Equal(t, int64(4), result.TotalItems)
	})

	t.Run("array CONTAINS with enum conversion", func(t *testing.T) {
		p, err := parser.NewParser(`features CONTAINS "bluetooth"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// Should find: Laptop(1), Phone(2), Mouse(4), Keyboard(5) = 4 items
		assert.Equal(t, 4, len(products))
		assert.Equal(t, int64(4), result.TotalItems)
	})

	t.Run("IN operator with enum conversion", func(t *testing.T) {
		p, err := parser.NewParser(`features IN ["usbc", "bluetooth"]`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		_, err = executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// This checks if features field (which is an array) contains any of the values
		// Since we're checking array field, CONTAINS would be more appropriate
		// But IN works with single values, so this might not match as expected
		// Let's test with a different approach - checking if the field value is in the array
		// Actually, IN checks if fieldValue is in the array, so for array fields it won't work as expected
		// This test demonstrates that IN doesn't work with array fields (which is expected)
		// The correct way is to use CONTAINS for array fields
		assert.Equal(t, 0, len(products)) // IN doesn't work with array fields
	})

	t.Run("multiple enum values in CONTAINS", func(t *testing.T) {
		// Test that we can query for multiple enum values
		// Note: CONTAINS with array field checks if any element matches
		p, err := parser.NewParser(`features CONTAINS "wifi"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// Should find: Phone(2), Tablet(3), Keyboard(5) = 3 items
		assert.Equal(t, 3, len(products))
		assert.Equal(t, int64(3), result.TotalItems)
	})

	t.Run("converter not applied to other fields", func(t *testing.T) {
		// Test that converter only applies to "features" field
		p, err := parser.NewParser(`name = "Laptop"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		_, err = executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 1, len(products))
		assert.Equal(t, "Laptop", products[0].Name)
	})

	t.Run("converter with non-matching enum value", func(t *testing.T) {
		// Test that non-matching enum values don't cause errors
		p, err := parser.NewParser(`features CONTAINS "nonexistent"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		_, err = executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 0, len(products))
	})

	t.Run("converter with integer values (no conversion needed)", func(t *testing.T) {
		// Test that integer values pass through when they don't match enum strings
		// Create a converter that handles integers
		intConverter := func(field string, value interface{}) (interface{}, error) {
			if field == "id" {
				// Convert string "1" to int 1
				if str, ok := value.(string); ok {
					switch str {
					case "1":
						return 1, nil
					case "2":
						return 2, nil
					}
				}
			}
			return value, nil
		}

		opts := query.DefaultExecutorOptions()
		opts.ValueConverter = intConverter
		opts.DefaultSortField = "id"
		executor := NewExecutor(data, opts)

		p, err := parser.NewParser(`id = "1"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithFeatures
		_, err = executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 1, len(products))
		assert.Equal(t, 1, products[0].ID)
	})
}

// TestMemoryExecutor_ArrayContains tests array CONTAINS functionality
func TestMemoryExecutor_ArrayContains(t *testing.T) {
	type ProductWithArray struct {
		ID   int
		Name string
		Tags []string
	}

	data := []ProductWithArray{
		{ID: 1, Name: "Product1", Tags: []string{"electronics", "popular"}},
		{ID: 2, Name: "Product2", Tags: []string{"accessories", "new"}},
		{ID: 3, Name: "Product3", Tags: []string{"electronics", "popular", "featured"}},
		{ID: 4, Name: "Product4", Tags: []string{"accessories"}},
	}

	opts := query.DefaultExecutorOptions()
	opts.DefaultSortField = "id"
	executor := NewExecutor(data, opts)
	ctx := context.Background()

	t.Run("array CONTAINS with string value", func(t *testing.T) {
		p, err := parser.NewParser(`tags CONTAINS "electronics"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithArray
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// Should find: Product1(1), Product3(3) = 2 items
		assert.Equal(t, 2, len(products))
		assert.Equal(t, int64(2), result.TotalItems)
	})

	t.Run("array CONTAINS with non-matching value", func(t *testing.T) {
		p, err := parser.NewParser(`tags CONTAINS "nonexistent"`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithArray
		_, err = executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		assert.Equal(t, 0, len(products))
	})

	t.Run("array CONTAINS with integer array", func(t *testing.T) {
		type ProductWithIntArray struct {
			ID       int
			Name     string
			Features []int
		}

		data := []ProductWithIntArray{
			{ID: 1, Name: "Product1", Features: []int{2, 3, 4}},
			{ID: 2, Name: "Product2", Features: []int{3, 5}},
			{ID: 3, Name: "Product3", Features: []int{2}},
		}

		opts := query.DefaultExecutorOptions()
		opts.DefaultSortField = "id"
		executor := NewExecutor(data, opts)

		p, err := parser.NewParser(`features CONTAINS 2`)
		require.NoError(t, err)
		q, err := p.Parse()
		require.NoError(t, err)

		var products []ProductWithIntArray
		result, err := executor.Execute(ctx, q, "", &products)
		require.NoError(t, err)
		// Should find: Product1(1), Product3(3) = 2 items
		assert.Equal(t, 2, len(products))
		assert.Equal(t, int64(2), result.TotalItems)
	})
}
