package parser

import (
	"testing"

	"github.com/hadi77ir/go-query/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_SimpleComparison(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "simple equality",
			input: "user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "user_id", comp.Field)
				assert.Equal(t, query.OpEqual, comp.Operator)
				assert.Equal(t, query.IntValue(123), comp.Value)
			},
		},
		{
			name:  "string value",
			input: `name = "John"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "name", comp.Field)
				assert.Equal(t, query.OpEqual, comp.Operator)
				assert.Equal(t, query.StringValue("John"), comp.Value)
			},
		},
		{
			name:  "greater than",
			input: "age > 18",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "age", comp.Field)
				assert.Equal(t, query.OpGreaterThan, comp.Operator)
				assert.Equal(t, query.IntValue(18), comp.Value)
			},
		},
		{
			name:  "greater than or equal",
			input: "score >= 100",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "score", comp.Field)
				assert.Equal(t, query.OpGreaterThanOrEqual, comp.Operator)
				assert.Equal(t, query.IntValue(100), comp.Value)
			},
		},
		{
			name:  "not equal",
			input: "status != deleted",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "status", comp.Field)
				assert.Equal(t, query.OpNotEqual, comp.Operator)
				assert.Equal(t, query.StringValue("deleted"), comp.Value)
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

func TestParser_BinaryOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "simple AND",
			input: "age > 18 and status = active",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp.Operator)

				leftComp, ok := binOp.Left.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "age", leftComp.Field)
				assert.Equal(t, query.OpGreaterThan, leftComp.Operator)

				rightComp, ok := binOp.Right.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "status", rightComp.Field)
				assert.Equal(t, query.OpEqual, rightComp.Operator)
			},
		},
		{
			name:  "simple OR",
			input: "type = admin or type = moderator",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpOr, binOp.Operator)
			},
		},
		{
			name:  "multiple AND",
			input: "a = 1 and b = 2 and c = 3",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				// Should be left-associative: ((a=1 AND b=2) AND c=3)
				binOp1, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp1.Operator)

				binOp2, ok := binOp1.Left.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp2.Operator)
			},
		},
		{
			name:  "mixed AND/OR with precedence",
			input: "a = 1 or b = 2 and c = 3",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				// Should be: a=1 OR (b=2 AND c=3) because AND has higher precedence
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpOr, binOp.Operator)

				// Right side should be an AND
				rightBinOp, ok := binOp.Right.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, rightBinOp.Operator)
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

func TestParser_Parentheses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "simple parentheses",
			input: "(age > 18 and status = active) or premium = true",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				// Should be: (age>18 AND status=active) OR premium=true
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpOr, binOp.Operator)

				// Left side should be an AND
				leftBinOp, ok := binOp.Left.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, leftBinOp.Operator)
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

func TestParser_QueryOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "page_size",
			input: "page_size = 50 user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 50, q.PageSize)
			},
		},
		{
			name:  "sort_by and sort_order",
			input: "sort_by = created_at sort_order = desc user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, "created_at", q.SortBy)
				assert.Equal(t, query.SortOrderDesc, q.SortOrder)
			},
		},
		{
			name:  "sort_order random",
			input: "sort_order = random status = active",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, query.SortOrderRandom, q.SortOrder)
			},
		},
		{
			name:  "cursor",
			input: "cursor = abc123 user_id = 123",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, "abc123", q.Cursor)
			},
		},
		{
			name:  "multiple options",
			input: "page_size = 25 sort_by = name sort_order = asc status = active",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 25, q.PageSize)
				assert.Equal(t, "name", q.SortBy)
				assert.Equal(t, query.SortOrderAsc, q.SortOrder)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "options at the end",
			input: "status = active sort_by = name sort_order = desc",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, "name", q.SortBy)
				assert.Equal(t, query.SortOrderDesc, q.SortOrder)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "options in the middle",
			input: "status = active page_size = 50 name = test",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 50, q.PageSize)
				require.NotNil(t, q.Filter)
			},
		},
		{
			name:  "options mixed with AND",
			input: "status = active and page_size = 20 and name = test",
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 20, q.PageSize)
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

func TestParser_ComplexExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "requirement example 1",
			input: "tag=account:123",
		},
		{
			name:  "requirement example 2",
			input: "user_id=123",
		},
		{
			name:  "requirement example 3",
			input: "created_at >= 2020-01-03-0415 or updated_at >= 2020-01-03-0415",
		},
		{
			name:  "complex with all features",
			input: "page_size = 20 sort_by = created_at sort_order = desc (status = active and verified = true) or premium = 1",
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

func TestParser_DateTimeValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "custom format",
			input: "created_at >= 2020-01-03-0415",
		},
		{
			name:  "date only",
			input: "date = 2020-01-03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)

			q, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, q)
			require.NotNil(t, q.Filter)

			comp, ok := q.Filter.(*query.ComparisonNode)
			require.True(t, ok)
			_, ok = comp.Value.(query.DateTimeValue)
			assert.True(t, ok, "expected DateTimeValue")
		})
	}
}

func TestParser_FloatValues(t *testing.T) {
	input := "price >= 19.99"
	parser, err := NewParser(input)
	require.NoError(t, err)

	q, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, q.Filter)

	comp, ok := q.Filter.(*query.ComparisonNode)
	require.True(t, ok)
	assert.Equal(t, "price", comp.Field)
	assert.Equal(t, query.FloatValue(19.99), comp.Value)
}

func TestParser_BoolValues(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"active = true", true},
		{"deleted = false", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			require.NoError(t, err)

			q, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, q.Filter)

			comp, ok := q.Filter.(*query.ComparisonNode)
			require.True(t, ok)
			assert.Equal(t, query.BoolValue(tt.expected), comp.Value)
		})
	}
}
