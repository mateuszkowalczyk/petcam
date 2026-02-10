// Package streamer manages HLS video streaming from the Raspberry Pi camera using FFmpeg.
// It provides lifecycle management with automatic cleanup and inactivity timeout.
package streamer

// TODO: add information about hardware acceleration after replacing streaming command with real one

import (
	"log"
	"time"
)

const hlsListSize = 3

type Streamer struct {
	settings Settings

	process *process

	keepAlive      chan struct{} // Receives signals to keep stream active
	streamingAlive chan struct{} // Signals when streaming is ready in response to keepAlive (buffered)
	quit           chan struct{} // Signals stream loop to stop
}

func NewStreamer(settings Settings) *Streamer {
	return &Streamer{
		settings: settings,

		process: nil,

		keepAlive:      make(chan struct{}),
		streamingAlive: make(chan struct{}, 1),
		quit:           make(chan struct{}),
	}
}

// Start begins the streaming loop in a goroutine. It must be called before using EnsureStreaming.
func (s *Streamer) Start() {
	go s.streamLoop()
}

// Stop terminates the streaming process, stops the streaming loop and cleans up resources.
func (s *Streamer) Stop() {
	if s.process != nil {
		s.process.Stop()
	}

	s.quit <- struct{}{}
}

// EnsureStreaming ensures the stream is active or starts it if necessary.
// Blocks until streaming is established. It must be called after Start.
func (s *Streamer) EnsureStreaming() {
	s.keepAlive <- struct{}{}
	<-s.streamingAlive
}

// streamLoop manages the streaming process lifecycle, handling:
// - Starting streaming process if not already running
// - Inactivity timeout (killing streaming process after 30 seconds of no http request)
// - Process exit monitoring
// - Shutdown on quit signal
func (s *Streamer) streamLoop() {
	for {
		select {
		case <-s.keepAlive:
			if s.process == nil {
				log.Println("starting streaming process...")
				s.process = NewProcess(s.settings)
				if err := s.process.Start(); err != nil {
					log.Fatalf("cannot start streaming process: %v\n", err)
				}
			}
			s.streamingAlive <- struct{}{}
		case <-time.After(30 * time.Second):
			if s.process != nil {
				log.Println("stopping streaming process due to inactivity...")
				s.process.Stop()
			}
		case <-s.processDone():
			s.process = nil
		case <-s.quit:
			return
		}
	}
}

func (s *Streamer) processDone() <-chan struct{} {
	if s.process == nil {
		return nil
	}

	return s.process.Done()
}
