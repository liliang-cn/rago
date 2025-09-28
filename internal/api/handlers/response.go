package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PagedResponse represents a paginated API response
type PagedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Total   int         `json:"total"`
	HasMore bool        `json:"has_more"`
}

// ListResponse represents a standard list response
type ListResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Total   int         `json:"total"`
}

// NewPagedResponse creates a new paged response
func NewPagedResponse(data interface{}, page, perPage, total int) PagedResponse {
	return PagedResponse{
		Success: true,
		Data:    data,
		Page:    page,
		PerPage: perPage,
		Total:   total,
		HasMore: page*perPage < total,
	}
}

// NewListResponse creates a new list response
func NewListResponse(data interface{}, total int) ListResponse {
	// Ensure data is never nil
	if data == nil {
		data = []interface{}{}
	}

	return ListResponse{
		Success: true,
		Data:    data,
		Total:   total,
	}
}

// SendListResponse sends a standard list response
func SendListResponse(c *gin.Context, data interface{}, total int) {
	c.JSON(http.StatusOK, NewListResponse(data, total))
}

// SendPagedResponse sends a paginated response
func SendPagedResponse(c *gin.Context, data interface{}, page, perPage, total int) {
	c.JSON(http.StatusOK, NewPagedResponse(data, page, perPage, total))
}

// GetPaginationParams extracts pagination parameters from request
func GetPaginationParams(c *gin.Context) (page int, perPage int) {
	page = 1
	perPage = 20 // Default per page

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if pp := c.Query("per_page"); pp != "" {
		if parsed, err := strconv.Atoi(pp); err == nil && parsed > 0 && parsed <= 100 {
			perPage = parsed
		}
	}

	return page, perPage
}