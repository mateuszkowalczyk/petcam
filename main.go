package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

const basePath = "/dev/shm"

var (
	streamPath    = path.Join(basePath, "stream")
	playlistPath  = path.Join(streamPath, "playlist.m3u8")
	keepAlive     = make(chan struct{})
	streamRunning = make(chan struct{})
)

func main() {
	fmt.Println("This is petcam 🐶🐱")
	fmt.Println("Starting streaming...")

	quit := make(chan struct{})
	go runStream(keepAlive, streamRunning, quit)
	defer func() { quit <- struct{}{} }()

	http.Handle("/segments/", http.StripPrefix("/segments/", http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", playlist)

	fmt.Println("Starting server...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}

	// TODO: handle Ctrl+C signal gracefully
}

func runStream(keepAlive <-chan struct{}, running chan<- struct{}, quit <-chan struct{}) {
	done := make(chan error, 1)
	var streamProcess *os.Process
	for {
		select {
		case <-keepAlive:
			if streamProcess == nil {
				var err error
				streamProcess, err = startStream(done)
				if err != nil {
					log.Fatalf("Cannot start stream process: %v", err)
				}
			}
			running <- struct{}{}
		case err := <-done:
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
		case <-quit:
			if streamProcess != nil {
				if err := streamProcess.Kill(); err != nil {
					log.Printf("Error while killing the process: %v", err)
				}
			}
			return
		}
	}
}

func startStream(done chan error) (*os.Process, error) {
	// Cleanup before start
	removeStreamDirectory()

	if _, err := os.Stat(basePath); err != nil {
		return nil, fmt.Errorf("%s cannot be accessed, but is required for the program to run: %v", basePath, err)
	}

	if err := os.MkdirAll(streamPath, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %v", streamPath, err)
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
		"-hls_base_url", "/segments/",
		playlistPath,
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot start streaming process: %v", err)
	}

	go func() {
		done <- cmd.Wait()
		removeStreamDirectory()
	}()

	// Wait until playlist file and at least 3 video segments are created
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case err := <-done:
			return nil, fmt.Errorf("cannot start streaming process: %w", err)
		case <-ticker.C:
			entries, err := os.ReadDir(streamPath)
			if err != nil {
				return nil, fmt.Errorf("cannot list files in the stream directory: %v", err)
			}

			if len(entries) >= 4 {
				return cmd.Process, nil
			}
		}
	}
}

func removeStreamDirectory() {
	if err := os.RemoveAll(streamPath); err != nil {
		log.Printf("Couldn't remove stream directory: %v", err)
	}
}

func playlist(w http.ResponseWriter, r *http.Request) {
	keepAlive <- struct{}{}
	<-streamRunning

	http.ServeFile(w, r, playlistPath)
}
