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
	address    = ":8080"
)

var (
	streamPath   = path.Join(basePath, "stream")
	playlistPath = path.Join(streamPath, "playlist.m3u8")
)

func main() {
	fmt.Println("this is petcam 🐶🐱")

	streamer := streamer.NewStreamer(streamPath, playlistPath, hlsBaseURL)
	streamer.Start()
	defer streamer.Stop()

	http.Handle(hlsBaseURL, http.StripPrefix(hlsBaseURL, http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		streamer.EnsureStreaming()
		http.ServeFile(w, r, playlistPath)
	})

	fmt.Printf("listening on: %v\n", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("cannot start server: %v\n", err)
	}

	// TODO: handle Ctrl+C signal gracefully
}
