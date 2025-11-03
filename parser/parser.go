package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	query "github.com/hadi77ir/go-query/query"
)

// Parser parses query strings into Query objects
type Parser struct {
	lexer   *Lexer
	curTok  Token
	peekTok Token
}

// NewParser creates a new parser for the given input
func NewParser(input string) (*Parser, error) {
	p := &Parser{lexer: NewLexer(input)}

	// Read two tokens to initialize curTok and peekTok
	if err := p.nextToken(); err != nil {
		return nil, err
	}
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	return p, nil
}

// nextToken advances the parser to the next token
func (p *Parser) nextToken() error {
	p.curTok = p.peekTok
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peekTok = tok
	return nil
}

// Parse parses the input and returns a Query
func (p *Parser) Parse() (*query.Query, error) {
	q := &query.Query{
		PageSize:  10, // default
		SortOrder: query.SortOrderAsc,
	}

	// Parse the entire query, extracting query options and building the filter
	filter, err := p.parseExpressionWithOptions(q)
	if err != nil {
		return nil, err
	}
	q.Filter = filter

	return q, nil
}

// parseExpressionWithOptions parses an expression while extracting query options
func (p *Parser) parseExpressionWithOptions(q *query.Query) (query.Node, error) {
	return p.parseOrExpressionWithOptions(q)
}

// parseOrExpressionWithOptions parses OR expressions while extracting query options
func (p *Parser) parseOrExpressionWithOptions(q *query.Query) (query.Node, error) {
	left, err := p.parseAndExpressionWithOptions(q)
	if err != nil {
		return nil, err
	}
	// If left is nil, it means we hit EOF (no filters, only query options)
	if left == nil {
		return nil, nil
	}

	for {
		// Check for OR operator
		if p.curTok.Type == TokenOr {
			if err := p.nextToken(); err != nil {
				return nil, err
			}
			// Check if there's actually a right-hand side
			if p.curTok.Type == TokenEOF || p.curTok.Type == TokenRightParen {
				return nil, fmt.Errorf("incomplete OR expression at position %d", p.curTok.Pos)
			}
			right, err := p.parseAndExpressionWithOptions(q)
			if err != nil {
				return nil, err
			}
			left = &query.BinaryOpNode{
				Operator: query.BinaryOpOr,
				Left:     left,
				Right:    right,
			}
			continue
		}

		// Try to extract query option
		if extracted, err := p.tryExtractQueryOption(q); err != nil {
			return nil, err
		} else if extracted {
			// Continue parsing after extracting the option
			continue
		}

		// No more OR or options, return what we have
		break
	}

	return left, nil
}

// parseAndExpressionWithOptions parses AND expressions while extracting query options
func (p *Parser) parseAndExpressionWithOptions(q *query.Query) (query.Node, error) {
	// Skip any query options at the start
	for {
		if p.curTok.Type == TokenEOF {
			return nil, nil
		}
		if p.curTok.Type != TokenIdentifier {
			break
		}
		extracted, err := p.tryExtractQueryOption(q)
		if err != nil {
			return nil, err
		}
		if !extracted {
			break
		}
	}

	left, err := p.parseComparisonWithOptions(q)
	if err != nil {
		return nil, err
	}
	// If left is nil, it means we hit EOF (no more filters)
	if left == nil {
		return nil, nil
	}

	for {
		// Try to extract query option first
		if extracted, err := p.tryExtractQueryOption(q); err != nil {
			return nil, err
		} else if extracted {
			// Successfully extracted, continue parsing
			// After extracting, check if we're at EOF or end of expression
			if p.curTok.Type == TokenEOF || p.curTok.Type == TokenRightParen {
				break
			}
			// If we're at an AND/OR, continue to handle it
			continue
		}

		// Explicit AND
		if p.curTok.Type == TokenAnd {
			if err := p.nextToken(); err != nil {
				return nil, err
			}
			// Check if there's actually a right-hand side
			if p.curTok.Type == TokenEOF || p.curTok.Type == TokenRightParen {
				return nil, fmt.Errorf("incomplete AND expression at position %d", p.curTok.Pos)
			}
			right, err := p.parseComparisonWithOptions(q)
			if err != nil {
				return nil, err
			}
			left = &query.BinaryOpNode{
				Operator: query.BinaryOpAnd,
				Left:     left,
				Right:    right,
			}
			continue
		}

		// Implicit AND - if we encounter another term without OR/AND/EOF/), treat it as AND
		if p.curTok.Type == TokenIdentifier || p.curTok.Type == TokenString || p.curTok.Type == TokenLeftParen {
			// But not if we're at the end or before a closing paren or explicit OR
			if p.curTok.Type == TokenRightParen || p.curTok.Type == TokenEOF {
				break
			}

			// Parse the next comparison with implicit AND
			right, err := p.parseComparisonWithOptions(q)
			if err != nil {
				return nil, err
			}
			left = &query.BinaryOpNode{
				Operator: query.BinaryOpAnd,
				Left:     left,
				Right:    right,
			}
			continue
		}

		break
	}

	return left, nil
}

