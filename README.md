# Petcam 🐶🐱

Keep an eye on your pets while you're away. A lightweight streaming server for Raspberry Pi cameras.

## Overview

Petcam captures video from a CSI camera (Raspberry Pi Camera Module) and streams it over HTTP using HLS (HTTP Live Streaming). It automatically starts streaming when someone connects and stops after a period of inactivity. Can run as a standalone process or as a systemd service for automatic startup and crash recovery.

## Features

- **On-demand streaming**: Stream starts only when a viewer connects
- **Auto-stop**: Automatically stops after 30 seconds of inactivity
- **HLS delivery**: Compatible with any browser or HLS player
- **Hardware acceleration**: Uses `rpicam-vid` with built-in hardware acceleration on Pi
- **Systemd service**: Includes systemd service template for automatic startup at boot and crash recovery
- **LED indicator**: Built-in LED lights up when camera is active (Raspberry Pi ACT LED by default)

## Requirements

- Raspberry Pi: `rpicam-vid` (libcamera), FFmpeg
- Local dev: Go 1.25+, FFmpeg, Webcam (V4L2 on Linux, AVFoundation on macOS)

## Tested Hardware

- Raspberry Pi Zero 2 W

## Quick Start

### Development (Linux or macOS)

Builds for local development using your webcam (V4L2 on Linux, AVFoundation on macOS) and starts the server:

```bash
make run
```

This compiles the binary and immediately starts the server on port 8080.

### Build and Deploy to Raspberry Pi

Prerequisites on the Pi:

```bash
sudo apt install ffmpeg
```

Deploys to `mk@rpi` by default:

```bash
# Builds and deploys to rpi:~/petcam/
make
```

Deploy to a different host or with a different LED:

```bash
make REMOTE_HOST=raspberrypi.local REMOTE_USER=pi
make LED_NAME=led0  # Use a different LED (default: ACT)
```

After deploying, SSH into the Pi and run the install script (one-time setup):

```bash
~/petcam/scripts/install.sh
```

This script will:

1. Disable default built-in LED behavior in config.txt file
2. Create service that sets up LED permissions on system startup (so petcam can control the LED without root)
3. Install and enable the petcam as a service
4. Reboot Pi to apply changes from the config.txt file

The petcam service will start automatically after reboot.

**View logs:**

```bash
journalctl -u petcam -f              # Live logs
journalctl -u petcam --since today   # Today's logs
journalctl -u petcam -n 50           # Last 50 lines
```

**Manage the service:**

```bash
sudo systemctl stop petcam             # Stop service
sudo systemctl restart petcam          # Restart service
sudo systemctl disable petcam          # Disable auto-start
~/petcam/scripts/uninstall.sh          # Remove services completely
```

### LED Indicator (Raspberry Pi)

The Raspberry Pi ACT LED can indicate when the camera is streaming. The install script automatically sets this up. The LED lights up when streaming starts and turns off when it stops.

To run manually without LED control:

````bash
~/petcam/petcam             # LED control disabled (default)

**Using a different LED:**

To use a different LED (from `/sys/class/leds/`):

```bash
make LED_NAME=led0
````

The Makefile will automatically replace `ACT` with your LED name in both the service file and the LED permissions script.

For LEDs other than ACT, you may need to manually edit the template to remove the `dtparam=act_led_trigger=none` line from `install.sh.template` before building.

## Usage

Once running, open `http://localhost:8080/` to get the HLS playlist, or use with any HLS player:

```bash
ffplay http://localhost:8080/
```

## Remote Access

The easy and secure way to access the camera from outside your network is using **Tailscale**:

1. Install Tailscale on the Pi: `curl -fsSL https://tailscale.com/install.sh | sh`
2. Start Tailscale: `sudo tailscale up`
3. Install Tailscale on your phone/laptop
4. Access using the Pi's Tailscale machine name or IP: `http://rpi:8080` or `http://<tailscale-ip>:8080`

Benefits: No port forwarding, automatic encryption, works behind NAT.

## Commands

| Command              | Description                                              |
| -------------------- | -------------------------------------------------------- |
| `make run`           | Build and run locally                                    |
| `make dev`           | Build for local development                              |
| `make`               | Build and deploy (binary + service file) to Raspberry Pi |
| `make LED_NAME=led0` | Deploy using a different LED (default: ACT)              |
| `make test`          | Run tests with race detection                            |
| `make clean`         | Remove build artifacts                                   |

## Project Structure

```
├── main.go              # HTTP server
├── streamer/            # Streaming logic
│   ├── streamer.go      # Stream lifecycle management
│   ├── process.go       # FFmpeg wrapper
│   ├── settings.go      # Configuration struct
│   └── led.go           # LED control
├── scripts/                               # Streaming shell scripts and systemd service
│   ├── stream_pi.sh                       # Raspberry Pi streaming
│   ├── stream_dev.sh                      # Development streaming
│   ├── install.sh.template                # Installation script template
│   ├── petcam.service.template            # Systemd service template
└── Makefile             # Build automation
```

## License

MIT
