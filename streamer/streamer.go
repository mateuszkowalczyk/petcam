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

	keepAlive     chan struct{}
	streaming     chan struct{}
	streamingDone chan error
	quit          chan struct{}
}

func NewStreamer(streamPath, playlistPath, hlsBaseURL string) *Streamer {
	return &Streamer{
		streamPath:   streamPath,
		playlistPath: playlistPath,
		hlsBaseURL:   hlsBaseURL,

		// TODO: improve channels naming
		keepAlive:     make(chan struct{}),
		streaming:     make(chan struct{}),
		streamingDone: make(chan error, 1),
		quit:          make(chan struct{}),
	}
}

// Start begins the streaming loop in a goroutine. It must be called before using EnsureStreaming.
func (s *Streamer) Start() {
	go s.streamLoop()
}

// Stop terminates the streaming process, stops the streaming loop and cleans up resources.
func (s *Streamer) Stop() {
	s.quit <- struct{}{}
}

// EnsureStreaming ensures the stream is active or starts it if necessary.
// Blocks until streaming is established. It must be called after Start.
func (s *Streamer) EnsureStreaming() {
	s.keepAlive <- struct{}{}
	<-s.streaming
}

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
		s.streamingDone <- cmd.Wait()
		s.removeStreamDirectory()
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
