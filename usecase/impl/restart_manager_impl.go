package impl

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	usecase "github.com/ca-srg/tosage/usecase/interface"
)

// RestartManagerImpl は RestartManager の実装
type RestartManagerImpl struct {
	mu             sync.Mutex
	restartPending bool
	restartReason  string
	restartTimer   *time.Timer
	executablePath string
	originalArgs   []string
}

// NewRestartManager は新しい RestartManager を作成する
func NewRestartManager() (usecase.RestartManager, error) {
	// 実行可能ファイルのパスを取得
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	return &RestartManagerImpl{
		executablePath: execPath,
		originalArgs:   os.Args[1:], // プログラム名を除く引数
	}, nil
}

// RequestRestart はアプリケーションの再起動をリクエストする
func (m *RestartManagerImpl) RequestRestart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.restartPending {
		return fmt.Errorf("restart already pending")
	}

	// 即座に再起動を実行
	return m.performRestart()
}

// ScheduleRestart は指定された秒数後に再起動をスケジュールする
func (m *RestartManagerImpl) ScheduleRestart(delaySec int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.restartPending {
		return fmt.Errorf("restart already pending")
	}

	m.restartPending = true

	// 既存のタイマーがあればキャンセル
	if m.restartTimer != nil {
		m.restartTimer.Stop()
	}

	// 新しいタイマーを設定
	m.restartTimer = time.AfterFunc(time.Duration(delaySec)*time.Second, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.restartPending {
			_ = m.performRestart()
		}
	})

	return nil
}

// CancelRestart はスケジュールされた再起動をキャンセルする
func (m *RestartManagerImpl) CancelRestart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.restartPending {
		return fmt.Errorf("no restart pending")
	}

	if m.restartTimer != nil {
		m.restartTimer.Stop()
		m.restartTimer = nil
	}

	m.restartPending = false
	m.restartReason = ""

	return nil
}

// IsRestartPending は再起動が保留中かどうかを返す
func (m *RestartManagerImpl) IsRestartPending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartPending
}

// GetRestartReason は再起動の理由を返す
func (m *RestartManagerImpl) GetRestartReason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartReason
}

// SetRestartReason は再起動の理由を設定する
func (m *RestartManagerImpl) SetRestartReason(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartReason = reason
}

// performRestart は実際の再起動を実行する
func (m *RestartManagerImpl) performRestart() error {
	// 環境変数を保持
	env := os.Environ()

	// 新しいプロセスを起動
	cmd := exec.Command(m.executablePath, m.originalArgs...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// プロセスグループを分離（親プロセスが終了しても子プロセスが継続するように）
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// 新しいプロセスを開始
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}

	// 現在のプロセスを終了
	// Note: ここで os.Exit を呼ぶと、デファードされた処理が実行されない
	// 代わりに、呼び出し元で適切にシャットダウン処理を行う必要がある
	go func() {
		// 少し待ってから終了（新しいプロセスが起動する時間を確保）
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()

	return nil
}

// RestartManagerForTesting はテスト用の RestartManager 実装
type RestartManagerForTesting struct {
	restartRequested bool
	restartReason    string
	mu               sync.Mutex
}

// NewRestartManagerForTesting はテスト用の RestartManager を作成する
func NewRestartManagerForTesting() usecase.RestartManager {
	return &RestartManagerForTesting{}
}

// RequestRestart はアプリケーションの再起動をリクエストする（テスト用）
func (m *RestartManagerForTesting) RequestRestart() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartRequested = true
	return nil
}

// ScheduleRestart は指定された秒数後に再起動をスケジュールする（テスト用）
func (m *RestartManagerForTesting) ScheduleRestart(delaySec int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartRequested = true
	return nil
}

// CancelRestart はスケジュールされた再起動をキャンセルする（テスト用）
func (m *RestartManagerForTesting) CancelRestart() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartRequested = false
	return nil
}

// IsRestartPending は再起動が保留中かどうかを返す（テスト用）
func (m *RestartManagerForTesting) IsRestartPending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartRequested
}

// GetRestartReason は再起動の理由を返す（テスト用）
func (m *RestartManagerForTesting) GetRestartReason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartReason
}

// SetRestartReason は再起動の理由を設定する（テスト用）
func (m *RestartManagerForTesting) SetRestartReason(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartReason = reason
}

// WasRestartRequested はテスト用：再起動がリクエストされたかを返す
func (m *RestartManagerForTesting) WasRestartRequested() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartRequested
}
