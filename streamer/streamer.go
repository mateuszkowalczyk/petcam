// Package streamer manages HLS video streaming from the Raspberry Pi camera using FFmpeg.
// It provides lifecycle management with automatic cleanup and inactivity timeout.
package streamer

// TODO: add information about hardware acceleration after replacing streaming command with real one

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const hlsListSize = 3

type Streamer struct {
	streamPath   string
	playlistPath string
	hlsBaseURL   string

	keepAlive     chan struct{} // Receives signals to keep stream active
	streaming     chan struct{} // Signals when streaming is ready (buffered)
	streamingDone chan error    // Receives FFmpeg process exit status (buffered)
	quit          chan struct{} // Signals goroutine to stop (buffered)
}

func NewStreamer(streamPath, playlistPath, hlsBaseURL string) *Streamer {
	return &Streamer{
		streamPath:   streamPath,
		playlistPath: playlistPath,
		hlsBaseURL:   hlsBaseURL,

		keepAlive:     make(chan struct{}),
		streaming:     make(chan struct{}, 1),
		streamingDone: make(chan error, 1),
		quit:          make(chan struct{}, 1),
	}
}

// Start begins the streaming loop in a goroutine. It must be called before using EnsureStreaming.
func (s *Streamer) Start() {
	go s.streamLoop()
}

// Stop terminates the streaming process, stops the streaming loop and cleans up resources.
func (s *Streamer) Stop() {
	// TODO: check if streamer is running before sending to `quit` channel
	s.quit <- struct{}{}
	<-s.streamingDone
}

// EnsureStreaming ensures the stream is active or starts it if necessary.
// Blocks until streaming is established. It must be called after Start.
func (s *Streamer) EnsureStreaming() {
	s.keepAlive <- struct{}{}
	<-s.streaming
}

// streamLoop manages the streaming process lifecycle, handling:
// - Starting streaming process if not already running
// - Inactivity timeout (killing streaming process after 30 seconds of no http request)
// - Process exit monitoring
// - Shutdown on quit signal
func (s *Streamer) streamLoop() {
	var streamProcess *os.Process

	for {
		select {
		case <-s.keepAlive:
			if streamProcess == nil {
				log.Println("starting streaming process...")
				var err error
				streamProcess, err = s.startStream()
				if err != nil {
					log.Fatalf("cannot start streaming process: %v\n", err)
				}
			}
			s.streaming <- struct{}{}
		case <-time.After(30 * time.Second):
			if streamProcess != nil {
				log.Println("stopping streaming process due to inactivity...")
				if err := streamProcess.Kill(); err != nil {
					log.Printf("error while killing the streaming process: %v\n", err)
				}
			}
		case err := <-s.streamingDone:
			streamProcess = nil
			if err != nil {
				log.Printf("error running streaming process: %v\n", err)
			}
		case <-s.quit:
			if streamProcess != nil {
				if err := streamProcess.Kill(); err != nil {
					log.Printf("error while killing the streaming process: %v\n", err)
				}
			}
			return
		}
	}
}

// startStream starts FFmpeg and waits for the playlist and initial segments
// to be created. Returns the process handle or an error if startup fails.
// Blocks until hlsListSize+1 files exist in streamPath.
// TODO: add information about Raspberry Pi cam process
func (s *Streamer) startStream() (*os.Process, error) {
	// Cleanup before start
	s.removeStreamDirectory()

	if err := os.MkdirAll(s.streamPath, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %v", s.streamPath, err)
	}

	cmd := exec.Command(
		"ffmpeg",
		"-f", "v4l2",
		"-i", "/dev/video0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-f", "hls",
		"-hls_time", "1",
		"-hls_list_size", strconv.Itoa(hlsListSize),
		"-hls_flags", "delete_segments",
		"-hls_base_url", s.hlsBaseURL,
		s.playlistPath,
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start streaming process: %v", err)
	}

	go func() {
		err := cmd.Wait()
		s.removeStreamDirectory()
		s.streamingDone <- err
	}()

	// Wait until playlist file and all initial video segments are created
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case err := <-s.streamingDone:
			return nil, fmt.Errorf("streaming process stopped unexpectedly: %w", err)
		case <-ticker.C:
			entries, err := os.ReadDir(s.streamPath)
			if err != nil {
				return nil, fmt.Errorf("cannot list files in the stream directory: %v", err)
			}

			if len(entries) >= hlsListSize+1 {
				return cmd.Process, nil
			}
		}
	}
}

func (s *Streamer) removeStreamDirectory() {
	if err := os.RemoveAll(s.streamPath); err != nil {
		log.Printf("couldn't remove stream directory: %v\n", err)
	}
}
