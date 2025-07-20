package impl

import (
	"errors"
	"testing"
	"time"
)

func TestStatusServiceImpl_BasicOperations(t *testing.T) {
	service := NewStatusService()

	// Test initial status
	status, err := service.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.IsRunning {
		t.Error("Expected IsRunning to be false initially")
	}
	if status.LastMetricsSentAt != nil {
		t.Error("Expected LastMetricsSentAt to be nil initially")
	}
	if status.TodayTokenCount != 0 {
		t.Error("Expected TodayTokenCount to be 0 initially")
	}

	// Test SetDaemonStarted
	startTime := time.Now()
	err = service.SetDaemonStarted(startTime)
	if err != nil {
		t.Fatalf("SetDaemonStarted failed: %v", err)
	}

	status, _ = service.GetStatus()
	if !status.IsRunning {
		t.Error("Expected IsRunning to be true after SetDaemonStarted")
	}
	if status.DaemonStartedAt == nil || !status.DaemonStartedAt.Equal(startTime) {
		t.Error("DaemonStartedAt not set correctly")
	}

	// Test UpdateLastMetricsSent
	sentTime := time.Now()
	err = service.UpdateLastMetricsSent(sentTime)
	if err != nil {
		t.Fatalf("UpdateLastMetricsSent failed: %v", err)
	}

	status, _ = service.GetStatus()
	if status.LastMetricsSentAt == nil || !status.LastMetricsSentAt.Equal(sentTime) {
		t.Error("LastMetricsSentAt not set correctly")
	}

	// Test UpdateNextMetricsSend
	nextTime := time.Now().Add(10 * time.Minute)
	err = service.UpdateNextMetricsSend(nextTime)
	if err != nil {
		t.Fatalf("UpdateNextMetricsSend failed: %v", err)
	}

	status, _ = service.GetStatus()
	if status.NextMetricsSendAt == nil || !status.NextMetricsSendAt.Equal(nextTime) {
		t.Error("NextMetricsSendAt not set correctly")
	}

	// Test UpdateTodayTokenCount
	err = service.UpdateTodayTokenCount(12345)
	if err != nil {
		t.Fatalf("UpdateTodayTokenCount failed: %v", err)
	}

	status, _ = service.GetStatus()
	if status.TodayTokenCount != 12345 {
		t.Errorf("Expected TodayTokenCount to be 12345, got %d", status.TodayTokenCount)
	}

	// Test SetDaemonStopped
	err = service.SetDaemonStopped()
	if err != nil {
		t.Fatalf("SetDaemonStopped failed: %v", err)
	}

	status, _ = service.GetStatus()
	if status.IsRunning {
		t.Error("Expected IsRunning to be false after SetDaemonStopped")
	}
	if status.DaemonStartedAt != nil {
		t.Error("Expected DaemonStartedAt to be nil after SetDaemonStopped")
	}
	if status.NextMetricsSendAt != nil {
		t.Error("Expected NextMetricsSendAt to be nil after SetDaemonStopped")
	}
}

func TestStatusServiceImpl_ErrorHandling(t *testing.T) {
	service := NewStatusService()

	// Test RecordError
	testErr := errors.New("test error")
	err := service.RecordError(testErr)
	if err != nil {
		t.Fatalf("RecordError failed: %v", err)
	}

	status, _ := service.GetStatus()
	if status.LastError == nil || status.LastError.Error() != testErr.Error() {
		t.Error("LastError not set correctly")
	}
	if status.LastErrorAt == nil {
		t.Error("LastErrorAt not set")
	}

	// Test ClearError
	err = service.ClearError()
	if err != nil {
		t.Fatalf("ClearError failed: %v", err)
	}

	status, _ = service.GetStatus()
	if status.LastError != nil {
		t.Error("Expected LastError to be nil after ClearError")
	}
	if status.LastErrorAt != nil {
		t.Error("Expected LastErrorAt to be nil after ClearError")
	}
}

func TestStatusServiceImpl_ConcurrentAccess(t *testing.T) {
	service := NewStatusService()
	done := make(chan bool)

	// Start multiple goroutines to test concurrent access
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Perform various operations
			_ = service.UpdateTodayTokenCount(int64(id))
			_ = service.UpdateLastMetricsSent(time.Now())
			_, _ = service.GetStatus()
			if id%2 == 0 {
				_ = service.RecordError(errors.New("concurrent error"))
			} else {
				_ = service.ClearError()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify service is still functional
	status, err := service.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed after concurrent access: %v", err)
	}
	if status == nil {
		t.Error("Expected non-nil status after concurrent access")
	}
}
