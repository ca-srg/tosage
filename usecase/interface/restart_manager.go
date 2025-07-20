package usecase

// RestartManager はアプリケーションの再起動を管理するインターフェース
type RestartManager interface {
	// RequestRestart はアプリケーションの再起動をリクエストする
	RequestRestart() error

	// ScheduleRestart は指定された秒数後に再起動をスケジュールする
	ScheduleRestart(delaySec int) error

	// CancelRestart はスケジュールされた再起動をキャンセルする
	CancelRestart() error

	// IsRestartPending は再起動が保留中かどうかを返す
	IsRestartPending() bool

	// GetRestartReason は再起動の理由を返す
	GetRestartReason() string

	// SetRestartReason は再起動の理由を設定する
	SetRestartReason(reason string)
}