// parseComparisonWithOptions parses a comparison expression while extracting query options
func (p *Parser) parseComparisonWithOptions(q *query.Query) (query.Node, error) {
	// Handle EOF (empty query)
	if p.curTok.Type == TokenEOF {
		return nil, nil
	}

	// Try to extract query option if we're at an identifier
	if p.curTok.Type == TokenIdentifier {
		if extracted, err := p.tryExtractQueryOption(q); err != nil {
			return nil, err
		} else if extracted {
			// Successfully extracted, return nil to indicate we need to continue parsing
			// The caller will handle what comes next
			return nil, nil
		}
	}

	if p.curTok.Type == TokenLeftParen {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpressionWithOptions(q)
		if err != nil {
			return nil, err
		}
		if p.curTok.Type != TokenRightParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.curTok.Pos)
		}
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return expr, nil
	}

	// Handle bare strings (e.g., "hello" or unquoted) as search terms
	if p.curTok.Type == TokenString {
		searchTerm := p.curTok.Value
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		// Create a CONTAINS comparison on the default search field
		return &query.ComparisonNode{
			Field:    "__DEFAULT_SEARCH__", // Special marker for default field
			Operator: query.OpContains,
			Value:    query.StringValue(searchTerm),
		}, nil
	}

	if p.curTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier at position %d, got %v", p.curTok.Pos, p.curTok.Type)
	}

	field := p.curTok.Value

	// Check if this is a query option (identifier followed by =)
	// We check peekTok without advancing yet
	isQueryOption := p.peekTok.Type == TokenOperator && p.peekTok.Value == "="
	if isQueryOption {
		// Try to extract as query option
		if extracted, err := p.tryExtractQueryOptionFromField(q, field); err != nil {
			return nil, err
		} else if extracted {
			// Successfully extracted, continue parsing
			return p.parseComparisonWithOptions(q)
		}
		// Not a query option, fall through to parse as comparison
	}

	// Advance to next token for normal comparison parsing
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Check if this is a bare identifier (search term) or a field name
	// If no operator follows, treat it as a search term
	isOperator := p.curTok.Type == TokenOperator ||
		p.curTok.Type == TokenLike ||
		p.curTok.Type == TokenNotLike ||
		p.curTok.Type == TokenContains ||
		p.curTok.Type == TokenIContains ||
		p.curTok.Type == TokenStartsWith ||
		p.curTok.Type == TokenEndsWith ||
		p.curTok.Type == TokenRegex ||
		p.curTok.Type == TokenIn ||
		p.curTok.Type == TokenNot

	if !isOperator {
		// This is a bare search term (identifier without operator)
		return &query.ComparisonNode{
			Field:    "__DEFAULT_SEARCH__",
			Operator: query.OpContains,
			Value:    query.StringValue(field),
		}, nil
	}

	// Parse operator - could be standard operator or keyword operator
	var operator query.ComparisonOperator
	switch p.curTok.Type {
	case TokenOperator:
		operator = query.ParseComparisonOperator(p.curTok.Value)
	case TokenLike:
		operator = query.OpLike
	case TokenContains:
		operator = query.OpContains
	case TokenIContains:
		operator = query.OpIContains
	case TokenStartsWith:
		operator = query.OpStartsWith
	case TokenEndsWith:
		operator = query.OpEndsWith
	case TokenRegex:
		operator = query.OpRegex
	case TokenIn:
		operator = query.OpIn
	case TokenNot:
		// Check for NOT LIKE or NOT IN
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if p.curTok.Type == TokenLike {
			operator = query.OpNotLike
		} else if p.curTok.Type == TokenIn {
			operator = query.OpNotIn
		} else {
			return nil, fmt.Errorf("unexpected token after NOT at position %d", p.curTok.Pos)
		}
	default:
		return nil, fmt.Errorf("expected operator at position %d", p.curTok.Pos)
	}

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Parse value - could be single value or array for IN/NOT IN
	var value interface{}
	var err error

	if operator == query.OpIn || operator == query.OpNotIn {
		// Expect array
		value, err = p.parseArray()
	} else {
		value, err = p.parseValue()
	}

	if err != nil {
		return nil, err
	}

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	return &query.ComparisonNode{
		Field:    field,
		Operator: operator,
		Value:    value,
	}, nil
}

// tryExtractQueryOption tries to extract a query option from the current position
// Returns true if an option was extracted, false otherwise
func (p *Parser) tryExtractQueryOption(q *query.Query) (bool, error) {
	if p.curTok.Type != TokenIdentifier {
		return false, nil
	}

	key := p.curTok.Value
	return p.tryExtractQueryOptionFromField(q, key)
}

