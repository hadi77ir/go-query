package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple comparison",
			input: "user_id = 123",
			expected: []Token{
				{Type: TokenIdentifier, Value: "user_id"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenNumber, Value: "123"},
				{Type: TokenEOF},
			},
		},
		{
			name:  "string value",
			input: `name = "John Doe"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "name"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenString, Value: "John Doe"},
				{Type: TokenEOF},
			},
		},
		{
			name:  "and operator",
			input: "age > 18 and status = active",
			expected: []Token{
				{Type: TokenIdentifier, Value: "age"},
				{Type: TokenOperator, Value: ">"},
				{Type: TokenNumber, Value: "18"},
				{Type: TokenAnd, Value: "and"},
				{Type: TokenIdentifier, Value: "status"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIdentifier, Value: "active"},
				{Type: TokenEOF},
			},
		},
		{
			name:  "or operator",
			input: "type = admin or type = moderator",
			expected: []Token{
				{Type: TokenIdentifier, Value: "type"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIdentifier, Value: "admin"},
				{Type: TokenOr, Value: "or"},
				{Type: TokenIdentifier, Value: "type"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenIdentifier, Value: "moderator"},
				{Type: TokenEOF},
			},
		},
		{
			name:  "parentheses",
			input: "(a = 1 and b = 2) or c = 3",
			expected: []Token{
				{Type: TokenLeftParen, Value: "("},
				{Type: TokenIdentifier, Value: "a"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenNumber, Value: "1"},
				{Type: TokenAnd, Value: "and"},
				{Type: TokenIdentifier, Value: "b"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenNumber, Value: "2"},
				{Type: TokenRightParen, Value: ")"},
				{Type: TokenOr, Value: "or"},
				{Type: TokenIdentifier, Value: "c"},
				{Type: TokenOperator, Value: "="},
				{Type: TokenNumber, Value: "3"},
				{Type: TokenEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.AllTokens()
			require.NoError(t, err)

			require.Equal(t, len(tt.expected), len(tokens), "token count mismatch")

			for i, expected := range tt.expected {
				assert.Equal(t, expected.Type, tokens[i].Type, "token %d type mismatch", i)
				if expected.Value != "" {
					assert.Equal(t, expected.Value, tokens[i].Value, "token %d value mismatch", i)
				}
			}
		})
	}
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"a = b", "="},
		{"a != b", "!="},
		{"a > b", ">"},
		{"a < b", "<"},
		{"a >= b", ">="},
		{"a <= b", "<="},
	}

	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.AllTokens()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 2)
			assert.Equal(t, TokenOperator, tokens[1].Type)
			assert.Equal(t, tt.operator, tokens[1].Value)
		})
	}
}

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		value string
	}{
		{"positive integer", "x = 123", "123"},
		{"negative integer", "x = -456", "-456"},
		{"float", "x = 3.14", "3.14"},
		{"negative float", "x = -2.5", "-2.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.AllTokens()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 3)
			assert.Equal(t, TokenNumber, tokens[2].Type)
			assert.Equal(t, tt.value, tokens[2].Value)
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quotes", `name = "John"`, "John"},
		{"single quotes", `name = 'Jane'`, "Jane"},
		{"with spaces", `text = "hello world"`, "hello world"},
		{"escaped quotes", `text = "He said \"hi\""`, `He said "hi"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.AllTokens()
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 3)
			assert.Equal(t, TokenString, tokens[2].Type)
			assert.Equal(t, tt.expected, tokens[2].Value)
		})
	}
}

func TestLexer_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unterminated string", `name = "hello`},
		{"invalid character", "a @ b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			_, err := lexer.AllTokens()
			assert.Error(t, err)
		})
	}
}

func TestLexer_ComplexQuery(t *testing.T) {
	input := `tag=account:123 and (created_at >= 2020-01-03-0415 or updated_at >= 2020-01-03-0415)`
	lexer := NewLexer(input)
	tokens, err := lexer.AllTokens()
	require.NoError(t, err)

	// Just verify we got reasonable tokens
	assert.Greater(t, len(tokens), 10)
	assert.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)
}
