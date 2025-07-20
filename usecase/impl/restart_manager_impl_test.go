package impl

import (
	"testing"
	"time"
)

func TestRestartManagerImpl_ScheduleRestart(t *testing.T) {
	manager, err := NewRestartManager()
	if err != nil {
		t.Fatalf("Failed to create restart manager: %v", err)
	}

	// 理由を設定
	manager.SetRestartReason("configuration changed")

	// 再起動をスケジュール（実際には実行しない）
	err = manager.ScheduleRestart(5)
	if err != nil {
		t.Fatalf("Failed to schedule restart: %v", err)
	}

	// 再起動が保留中であることを確認
	if !manager.IsRestartPending() {
		t.Error("Restart should be pending")
	}

	// 理由が設定されていることを確認
	if manager.GetRestartReason() != "configuration changed" {
		t.Errorf("Expected reason 'configuration changed', got '%s'", manager.GetRestartReason())
	}

	// キャンセル
	err = manager.CancelRestart()
	if err != nil {
		t.Fatalf("Failed to cancel restart: %v", err)
	}

	// 再起動が保留中でないことを確認
	if manager.IsRestartPending() {
		t.Error("Restart should not be pending after cancel")
	}
}

func TestRestartManagerImpl_CancelNoPending(t *testing.T) {
	manager, err := NewRestartManager()
	if err != nil {
		t.Fatalf("Failed to create restart manager: %v", err)
	}

	// 保留中の再起動がない状態でキャンセル
	err = manager.CancelRestart()
	if err == nil {
		t.Error("Expected error when canceling with no pending restart")
	}
}

func TestRestartManagerImpl_MultipleSchedule(t *testing.T) {
	manager, err := NewRestartManager()
	if err != nil {
		t.Fatalf("Failed to create restart manager: %v", err)
	}

	// 最初の再起動をスケジュール
	err = manager.ScheduleRestart(5)
	if err != nil {
		t.Fatalf("Failed to schedule first restart: %v", err)
	}

	// 2回目のスケジュールは失敗するはず
	err = manager.ScheduleRestart(3)
	if err == nil {
		t.Error("Expected error when scheduling restart while one is already pending")
	}
}

func TestRestartManagerForTesting(t *testing.T) {
	manager := NewRestartManagerForTesting()

	// 初期状態では再起動はリクエストされていない
	if manager.IsRestartPending() {
		t.Error("Initial state should not have pending restart")
	}

	// 再起動をリクエスト
	err := manager.RequestRestart()
	if err != nil {
		t.Fatalf("Failed to request restart: %v", err)
	}

	// テスト用実装の特別なメソッドでチェック
	testManager := manager.(*RestartManagerForTesting)
	if !testManager.WasRestartRequested() {
		t.Error("Restart should have been requested")
	}

	// キャンセル
	err = manager.CancelRestart()
	if err != nil {
		t.Fatalf("Failed to cancel restart: %v", err)
	}

	if testManager.WasRestartRequested() {
		t.Error("Restart should not be requested after cancel")
	}
}

func TestRestartManagerImpl_ScheduleAndAutoCancel(t *testing.T) {
	manager, err := NewRestartManager()
	if err != nil {
		t.Fatalf("Failed to create restart manager: %v", err)
	}

	// 短い遅延でスケジュール
	err = manager.ScheduleRestart(1)
	if err != nil {
		t.Fatalf("Failed to schedule restart: %v", err)
	}

	// すぐにキャンセル
	time.Sleep(100 * time.Millisecond)
	err = manager.CancelRestart()
	if err != nil {
		t.Fatalf("Failed to cancel restart: %v", err)
	}

	// タイマーが発火する時間まで待つ
	time.Sleep(1 * time.Second)

	// まだ保留中でないことを確認
	if manager.IsRestartPending() {
		t.Error("Restart should not be pending after cancel")
	}
}