// tryExtractQueryOptionFromField tries to extract a query option given the field name
// Returns true if an option was extracted, false otherwise
func (p *Parser) tryExtractQueryOptionFromField(q *query.Query, key string) (bool, error) {
	// Check if next token is =
	if p.peekTok.Type != TokenOperator || p.peekTok.Value != "=" {
		return false, nil
	}

	lowerKey := strings.ToLower(key)

	switch lowerKey {
	case "sort_by":
		if err := p.nextToken(); err != nil {
			return false, err
		}
		if p.curTok.Type != TokenOperator || p.curTok.Value != "=" {
			return false, fmt.Errorf("expected '=' after sort_by")
		}
		if err := p.nextToken(); err != nil {
			return false, err
		}
		q.SortBy = p.getValue()
		if err := p.nextToken(); err != nil {
			return false, err
		}
		return true, nil

	case "sort_order":
		if err := p.nextToken(); err != nil {
			return false, err
		}
		if p.curTok.Type != TokenOperator || p.curTok.Value != "=" {
			return false, fmt.Errorf("expected '=' after sort_order")
		}
		if err := p.nextToken(); err != nil {
			return false, err
		}
		q.SortOrder = query.ParseSortOrder(p.getValue())
		if err := p.nextToken(); err != nil {
			return false, err
		}
		return true, nil

	case "page_size":
		if err := p.nextToken(); err != nil {
			return false, err
		}
		if p.curTok.Type != TokenOperator || p.curTok.Value != "=" {
			return false, fmt.Errorf("expected '=' after page_size")
		}
		if err := p.nextToken(); err != nil {
			return false, err
		}
		val := p.getValue()
		size, err := strconv.Atoi(val)
		if err != nil {
			return false, fmt.Errorf("invalid page_size: %s", val)
		}
		q.PageSize = size
		if err := p.nextToken(); err != nil {
			return false, err
		}
		return true, nil

		// Note: cursor is no longer part of Query - it should be passed separately to Execute
	}

	return false, nil
}

// parseExpression parses a logical expression with AND/OR operators
func (p *Parser) parseExpression() (query.Node, error) {
	return p.parseOrExpression()
}

