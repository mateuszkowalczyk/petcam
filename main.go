package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
)

const basePath = "/dev/shm"

var (
	streamPath   = path.Join(basePath, "stream")
	playlistPath = path.Join(streamPath, "playlist.m3u8")
)

func main() {
	fmt.Println("This is petcam 🐶🐱")
	fmt.Println("Starting streaming...")
	go startStream()

	http.Handle("/segments/", http.StripPrefix("/segments/", http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", playlist)

	fmt.Println("Starting server...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}

	// TODO: stop stream process on exit
	// TODO: stop stream process after a while of inactivity and restart it on first request
}

func startStream() {
	if _, err := os.Stat(basePath); err != nil {
		log.Fatalf("%s cannot be accessed, but is required for the program to run", basePath)
	}

	if err := os.MkdirAll(streamPath, 0o755); err != nil {
		log.Fatalf("Cannot create directory: %s", streamPath)
	}

	defer func() { _ = os.RemoveAll(streamPath) }()

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

	if err := cmd.Run(); err != nil {
		log.Fatalf("Cannot run streaming proces: %v", err)
	}
}

func playlist(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(playlistPath); err != nil {
		log.Printf("Stream not accessible: %v", err)
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, playlistPath)
}
