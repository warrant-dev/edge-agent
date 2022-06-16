package edge

import (
	"fmt"
	"net/http"
)

const (
	ErrorCacheNotReady            = "cache_not_ready"
	ErrorInternalError            = "internal_error"
	ErrorInvalidRequest           = "invalid_request"
	ErrorInvalidParameter         = "invalid_parameter"
	ErrorMissingRequiredParameter = "missing_required_parameter"
	ErrorUnauthorized             = "unauthorized"
)

type GenericError struct {
	Tag     string `json:"-"`
	Code    string `json:"code"`
	Status  int    `json:"-"`
	Message string `json:"message"`
}

type Error interface {
	GetTag() string
	GetStatus() int
}

func (err *GenericError) GetTag() string {
	return err.Tag
}

func (err *GenericError) GetStatus() int {
	return err.Status
}

func (err *GenericError) Error() string {
	return fmt.Sprintf("%s: %s", err.GetTag(), err.Message)
}

func NewGenericError(tag string, code string, status int, msg string) *GenericError {
	return &GenericError{
		Tag:     tag,
		Code:    code,
		Status:  status,
		Message: msg,
	}
}

// CacheNotReady type
type CacheNotReady struct {
	*GenericError
}

func NewCacheNotReady() *CacheNotReady {
	return &CacheNotReady{
		GenericError: NewGenericError(
			"CacheNotReady",
			ErrorCacheNotReady,
			http.StatusServiceUnavailable,
			"Edge cache not ready",
		),
	}
}

// InternalError type
type InternalError struct {
	*GenericError
}

func NewInternalError(msg string) *InternalError {
	return &InternalError{
		GenericError: NewGenericError(
			"InternalError",
			ErrorInternalError,
			http.StatusInternalServerError,
			msg,
		),
	}
}

// InvalidRequestError type
type InvalidRequestError struct {
	*GenericError
}

func NewInvalidRequestError(msg string) *InvalidRequestError {
	return &InvalidRequestError{
		GenericError: NewGenericError(
			"InvalidRequestError",
			ErrorInvalidRequest,
			http.StatusBadRequest,
			msg,
		),
	}
}

// InvalidParameterError type
type InvalidParameterError struct {
	*GenericError
	Parameter string `json:"parameter"`
}

func NewInvalidParameterError(paramName string, msg string) *InvalidParameterError {
	return &InvalidParameterError{
		GenericError: NewGenericError(
			"InvalidParameterError",
			ErrorInvalidParameter,
			http.StatusBadRequest,
			msg,
		),
		Parameter: paramName,
	}
}

func (err *InvalidParameterError) Error() string {
	return fmt.Sprintf("%s: Invalid parameter %s, %s", err.GetTag(), err.Parameter, err.Message)
}

// MissingRequiredParameterError type
type MissingRequiredParameterError struct {
	*GenericError
	Parameter string `json:"parameter"`
}

func NewMissingRequiredParameterError(parameterName string) *MissingRequiredParameterError {
	return &MissingRequiredParameterError{
		GenericError: NewGenericError(
			"MissingRequiredParameterError",
			ErrorMissingRequiredParameter,
			http.StatusBadRequest,
			fmt.Sprintf("Missing required parameter %s", parameterName),
		),
		Parameter: parameterName,
	}
}
