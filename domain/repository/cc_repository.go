package repository

import (
	"time"

	"github.com/ca-srg/tosage/domain/entity"
)

// CcRepository defines the interface for accessing cc data
type CcRepository interface {
	// FindAll returns all cc entries
	FindAll() ([]*entity.CcEntry, error)

	// FindByID returns a cc entry by its ID
	FindByID(id string) (*entity.CcEntry, error)

	// FindByDateRange returns cc entries within a date range
	FindByDateRange(start, end time.Time) ([]*entity.CcEntry, error)

	// FindByDate returns cc entries for a specific date
	FindByDate(date time.Time) ([]*entity.CcEntry, error)

	// FindByProject returns cc entries for a specific project
	FindByProject(projectPath string) ([]*entity.CcEntry, error)

	// FindBySession returns cc entries for a specific session
	FindBySession(sessionID string) ([]*entity.CcEntry, error)

	// FindByModel returns cc entries for a specific model
	FindByModel(model string) ([]*entity.CcEntry, error)

	// FindByProjectAndDateRange returns cc entries for a project within a date range
	FindByProjectAndDateRange(projectPath string, start, end time.Time) ([]*entity.CcEntry, error)

	// ExistsByID checks if a cc entry exists with the given ID
	ExistsByID(id string) (bool, error)

	// ExistsByMessageID checks if a cc entry exists with the given message ID
	ExistsByMessageID(messageID string) (bool, error)

	// ExistsByRequestID checks if a cc entry exists with the given request ID
	ExistsByRequestID(requestID string) (bool, error)

	// CountAll returns the total number of cc entries
	CountAll() (int, error)

	// CountByDateRange returns the number of entries within a date range
	CountByDateRange(start, end time.Time) (int, error)

	// GetDistinctProjects returns all unique project paths
	GetDistinctProjects() ([]string, error)

	// GetDistinctModels returns all unique model names
	GetDistinctModels() ([]string, error)

	// GetDistinctSessions returns all unique session IDs
	GetDistinctSessions() ([]string, error)

	// GetDateRange returns the earliest and latest dates with cc entries
	GetDateRange() (start, end time.Time, err error)

	// Save persists a cc entry (for future extension)
	Save(entry *entity.CcEntry) error

	// SaveAll persists multiple cc entries (for future extension)
	SaveAll(entries []*entity.CcEntry) error

	// DeleteByID deletes a cc entry by ID (for future extension)
	DeleteByID(id string) error

	// DeleteByDateRange deletes entries within a date range (for future extension)
	DeleteByDateRange(start, end time.Time) error
}

// CcRepositoryError represents errors from the cc repository
type CcRepositoryError struct {
	Operation string
	Err       error
}

func (e *CcRepositoryError) Error() string {
	if e.Err != nil {
		return "cc repository error in " + e.Operation + ": " + e.Err.Error()
	}
	return "cc repository error in " + e.Operation
}

func (e *CcRepositoryError) Unwrap() error {
	return e.Err
}

// NewCcRepositoryError creates a new cc repository error
func NewCcRepositoryError(operation string, err error) error {
	return &CcRepositoryError{
		Operation: operation,
		Err:       err,
	}
}

// Common error types
var (
	// ErrCcNotFound is returned when a cc entry is not found
	ErrCcNotFound = &CcRepositoryError{Operation: "find", Err: nil}

	// ErrCcAlreadyExists is returned when trying to save a duplicate entry
	ErrCcAlreadyExists = &CcRepositoryError{Operation: "save", Err: nil}

	// ErrInvalidDateRange is returned when the date range is invalid
	ErrInvalidDateRange = &CcRepositoryError{Operation: "validate", Err: nil}
)
