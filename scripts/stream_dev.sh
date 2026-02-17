#!/bin/bash
set -euo pipefail

# Detect OS and use appropriate camera input
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS - use AVFoundation (default camera)
  INPUT="-f avfoundation -framerate 30 -i default"
elif [[ "$OSTYPE" == "linux"* ]]; then
  # Linux - use V4L2
  INPUT="-f v4l2 -framerate 30 -i /dev/video0"
else
  echo "Unsupported OS: $OSTYPE" >&2
  exit 1
fi

ffmpeg -hide_banner $INPUT \
  -c:v libx264 -preset veryfast -tune zerolatency -pix_fmt yuv420p \
  -f hls \
  -hls_time 1 \
  -hls_list_size "${HLS_LIST_SIZE}" \
  -hls_delete_threshold 10 \
  -hls_flags delete_segments+program_date_time \
  -hls_base_url "${HLS_BASE_URL}" \
  "${PLAYLIST_PATH}"
