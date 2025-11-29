package errors

import (
	"errors"
	"fmt"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Domain errors
	ErrorTypeValidation    ErrorType = "validation_error"
	ErrorTypeNotFound      ErrorType = "not_found"
	ErrorTypeAlreadyExists ErrorType = "already_exists"

	// Infrastructure errors
	ErrorTypeStorage   ErrorType = "storage_error"
	ErrorTypeMessaging ErrorType = "messaging_error"
	ErrorTypeExternal  ErrorType = "external_service_error"

	// Processing errors
	ErrorTypeProcessing   ErrorType = "processing_error"
	ErrorTypeTimeout      ErrorType = "timeout_error"
	ErrorTypeCancellation ErrorType = "cancellation_error"

	// System errors
	ErrorTypeInternal      ErrorType = "internal_error"
	ErrorTypeConfiguration ErrorType = "configuration_error"
)

// AppError represents a custom application error
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap implements the unwrap interface for errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new AppError
func New(errType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, preserve the original type unless explicitly overridden
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			Type:    errType,
			Message: message,
			Err:     appErr,
			Context: appErr.Context,
		}
	}

	return &AppError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// Is checks if the error is of a specific type
func Is(err error, errType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

// Common error constructors

// Validation errors
func NewValidationError(message string) *AppError {
	return New(ErrorTypeValidation, message)
}

func WrapValidationError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeValidation, message)
}

// Not found errors
func NewNotFoundError(resource string) *AppError {
	return New(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource))
}

func WrapNotFoundError(err error, resource string) *AppError {
	return Wrap(err, ErrorTypeNotFound, fmt.Sprintf("%s not found", resource))
}

// Already exists errors
func NewAlreadyExistsError(resource string) *AppError {
	return New(ErrorTypeAlreadyExists, fmt.Sprintf("%s already exists", resource))
}

// Storage errors
func NewStorageError(message string) *AppError {
	return New(ErrorTypeStorage, message)
}

func WrapStorageError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeStorage, message)
}

// Messaging errors
func NewMessagingError(message string) *AppError {
	return New(ErrorTypeMessaging, message)
}

func WrapMessagingError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeMessaging, message)
}

// Processing errors
func NewProcessingError(message string) *AppError {
	return New(ErrorTypeProcessing, message)
}

func WrapProcessingError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeProcessing, message)
}

// Timeout errors
func NewTimeoutError(message string) *AppError {
	return New(ErrorTypeTimeout, message)
}

func WrapTimeoutError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeTimeout, message)
}

// Internal errors
func NewInternalError(message string) *AppError {
	return New(ErrorTypeInternal, message)
}

func WrapInternalError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeInternal, message)
}

// Configuration errors
func NewConfigurationError(message string) *AppError {
	return New(ErrorTypeConfiguration, message)
}

func WrapConfigurationError(err error, message string) *AppError {
	return Wrap(err, ErrorTypeConfiguration, message)
}

func IsNonRetryable(err error) bool {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return false
	}

	switch appErr.Type {
	case ErrorTypeValidation,
		ErrorTypeNotFound,
		ErrorTypeAlreadyExists,
		ErrorTypeProcessing,
		ErrorTypeConfiguration,
		ErrorTypeInternal:
		return true

	case ErrorTypeStorage,
		ErrorTypeMessaging,
		ErrorTypeExternal,
		ErrorTypeTimeout:
		return false

	default:
		return false
	}
}
