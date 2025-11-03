package query

import (
	"strings"
	"time"
)

// NodeType represents the type of AST node
type NodeType int

const (
	NodeTypeBinaryOp NodeType = iota
	NodeTypeComparison
	NodeTypeLiteral
	NodeTypeIdentifier
)

// Node is the interface that all AST nodes implement
type Node interface {
	Type() NodeType
}

// BinaryOperator represents a binary logical operator
type BinaryOperator int

const (
	// BinaryOpAnd represents the AND operator
	BinaryOpAnd BinaryOperator = iota
	// BinaryOpOr represents the OR operator
	BinaryOpOr
)

// String returns the string representation of BinaryOperator
func (bo BinaryOperator) String() string {
	switch bo {
	case BinaryOpAnd:
		return "and"
	case BinaryOpOr:
		return "or"
	default:
		return "and" // Default to and
	}
}

// ParseBinaryOperator parses a string into a BinaryOperator enum value
func ParseBinaryOperator(s string) BinaryOperator {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "and":
		return BinaryOpAnd
	case "or":
		return BinaryOpOr
	default:
		return BinaryOpAnd // Default to and
	}
}

// BinaryOpNode represents a binary operation (AND, OR)
type BinaryOpNode struct {
	Operator BinaryOperator
	Left     Node
	Right    Node
}

func (n *BinaryOpNode) Type() NodeType { return NodeTypeBinaryOp }

// ComparisonNode represents a comparison operation
type ComparisonNode struct {
	Field    string
	Operator ComparisonOperator
	Value    interface{}
}

func (n *ComparisonNode) Type() NodeType { return NodeTypeComparison }

// Value types for easier type assertion
type StringValue string
type IntValue int64
type FloatValue float64
type BoolValue bool
type DateTimeValue time.Time

// SortOrder represents the sort order direction
type SortOrder int

const (
	// SortOrderAsc sorts in ascending order
	SortOrderAsc SortOrder = iota
	// SortOrderDesc sorts in descending order
	SortOrderDesc
	// SortOrderRandom sorts in random order
	SortOrderRandom
)

// String returns the string representation of SortOrder
func (so SortOrder) String() string {
	switch so {
	case SortOrderAsc:
		return "asc"
	case SortOrderDesc:
		return "desc"
	case SortOrderRandom:
		return "random"
	default:
		return "asc" // Default to asc
	}
}

// ParseSortOrder parses a string into a SortOrder enum value
// Returns SortOrderAsc as default for empty or invalid values
func ParseSortOrder(s string) SortOrder {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "asc":
		return SortOrderAsc
	case "desc":
		return SortOrderDesc
	case "random":
		return SortOrderRandom
	default:
		return SortOrderAsc // Default to asc
	}
}

// Query represents a parsed query with all its components
type Query struct {
	Filter    Node
	SortBy    string
	SortOrder SortOrder
	PageSize  int
}
