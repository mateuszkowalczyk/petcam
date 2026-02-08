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
	streamPath   = path.Join(basePath, "stream")
	playlistPath = path.Join(streamPath, "playlist.m3u8")
	keepAlive    = make(chan struct{})
)

func main() {
	fmt.Println("This is petcam 🐶🐱")
	fmt.Println("Starting streaming...")

	quit := make(chan struct{})
	go runStream(keepAlive, quit)
	defer func() { quit <- struct{}{} }()

	http.Handle("/segments/", http.StripPrefix("/segments/", http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", playlist)

	fmt.Println("Starting server...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}

	// TODO: handle Ctrl+C signal gracefully
}

func runStream(keepAlive <-chan struct{}, quit <-chan struct{}) {
	done := make(chan error, 1)
	var streamProcess *os.Process
	for {
		select {
		case <-keepAlive:
			if streamProcess == nil {
				streamProcess = startStream(done)
			}
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

func startStream(done chan<- error) *os.Process {
	if _, err := os.Stat(basePath); err != nil {
		log.Fatalf("%s cannot be accessed, but is required for the program to run", basePath)
	}

	if err := os.MkdirAll(streamPath, 0o755); err != nil {
		log.Fatalf("Cannot create directory: %s", streamPath)
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
		log.Fatalf("Cannot start streaming proces: %v", err)
	}

	go func() {
		done <- cmd.Wait()
		if err := os.RemoveAll(streamPath); err != nil {
			log.Printf("Couldn't remove stream directory: %v", err)
		}
	}()

	return cmd.Process
}

func playlist(w http.ResponseWriter, r *http.Request) {
	keepAlive <- struct{}{}
	// TODO: wait for process to start to prevent from 404 on first request

	if _, err := os.Stat(playlistPath); err != nil {
		log.Printf("Stream not accessible: %v", err)
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, playlistPath)
}
