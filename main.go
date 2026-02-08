package main

import (
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/mateuszkowalczyk/petcam/streamer"
)

const (
	basePath   = "/dev/shm"
	hlsBaseURL = "/segments/"
)

var (
	streamPath   = path.Join(basePath, "stream")
	playlistPath = path.Join(streamPath, "playlist.m3u8")
)

func main() {
	fmt.Println("This is petcam 🐶🐱")
	fmt.Println("Starting streaming...")

	streamer := streamer.NewStreamer(streamPath, playlistPath, hlsBaseURL)
	streamer.Start()
	defer streamer.Stop()

	http.Handle(hlsBaseURL, http.StripPrefix(hlsBaseURL, http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", playlist)

	fmt.Println("Starting server...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Cannot start server: %v", err)
	}

	// TODO: handle Ctrl+C signal gracefully
}

func playlist(w http.ResponseWriter, r *http.Request) {
	keepAlive <- struct{}{}
	<-streamRunning

	http.ServeFile(w, r, playlistPath)
}
