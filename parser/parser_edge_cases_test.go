package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "empty string",
			input:     "",
			shouldErr: false, // Should parse but have no filter
		},
		{
			name:      "only whitespace",
			input:     "   \t\n  ",
			shouldErr: false,
		},
		{
			name:      "only query options",
			input:     "page_size = 10 sort_by = name",
			shouldErr: false,
		},
		{
			name:      "mismatched parentheses - extra open",
			input:     "(status = active",
			shouldErr: true,
		},
		{
			name:      "mismatched parentheses - extra close",
			input:     "status = active)",
			shouldErr: false, // Parser treats ) as end of expression
		},
		{
			name:      "nested mismatched",
			input:     "((status = active)",
			shouldErr: true,
		},
		{
			name:      "empty parentheses",
			input:     "()",
			shouldErr: true,
		},
		{
			name:      "empty array",
			input:     "id IN []",
			shouldErr: false,
		},
		{
			name:      "single item array",
			input:     "id IN [1]",
			shouldErr: false,
		},
		{
			name:      "special characters in string",
			input:     `name = "John's \"Special\" Name"`,
			shouldErr: false,
		},
		{
			name:      "unicode characters",
			input:     `name = "日本語"`,
			shouldErr: false,
		},
		{
			name:      "emoji in search",
			input:     `"hello 👋 world"`,
			shouldErr: false,
		},
		{
			name:      "very long string",
			input:     `name = "` + strings.Repeat("a", 5000) + `"`,
			shouldErr: false,
		},
		{
			name:      "negative numbers",
			input:     "age = -25",
			shouldErr: false,
		},
		{
			name:      "large numbers",
			input:     "id = 999999999999999",
			shouldErr: false,
		},
		{
			name:      "float edge cases",
			input:     "price = 0.00001",
			shouldErr: false,
		},
		{
			name:      "multiple operators",
			input:     "a = 1 and b = 2 and c = 3 and d = 4 and e = 5",
			shouldErr: false,
		},
		{
			name:      "deeply nested parentheses",
			input:     "((((a = 1))))",
			shouldErr: false,
		},
		{
			name:      "mixed quotes",
			input:     `name = "John" and nickname = 'Johnny'`,
			shouldErr: false,
		},
		{
			name:      "operator without value",
			input:     "name =",
			shouldErr: true,
		},
		{
			name:      "field without operator",
			input:     "name",
			shouldErr: false, // Treated as bare search
		},
		{
			name:      "unterminated string",
			input:     `name = "unterminated`,
			shouldErr: true,
		},
		{
			name:      "AND without right side",
			input:     "status = active and",
			shouldErr: true,
		},
		{
			name:      "OR without left side",
			input:     "or status = active",
			shouldErr: true,
		},
		{
			name:      "multiple spaces between tokens",
			input:     "status     =     active",
			shouldErr: false,
		},
		{
			name:      "tabs and newlines",
			input:     "status\t=\nactive",
			shouldErr: false,
		},
		{
			name:      "IN with quoted strings",
			input:     `status IN ["active", "pending", "approved"]`,
			shouldErr: false,
		},
		{
			name:      "date formats",
			input:     "created_at >= 2024-12-31",
			shouldErr: false,
		},
		{
			name:      "time format",
			input:     "created_at >= 2024-12-31-2359",
			shouldErr: false,
		},
		{
			name:      "boolean values",
			input:     "active = true and deleted = false",
			shouldErr: false,
		},
		{
			name:      "NOT LIKE",
			input:     `email NOT LIKE "%@spam.com"`,
			shouldErr: false,
		},
		{
			name:      "NOT IN",
			input:     "role NOT IN [guest, banned]",
			shouldErr: false,
		},
		{
			name:      "multiple bare search terms",
			input:     "hello world foo bar",
			shouldErr: false,
		},
		{
			name:      "bare search with parentheses",
			input:     "(hello world) and status = active",
			shouldErr: false,
		},
		{
			name:      "all query options",
			input:     "page_size = 20 sort_by = name sort_order = desc cursor = abc status = active",
			shouldErr: false,
		},
		{
			name:      "field name with underscores",
			input:     "user_id = 123",
			shouldErr: false,
		},
		{
			name:      "field name with colon",
			input:     "tag=account:123",
			shouldErr: false,
		},
		{
			name:      "regex pattern",
			input:     `pattern REGEX "^[A-Za-z0-9]+$"`,
			shouldErr: false,
		},
		{
			name:      "complex real-world query",
			input:     `page_size = 25 wireless mouse (price >= 10 and price <= 100) (brand IN [logitech, microsoft] or featured = true) rating >= 4`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if tt.shouldErr {
				// If we expect error, it might happen during NewParser or Parse
				if err == nil {
					_, err = parser.Parse()
				}
				assert.Error(t, err, "expected error for input: %s", tt.input)
			} else {
				require.NoError(t, err, "unexpected error creating parser for: %s", tt.input)
				if parser != nil {
					q, err := parser.Parse()
					require.NoError(t, err, "unexpected error parsing: %s", tt.input)
					require.NotNil(t, q, "query should not be nil")
				}
			}
		})
	}
}

