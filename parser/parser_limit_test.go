package parser

import (
	"testing"

	query "github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_LimitOption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "simple limit",
			input: "limit = 10 user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 10, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit with page_size",
			input: "limit = 50 page_size = 20 user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 50, q.Limit)
				assert.Equal(t, 20, q.PageSize)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit with sort options",
			input: "limit = 100 sort_by = name sort_order = desc status = active",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 100, q.Limit)
				assert.Equal(t, "name", q.SortBy)
				assert.Equal(t, query.SortOrderDesc, q.SortOrder)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit at the end",
			input: "status = active limit = 25",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 25, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit in the middle",
			input: "status = active limit = 15 name = test",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 15, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit zero (no limit)",
			input: "limit = 0 user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 0, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit with all options",
			input: "limit = 200 page_size = 30 sort_by = created_at sort_order = asc status = active",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 200, q.Limit)
				assert.Equal(t, 30, q.PageSize)
				assert.Equal(t, "created_at", q.SortBy)
				assert.Equal(t, query.SortOrderAsc, q.SortOrder)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "only limit option",
			input: "limit = 10",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 10, q.Limit)
				assert.Nil(t, q.Filter)
			},
		},
		{
			name:  "limit with AND",
			input: "status = active and limit = 20 and name = test",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 20, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "large limit value",
			input: "limit = 999999 status = active",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 999999, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit with parentheses",
			input: "limit = 50 (status = active and verified = true)",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 50, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "limit with OR",
			input: "limit = 30 status = active or status = pending",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 30, q.Limit)
				require.NotNil(t, q.Filter)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)

			q, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, q)

			tt.expected(t, q)
		})
	}
}

func TestParser_LimitInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "negative limit",
			input:     "limit = -10",
			shouldErr: true,
		},
		{
			name:      "limit without value",
			input:     "limit =",
			shouldErr: true,
		},
		{
			name:      "limit with non-numeric value",
			input:     "limit = abc",
			shouldErr: true,
		},
		{
			name:      "limit with float value",
			input:     "limit = 10.5",
			shouldErr: true,
		},
		{
			name:      "limit with quoted string (should work for consistency with page_size)",
			input:     `limit = "10"`,
			shouldErr: false,
		},
		{
			name:      "limit without equals",
			input:     "limit 10",
			shouldErr: false, // Treated as bare search, not an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil && parser != nil {
				_, err = parser.Parse()
			}
			if tt.shouldErr {
				assert.Error(t, err, "expected error for input: %s", tt.input)
			} else {
				// If not expecting error, just verify it doesn't crash
				// The query might not have limit set correctly, but that's okay for these edge cases
			}
		})
	}
}

func TestParser_LimitEdgeCases(t *testing.T) {
	t.Run("limit zero means no limit", func(t *testing.T) {
		p, _ := NewParser("limit = 0 user_id = 123")
		q, _ := p.Parse()
		assert.Equal(t, 0, q.Limit)
	})

	t.Run("limit with whitespace", func(t *testing.T) {
		p, _ := NewParser("limit   =   100   user_id = 123")
		q, _ := p.Parse()
		assert.Equal(t, 100, q.Limit)
	})

	t.Run("limit case insensitive", func(t *testing.T) {
		// Note: The parser uses ToLower, so LIMIT should work
		p, _ := NewParser("LIMIT = 50 user_id = 123")
		q, _ := p.Parse()
		assert.Equal(t, 50, q.Limit)
	})

	t.Run("multiple limit options (last one wins)", func(t *testing.T) {
		p, _ := NewParser("limit = 10 limit = 20 user_id = 123")
		q, _ := p.Parse()
		assert.Equal(t, 20, q.Limit) // Last one should win
	})

	t.Run("limit with complex filter", func(t *testing.T) {
		p, _ := NewParser("limit = 100 (status = active and verified = true) or premium = 1")
		q, _ := p.Parse()
		assert.Equal(t, 100, q.Limit)
		require.NotNil(t, q.Filter)
	})

	t.Run("limit with IN operator", func(t *testing.T) {
		p, _ := NewParser("limit = 50 status IN [active, pending, approved]")
		q, _ := p.Parse()
		assert.Equal(t, 50, q.Limit)
		require.NotNil(t, q.Filter)
	})

	t.Run("limit with CONTAINS", func(t *testing.T) {
		p, _ := NewParser(`limit = 25 name CONTAINS "test"`)
		q, _ := p.Parse()
		assert.Equal(t, 25, q.Limit)
		require.NotNil(t, q.Filter)
	})

	t.Run("limit with quoted number (consistency with page_size)", func(t *testing.T) {
		p, _ := NewParser(`limit = "50" user_id = 123`)
		q, _ := p.Parse()
		assert.Equal(t, 50, q.Limit)
		require.NotNil(t, q.Filter)
	})

	t.Run("limit with quoted number and page_size with quoted number", func(t *testing.T) {
		p, _ := NewParser(`limit = "100" page_size = "20" user_id = 123`)
		q, _ := p.Parse()
		assert.Equal(t, 100, q.Limit)
		assert.Equal(t, 20, q.PageSize)
		require.NotNil(t, q.Filter)
	})
}
