// Package streamer manages HLS video streaming from the Raspberry Pi camera
// using FFmpeg and rpicam-vid, the latter providing hardware acceleration.
// It provides lifecycle management with automatic cleanup and inactivity timeout.
package streamer

// TODO: add information about hardware acceleration after replacing streaming command with real one

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const hlsListSize = 3

type Streamer struct {
	settings Settings

	process *process

	keepAlive      chan struct{} // Receives signals to keep stream active
	streamingAlive chan struct{} // Signals when streaming is ready in response to keepAlive (buffered)
	stop           chan struct{} // Signals stream loop to stop
	stopped        chan struct{} // Signals when stream has been stopped

	stopOnce sync.Once
	waitOnce sync.Once

	err error // Stores error from stream loop
}

func NewStreamer(settings Settings) *Streamer {
	return &Streamer{
		settings: settings,

		process: nil,

		keepAlive:      make(chan struct{}, 1),
		streamingAlive: make(chan struct{}, 1),
		stop:           make(chan struct{}),
		stopped:        make(chan struct{}, 1),
	}
}

// Start begins the streaming loop in a goroutine. It must be called before using EnsureStreaming.
func (s *Streamer) Start() {
	go s.streamLoop()
}

// Stop terminates the streaming process, stops the streaming loop and cleans up resources.
// Blocks until cleanup is complete.
func (s *Streamer) Stop() {
	s.stopOnce.Do(func() {
		close(s.stop)
		<-s.stopped
	})
}

// Wait blocks until the streamer has stopped and returns any error that occurred.
// It is safe to call Wait multiple times; only the first call will block.
func (s *Streamer) Wait() error {
	s.waitOnce.Do(func() {
		<-s.stopped
	})
	return s.err
}

// EnsureStreaming ensures the stream is active or starts it if necessary.
// Blocks until streaming is established. It must be called after Start.
// Returns immediately if Stop() has been called or if an error occurs.
func (s *Streamer) EnsureStreaming() {
	s.keepAlive <- struct{}{}
	select {
	case <-s.streamingAlive:
	case <-s.stop:
	case <-s.stopped:
		// Streamer stopped (possibly due to error)
	}
}

// streamLoop manages the streaming process lifecycle, handling:
// - Starting streaming process if not already running
// - Inactivity timeout (killing streaming process after 30 seconds of no http request)
// - Process exit monitoring
// - Shutdown on stop signal
func (s *Streamer) streamLoop() {
	for {
		select {
		case <-s.keepAlive:
			if s.process == nil {
				log.Println("starting streaming process...")
				s.process = NewProcess(s.settings)
				if err := s.process.Start(); err != nil {
					s.err = fmt.Errorf("cannot start streaming process: %w", err)
					close(s.stopped)
					return
				}
			}
			s.streamingAlive <- struct{}{}
		case <-time.After(s.settings.InactivityTimeout):
			if s.process != nil {
				log.Println("stopping streaming process due to inactivity...")
				s.process.Stop()
				s.process = nil
			}
		case <-s.processDone():
			s.process = nil
		case <-s.stop:
			if s.process != nil {
				s.process.Stop()
			}
			close(s.stopped)
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
