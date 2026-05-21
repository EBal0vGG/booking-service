package domain

// Pagination describes paginated list response.
type Pagination struct {
	Page     int
	PageSize int
	Total    int
}
