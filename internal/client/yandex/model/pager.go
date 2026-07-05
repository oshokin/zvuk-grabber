package model

// Pager represents paginated list metadata from API responses.
type Pager struct {
	// Total is the total number of items across all pages.
	Total int `json:"total"`
	// Page is the current one-based page number.
	Page int `json:"page"`
	// PerPage is the number of items returned per page.
	PerPage int `json:"perPage"`
}
