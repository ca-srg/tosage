package domain

import (
	"fmt"
)

// ErrorCode represents the type of domain error
type ErrorCode string

const (
	// ErrCodeNotFound indicates that a requested resource was not found
	ErrCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrCodeInvalidInput indicates that the input provided is invalid
	ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrCodeBusinessRule indicates a business rule violation
	ErrCodeBusinessRule ErrorCode = "BUSINESS_RULE_VIOLATION"

	// ErrCodeCalculation indicates an error during calculation
	ErrCodeCalculation ErrorCode = "CALCULATION_ERROR"

	// ErrCodeRepository indicates a repository operation error
	ErrCodeRepository ErrorCode = "REPOSITORY_ERROR"

	// ErrCodeInvalidState indicates an invalid state transition
	ErrCodeInvalidState ErrorCode = "INVALID_STATE"

	// ErrCodeCursorToken indicates a Cursor token-related error
	ErrCodeCursorToken ErrorCode = "CURSOR_TOKEN_ERROR"

	// ErrCodeCursorAPI indicates a Cursor API communication error
	ErrCodeCursorAPI ErrorCode = "CURSOR_API_ERROR"

	// ErrCodeCursorDatabase indicates a Cursor database access error
	ErrCodeCursorDatabase ErrorCode = "CURSOR_DATABASE_ERROR"
)

// DomainError represents a domain-specific error
type DomainError struct {
	Code    ErrorCode
	Message string
	Details map[string]interface{}
	Err     error
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *DomainError) WithDetails(key string, value interface{}) *DomainError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewDomainError creates a new domain error
func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// NewDomainErrorWithCause creates a new domain error with an underlying cause
func NewDomainErrorWithCause(code ErrorCode, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
		Err:     err,
	}
}

// Common domain errors

// ErrNotFound creates a not found error
func ErrNotFound(resource string, id string) *DomainError {
	return NewDomainError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource)).
		WithDetails("resource", resource).
		WithDetails("id", id)
}

// ErrInvalidInput creates an invalid input error
func ErrInvalidInput(field string, reason string) *DomainError {
	return NewDomainError(ErrCodeInvalidInput, fmt.Sprintf("invalid %s: %s", field, reason)).
		WithDetails("field", field).
		WithDetails("reason", reason)
}

// ErrBusinessRule creates a business rule violation error
func ErrBusinessRule(rule string, violation string) *DomainError {
	return NewDomainError(ErrCodeBusinessRule, fmt.Sprintf("business rule violation: %s", violation)).
		WithDetails("rule", rule).
		WithDetails("violation", violation)
}

// ErrCalculation creates a calculation error
func ErrCalculation(operation string, reason string) *DomainError {
	return NewDomainError(ErrCodeCalculation, fmt.Sprintf("calculation error in %s: %s", operation, reason)).
		WithDetails("operation", operation).
		WithDetails("reason", reason)
}

// ErrRepository creates a repository error
func ErrRepository(operation string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeRepository, fmt.Sprintf("repository error in %s", operation), err).
		WithDetails("operation", operation)
}

// ErrInvalidState creates an invalid state error
func ErrInvalidState(entity string, currentState string, attemptedAction string) *DomainError {
	return NewDomainError(ErrCodeInvalidState,
		fmt.Sprintf("invalid state transition for %s: cannot %s in state %s", entity, attemptedAction, currentState)).
		WithDetails("entity", entity).
		WithDetails("currentState", currentState).
		WithDetails("attemptedAction", attemptedAction)
}

// IsErrorCode checks if an error has a specific error code
func IsErrorCode(err error, code ErrorCode) bool {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr.Code == code
	}
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr.Code
	}
	return ""
}

// Cursor-specific errors

// ErrCursorToken creates a Cursor token error
func ErrCursorToken(reason string) *DomainError {
	return NewDomainError(ErrCodeCursorToken, fmt.Sprintf("cursor token error: %s", reason)).
		WithDetails("reason", reason)
}

// ErrCursorTokenWithCause creates a Cursor token error with cause
func ErrCursorTokenWithCause(reason string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeCursorToken, fmt.Sprintf("cursor token error: %s", reason), err).
		WithDetails("reason", reason)
}

// ErrCursorAPI creates a Cursor API error
func ErrCursorAPI(operation string, statusCode int, response string) *DomainError {
	return NewDomainError(ErrCodeCursorAPI, fmt.Sprintf("cursor API error in %s", operation)).
		WithDetails("operation", operation).
		WithDetails("statusCode", statusCode).
		WithDetails("response", response)
}

// ErrCursorAPIWithCause creates a Cursor API error with cause
func ErrCursorAPIWithCause(operation string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeCursorAPI, fmt.Sprintf("cursor API error in %s", operation), err).
		WithDetails("operation", operation)
}

// ErrCursorDatabase creates a Cursor database error
func ErrCursorDatabase(operation string, path string) *DomainError {
	return NewDomainError(ErrCodeCursorDatabase, fmt.Sprintf("cursor database error in %s", operation)).
		WithDetails("operation", operation).
		WithDetails("path", path)
}

// ErrCursorDatabaseWithCause creates a Cursor database error with cause
func ErrCursorDatabaseWithCause(operation string, path string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeCursorDatabase, fmt.Sprintf("cursor database error in %s", operation), err).
		WithDetails("operation", operation).
		WithDetails("path", path)
}
