package impl

import (
	"sync"
	"time"

	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// StatusServiceImpl implements StatusService
type StatusServiceImpl struct {
	mu     sync.RWMutex
	status *usecase.StatusInfo
}

// NewStatusService creates a new instance of StatusService
func NewStatusService() usecase.StatusService {
	return &StatusServiceImpl{
		status: &usecase.StatusInfo{
			IsRunning: false,
		},
	}
}

// GetStatus returns the current status information
func (s *StatusServiceImpl) GetStatus() (*usecase.StatusInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid concurrent modification
	statusCopy := &usecase.StatusInfo{
		IsRunning:         s.status.IsRunning,
		LastMetricsSentAt: s.status.LastMetricsSentAt,
		NextMetricsSendAt: s.status.NextMetricsSendAt,
		TodayTokenCount:   s.status.TodayTokenCount,
		LastError:         s.status.LastError,
		LastErrorAt:       s.status.LastErrorAt,
		DaemonStartedAt:   s.status.DaemonStartedAt,
	}

	return statusCopy, nil
}

// UpdateLastMetricsSent updates the last metrics sent timestamp
func (s *StatusServiceImpl) UpdateLastMetricsSent(sentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.LastMetricsSentAt = &sentAt
	return nil
}

// UpdateNextMetricsSend updates the next metrics send timestamp
func (s *StatusServiceImpl) UpdateNextMetricsSend(nextAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.NextMetricsSendAt = &nextAt
	return nil
}

// UpdateTodayTokenCount updates today's token count
func (s *StatusServiceImpl) UpdateTodayTokenCount(count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.TodayTokenCount = count
	return nil
}

// RecordError records an error that occurred
func (s *StatusServiceImpl) RecordError(err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.status.LastError = err
	s.status.LastErrorAt = &now
	return nil
}

// ClearError clears the last error
func (s *StatusServiceImpl) ClearError() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.LastError = nil
	s.status.LastErrorAt = nil
	return nil
}

// SetDaemonStarted sets the daemon started timestamp
func (s *StatusServiceImpl) SetDaemonStarted(startedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.IsRunning = true
	s.status.DaemonStartedAt = &startedAt
	return nil
}

// SetDaemonStopped clears the daemon runtime information
func (s *StatusServiceImpl) SetDaemonStopped() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.IsRunning = false
	s.status.DaemonStartedAt = nil
	s.status.NextMetricsSendAt = nil
	return nil
}
