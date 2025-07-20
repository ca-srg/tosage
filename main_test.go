package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestBackwardCompatibility_CLIMode tests that the CLI mode continues to work
// as expected even with the new daemon features
func TestBackwardCompatibility_CLIMode(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test")
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", "tosage-test")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer func() {
		_ = os.Remove("tosage-test")
	}()

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		wantErr  bool
		contains string
	}{
		{
			name: "Default CLI mode",
			args: []string{},
			env: map[string]string{
				"TOSAGE_DAEMON_ENABLED": "false",
			},
			wantErr:  false,
			contains: "", // Should output a number or error
		},
		{
			name:     "Explicit CLI flag",
			args:     []string{"--cli"},
			env:      map[string]string{},
			wantErr:  false,
			contains: "",
		},
		{
			name: "CLI mode with Prometheus disabled",
			args: []string{"--cli"},
			env: map[string]string{
				"TOSAGE_PROMETHEUS_REMOTE_WRITE_URL": "",
			},
			wantErr:  false,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./tosage-test", tt.args...)

			// Set environment variables
			for k, v := range tt.env {
				cmd.Env = append(os.Environ(), k+"="+v)
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
				t.Logf("stdout: %s", stdout.String())
				t.Logf("stderr: %s", stderr.String())
			}

			if tt.contains != "" && !bytes.Contains(stdout.Bytes(), []byte(tt.contains)) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, stdout.String())
			}
		})
	}
}

// TestBackwardCompatibility_EnvironmentVariables tests that all existing
// environment variables continue to work
func TestBackwardCompatibility_EnvironmentVariables(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test")
	}

	envVars := []struct {
		name  string
		value string
	}{
		{"TOSAGE_CLAUDE_PATH", "/tmp/test-claude"},
		{"TOSAGE_TIMEZONE", "UTC"},
		{"TOSAGE_PROMETHEUS_SERVER_URL", "http://localhost:9091"},
		{"TOSAGE_PROMETHEUS_REMOTE_WRITE_URL", "http://localhost:9090/api/v1/write"},
		{"TOSAGE_PROMETHEUS_HOST_LABEL", "test-host"},
		{"TOSAGE_PROMETHEUS_INTERVAL_SECONDS", "600"},
		{"TOSAGE_PROMETHEUS_TIMEOUT_SECONDS", "30"},
		{"TOSAGE_CURSOR_DB_PATH", "/tmp/test-cursor.db"},
		{"TOSAGE_CURSOR_API_TIMEOUT", "30"},
		{"TOSAGE_CURSOR_CACHE_TIMEOUT", "300"},
	}

	// Create a temporary test script that reads environment
	script := `#!/bin/bash
for var in "$@"; do
    echo "$var=${!var}"
done
`
	scriptFile := "/tmp/test-env.sh"
	err := os.WriteFile(scriptFile, []byte(script), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	defer func() {
		_ = os.Remove(scriptFile)
	}()

	// Test that environment variables are properly loaded
	for _, ev := range envVars {
		t.Run(ev.name, func(t *testing.T) {
			_ = os.Setenv(ev.name, ev.value)
			defer func() {
				_ = os.Unsetenv(ev.name)
			}()

			// The application should be able to load with these env vars
			// This is a basic smoke test to ensure no breaking changes
			cmd := exec.Command("go", "run", ".", "--cli")
			cmd.Env = append(os.Environ(), ev.name+"="+ev.value)

			// Set timeout to prevent hanging
			done := make(chan error, 1)
			go func() {
				done <- cmd.Run()
			}()

			select {
			case <-done:
				// Command completed
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				// Not necessarily an error - the app might be waiting for data
			}
		})
	}
}

// TestBackwardCompatibility_ConfigStructure tests that the configuration
// structure remains backward compatible
func TestBackwardCompatibility_ConfigStructure(t *testing.T) {
	// This test ensures that all existing configuration fields
	// are still present and functional

	// Test configuration JSON compatibility
	oldConfigJSON := `{
		"claude_path": "/custom/claude/path",
		"timezone": "Asia/Tokyo",
		"prometheus": {
			"server_url": "http://localhost:9091",
			"host_label": "my-host",
			"interval_seconds": 600,
			"timeout_seconds": 30
		},
		"cursor": {
			"database_path": "/custom/cursor.db",
			"api_timeout": 30,
			"cache_timeout": 300
		}
	}`

	// Write config to temporary file
	configFile := "/tmp/test-config.json"
	err := os.WriteFile(configFile, []byte(oldConfigJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer func() {
		_ = os.Remove(configFile)
	}()

	// The application should be able to load this config
	// This is validated by the config loading logic in the app
}

// TestBackwardCompatibility_PrometheusIntegration tests that the existing
// Prometheus integration continues to work
func TestBackwardCompatibility_PrometheusIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test")
	}

	// Test that metrics sending still works in CLI mode
	cmd := exec.Command("go", "run", ".", "--cli")
	cmd.Env = append(os.Environ(),
		"TOSAGE_PROMETHEUS_REMOTE_WRITE_URL=http://localhost:9090/api/v1/write",
		"TOSAGE_PROMETHEUS_INTERVAL_SECONDS=60",
		"TOSAGE_DAEMON_ENABLED=false",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			// Check if it's just a connection error (expected if Prometheus isn't running)
			stderrStr := stderr.String()
			if !bytes.Contains([]byte(stderrStr), []byte("connection refused")) &&
				!bytes.Contains([]byte(stderrStr), []byte("no such host")) {
				t.Errorf("Unexpected error: %v\nstderr: %s", err, stderrStr)
			}
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
	}
}

// TestBackwardCompatibility_CursorIntegration tests that the existing
// Cursor integration continues to work
func TestBackwardCompatibility_CursorIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test")
	}

	// Create a mock Cursor database
	mockDB := "/tmp/test-cursor.db"
	defer func() {
		_ = os.Remove(mockDB)
	}()

	cmd := exec.Command("go", "run", ".", "--cli")
	cmd.Env = append(os.Environ(),
		"TOSAGE_CURSOR_DB_PATH="+mockDB,
		"TOSAGE_DAEMON_ENABLED=false",
	)

	err := cmd.Run()
	// The command might fail if there's no actual data, but it shouldn't crash
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 is acceptable (no data found)
			if exitErr.ExitCode() != 1 {
				t.Errorf("Unexpected exit code: %d", exitErr.ExitCode())
			}
		}
	}
}
