package streamer

import "time"

type Settings struct {
	StreamPath   string
	PlaylistPath string
	HlsBaseURL   string

	InactivityTimeout time.Duration
	Command           string // Command to run, defaults to "ffmpeg" if empty
}
