package query

import (
	"errors"
	"fmt"
)

// Sentinel errors - use with errors.Is() for matching
var (
	// ErrNoRecordsFound is returned when a query executes successfully but returns no results
	ErrNoRecordsFound = errors.New("no records found")

	// ErrInvalidFieldName is returned when a field name contains invalid characters (SQL injection protection)
	ErrInvalidFieldName = errors.New("invalid field name")

	// ErrFieldNotAllowed is returned when a field is not in the AllowedFields whitelist
	ErrFieldNotAllowed = errors.New("field not allowed")

	// ErrInvalidQuery is returned when the query structure is invalid
	ErrInvalidQuery = errors.New("invalid query")

	// ErrInvalidCursor is returned when a cursor string cannot be decoded
	ErrInvalidCursor = errors.New("invalid cursor")

	// ErrPageSizeExceeded is returned when requested page size exceeds maximum
	ErrPageSizeExceeded = errors.New("page size exceeds maximum")

	// ErrRegexNotSupported is returned when REGEX operator is used but disabled
	ErrRegexNotSupported = errors.New("regex operator not supported")

	// ErrRandomOrderNotAllowed is returned when random order is requested but disabled
	ErrRandomOrderNotAllowed = errors.New("random order not allowed")

	// ErrExecutionFailed is returned when query execution fails at database level
	ErrExecutionFailed = errors.New("query execution failed")

	// ErrInvalidDestination is returned when the destination parameter is invalid
	ErrInvalidDestination = errors.New("invalid destination")
)

// FieldError wraps an error with field name information
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("field '%s': %v", e.Field, e.Err)
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

// NewFieldError creates a new FieldError
func NewFieldError(field string, err error) error {
	return &FieldError{
		Field: field,
		Err:   err,
	}
}

// InvalidFieldNameError creates an error for invalid field names
func InvalidFieldNameError(field string) error {
	return NewFieldError(field, ErrInvalidFieldName)
}

// FieldNotAllowedError creates an error for fields not in AllowedFields
func FieldNotAllowedError(field string) error {
	return NewFieldError(field, ErrFieldNotAllowed)
}

// ExecutionError wraps a database execution error
type ExecutionError struct {
	Operation string
	Err       error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// NewExecutionError creates a new ExecutionError
func NewExecutionError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &ExecutionError{
		Operation: operation,
		Err:       err,
	}
}
