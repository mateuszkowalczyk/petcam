package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createFakeFFmpeg creates an executable shell script that simulates FFmpeg.
// The script will be created in the given directory and execute the provided shell commands.
// Returns the full path to the script.
func createFakeFFmpeg(t *testing.T, dir string, script string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "fake_ffmpeg.sh")
	content := fmt.Sprintf("#!/bin/bash\n%s", script)

	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to create fake ffmpeg script: %v", err)
	}

	return scriptPath
}

// createTestSettings creates Settings with a temporary directory for streaming.
// The timeout is set to 100ms for fast test execution.
// If command is provided, it will be used as the streaming command.
// Returns the Settings struct (t.TempDir() handles cleanup automatically).
func createTestSettings(t *testing.T, command string) Settings {
	t.Helper()

	tempDir := t.TempDir()
	streamPath := filepath.Join(tempDir, "stream")
	playlistPath := filepath.Join(streamPath, "playlist.m3u8")

	return Settings{
		StreamPath:        streamPath,
		PlaylistPath:      playlistPath,
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 100 * time.Millisecond,
		Command:           command,
	}
}

// waitForFile polls for a file to exist, returning an error if it doesn't exist within timeout.
func waitForFile(t *testing.T, path string, timeout time.Duration) error {
	t.Helper()

	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for file: %s", path)
		case <-ticker.C:
			if _, err := os.Stat(path); err == nil {
				return nil
			}
		}
	}
}

// waitForCondition polls a condition function until it returns true or timeout expires.
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration) bool {
	t.Helper()

	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		}
	}
}
