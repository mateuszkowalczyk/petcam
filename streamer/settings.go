package streamer

import "time"

type Settings struct {
	StreamPath   string
	PlaylistPath string
	HlsBaseURL   string

	InactivityTimeout time.Duration
	Command           string // Command to run, defaults to "ffmpeg" if empty
	LEDName           string // LED name in /sys/class/leds/ (e.g., "ACT"), empty to disable LED control
}
