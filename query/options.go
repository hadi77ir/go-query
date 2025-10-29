package query

// ExecutorOptions contains configuration options for query executors
type ExecutorOptions struct {
	// MaxPageSize is the maximum allowed page size
	MaxPageSize int

	// DefaultPageSize is the default page size when not specified
	DefaultPageSize int

	// DefaultSortField is the default field to sort by
	DefaultSortField string

	// DefaultSortOrder is the default sort order
	DefaultSortOrder SortOrder

	// AllowRandomOrder determines if random ordering is allowed
	AllowRandomOrder bool

	// RandomFunctionName is the SQL function name to use for random ordering
	// Defaults to "RANDOM()" (PostgreSQL, SQLite). Use "RAND()" for MySQL
	// This only applies to SQL-based executors (GORM)
	RandomFunctionName string

	// IDFieldName is the name of the ID field used for cursor-based pagination
	// Defaults to "_id" for MongoDB, "id" for GORM, empty for Memory executor
	// This field is used when sorting by a different field to handle ties
	IDFieldName string

	// DefaultSearchField is the field used for bare string searches
	// When a bare string is encountered (e.g., "hello" without field name),
	// it will search this field using CONTAINS
	DefaultSearchField string

	// AllowedFields is a whitelist of fields that can be queried
	// Empty list means all fields are allowed (no restriction)
	// This is a security feature to prevent querying sensitive fields
	AllowedFields []string

	// DisableRegex disables REGEX operator support
	// Set to true for databases that don't support regex (e.g., SQLite without extension)
	// When disabled, queries with REGEX will return a clear error
	DisableRegex bool
}

// DefaultExecutorOptions returns default executor options
func DefaultExecutorOptions() *ExecutorOptions {
	return &ExecutorOptions{
		MaxPageSize:        100,
		DefaultPageSize:    10,
		DefaultSortField:   "_id",
		DefaultSortOrder:   SortOrderAsc,
		AllowRandomOrder:   true,
		RandomFunctionName: "RANDOM()", // PostgreSQL, SQLite default. Use "RAND()" for MySQL
		DefaultSearchField: "name",     // Default to searching "name" field
	}
}

// ValidatePageSize validates and adjusts the page size based on options
func (o *ExecutorOptions) ValidatePageSize(size int) int {
	if size <= 0 {
		return o.DefaultPageSize
	}
	// Only cap if MaxPageSize is set (> 0)
	if o.MaxPageSize > 0 && size > o.MaxPageSize {
		return o.MaxPageSize
	}
	return size
}

// IsFieldAllowed checks if a field is in the allowed fields list
// Returns true if AllowedFields is empty (no restriction) or field is in the list
func (o *ExecutorOptions) IsFieldAllowed(field string) bool {
	// Empty list means all fields allowed
	if len(o.AllowedFields) == 0 {
		return true
	}

	// Check if field is in allowed list
	for _, allowed := range o.AllowedFields {
		if allowed == field {
			return true
		}
	}
	return false
}
