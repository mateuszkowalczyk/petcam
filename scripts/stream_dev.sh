#!/bin/bash
set -euo pipefail

ffmpeg -hide_banner -f v4l2 -i /dev/video0 \
  -c:v libx264 -preset veryfast -tune zerolatency \
  -f hls \
  -hls_time 1 \
  -hls_list_size "${HLS_LIST_SIZE}" \
  -hls_flags delete_segments \
  -hls_base_url "${HLS_BASE_URL}" \
  "${PLAYLIST_PATH}"
