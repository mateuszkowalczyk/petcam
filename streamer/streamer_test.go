package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestSingleUserFlow verifies the default scenario:
// User connects → stream starts → disconnect → timeout → process stops
func TestSingleUserFlow(t *testing.T) {
	tempDir := t.TempDir()

	// Create a fake FFmpeg that creates 4 segment files and stays running
	// Uses environment variables passed by process.go
	fakeScript := `STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
# Keep running until killed
sleep 300
`
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	settings := createTestSettings(t, fakeCmd)

	s := NewStreamer(settings)
	s.Start()

	// Simulate user opening browser (HTTP request)
	s.EnsureStreaming()

	// Verify process started by checking segment files exist
	segmentPath := filepath.Join(settings.StreamPath, "segment_0.ts")
	if err := waitForFile(t, segmentPath, 500*time.Millisecond); err != nil {
		t.Fatalf("stream didn't start, segment file not created: %v", err)
	}

	// User disconnects - don't call EnsureStreaming again
	// Wait for inactivity timeout (100ms in tests)
	time.Sleep(150 * time.Millisecond)

	s.Stop()

	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}

// TestMultipleUsersSimultaneous verifies that multiple concurrent requests
// to EnsureStreaming() only start ONE streaming process.
// This is the real-world scenario: 5 users open the page at the same time.
func TestMultipleUsersSimultaneous(t *testing.T) {
	// Create temp dir for fake FFmpeg and stream output
	tempDir := t.TempDir()
	counterFile := filepath.Join(tempDir, "invocation_count.txt")

	// Create a fake FFmpeg that:
	// 1. Creates segment files
	// 2. Increments an invocation counter (to track how many times it was started)
	// 3. Writes its PID to track unique processes
	pidFile := filepath.Join(tempDir, "process.pid")
	fakeScript := fmt.Sprintf(`STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
# Increment invocation counter
count=$(cat "%s" 2>/dev/null || echo "0")
echo $((count + 1)) > "%s"
# Write PID
echo $$ > "%s"
# Keep running
sleep 300
`, counterFile, counterFile, pidFile)
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	// Create settings with longer timeout (1s) so process stays alive during test
	settings := Settings{
		StreamPath:        filepath.Join(tempDir, "stream"),
		PlaylistPath:      filepath.Join(tempDir, "stream", "playlist.m3u8"),
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 1 * time.Second,
		Command:           fakeCmd,
	}

	s := NewStreamer(settings)
	s.Start()

	// Launch 5 goroutines simultaneously to simulate 5 users hitting the endpoint
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.EnsureStreaming()
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("goroutine reported error: %v", err)
		}
	}

	counterData, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("could not read invocation counter: %v", err)
	}

	var invocationCount int
	if _, err := fmt.Sscanf(string(counterData), "%d", &invocationCount); err != nil {
		t.Fatalf("failed to parse invocation counter: %v", err)
	}

	if invocationCount != 1 {
		t.Errorf("expected 1 process invocation, but got %d (race condition: multiple processes started)", invocationCount)
	}

	// Verify segments were created
	segmentPath := filepath.Join(settings.StreamPath, "segment_0.ts")
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		t.Fatalf("segment file not created, stream didn't start properly")
	}

	s.Stop()
	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}

// TestUserReconnectBeforeTimeout verifies that when a user reconnects
// within the inactivity timeout period, the same process continues running.
// This is the "user refreshes page quickly" scenario.
func TestUserReconnectBeforeTimeout(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "process.pid")

	// Create a fake FFmpeg that writes its PID on startup
	fakeScript := fmt.Sprintf(`STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
echo $$ > "%s"
sleep 300
`, pidFile)
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	settings := Settings{
		StreamPath:        filepath.Join(tempDir, "stream"),
		PlaylistPath:      filepath.Join(tempDir, "stream", "playlist.m3u8"),
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 100 * time.Millisecond,
		Command:           fakeCmd,
	}

	s := NewStreamer(settings)
	s.Start()

	// First connection - start the stream
	s.EnsureStreaming()

	// Verify process started and get PID
	if err := waitForFile(t, pidFile, 500*time.Millisecond); err != nil {
		t.Fatalf("stream didn't start, PID file not created: %v", err)
	}
	firstPIDBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file: %v", err)
	}
	firstPID := string(firstPIDBytes)

	// User disconnects (no calls for 50ms - less than the 100ms timeout)
	time.Sleep(50 * time.Millisecond)

	// User reconnects (within timeout window)
	s.EnsureStreaming()

	// Verify it's still the same process (same PID)
	secondPIDBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file on reconnect: %v", err)
	}
	secondPID := string(secondPIDBytes)

	if firstPID != secondPID {
		t.Errorf("process was restarted on reconnect (PID changed from %s to %s)", firstPID, secondPID)
	}

	s.Stop()
	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}

