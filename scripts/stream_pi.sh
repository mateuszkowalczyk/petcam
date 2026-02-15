#!/bin/bash
set -euo pipefail

rpicam-vid -t 0 --width 1920 --height 1080 --inline --verbose 0 -o - |
  ffmpeg -hide_banner -i pipe:0 -c:v copy -f hls \
    -hls_time 1 \
    -hls_list_size "${HLS_LIST_SIZE}" \
    -hls_flags delete_segments \
    -hls_base_url "${HLS_BASE_URL}" \
    "${PLAYLIST_PATH}"
