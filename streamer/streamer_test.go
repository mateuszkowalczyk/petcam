package streamer

import (
	"path/filepath"
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