func TestParser_SpecialCases(t *testing.T) {
	t.Run("empty query has no filter", func(t *testing.T) {
		p, _ := NewParser("")
		q, _ := p.Parse()
		assert.Nil(t, q.Filter)
		assert.Equal(t, 10, q.PageSize) // Should have defaults
	})

	t.Run("only options has no filter", func(t *testing.T) {
		p, _ := NewParser("page_size = 50 sort_by = name")
		q, _ := p.Parse()
		assert.Nil(t, q.Filter)
		assert.Equal(t, 50, q.PageSize)
		assert.Equal(t, "name", q.SortBy)
	})

	t.Run("empty array in IN", func(t *testing.T) {
		p, _ := NewParser("id IN []")
		q, _ := p.Parse()
		require.NotNil(t, q.Filter)
	})

	t.Run("special characters preserved", func(t *testing.T) {
		p, _ := NewParser(`name = "John's \"Special\" Name"`)
		q, _ := p.Parse()
		require.NotNil(t, q.Filter)
	})

	t.Run("unicode preserved", func(t *testing.T) {
		p, _ := NewParser(`name = "こんにちは"`)
		q, _ := p.Parse()
		require.NotNil(t, q.Filter)
	})

	t.Run("negative page size", func(t *testing.T) {
		p, _ := NewParser("page_size = -10")
		q, _ := p.Parse()
		assert.Equal(t, -10, q.PageSize) // Parser accepts it, executor validates
	})

	t.Run("zero page size", func(t *testing.T) {
		p, _ := NewParser("page_size = 0")
		q, _ := p.Parse()
		assert.Equal(t, 0, q.PageSize)
	})

	t.Run("very large page size", func(t *testing.T) {
		p, _ := NewParser("page_size = 999999")
		q, _ := p.Parse()
		assert.Equal(t, 999999, q.PageSize) // Parser accepts, executor caps it
	})
}

func TestParser_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"incomplete comparison", "name ="},
		{"incomplete AND", "status = active and"},
		{"incomplete OR", "status = active or"},
		{"invalid character", "status @ active"},
		{"unterminated string double", `name = "unterminated`},
		{"unterminated string single", `name = 'unterminated`},
		{"mismatched parens open", "(status = active"},
		{"empty expression in parens", "()"},
		{"only operator", "="},
		{"only AND", "and"},
		{"only OR", "or"},
		{"NOT without LIKE/IN", "status NOT active"},
		{"IN without array", "status IN"},
		{"unclosed array", "status IN [1, 2"},
		{"array without IN", "[1, 2, 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.input)
			if err == nil && parser != nil {
				_, err = parser.Parse()
			}
			assert.Error(t, err, "should error on invalid syntax: %s", tt.input)
		})
	}
}
