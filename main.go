package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mateuszkowalczyk/petcam/streamer"
)

const (
	basePath   = "/dev/shm"
	hlsBaseURL = "/segments/"
	address    = ":8080"
)

var (
	streamPath   = filepath.Join(basePath, "stream")
	playlistPath = filepath.Join(streamPath, "playlist.m3u8")
)

func main() {
	fmt.Println("this is petcam 🐶🐱")

	streamer := streamer.NewStreamer(
		streamer.Settings{
			StreamPath:        streamPath,
			PlaylistPath:      playlistPath,
			HlsBaseURL:        hlsBaseURL,
			InactivityTimeout: 30 * time.Second,
			Command:           filepath.Join("scripts", "stream.sh"),
		})
	streamer.Start()

	go func() {
		if err := streamer.Wait(); err != nil {
			log.Fatalf("streamer error: %v\n", err)
		}
	}()

	http.Handle(hlsBaseURL, http.StripPrefix(hlsBaseURL, http.FileServer(http.Dir(streamPath))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		streamer.EnsureStreaming()
		http.ServeFile(w, r, playlistPath)
	})

	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			streamer.Stop()
			log.Fatalf("server error: %v\n", err)
		}
	}()
	fmt.Printf("listening on: %v\n", address)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("shutting down...")
	streamer.Stop()
}
