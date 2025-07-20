package repository

import (
	"github.com/ca-srg/tosage/domain/valueobject"
)

// CursorTokenRepository defines the interface for retrieving Cursor authentication tokens
type CursorTokenRepository interface {
	// GetToken retrieves the current Cursor authentication token from the database
	GetToken() (*valueobject.CursorToken, error)
}
