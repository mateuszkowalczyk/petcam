package streamer

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const ledBasePath = "/sys/class/leds"

// setLED writes a value (0 or 1) to the LED brightness file.
// If ledName is empty, this function does nothing.
// Logs a warning if the LED file doesn't exist.
func setLED(ledName string, value int) {
	if ledName == "" {
		return
	}

	ledPath := filepath.Join(ledBasePath, ledName, "brightness")
	content := fmt.Sprintf("%d\n", value)

	if _, err := os.Stat(ledPath); err != nil {
		slog.Warn("LED brightness file not found", "led", ledName, "path", ledPath)
		return
	}

	if err := os.WriteFile(ledPath, []byte(content), 0); err != nil {
		slog.Warn("failed to set LED brightness", "led", ledName, "value", value, "err", err)
	}
}

func turnOnLED(ledName string) {
	setLED(ledName, 1)
}

func turnOffLED(ledName string) {
	setLED(ledName, 0)
}
