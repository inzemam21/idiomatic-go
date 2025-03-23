package custom_errors

import (
	"errors"
	"net/http"
)

var (
	ErrBadRequest          = NewAPIError(http.StatusBadRequest, "bad_request", "Invalid request")
	ErrUnauthorized        = NewAPIError(http.StatusUnauthorized, "unauthorized", "Authentication failed")
	ErrForbidden           = NewAPIError(http.StatusForbidden, "forbidden", "Permission denied")
	ErrNotFound            = NewAPIError(http.StatusNotFound, "not_found", "Resource not found")
	ErrInternalServerError = NewAPIError(http.StatusInternalServerError, "internal_server_error", "Something went wrong")
)

type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
}

func NewAPIError(statusCode int, code, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

func (e *APIError) Error() string {
	return e.Message
}

// IsAPIError checks if an error is an APIError
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
