package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenOperator
	TokenAnd
	TokenOr
	TokenLeftParen
	TokenRightParen
	TokenComma
	TokenLeftBracket
	TokenRightBracket
	TokenLike
	TokenNotLike
	TokenContains
	TokenIContains
	TokenStartsWith
	TokenEndsWith
	TokenRegex
	TokenIn
	TokenNotIn
	TokenNot
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer tokenizes the input query string
type Lexer struct {
	input string
	pos   int
	ch    rune
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar reads the next character
func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = rune(l.input[l.pos])
	}
	l.pos++
}

// peekChar looks at the next character without advancing
func (l *Lexer) peekChar() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

// skipWhitespace skips over whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()

	startPos := l.pos - 1

	switch l.ch {
	case 0:
		return Token{Type: TokenEOF, Pos: startPos}, nil
	case '(':
		tok := Token{Type: TokenLeftParen, Value: string(l.ch), Pos: startPos}
		l.readChar()
		return tok, nil
	case ')':
		tok := Token{Type: TokenRightParen, Value: string(l.ch), Pos: startPos}
		l.readChar()
		return tok, nil
	case '[':
		tok := Token{Type: TokenLeftBracket, Value: string(l.ch), Pos: startPos}
		l.readChar()
		return tok, nil
	case ']':
		tok := Token{Type: TokenRightBracket, Value: string(l.ch), Pos: startPos}
		l.readChar()
		return tok, nil
	case ',':
		tok := Token{Type: TokenComma, Value: string(l.ch), Pos: startPos}
		l.readChar()
		return tok, nil
	case '"', '\'':
		return l.readString()
	case '=', '!', '>', '<':
		return l.readOperator()
	default:
		if unicode.IsLetter(l.ch) || l.ch == '_' {
			return l.readIdentifier()
		}
		if unicode.IsDigit(l.ch) {
			return l.readNumber()
		}
		if l.ch == '-' {
			// Look ahead to determine if this is a negative number or part of an identifier
			next := l.peekChar()
			if unicode.IsDigit(next) {
				return l.readNumber()
			}
			// Otherwise treat it as part of identifier (for dates like 2020-01-03-0415)
			return l.readIdentifier()
		}
		return Token{}, fmt.Errorf("unexpected character '%c' at position %d", l.ch, startPos)
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() (Token, error) {
	startPos := l.pos - 1
	var sb strings.Builder

	for unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' || l.ch == ':' || l.ch == '-' {
		sb.WriteRune(l.ch)
		l.readChar()
	}

	value := sb.String()
	lowerValue := strings.ToLower(value)
	
	// Check for keywords
	switch lowerValue {
	case "and":
		return Token{Type: TokenAnd, Value: value, Pos: startPos}, nil
	case "or":
		return Token{Type: TokenOr, Value: value, Pos: startPos}, nil
	case "not":
		return Token{Type: TokenNot, Value: value, Pos: startPos}, nil
	case "like":
		return Token{Type: TokenLike, Value: value, Pos: startPos}, nil
	case "contains":
		return Token{Type: TokenContains, Value: value, Pos: startPos}, nil
	case "icontains":
		return Token{Type: TokenIContains, Value: value, Pos: startPos}, nil
	case "starts_with":
		return Token{Type: TokenStartsWith, Value: value, Pos: startPos}, nil
	case "ends_with":
		return Token{Type: TokenEndsWith, Value: value, Pos: startPos}, nil
	case "regex":
		return Token{Type: TokenRegex, Value: value, Pos: startPos}, nil
	case "in":
		return Token{Type: TokenIn, Value: value, Pos: startPos}, nil
	}
	
	return Token{Type: TokenIdentifier, Value: value, Pos: startPos}, nil
}

// readNumber reads a number token
func (l *Lexer) readNumber() (Token, error) {
	startPos := l.pos - 1
	var sb strings.Builder

	// Handle negative numbers
	if l.ch == '-' {
		sb.WriteRune(l.ch)
		l.readChar()
	}

	for unicode.IsDigit(l.ch) || l.ch == '.' {
		sb.WriteRune(l.ch)
		l.readChar()
	}

	// Check if this might be a date/datetime (e.g., 2020-01-03 or 2020-01-03-0415)
	// If we see a hyphen followed by digits, continue reading as identifier
	if l.ch == '-' && unicode.IsDigit(l.peekChar()) {
		// This looks like a date, switch to identifier mode
		for unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' || l.ch == ':' || l.ch == '-' {
			sb.WriteRune(l.ch)
			l.readChar()
		}
		return Token{Type: TokenIdentifier, Value: sb.String(), Pos: startPos}, nil
	}

	return Token{Type: TokenNumber, Value: sb.String(), Pos: startPos}, nil
}

// readString reads a quoted string
func (l *Lexer) readString() (Token, error) {
	startPos := l.pos - 1
	quote := l.ch
	l.readChar()

	var sb strings.Builder
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' && l.peekChar() == quote {
			l.readChar()
			sb.WriteRune(quote)
			l.readChar()
		} else {
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}

	if l.ch == 0 {
		return Token{}, fmt.Errorf("unterminated string at position %d", startPos)
	}

	l.readChar() // consume closing quote
	return Token{Type: TokenString, Value: sb.String(), Pos: startPos}, nil
}

// readOperator reads an operator token
func (l *Lexer) readOperator() (Token, error) {
	startPos := l.pos - 1
	var sb strings.Builder

	sb.WriteRune(l.ch)
	l.readChar()

	// Handle two-character operators
	if l.ch == '=' {
		sb.WriteRune(l.ch)
		l.readChar()
	}

	return Token{Type: TokenOperator, Value: sb.String(), Pos: startPos}, nil
}

// AllTokens returns all tokens from the input (useful for debugging)
func (l *Lexer) AllTokens() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}
