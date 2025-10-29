package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutorOptions_ValidatePageSize(t *testing.T) {
	opts := &ExecutorOptions{
		MaxPageSize:     100,
		DefaultPageSize: 10,
	}

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero returns default", 0, 10},
		{"negative returns default", -5, 10},
		{"within range unchanged", 50, 50},
		{"exceeds max returns max", 200, 100},
		{"at max unchanged", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := opts.ValidatePageSize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultExecutorOptions(t *testing.T) {
	opts := DefaultExecutorOptions()

	assert.Equal(t, 100, opts.MaxPageSize)
	assert.Equal(t, 10, opts.DefaultPageSize)
	assert.Equal(t, "_id", opts.DefaultSortField)
	assert.Equal(t, SortOrderAsc, opts.DefaultSortOrder)
	assert.True(t, opts.AllowRandomOrder)
	assert.Equal(t, "RANDOM()", opts.RandomFunctionName)
	assert.Empty(t, opts.IDFieldName) // Empty by default (executors set their own defaults)
	assert.Equal(t, "name", opts.DefaultSearchField)
	assert.Empty(t, opts.AllowedFields) // No restrictions by default
}

func TestExecutorOptions_EdgeCases(t *testing.T) {
	t.Run("zero max page size", func(t *testing.T) {
		opts := &ExecutorOptions{
			MaxPageSize:     0,
			DefaultPageSize: 10,
		}
		// When MaxPageSize is 0, there's no maximum limit
		result := opts.ValidatePageSize(5)
		// Returns the requested size (no capping since max is 0)
		assert.Equal(t, 5, result)

		// Large request also works (no limit)
		result = opts.ValidatePageSize(1000)
		assert.Equal(t, 1000, result)
	})

	t.Run("default larger than max", func(t *testing.T) {
		opts := &ExecutorOptions{
			MaxPageSize:     10,
			DefaultPageSize: 50,
		}
		result := opts.ValidatePageSize(0)
		assert.Equal(t, 50, result)
	})
}

func TestExecutorOptions_IsFieldAllowed(t *testing.T) {
	t.Run("empty list allows all fields", func(t *testing.T) {
		opts := &ExecutorOptions{
			AllowedFields: []string{},
		}
		assert.True(t, opts.IsFieldAllowed("any_field"))
		assert.True(t, opts.IsFieldAllowed("password"))
		assert.True(t, opts.IsFieldAllowed("ssn"))
	})

	t.Run("nil list allows all fields", func(t *testing.T) {
		opts := &ExecutorOptions{
			AllowedFields: nil,
		}
		assert.True(t, opts.IsFieldAllowed("any_field"))
	})

	t.Run("restricted list only allows specified fields", func(t *testing.T) {
		opts := &ExecutorOptions{
			AllowedFields: []string{"name", "email", "age"},
		}
		assert.True(t, opts.IsFieldAllowed("name"))
		assert.True(t, opts.IsFieldAllowed("email"))
		assert.True(t, opts.IsFieldAllowed("age"))
		assert.False(t, opts.IsFieldAllowed("password"))
		assert.False(t, opts.IsFieldAllowed("ssn"))
		assert.False(t, opts.IsFieldAllowed("credit_card"))
	})

	t.Run("case sensitive matching", func(t *testing.T) {
		opts := &ExecutorOptions{
			AllowedFields: []string{"Name", "Email"},
		}
		assert.True(t, opts.IsFieldAllowed("Name"))
		assert.False(t, opts.IsFieldAllowed("name")) // Case sensitive
		assert.True(t, opts.IsFieldAllowed("Email"))
		assert.False(t, opts.IsFieldAllowed("email"))
	})

	t.Run("special field names", func(t *testing.T) {
		opts := &ExecutorOptions{
			AllowedFields: []string{"_id", "created_at", "user.name"},
		}
		assert.True(t, opts.IsFieldAllowed("_id"))
		assert.True(t, opts.IsFieldAllowed("created_at"))
		assert.True(t, opts.IsFieldAllowed("user.name"))
		assert.False(t, opts.IsFieldAllowed("user.password"))
	})
}
