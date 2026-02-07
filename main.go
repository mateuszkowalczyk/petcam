package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
)

func main() {
	fmt.Println("This is petcam 🐶🐱")
	fmt.Println("Starting streaming...")
	go startStream()

	// TODO: remove, just for initial quick manual testing
	time.Sleep(20 * time.Second)
}

func startStream() {
	basePath := "/dev/shm"
	if _, err := os.Stat(basePath); err != nil {
		log.Fatalf("%s cannot be accessed, but is required for the program to run", basePath)
	}

	streamPath := path.Join(basePath, "stream")
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
		path.Join(streamPath, "stream.m3u8"),
	)

	if err := cmd.Run(); err != nil {
		log.Fatal("Cannot run streaming process")
	}
}
