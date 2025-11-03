package parser

import (
	"testing"

	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_StringMatchingOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "LIKE operator",
			input: `name LIKE "%John%"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "name", comp.Field)
				assert.Equal(t, query.OpLike, comp.Operator)
				assert.Equal(t, query.StringValue("%John%"), comp.Value)
			},
		},
		{
			name:  "NOT LIKE operator",
			input: `email NOT LIKE "%@spam.com"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "email", comp.Field)
				assert.Equal(t, query.OpNotLike, comp.Operator)
			},
		},
		{
			name:  "CONTAINS operator",
			input: "description CONTAINS error",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "description", comp.Field)
				assert.Equal(t, query.OpContains, comp.Operator)
			},
		},
		{
			name:  "ICONTAINS operator",
			input: "title ICONTAINS hello",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "title", comp.Field)
				assert.Equal(t, query.OpIContains, comp.Operator)
			},
		},
		{
			name:  "STARTS_WITH operator",
			input: `path STARTS_WITH "/api"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "path", comp.Field)
				assert.Equal(t, query.OpStartsWith, comp.Operator)
			},
		},
		{
			name:  "ENDS_WITH operator",
			input: `filename ENDS_WITH ".pdf"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "filename", comp.Field)
				assert.Equal(t, query.OpEndsWith, comp.Operator)
			},
		},
		{
			name:  "REGEX operator",
			input: `pattern REGEX "^[A-Z][0-9]+"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "pattern", comp.Field)
				assert.Equal(t, query.OpRegex, comp.Operator)
				assert.Equal(t, query.StringValue("^[A-Z][0-9]+"), comp.Value)
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

func TestParser_ArrayOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "IN with integers",
			input: "status IN [1, 2, 3]",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "status", comp.Field)
				assert.Equal(t, query.OpIn, comp.Operator)

				arr, ok := comp.Value.(query.ArrayValue)
				require.True(t, ok)
				assert.Len(t, arr, 3)
				assert.Equal(t, query.IntValue(1), arr[0])
				assert.Equal(t, query.IntValue(2), arr[1])
				assert.Equal(t, query.IntValue(3), arr[2])
			},
		},
		{
			name:  "IN with strings",
			input: `role IN ["admin", "moderator", "user"]`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "role", comp.Field)
				assert.Equal(t, query.OpIn, comp.Operator)

				arr, ok := comp.Value.(query.ArrayValue)
				require.True(t, ok)
				assert.Len(t, arr, 3)
			},
		},
		{
			name:  "NOT IN operator",
			input: "country NOT IN [US, UK, CA]",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "country", comp.Field)
				assert.Equal(t, query.OpNotIn, comp.Operator)

				arr, ok := comp.Value.(query.ArrayValue)
				require.True(t, ok)
				assert.Len(t, arr, 3)
			},
		},
		{
			name:  "IN with mixed types",
			input: `id IN [123, "abc", 456]`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)

				arr, ok := comp.Value.(query.ArrayValue)
				require.True(t, ok)
				assert.Len(t, arr, 3)
				assert.Equal(t, query.IntValue(123), arr[0])
				assert.Equal(t, query.StringValue("abc"), arr[1])
				assert.Equal(t, query.IntValue(456), arr[2])
			},
		},
		{
			name:  "empty array",
			input: "id IN []",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)

				arr, ok := comp.Value.(query.ArrayValue)
				require.True(t, ok)
				assert.Len(t, arr, 0)
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

func TestParser_ComplexWithNewOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "mixed operators with AND",
			input: "name CONTAINS john and age >= 18",
		},
		{
			name:  "IN with OR",
			input: "status IN [active, pending] or priority = high",
		},
		{
			name:  "LIKE with NOT IN",
			input: `email LIKE "%@example.com" and role NOT IN [guest, banned]`,
		},
		{
			name:  "all features combined",
			input: `page_size = 20 sort_by = created_at (title CONTAINS "search" or description ICONTAINS "search") and status IN [published, approved]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)

			q, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, q)
		})
	}
}
