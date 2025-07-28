package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainError(t *testing.T) {
	t.Run("NewDomainError", func(t *testing.T) {
		err := NewDomainError(ErrCodeNotFound, "user not found")

		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeNotFound, err.Code)
		assert.Equal(t, "user not found", err.Message)
		assert.Equal(t, "[NOT_FOUND] user not found", err.Error())
		assert.NotNil(t, err.Details)
		assert.Nil(t, err.Err)
	})

	t.Run("NewDomainErrorWithCause", func(t *testing.T) {
		cause := errors.New("database connection failed")
		err := NewDomainErrorWithCause(ErrCodeRepository, "failed to fetch user", cause)

		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeRepository, err.Code)
		assert.Equal(t, "failed to fetch user", err.Message)
		assert.Equal(t, "[REPOSITORY_ERROR] failed to fetch user: database connection failed", err.Error())
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("WithDetails", func(t *testing.T) {
		err := NewDomainError(ErrCodeInvalidInput, "invalid email").
			WithDetails("field", "email").
			WithDetails("value", "not-an-email")

		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "not-an-email", err.Details["value"])
	})
}

func TestCommonErrors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		err := ErrNotFound("user", "123")

		assert.Equal(t, ErrCodeNotFound, err.Code)
		assert.Contains(t, err.Message, "user not found")
		assert.Equal(t, "user", err.Details["resource"])
		assert.Equal(t, "123", err.Details["id"])
	})

	t.Run("ErrInvalidInput", func(t *testing.T) {
		err := ErrInvalidInput("email", "must be a valid email address")

		assert.Equal(t, ErrCodeInvalidInput, err.Code)
		assert.Contains(t, err.Message, "invalid email")
		assert.Contains(t, err.Message, "must be a valid email address")
		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "must be a valid email address", err.Details["reason"])
	})

	t.Run("ErrBusinessRule", func(t *testing.T) {
		err := ErrBusinessRule("age_restriction", "must be 18 or older")

		assert.Equal(t, ErrCodeBusinessRule, err.Code)
		assert.Contains(t, err.Message, "business rule violation")
		assert.Contains(t, err.Message, "must be 18 or older")
		assert.Equal(t, "age_restriction", err.Details["rule"])
		assert.Equal(t, "must be 18 or older", err.Details["violation"])
	})

	t.Run("ErrCalculation", func(t *testing.T) {
		err := ErrCalculation("token_count", "overflow detected")

		assert.Equal(t, ErrCodeCalculation, err.Code)
		assert.Contains(t, err.Message, "calculation error in token_count")
		assert.Contains(t, err.Message, "overflow detected")
		assert.Equal(t, "token_count", err.Details["operation"])
		assert.Equal(t, "overflow detected", err.Details["reason"])
	})

	t.Run("ErrRepository", func(t *testing.T) {
		cause := errors.New("connection timeout")
		err := ErrRepository("save_user", cause)

		assert.Equal(t, ErrCodeRepository, err.Code)
		assert.Contains(t, err.Message, "repository error in save_user")
		assert.Equal(t, "save_user", err.Details["operation"])
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("ErrInvalidState", func(t *testing.T) {
		err := ErrInvalidState("order", "cancelled", "ship")

		assert.Equal(t, ErrCodeInvalidState, err.Code)
		assert.Contains(t, err.Message, "invalid state transition for order")
		assert.Contains(t, err.Message, "cannot ship in state cancelled")
		assert.Equal(t, "order", err.Details["entity"])
		assert.Equal(t, "cancelled", err.Details["currentState"])
		assert.Equal(t, "ship", err.Details["attemptedAction"])
	})
}

func TestCursorErrors(t *testing.T) {
	t.Run("ErrCursorToken", func(t *testing.T) {
		err := ErrCursorToken("token expired")

		assert.Equal(t, ErrCodeCursorToken, err.Code)
		assert.Contains(t, err.Message, "cursor token error")
		assert.Contains(t, err.Message, "token expired")
		assert.Equal(t, "token expired", err.Details["reason"])
	})

	t.Run("ErrCursorTokenWithCause", func(t *testing.T) {
		cause := errors.New("JWT validation failed")
		err := ErrCursorTokenWithCause("invalid signature", cause)

		assert.Equal(t, ErrCodeCursorToken, err.Code)
		assert.Contains(t, err.Message, "cursor token error")
		assert.Contains(t, err.Message, "invalid signature")
		assert.Equal(t, "invalid signature", err.Details["reason"])
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("ErrCursorAPI", func(t *testing.T) {
		err := ErrCursorAPI("get_usage", 401, "Unauthorized")

		assert.Equal(t, ErrCodeCursorAPI, err.Code)
		assert.Contains(t, err.Message, "cursor API error in get_usage")
		assert.Equal(t, "get_usage", err.Details["operation"])
		assert.Equal(t, 401, err.Details["statusCode"])
		assert.Equal(t, "Unauthorized", err.Details["response"])
	})

	t.Run("ErrCursorDatabase", func(t *testing.T) {
		err := ErrCursorDatabase("open", "/path/to/db.sqlite")

		assert.Equal(t, ErrCodeCursorDatabase, err.Code)
		assert.Contains(t, err.Message, "cursor database error in open")
		assert.Equal(t, "open", err.Details["operation"])
		assert.Equal(t, "/path/to/db.sqlite", err.Details["path"])
	})
}

func TestTimezoneErrors(t *testing.T) {
	t.Run("ErrTimezone", func(t *testing.T) {
		err := ErrTimezone("parse", "invalid timezone format")

		assert.Equal(t, ErrCodeTimezone, err.Code)
		assert.Contains(t, err.Message, "timezone error in parse")
		assert.Contains(t, err.Message, "invalid timezone format")
		assert.Equal(t, "parse", err.Details["operation"])
		assert.Equal(t, "invalid timezone format", err.Details["reason"])
	})

	t.Run("ErrTimezoneDetection", func(t *testing.T) {
		err := ErrTimezoneDetection("UTC")

		assert.Equal(t, ErrCodeTimezone, err.Code)
		assert.Contains(t, err.Message, "failed to detect system timezone")
		assert.Contains(t, err.Message, "using fallback")
		assert.Equal(t, "UTC", err.Details["fallback"])
	})

	t.Run("ErrTimezoneParse", func(t *testing.T) {
		cause := errors.New("unknown timezone")
		err := ErrTimezoneParse("Invalid/Zone", cause)

		assert.Equal(t, ErrCodeTimezone, err.Code)
		assert.Contains(t, err.Message, "failed to parse timezone")
		assert.Contains(t, err.Message, "Invalid/Zone")
		assert.Equal(t, "Invalid/Zone", err.Details["timezoneName"])
		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestCSVExportErrors(t *testing.T) {
	t.Run("ErrCSVExport", func(t *testing.T) {
		err := ErrCSVExport("write", "disk full")

		assert.Equal(t, ErrCodeCSVExport, err.Code)
		assert.Contains(t, err.Message, "CSV export error in write")
		assert.Contains(t, err.Message, "disk full")
		assert.Equal(t, "write", err.Details["operation"])
		assert.Equal(t, "disk full", err.Details["reason"])
	})

	t.Run("ErrCSVExportWithCause", func(t *testing.T) {
		cause := errors.New("IO error")
		err := ErrCSVExportWithCause("encode", "failed to encode data", cause)

		assert.Equal(t, ErrCodeCSVExport, err.Code)
		assert.Contains(t, err.Message, "CSV export error in encode")
		assert.Contains(t, err.Message, "failed to encode data")
		assert.Equal(t, "encode", err.Details["operation"])
		assert.Equal(t, "failed to encode data", err.Details["reason"])
		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestFileOperationErrors(t *testing.T) {
	t.Run("ErrFileOperation", func(t *testing.T) {
		err := ErrFileOperation("create", "/path/to/file.csv", "permission denied")

		assert.Equal(t, ErrCodeFileOperation, err.Code)
		assert.Contains(t, err.Message, "file operation error in create")
		assert.Contains(t, err.Message, "permission denied")
		assert.Equal(t, "create", err.Details["operation"])
		assert.Equal(t, "/path/to/file.csv", err.Details["path"])
		assert.Equal(t, "permission denied", err.Details["reason"])
	})

	t.Run("ErrFileOperationWithCause", func(t *testing.T) {
		cause := errors.New("EACCES")
		err := ErrFileOperationWithCause("write", "/path/to/file.csv", cause)

		assert.Equal(t, ErrCodeFileOperation, err.Code)
		assert.Contains(t, err.Message, "file operation error in write")
		assert.Equal(t, "write", err.Details["operation"])
		assert.Equal(t, "/path/to/file.csv", err.Details["path"])
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("ErrFilePermission", func(t *testing.T) {
		err := ErrFilePermission("/etc/passwd", "write")

		assert.Equal(t, ErrCodeFileOperation, err.Code)
		assert.Contains(t, err.Message, "insufficient permissions for file")
		assert.Contains(t, err.Message, "/etc/passwd")
		assert.Equal(t, "/etc/passwd", err.Details["path"])
		assert.Equal(t, "write", err.Details["requiredPermission"])
	})

	t.Run("ErrPathTraversal", func(t *testing.T) {
		err := ErrPathTraversal("../../../etc/passwd")

		assert.Equal(t, ErrCodeFileOperation, err.Code)
		assert.Contains(t, err.Message, "path contains directory traversal")
		assert.Equal(t, "../../../etc/passwd", err.Details["path"])
		assert.Equal(t, "directory_traversal", err.Details["securityViolation"])
	})

	t.Run("ErrSystemDirectory", func(t *testing.T) {
		err := ErrSystemDirectory("/etc/test.csv")

		assert.Equal(t, ErrCodeFileOperation, err.Code)
		assert.Contains(t, err.Message, "cannot write to system directory")
		assert.Equal(t, "/etc/test.csv", err.Details["path"])
		assert.Equal(t, "system_directory_access", err.Details["securityViolation"])
	})
}

func TestDataCollectionErrors(t *testing.T) {
	t.Run("ErrDataCollection", func(t *testing.T) {
		err := ErrDataCollection("claude_code", "API rate limit exceeded")

		assert.Equal(t, ErrCodeDataCollection, err.Code)
		assert.Contains(t, err.Message, "data collection error from claude_code")
		assert.Contains(t, err.Message, "API rate limit exceeded")
		assert.Equal(t, "claude_code", err.Details["source"])
		assert.Equal(t, "API rate limit exceeded", err.Details["reason"])
	})

	t.Run("ErrDataCollectionWithCause", func(t *testing.T) {
		cause := errors.New("network timeout")
		err := ErrDataCollectionWithCause("cursor", "connection failed", cause)

		assert.Equal(t, ErrCodeDataCollection, err.Code)
		assert.Contains(t, err.Message, "data collection error from cursor")
		assert.Contains(t, err.Message, "connection failed")
		assert.Equal(t, "cursor", err.Details["source"])
		assert.Equal(t, "connection failed", err.Details["reason"])
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("ErrNoDataAvailable", func(t *testing.T) {
		err := ErrNoDataAvailable("bedrock", "2025-01-01 to 2025-01-31")

		assert.Equal(t, ErrCodeDataCollection, err.Code)
		assert.Contains(t, err.Message, "no data available from bedrock")
		assert.Contains(t, err.Message, "2025-01-01 to 2025-01-31")
		assert.Equal(t, "bedrock", err.Details["source"])
		assert.Equal(t, "2025-01-01 to 2025-01-31", err.Details["timeRange"])
	})
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsErrorCode", func(t *testing.T) {
		err := ErrNotFound("user", "123")

		assert.True(t, IsErrorCode(err, ErrCodeNotFound))
		assert.False(t, IsErrorCode(err, ErrCodeInvalidInput))

		standardErr := errors.New("some error")
		assert.False(t, IsErrorCode(standardErr, ErrCodeNotFound))
	})

	t.Run("GetErrorCode", func(t *testing.T) {
		err := ErrInvalidInput("email", "invalid format")

		assert.Equal(t, ErrCodeInvalidInput, GetErrorCode(err))

		standardErr := errors.New("some error")
		assert.Equal(t, ErrorCode(""), GetErrorCode(standardErr))
	})
}
