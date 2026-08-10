package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a structured application error
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// BadRequest creates a 400 error
func BadRequest(message string, err error) *AppError {
	return New(http.StatusBadRequest, message, err)
}

// Unauthorized creates a 401 error
func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message, nil)
}

// Forbidden creates a 403 error
func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, message, nil)
}

// NotFound creates a 404 error
func NotFound(message string) *AppError {
	return New(http.StatusNotFound, message, nil)
}

// Conflict creates a 409 error
func Conflict(message string, err error) *AppError {
	return New(http.StatusConflict, message, err)
}

// Internal creates a 500 error
func Internal(message string, err error) *AppError {
	return New(http.StatusInternalServerError, message, err)
}
