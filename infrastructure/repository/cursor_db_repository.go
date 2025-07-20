package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ca-srg/tosage/domain"
	"github.com/ca-srg/tosage/domain/repository"
	"github.com/ca-srg/tosage/domain/valueobject"
	_ "github.com/mattn/go-sqlite3"
)

// CursorDBRepository implements the CursorTokenRepository interface
type CursorDBRepository struct {
	customDBPath string
}

// NewCursorDBRepository creates a new CursorDBRepository instance
func NewCursorDBRepository(customDBPath string) repository.CursorTokenRepository {
	return &CursorDBRepository{
		customDBPath: customDBPath,
	}
}

// GetToken retrieves the Cursor authentication token from the SQLite database
func (r *CursorDBRepository) GetToken() (*valueobject.CursorToken, error) {
	dbPath := r.getCursorDBPath()

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, domain.ErrCursorDatabase("GetToken", dbPath).
			WithDetails("error", "database file not found")
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, domain.ErrCursorDatabaseWithCause("GetToken", dbPath, err)
	}
	defer func() {
		_ = db.Close()
	}()

	// Query for the access token
	var tokenValue string
	err = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/accessToken'").Scan(&tokenValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrCursorDatabase("GetToken", dbPath).
				WithDetails("error", "no token found in database")
		}
		return nil, domain.ErrCursorDatabaseWithCause("GetToken", dbPath, err)
	}

	// Create CursorToken value object with the JWT token
	// Note: CursorToken will handle the session token format internally
	cursorToken, err := valueobject.NewCursorToken(tokenValue)
	if err != nil {
		return nil, domain.ErrCursorTokenWithCause("invalid token format", err)
	}

	return cursorToken, nil
}

// getCursorDBPath determines the path to the Cursor SQLite database
func (r *CursorDBRepository) getCursorDBPath() string {
	// Use custom path if provided
	if r.customDBPath != "" {
		return r.customDBPath
	}

	// Determine default path based on platform
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		homeDir = "."
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/Cursor/User/globalStorage/state.vscdb
		return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb")
	case "windows":
		// Windows: %APPDATA%\Cursor\User\globalStorage\state.vscdb
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb")
	case "linux":
		// Linux: ~/.config/Cursor/User/globalStorage/state.vscdb
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configDir, "Cursor", "User", "globalStorage", "state.vscdb")
	default:
		// Default fallback
		return filepath.Join(homeDir, ".config", "Cursor", "User", "globalStorage", "state.vscdb")
	}
}
