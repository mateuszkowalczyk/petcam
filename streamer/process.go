package streamer

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

type process struct {
	settings Settings

	cmd     *exec.Cmd
	stopped chan struct{}
}

func NewProcess(settings Settings) *process {
	return &process{
		settings: settings,
		stopped:  make(chan struct{}, 1), // why buffered if we use close?
	}
}

func (p *process) Start() error {
	// Cleanup before start
	p.removeStreamDirectory()

	if err := os.MkdirAll(p.settings.StreamPath, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %s: %v", p.settings.StreamPath, err)
	}

	cmd := exec.Command(p.settings.Command)
	// Run in a new process group so children (e.g., rpicam-vid, ffmpeg) can be killed together.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Env = append(os.Environ(),
		"HLS_LIST_SIZE="+strconv.Itoa(hlsListSize),
		"HLS_BASE_URL="+p.settings.HlsBaseURL,
		"PLAYLIST_PATH="+p.settings.PlaylistPath,
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start streaming process: %v", err)
	}
	p.cmd = cmd

	go func() {
		// use Wait() in case the process is killed externally
		if err := cmd.Wait(); err != nil {
			log.Printf("streaming process stopped: %v\n", err)
		}

		p.removeStreamDirectory()
		close(p.stopped)
	}()

	return p.waitForStart()
}

func (p *process) Stop() {
	// SIGKILL with negative PID terminates the entire process group.
	// This ensures all children spawned by the shell script (e.g., rpicam-vid, ffmpeg)
	// are terminated.
	if p.cmd != nil && p.cmd.Process != nil {
		if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("failed to send SIGKILL: %v\n", err)
		}
	}
	<-p.stopped
}

func (p *process) Done() <-chan struct{} {
	return p.stopped
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
