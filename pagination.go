package main

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page        int
	PerPage     int
	TotalItems  int
	TotalPages  int
	Offset      int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	StartItem   int
	EndItem     int
	PageNumbers []int
}

// GetPaginationParams extracts and validates pagination parameters from request
func GetPaginationParams(r *http.Request, totalItems int, defaultPerPage int) PaginationParams {
	// Get page number from query params
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Get items per page from query params
	perPage := defaultPerPage
	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalItems) / float64(perPage)))

	// Ensure page is within bounds
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Calculate start and end item numbers
	startItem := offset + 1
	endItem := offset + perPage
	if endItem > totalItems {
		endItem = totalItems
	}
	if totalItems == 0 {
		startItem = 0
	}

	// Generate page numbers for pagination UI
	pageNumbers := generatePageNumbers(page, totalPages)

	return PaginationParams{
		Page:        page,
		PerPage:     perPage,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		Offset:      offset,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		StartItem:   startItem,
		EndItem:     endItem,
		PageNumbers: pageNumbers,
	}
}

// generatePageNumbers generates page numbers for pagination UI
func generatePageNumbers(currentPage, totalPages int) []int {
	if totalPages <= 7 {
		// Show all pages if 7 or fewer
		numbers := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			numbers[i] = i + 1
		}
		return numbers
	}

	// Show current page, 2 before, 2 after, first, and last
	numbers := []int{}

	// Always show first page
	numbers = append(numbers, 1)

	// Add ellipsis placeholder (-1) if needed
	if currentPage > 4 {
		numbers = append(numbers, -1)
	}

	// Add pages around current
	start := currentPage - 2
	if start < 2 {
		start = 2
	}
	end := currentPage + 2
	if end > totalPages-1 {
		end = totalPages - 1
	}

	for i := start; i <= end; i++ {
		numbers = append(numbers, i)
	}

	// Add ellipsis placeholder if needed
	if currentPage < totalPages-3 {
		numbers = append(numbers, -1)
	}

	// Always show last page
	if totalPages > 1 {
		numbers = append(numbers, totalPages)
	}

	return numbers
}

// BuildPaginationQuery builds query string with pagination params
func BuildPaginationQuery(baseURL string, page int, params map[string]string) string {
	query := fmt.Sprintf("%s?page=%d", baseURL, page)
	for key, value := range params {
		query += fmt.Sprintf("&%s=%s", key, value)
	}
	return query
}
