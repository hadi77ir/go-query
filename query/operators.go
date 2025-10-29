package query

import "strings"

// ComparisonOperator represents a comparison operator
type ComparisonOperator int

const (
	// Comparison operators
	OpEqual ComparisonOperator = iota
	OpNotEqual
	OpGreaterThan
	OpGreaterThanOrEqual
	OpLessThan
	OpLessThanOrEqual

	// String matching operators
	OpLike
	OpNotLike
	OpContains
	OpIContains
	OpStartsWith
	OpEndsWith
	OpRegex

	// Array/Set operators
	OpIn
	OpNotIn
)

// String returns the string representation of ComparisonOperator
func (co ComparisonOperator) String() string {
	switch co {
	case OpEqual:
		return "="
	case OpNotEqual:
		return "!="
	case OpGreaterThan:
		return ">"
	case OpGreaterThanOrEqual:
		return ">="
	case OpLessThan:
		return "<"
	case OpLessThanOrEqual:
		return "<="
	case OpLike:
		return "LIKE"
	case OpNotLike:
		return "NOT LIKE"
	case OpContains:
		return "CONTAINS"
	case OpIContains:
		return "ICONTAINS"
	case OpStartsWith:
		return "STARTS_WITH"
	case OpEndsWith:
		return "ENDS_WITH"
	case OpRegex:
		return "REGEX"
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	default:
		return "=" // Default to equal
	}
}

// ParseComparisonOperator parses a string into a ComparisonOperator enum value
func ParseComparisonOperator(s string) ComparisonOperator {
	s = strings.TrimSpace(s)
	switch s {
	case "=":
		return OpEqual
	case "!=":
		return OpNotEqual
	case ">":
		return OpGreaterThan
	case ">=":
		return OpGreaterThanOrEqual
	case "<":
		return OpLessThan
	case "<=":
		return OpLessThanOrEqual
	case "LIKE":
		return OpLike
	case "NOT LIKE":
		return OpNotLike
	case "CONTAINS":
		return OpContains
	case "ICONTAINS":
		return OpIContains
	case "STARTS_WITH":
		return OpStartsWith
	case "ENDS_WITH":
		return OpEndsWith
	case "REGEX":
		return OpRegex
	case "IN":
		return OpIn
	case "NOT IN":
		return OpNotIn
	default:
		return OpEqual // Default to equal
	}
}

// Legacy constants for backward compatibility (deprecated, use enum values)
const (
	OpEqualStr              = "="
	OpNotEqualStr           = "!="
	OpGreaterThanStr        = ">"
	OpGreaterThanOrEqualStr = ">="
	OpLessThanStr           = "<"
	OpLessThanOrEqualStr    = "<="
	OpLikeStr               = "LIKE"
	OpNotLikeStr            = "NOT LIKE"
	OpContainsStr           = "CONTAINS"
	OpIContainsStr          = "ICONTAINS"
	OpStartsWithStr         = "STARTS_WITH"
	OpEndsWithStr           = "ENDS_WITH"
	OpRegexStr              = "REGEX"
	OpInStr                 = "IN"
	OpNotInStr              = "NOT IN"
)

// IsValidOperator checks if an operator string is valid
func IsValidOperator(op string) bool {
	parsed := ParseComparisonOperator(op)
	// Check if the parsed value matches the string representation
	// This handles both valid operators and invalid ones that default to OpEqual
	return parsed.String() == op || op == "="
}

// ArrayValue represents an array of values
type ArrayValue []interface{}
