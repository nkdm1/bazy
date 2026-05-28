package types

import (
	"net/http"
)

type ErrorApi interface {
	error
	Code() int
}

type ErrorApiWithData interface {
	ErrorApi
	ErrorData() map[string]any
}

type basicApiError struct {
	Message string
	Status  int
}

func (e *basicApiError) Error() string {
	return e.Message
}

func (e *basicApiError) Code() int {
	return e.Status
}

var (
	ErrInternalServer = &basicApiError{
		Message: "internal server error",
		Status:  http.StatusInternalServerError,
	}
	ErrNotFound = &basicApiError{
		Message: "record not found",
		Status:  http.StatusNotFound,
	}
	ErrTimeout = &basicApiError{
		Message: "database timeout",
		Status:  http.StatusServiceUnavailable,
	}
	ErrInvalidEmailOrPassword = &basicApiError{
		Message: "invalid email or password",
		Status:  http.StatusUnauthorized,
	}
	ErrInvalidJsonBody = &basicApiError{
		Message: "invalid json body",
		Status:  http.StatusBadRequest,
	}
	ErrPayloadTooLarge = &basicApiError{
		Message: "request's body is too large",
		Status:  http.StatusRequestEntityTooLarge,
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
