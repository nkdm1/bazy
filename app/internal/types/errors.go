package types

import (
	"net/http"
)

var (
	ErrInternalServer = &apiError{
		Message: "internal server error",
		Status:  http.StatusInternalServerError,
	}
	ErrNotFound = &apiError{
		Message: "record not found",
		Status:  http.StatusNotFound,
	}
	ErrTimeout = &apiError{
		Message: "database timeout",
		Status:  http.StatusServiceUnavailable,
	}
	ErrInvalidEmailOrPassword = &apiError{
		Message: "invalid email or password",
		Status:  http.StatusUnauthorized,
	}
	ErrInvalidJsonBody = &apiError{
		Message: "invalid json body",
		Status:  http.StatusBadRequest,
	}
	ErrPayloadTooLarge = &apiError{
		Message: "request's body is too large",
		Status:  http.StatusRequestEntityTooLarge,
	}
	ErrNullPassword = &apiError{
		Message: "password to that account has not been created yet",
		Status:  http.StatusConflict,
	}
	ErrInvalidEmailFormat = &apiError{
		Message: "invalid email format",
		Status:  http.StatusBadRequest,
	}
	ErrInvalidPayload = &apiError{
		Message: "invalid payload values",
		Status:  http.StatusBadRequest,
	}
	ErrInvalidToken = &apiError{
		Message: "invalid token value",
		Status:  http.StatusBadRequest,
	}
	ErrUnauthorized = &apiError{
		Message: "unauthorized, log in to continue",
		Status:  http.StatusUnauthorized,
	}
)

type ErrMissingRequiredFields struct {
	Fields []string
}

func (e *ErrMissingRequiredFields) Error() string {
	return "missing required fields"
}
func (e *ErrMissingRequiredFields) Code() int {
	return http.StatusBadRequest
}
func (e *ErrMissingRequiredFields) ErrorData() map[string]any {
	return map[string]any{
		"fields": e.Fields,
	}
}

// ====================================

type ErrorApi interface {
	error
	Code() int
}

type ErrorApiWithData interface {
	ErrorApi
	ErrorData() map[string]any
}

type apiError struct {
	Message string
	Status  int
}

func (e *apiError) Error() string {
	return e.Message
}

func (e *apiError) Code() int {
	return e.Status
}