// TestUserReconnectAfterTimeout verifies that when a user reconnects
// AFTER the inactivity timeout period, a NEW process is started.
// This is the "user returns after a break" scenario.
func TestUserReconnectAfterTimeout(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "process.pid")

	// Create a fake FFmpeg that writes its PID and timestamp on startup
	fakeScript := fmt.Sprintf(`STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
echo "$(date +%%s):$$" >> "%s"
sleep 300
`, pidFile)
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	settings := Settings{
		StreamPath:        filepath.Join(tempDir, "stream"),
		PlaylistPath:      filepath.Join(tempDir, "stream", "playlist.m3u8"),
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 100 * time.Millisecond,
		Command:           fakeCmd,
	}

	s := NewStreamer(settings)
	s.Start()

	// First connection - start the stream
	s.EnsureStreaming()

	// Verify process started and get first PID
	if err := waitForFile(t, pidFile, 500*time.Millisecond); err != nil {
		t.Fatalf("stream didn't start, PID file not created: %v", err)
	}
	firstPIDData, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file: %v", err)
	}
	firstPID := string(firstPIDData)

	// User disconnects - wait LONGER than timeout (150ms > 100ms)
	// This should trigger process termination due to inactivity
	time.Sleep(150 * time.Millisecond)

	// User reconnects (AFTER timeout - should start NEW process)
	s.EnsureStreaming()

	// Verify a NEW process was started (different PID/timestamp)
	secondPIDData, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file on reconnect: %v", err)
	}
	secondPID := string(secondPIDData)

	if firstPID == secondPID {
		t.Errorf("expected new process after timeout, but got same process data (old: %s, new: %s)", firstPID, secondPID)
	}

	s.Stop()
	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}

// TestAppShutdownWhileStreaming verifies clean shutdown when Stop() is called
// while streaming is active. This simulates receiving SIGTERM during streaming.
func TestAppShutdownWhileStreaming(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "process.pid")

	// Create a fake FFmpeg that writes its PID and stays running
	fakeScript := fmt.Sprintf(`STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
echo $$ > "%s"
sleep 300
`, pidFile)
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	settings := Settings{
		StreamPath:        filepath.Join(tempDir, "stream"),
		PlaylistPath:      filepath.Join(tempDir, "stream", "playlist.m3u8"),
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 10 * time.Second,
		Command:           fakeCmd,
	}

	s := NewStreamer(settings)
	s.Start()

	s.EnsureStreaming()

	if err := waitForFile(t, pidFile, 500*time.Millisecond); err != nil {
		t.Fatalf("stream didn't start: %v", err)
	}

	// Simulate app shutdown (SIGTERM) - call Stop() while streaming is active
	// This should gracefully terminate the process
	done := make(chan struct{})
	var stopErr error
	go func() {
		s.Stop()
		stopErr = s.Wait()
		close(done)
	}()

	// Stop should complete within reasonable time (not hang)
	select {
	case <-done:
		// Good - Stop() completed
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() timed out - possible deadlock or hanging process")
	}

	if stopErr != nil {
		t.Fatalf("streamer reported error during shutdown: %v", stopErr)
	}
}

// TestFFmpegCrashRecovery verifies that when FFmpeg crashes unexpectedly,
// the next HTTP request starts a fresh process.
// This simulates the camera disconnecting or FFmpeg failing.
func TestFFmpegCrashRecovery(t *testing.T) {
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "process.pid")

	fakeScript := fmt.Sprintf(`STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
echo $$ > "%s"
# Must stay alive long enough for waitForStart() to verify files exist (process.Start()
# waits for 4 files), then crash to simulate runtime failure (camera disconnect, etc)
sleep 0.3
exit 1
`, pidFile)
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	settings := Settings{
		StreamPath:        filepath.Join(tempDir, "stream"),
		PlaylistPath:      filepath.Join(tempDir, "stream", "playlist.m3u8"),
		HlsBaseURL:        "/segments/",
		InactivityTimeout: 10 * time.Second,
		Command:           fakeCmd,
	}

	s := NewStreamer(settings)
	s.Start()

	s.EnsureStreaming()

	if err := waitForFile(t, pidFile, 500*time.Millisecond); err != nil {
		t.Fatalf("stream didn't start: %v", err)
	}

	firstPIDBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file: %v", err)
	}
	firstPID := string(firstPIDBytes)

	// Wait for the fake FFmpeg to crash (sleeps 300ms then exits with error)
	time.Sleep(500 * time.Millisecond)

	// Remove the old PID file so we can detect when the NEW process writes its PID
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove PID file: %v", err)
	}

	s.EnsureStreaming()

	if err := waitForFile(t, pidFile, 500*time.Millisecond); err != nil {
		t.Fatalf("new stream didn't start after crash: %v", err)
	}

	secondPIDBytes, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("could not read PID file after reconnect: %v", err)
	}
	secondPID := string(secondPIDBytes)

	// Different PIDs prove that crash recovery worked: the old process died and a new one was started
	if firstPID == secondPID {
		t.Errorf("expected new process after crash, but got same PID (old: %s, new: %s)", firstPID, secondPID)
	}

	s.Stop()
	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}
