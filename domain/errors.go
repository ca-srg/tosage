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

	// ErrCodeTimezone indicates a timezone-related error
	ErrCodeTimezone ErrorCode = "TIMEZONE_ERROR"

	// ErrCodeCSVExport indicates a CSV export-related error
	ErrCodeCSVExport ErrorCode = "CSV_EXPORT_ERROR"

	// ErrCodeFileOperation indicates a file operation error
	ErrCodeFileOperation ErrorCode = "FILE_OPERATION_ERROR"

	// ErrCodeDataCollection indicates a data collection error
	ErrCodeDataCollection ErrorCode = "DATA_COLLECTION_ERROR"
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

// Timezone-specific errors

// ErrTimezone creates a timezone error
func ErrTimezone(operation string, reason string) *DomainError {
	return NewDomainError(ErrCodeTimezone, fmt.Sprintf("timezone error in %s: %s", operation, reason)).
		WithDetails("operation", operation).
		WithDetails("reason", reason)
}

// ErrTimezoneWithCause creates a timezone error with cause
func ErrTimezoneWithCause(operation string, reason string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeTimezone, fmt.Sprintf("timezone error in %s: %s", operation, reason), err).
		WithDetails("operation", operation).
		WithDetails("reason", reason)
}

// ErrTimezoneDetection creates a timezone detection error
func ErrTimezoneDetection(fallbackLocation string) *DomainError {
	return NewDomainError(ErrCodeTimezone, "failed to detect system timezone, using fallback").
		WithDetails("fallback", fallbackLocation)
}

// ErrTimezoneParse creates a timezone parsing error
func ErrTimezoneParse(timezoneName string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeTimezone, fmt.Sprintf("failed to parse timezone: %s", timezoneName), err).
		WithDetails("timezoneName", timezoneName)
}

// CSV Export errors

// ErrCSVExport creates a CSV export error
func ErrCSVExport(operation string, reason string) *DomainError {
	return NewDomainError(ErrCodeCSVExport, fmt.Sprintf("CSV export error in %s: %s", operation, reason)).
		WithDetails("operation", operation).
		WithDetails("reason", reason)
}

// ErrCSVExportWithCause creates a CSV export error with cause
func ErrCSVExportWithCause(operation string, reason string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeCSVExport, fmt.Sprintf("CSV export error in %s: %s", operation, reason), err).
		WithDetails("operation", operation).
		WithDetails("reason", reason)
}

// File operation errors

// ErrFileOperation creates a file operation error
func ErrFileOperation(operation string, path string, reason string) *DomainError {
	return NewDomainError(ErrCodeFileOperation, fmt.Sprintf("file operation error in %s: %s", operation, reason)).
		WithDetails("operation", operation).
		WithDetails("path", path).
		WithDetails("reason", reason)
}

// ErrFileOperationWithCause creates a file operation error with cause
func ErrFileOperationWithCause(operation string, path string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeFileOperation, fmt.Sprintf("file operation error in %s", operation), err).
		WithDetails("operation", operation).
		WithDetails("path", path)
}

// ErrFilePermission creates a file permission error
func ErrFilePermission(path string, requiredPermission string) *DomainError {
	return NewDomainError(ErrCodeFileOperation, fmt.Sprintf("insufficient permissions for file: %s", path)).
		WithDetails("path", path).
		WithDetails("requiredPermission", requiredPermission)
}

// ErrPathTraversal creates a path traversal error
func ErrPathTraversal(path string) *DomainError {
	return NewDomainError(ErrCodeFileOperation, "path contains directory traversal").
		WithDetails("path", path).
		WithDetails("securityViolation", "directory_traversal")
}

// ErrSystemDirectory creates a system directory access error
func ErrSystemDirectory(path string) *DomainError {
	return NewDomainError(ErrCodeFileOperation, "cannot write to system directory").
		WithDetails("path", path).
		WithDetails("securityViolation", "system_directory_access")
}

// Data collection errors

// ErrDataCollection creates a data collection error
func ErrDataCollection(source string, reason string) *DomainError {
	return NewDomainError(ErrCodeDataCollection, fmt.Sprintf("data collection error from %s: %s", source, reason)).
		WithDetails("source", source).
		WithDetails("reason", reason)
}

// ErrDataCollectionWithCause creates a data collection error with cause
func ErrDataCollectionWithCause(source string, reason string, err error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeDataCollection, fmt.Sprintf("data collection error from %s: %s", source, reason), err).
		WithDetails("source", source).
		WithDetails("reason", reason)
}

// ErrNoDataAvailable creates a no data available error
func ErrNoDataAvailable(source string, timeRange string) *DomainError {
	return NewDomainError(ErrCodeDataCollection, fmt.Sprintf("no data available from %s for %s", source, timeRange)).
		WithDetails("source", source).
		WithDetails("timeRange", timeRange)
}
