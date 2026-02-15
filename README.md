# Petcam 🐶🐱

Lightweight HLS streaming server for Raspberry Pi cameras.

## Overview

Petcam captures video from a CSI camera (Raspberry Pi Camera Module) and streams it over HTTP using HLS (HTTP Live Streaming). It automatically starts streaming when someone connects and stops after a period of inactivity.

## Features

- **On-demand streaming**: Stream starts only when a viewer connects
- **Auto-stop**: Automatically stops after 30 seconds of inactivity
- **HLS delivery**: Compatible with any browser or HLS player
- **Hardware acceleration**: Uses `rpicam-vid` with built-in hardware acceleration on Pi

## Requirements

- Raspberry Pi: `rpicam-vid` (libcamera), FFmpeg
- Local dev: Go 1.25+, FFmpeg, Webcam (V4L2 on Linux, AVFoundation on macOS)

## Quick Start

### Development (Linux or macOS)

```bash
make run
```

### Deploy to Raspberry Pi

Prerequisites on the Pi:

```bash
sudo apt install ffmpeg
```

Deploys to `mk@rpi` by default:

```bash
make                    # Builds and deploys to rpi:~/petcam/
```

Deploy to a different host:

```bash
make REMOTE_HOST=raspberrypi.local REMOTE_USER=pi
```

## Usage

Once running, open `http://localhost:8080/` to get the HLS playlist, or use with any HLS player:

```bash
ffplay http://localhost:8080/
```

## Remote Access

The easiest and most secure way to access the camera from outside your network is using **Tailscale**:

1. Install Tailscale on the Pi: `curl -fsSL https://tailscale.com/install.sh | sh`
2. Start Tailscale: `sudo tailscale up`
3. Install Tailscale on your phone/laptop
4. Access using the Pi's Tailscale machine name or IP: `http://rpi:8080` or `http://<tailscale-ip>:8080`

Benefits: No port forwarding, automatic encryption, works behind NAT.

## Commands

| Command      | Description                      |
| ------------ | -------------------------------- |
| `make run`   | Build and run locally            |
| `make dev`   | Build for local development      |
| `make`       | Build and deploy to Raspberry Pi |
| `make test`  | Run tests with race detection    |
| `make clean` | Remove build artifacts           |

## Project Structure

```
├── main.go              # HTTP server
├── streamer/            # Streaming logic
│   ├── streamer.go      # Stream lifecycle management
│   ├── process.go       # FFmpeg wrapper
│   └── settings.go      # Configuration
├── scripts/             # Streaming shell scripts
│   ├── stream_pi.sh     # Raspberry Pi streaming
│   └── stream_dev.sh    # Development streaming
└── Makefile             # Build automation
```

## License

MIT
