package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "error without wrapped error",
			err: &AppError{
				Code:    400,
				Message: "bad request",
				Err:     nil,
			},
			expected: "bad request",
		},
		{
			name: "error with wrapped error",
			err: &AppError{
				Code:    500,
				Message: "internal error",
				Err:     errors.New("database connection failed"),
			},
			expected: "internal error: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	wrappedErr := errors.New("original error")
	appErr := &AppError{
		Code:    500,
		Message: "wrapped error",
		Err:     wrappedErr,
	}

	unwrapped := appErr.Unwrap()
	if unwrapped != wrappedErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, wrappedErr)
	}

	// Test unwrap with nil
	appErrNil := &AppError{
		Code:    400,
		Message: "no wrapped error",
		Err:     nil,
	}
	if appErrNil.Unwrap() != nil {
		t.Errorf("Unwrap() should return nil when Err is nil")
	}
}

func TestNew(t *testing.T) {
	wrappedErr := errors.New("wrapped")
	err := New(418, "I'm a teapot", wrappedErr)

	if err.Code != 418 {
		t.Errorf("Code = %d, want 418", err.Code)
	}
	if err.Message != "I'm a teapot" {
		t.Errorf("Message = %q, want %q", err.Message, "I'm a teapot")
	}
	if err.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", err.Err, wrappedErr)
	}
}

func TestBadRequest(t *testing.T) {
	wrappedErr := errors.New("invalid input")
	err := BadRequest("validation failed", wrappedErr)

	if err.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
	if err.Message != "validation failed" {
		t.Errorf("Message = %q, want %q", err.Message, "validation failed")
	}
	if err.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", err.Err, wrappedErr)
	}

	// Test without wrapped error
	err2 := BadRequest("bad input", nil)
	if err2.Err != nil {
		t.Errorf("Err should be nil, got %v", err2.Err)
	}
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("invalid credentials")

	if err.Code != http.StatusUnauthorized {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusUnauthorized)
	}
	if err.Message != "invalid credentials" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid credentials")
	}
	if err.Err != nil {
		t.Errorf("Err should be nil, got %v", err.Err)
	}
}

func TestForbidden(t *testing.T) {
	err := Forbidden("access denied")

	if err.Code != http.StatusForbidden {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusForbidden)
	}
	if err.Message != "access denied" {
		t.Errorf("Message = %q, want %q", err.Message, "access denied")
	}
	if err.Err != nil {
		t.Errorf("Err should be nil, got %v", err.Err)
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	if err.Code != http.StatusNotFound {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusNotFound)
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %q, want %q", err.Message, "resource not found")
	}
	if err.Err != nil {
		t.Errorf("Err should be nil, got %v", err.Err)
	}
}

func TestConflict(t *testing.T) {
	wrappedErr := errors.New("duplicate key")
	err := Conflict("resource already exists", wrappedErr)

	if err.Code != http.StatusConflict {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusConflict)
	}
	if err.Message != "resource already exists" {
		t.Errorf("Message = %q, want %q", err.Message, "resource already exists")
	}
	if err.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", err.Err, wrappedErr)
	}

	// Test without wrapped error
	err2 := Conflict("conflict", nil)
	if err2.Err != nil {
		t.Errorf("Err should be nil, got %v", err2.Err)
	}
}

func TestInternal(t *testing.T) {
	wrappedErr := errors.New("database error")
	err := Internal("server error", wrappedErr)

	if err.Code != http.StatusInternalServerError {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusInternalServerError)
	}
	if err.Message != "server error" {
		t.Errorf("Message = %q, want %q", err.Message, "server error")
	}
	if err.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", err.Err, wrappedErr)
	}

	// Test without wrapped error
	err2 := Internal("error", nil)
	if err2.Err != nil {
		t.Errorf("Err should be nil, got %v", err2.Err)
	}
}

func TestAppError_AsError(t *testing.T) {
	// Test that AppError can be used as a standard error
	var err error = BadRequest("test", nil)
	if err == nil {
		t.Error("AppError should be usable as error interface")
	}
}

func TestErrorsIs(t *testing.T) {
	// Test errors.Is compatibility
	wrappedErr := errors.New("original")
	appErr := BadRequest("wrapped", wrappedErr)

	// Should be able to unwrap to the original error
	if !errors.Is(appErr, wrappedErr) {
		t.Error("errors.Is should find wrapped error")
	}
}
