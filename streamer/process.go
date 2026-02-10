package streamer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type process struct {
	settings Settings

	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

func NewProcess(settings Settings) *process {
	ctx, cancel := context.WithCancel(context.Background())

	return &process{
		settings: settings,

		ctx:     ctx,
		cancel:  cancel,
		stopped: make(chan struct{}, 1),
	}
}

func (p *process) Start() error {
	// Cleanup before start
	p.removeStreamDirectory()

	if err := os.MkdirAll(p.settings.StreamPath, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %s: %v", p.settings.StreamPath, err)
	}

	cmd := exec.CommandContext(
		p.ctx,
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
		"-hls_base_url", p.settings.HlsBaseURL,
		p.settings.PlaylistPath,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start streaming process: %v", err)
	}

	go func() {
		// use Wait() in case the process is killed externally
		if err := cmd.Wait(); err != nil {
			log.Printf("streaming process stopped: %v\n", err)
		}

		p.cancel()
		p.removeStreamDirectory()
		close(p.stopped)
	}()

	return p.waitForStart()
}

func (p *process) Stop() {
	if p.ctx.Err() != nil {
		return
	}

	p.cancel()
	<-p.stopped
}

func (p *process) Done() <-chan struct{} {
	return p.ctx.Done()
}

func (p *process) removeStreamDirectory() {
	if err := os.RemoveAll(p.settings.StreamPath); err != nil {
		log.Printf("couldn't remove stream directory: %v\n", err)
	}
}

func (p *process) waitForStart() error {
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-p.stopped:
			return errors.New("streaming process stopped unexpectedly")
		case <-ticker.C:
			entries, err := os.ReadDir(p.settings.StreamPath)
			if err != nil {
				return fmt.Errorf("cannot list files in the stream directory: %v", err)
			}

			if len(entries) >= hlsListSize+1 {
				return nil
			}
		}
	}
}
