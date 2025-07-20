package usecase

import (
	"time"
)

// StatusInfo represents the current status of the application
type StatusInfo struct {
	// IsRunning indicates whether the daemon is currently running
	IsRunning bool

	// LastMetricsSentAt is the timestamp of the last successful metrics send
	LastMetricsSentAt *time.Time

	// NextMetricsSendAt is the timestamp when the next metrics send is scheduled
	NextMetricsSendAt *time.Time

	// TodayTokenCount is the total token count for today
	TodayTokenCount int64

	// LastError is the last error that occurred (if any)
	LastError error

	// LastErrorAt is the timestamp of the last error
	LastErrorAt *time.Time

	// DaemonStartedAt is the timestamp when the daemon was started
	DaemonStartedAt *time.Time
}

// StatusService provides status information about the application
type StatusService interface {
	// GetStatus returns the current status information
	GetStatus() (*StatusInfo, error)

	// UpdateLastMetricsSent updates the last metrics sent timestamp
	UpdateLastMetricsSent(sentAt time.Time) error

	// UpdateNextMetricsSend updates the next metrics send timestamp
	UpdateNextMetricsSend(nextAt time.Time) error

	// UpdateTodayTokenCount updates today's token count
	UpdateTodayTokenCount(count int64) error

	// RecordError records an error that occurred
	RecordError(err error) error

	// ClearError clears the last error
	ClearError() error

	// SetDaemonStarted sets the daemon started timestamp
	SetDaemonStarted(startedAt time.Time) error

	// SetDaemonStopped clears the daemon runtime information
	SetDaemonStopped() error
}