// parseOrExpression parses OR expressions (lower precedence)
func (p *Parser) parseOrExpression() (query.Node, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.curTok.Type == TokenOr {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = &query.BinaryOpNode{
			Operator: query.BinaryOpOr,
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// parseAndExpression parses AND expressions (higher precedence)
func (p *Parser) parseAndExpression() (query.Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for {
		// Explicit AND
		if p.curTok.Type == TokenAnd {
			if err := p.nextToken(); err != nil {
				return nil, err
			}
			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			left = &query.BinaryOpNode{
				Operator: query.BinaryOpAnd,
				Left:     left,
				Right:    right,
			}
			continue
		}

		// Implicit AND - if we encounter another term without OR/AND/EOF/), treat it as AND
		if p.curTok.Type == TokenIdentifier || p.curTok.Type == TokenString || p.curTok.Type == TokenLeftParen {
			// But not if we're at the end or before a closing paren or explicit OR
			if p.curTok.Type == TokenRightParen || p.curTok.Type == TokenEOF {
				break
			}

			// Parse the next comparison with implicit AND
			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			left = &query.BinaryOpNode{
				Operator: query.BinaryOpAnd,
				Left:     left,
				Right:    right,
			}
			continue
		}

		break
	}

	return left, nil
}

// parseComparison parses a comparison expression (field operator value)
func (p *Parser) parseComparison() (query.Node, error) {
	if p.curTok.Type == TokenLeftParen {
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.curTok.Type != TokenRightParen {
			return nil, fmt.Errorf("expected ')' at position %d", p.curTok.Pos)
		}
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		return expr, nil
	}

	// Handle bare strings (e.g., "hello" or unquoted) as search terms
	if p.curTok.Type == TokenString {
		searchTerm := p.curTok.Value
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		// Create a CONTAINS comparison on the default search field
		return &query.ComparisonNode{
			Field:    "__DEFAULT_SEARCH__", // Special marker for default field
			Operator: query.OpContains,
			Value:    query.StringValue(searchTerm),
		}, nil
	}

	if p.curTok.Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier at position %d, got %v", p.curTok.Pos, p.curTok.Type)
	}

	field := p.curTok.Value
	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Check if this is a bare identifier (search term) or a field name
	// If no operator follows, treat it as a search term
	isOperator := p.curTok.Type == TokenOperator ||
		p.curTok.Type == TokenLike ||
		p.curTok.Type == TokenNotLike ||
		p.curTok.Type == TokenContains ||
		p.curTok.Type == TokenIContains ||
		p.curTok.Type == TokenStartsWith ||
		p.curTok.Type == TokenEndsWith ||
		p.curTok.Type == TokenRegex ||
		p.curTok.Type == TokenIn ||
		p.curTok.Type == TokenNot

	if !isOperator {
		// This is a bare search term (identifier without operator)
		return &query.ComparisonNode{
			Field:    "__DEFAULT_SEARCH__",
			Operator: query.OpContains,
			Value:    query.StringValue(field),
		}, nil
	}

	// Parse operator - could be standard operator or keyword operator
	var operator query.ComparisonOperator
	switch p.curTok.Type {
	case TokenOperator:
		operator = query.ParseComparisonOperator(p.curTok.Value)
	case TokenLike:
		operator = query.OpLike
	case TokenContains:
		operator = query.OpContains
	case TokenIContains:
		operator = query.OpIContains
	case TokenStartsWith:
		operator = query.OpStartsWith
	case TokenEndsWith:
		operator = query.OpEndsWith
	case TokenRegex:
		operator = query.OpRegex
	case TokenIn:
		operator = query.OpIn
	case TokenNot:
		// Check for NOT LIKE or NOT IN
		if err := p.nextToken(); err != nil {
			return nil, err
		}
		if p.curTok.Type == TokenLike {
			operator = query.OpNotLike
		} else if p.curTok.Type == TokenIn {
			operator = query.OpNotIn
		} else {
			return nil, fmt.Errorf("unexpected token after NOT at position %d", p.curTok.Pos)
		}
	default:
		return nil, fmt.Errorf("expected operator at position %d", p.curTok.Pos)
	}

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Parse value - could be single value or array for IN/NOT IN
	var value interface{}
	var err error

	if operator == query.OpIn || operator == query.OpNotIn {
		// Expect array
		value, err = p.parseArray()
	} else {
		value, err = p.parseValue()
	}

	if err != nil {
		return nil, err
	}

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	return &query.ComparisonNode{
		Field:    field,
		Operator: operator,
		Value:    value,
	}, nil
}

// parseValue parses a value (string, number, or identifier)
func (p *Parser) parseValue() (interface{}, error) {
	switch p.curTok.Type {
	case TokenString:
		return query.StringValue(p.curTok.Value), nil
	case TokenNumber:
		// Try to parse as int first
		if !strings.Contains(p.curTok.Value, ".") {
			i, err := strconv.ParseInt(p.curTok.Value, 10, 64)
			if err == nil {
				return query.IntValue(i), nil
			}
		}
		// Parse as float
		f, err := strconv.ParseFloat(p.curTok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", p.curTok.Value)
		}
		return query.FloatValue(f), nil
	case TokenIdentifier:
		// Try to parse as datetime (format: 2020-01-03-0415)
		val := p.curTok.Value
		if t, err := parseDateTime(val); err == nil {
			return query.DateTimeValue(t), nil
		}
		// Try to parse as boolean
		if strings.ToLower(val) == "true" {
			return query.BoolValue(true), nil
		}
		if strings.ToLower(val) == "false" {
			return query.BoolValue(false), nil
		}
		// Treat as string
		return query.StringValue(val), nil
	default:
		return nil, fmt.Errorf("unexpected token type for value at position %d", p.curTok.Pos)
	}
}

// getValue returns the string value of the current token
func (p *Parser) getValue() string {
	switch p.curTok.Type {
	case TokenString, TokenIdentifier, TokenNumber:
		return p.curTok.Value
	default:
		return ""
	}
}

// parseArray parses an array literal [value1, value2, ...]
func (p *Parser) parseArray() (interface{}, error) {
	if p.curTok.Type != TokenLeftBracket {
		return nil, fmt.Errorf("expected '[' at position %d", p.curTok.Pos)
	}

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	var values []interface{}

	// Handle empty array
	if p.curTok.Type == TokenRightBracket {
		return query.ArrayValue(values), nil
	}

	// Parse first value
	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	values = append(values, val)

	if err := p.nextToken(); err != nil {
		return nil, err
	}

	// Parse remaining values
	for p.curTok.Type == TokenComma {
		if err := p.nextToken(); err != nil {
			return nil, err
		}

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, val)

		if err := p.nextToken(); err != nil {
			return nil, err
		}
	}

	if p.curTok.Type != TokenRightBracket {
		return nil, fmt.Errorf("expected ']' at position %d", p.curTok.Pos)
	}

	return query.ArrayValue(values), nil
}

// parseDateTime attempts to parse various datetime formats
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02-1504",     // 2020-01-03-0415
		"2006-01-02T15:04:05", // ISO 8601
		"2006-01-02 15:04:05", // Standard datetime
		"2006-01-02",          // Date only
		time.RFC3339,          // RFC3339
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %s", s)
}
