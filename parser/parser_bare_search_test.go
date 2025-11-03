package parser

import (
	"testing"

	"github.com/hadi77ir/go-query/v2/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_BareSearchTerms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*testing.T, *query.Query)
	}{
		{
			name:  "single bare string",
			input: `"hello"`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", comp.Field)
				assert.Equal(t, query.OpContains, comp.Operator)
				assert.Equal(t, query.StringValue("hello"), comp.Value)
			},
		},
		{
			name:  "single bare identifier",
			input: "hello",
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				comp, ok := q.Filter.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", comp.Field)
				assert.Equal(t, query.OpContains, comp.Operator)
				assert.Equal(t, query.StringValue("hello"), comp.Value)
			},
		},
		{
			name:  "bare string with field comparison",
			input: `name LIKE "John%" hello`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp.Operator)

				// Left should be the LIKE comparison
				leftComp, ok := binOp.Left.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "name", leftComp.Field)
				assert.Equal(t, query.OpLike, leftComp.Operator)

				// Right should be the bare search term
				rightComp, ok := binOp.Right.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", rightComp.Field)
				assert.Equal(t, query.OpContains, rightComp.Operator)
				assert.Equal(t, query.StringValue("hello"), rightComp.Value)
			},
		},
		{
			name:  "multiple bare terms",
			input: `hello world`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp.Operator)

				// Both should be search terms
				leftComp, ok := binOp.Left.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", leftComp.Field)

				rightComp, ok := binOp.Right.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", rightComp.Field)
			},
		},
		{
			name:  "bare term with parentheses",
			input: `(status = active or status = pending) hello`,
			expected: func(t *testing.T, q *query.Query) {
				require.NotNil(t, q.Filter)
				binOp, ok := q.Filter.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpAnd, binOp.Operator)

				// Left should be the parenthesized expression
				leftBinOp, ok := binOp.Left.(*query.BinaryOpNode)
				require.True(t, ok)
				assert.Equal(t, query.BinaryOpOr, leftBinOp.Operator)

				// Right should be the bare search term
				rightComp, ok := binOp.Right.(*query.ComparisonNode)
				require.True(t, ok)
				assert.Equal(t, "__DEFAULT_SEARCH__", rightComp.Field)
			},
		},
		{
			name:  "complex query with bare terms",
			input: `page_size = 10 wireless mouse price < 100`,
			expected: func(t *testing.T, q *query.Query) {
				assert.Equal(t, 10, q.PageSize)
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

func TestParser_ParenthesesNesting(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple parentheses",
			input: "(status = active)",
		},
		{
			name:  "nested parentheses",
			input: "((status = active) and (age > 18))",
		},
		{
			name:  "complex nesting",
			input: "((a = 1 and b = 2) or (c = 3 and d = 4)) and e = 5",
		},
		{
			name:  "multiple levels",
			input: "(((a = 1) or (b = 2)) and ((c = 3) or (d = 4)))",
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
		})
	}
}
