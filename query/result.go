package query

// Result represents the standardized result of a query execution
// Note: Actual data is stored in the destination variable passed to Execute
type Result struct {
	// NextPageCursor is the cursor for the next page (empty if no next page)
	NextPageCursor string `json:"next_page_cursor"`

	// PrevPageCursor is the cursor for the previous page (empty if no previous page)
	PrevPageCursor string `json:"prev_page_cursor"`

	// TotalItems is the total number of items matching the query
	TotalItems int64 `json:"total_items"`

	// ShowingFrom is the starting index (1-based) of items in current page
	ShowingFrom int `json:"showing_from"`

	// ShowingTo is the ending index (1-based) of items in current page
	ShowingTo int `json:"showing_to"`

	// ItemsReturned is the number of items returned in this page
	ItemsReturned int `json:"items_returned"`

	// Error contains any error that occurred during execution
	Error error `json:"error,omitempty"`
}

// HasNextPage returns true if there is a next page available
func (r *Result) HasNextPage() bool {
	return r.NextPageCursor != ""
}

// HasPrevPage returns true if there is a previous page available
func (r *Result) HasPrevPage() bool {
	return r.PrevPageCursor != ""
}

// IsEmpty returns true if the result contains no data
func (r *Result) IsEmpty() bool {
	return r.ItemsReturned == 0
}
