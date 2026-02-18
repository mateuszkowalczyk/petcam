package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mateuszkowalczyk/petcam/streamer"
)

const (
	basePath    = "/dev/shm"
	hlsBaseURL  = "/segments/"
	defaultPort = "8080"
)

var (
	port         = flag.String("port", defaultPort, "HTTP server port")
	ledName      = flag.String("led", "", "LED name in /sys/class/leds/ (e.g., 'ACT'), empty to disable LED control")
	streamPath   = filepath.Join(basePath, "stream")
	playlistPath = filepath.Join(streamPath, "playlist.m3u8")
)

func main() {
	flag.Parse()
	address := ":" + *port

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	fmt.Println("this is petcam 🐶🐱")

	streamer := streamer.NewStreamer(
		streamer.Settings{
			StreamPath:        streamPath,
			PlaylistPath:      playlistPath,
			HlsBaseURL:        hlsBaseURL,
			InactivityTimeout: 30 * time.Second,
			Command:           filepath.Join("scripts", "stream.sh"),
			LEDName:           *ledName,
		})
	streamer.Start()

	go func() {
		if err := streamer.Wait(); err != nil {
			slog.Error("streamer error", "err", err)
			os.Exit(1)
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
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	fmt.Printf("listening on: %v\n", address)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("shutting down...")
	streamer.Stop()
}
