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
	// Create temp dir for fake FFmpeg and stream output
	tempDir := t.TempDir()

	// Create a fake FFmpeg that creates 4 segment files and stays running
	// Use ${!#} to get the last argument (playlist path) reliably
	fakeScript := `PLAYLIST_PATH="${!#}"
STREAM_DIR=$(dirname "$PLAYLIST_PATH")
mkdir -p "$STREAM_DIR"
touch "$STREAM_DIR/segment_0.ts"
touch "$STREAM_DIR/segment_1.ts"
touch "$STREAM_DIR/segment_2.ts"
touch "$STREAM_DIR/segment_3.ts"
# Keep running until killed
sleep 300
`
	fakeCmd := createFakeFFmpeg(t, tempDir, fakeScript)

	// Create settings with 100ms timeout for fast testing
	settings := createTestSettings(t, fakeCmd)

	// Create and start streamer
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

	// Stop the streamer and verify clean shutdown
	s.Stop()

	// Verify no errors occurred
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
	fakeScript := fmt.Sprintf(`PLAYLIST_PATH="${!#}"
STREAM_DIR=$(dirname "$PLAYLIST_PATH")
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

	// Create and start streamer
	s := NewStreamer(settings)
	s.Start()

	// Launch 5 goroutines simultaneously to simulate 5 users hitting the endpoint
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.EnsureStreaming()
		}(i)
	}

	// Wait for all goroutines to complete
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

	// Clean shutdown
	s.Stop()
	if err := s.Wait(); err != nil {
		t.Fatalf("streamer reported error: %v", err)
	}
}
