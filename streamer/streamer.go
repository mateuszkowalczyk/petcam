package streamer

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type Streamer struct {
	streamPath   string
	playlistPath string
	hlsBaseURL   string

	keepAlive chan struct{}
	running   chan struct{}
	done      chan error
	quit      chan struct{}
}

func NewStreamer(streamPath, playlistPath, hlsBaseURL string) *Streamer {
	return &Streamer{
		streamPath:   streamPath,
		playlistPath: playlistPath,
		hlsBaseURL:   hlsBaseURL,

		// TODO: improve channels naming
		keepAlive: make(chan struct{}),
		running:   make(chan struct{}),
		done:      make(chan error, 1),
		quit:      make(chan struct{}),
	}
}

func (s *Streamer) Start() {
	go s.streamLoop()
}

func (s *Streamer) Stop() {
	s.quit <- struct{}{}
}

func (s *Streamer) streamLoop() {
	var streamProcess *os.Process

	for {
		select {
		case <-s.keepAlive:
			if streamProcess == nil {
				var err error
				streamProcess, err = s.startStream()
				if err != nil {
					log.Fatalf("Cannot start stream process: %v", err)
				}
			}
			s.running <- struct{}{}
		case err := <-s.done:
			streamProcess = nil
			if err != nil {
				log.Printf("Error running command: %v", err)
			}
		case <-time.After(30 * time.Second):
			if streamProcess != nil {
				if err := streamProcess.Kill(); err != nil {
					log.Printf("Error while killing the process: %v", err)
				}
				log.Println("Stopping streaming process due to inactivity")
			}
		case <-s.quit:
			if streamProcess != nil {
				if err := streamProcess.Kill(); err != nil {
					log.Printf("Error while killing the process: %v", err)
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
		"-hls_list_size", "3",
		"-hls_flags", "delete_segments",
		"-hls_base_url", s.hlsBaseURL,
		s.playlistPath,
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start streaming process: %v", err)
	}

	go func() {
		s.done <- cmd.Wait()
		s.removeStreamDirectory()
	}()

	// Wait until playlist file and at least 3 video segments are created
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case err := <-s.done:
			return nil, fmt.Errorf("cannot start streaming process: %w", err)
		case <-ticker.C:
			entries, err := os.ReadDir(s.streamPath)
			if err != nil {
				return nil, fmt.Errorf("cannot list files in the stream directory: %v", err)
			}

			if len(entries) >= 4 {
				return cmd.Process, nil
			}
		}
	}
}

func (s *Streamer) removeStreamDirectory() {
	if err := os.RemoveAll(s.streamPath); err != nil {
		log.Printf("Couldn't remove stream directory: %v", err)
	}
}
