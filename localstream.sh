#!/bin/sh
mkdir -p /dev/shm/stream && ffmpeg -f v4l2 -i /dev/video0 -c:v libx264 -preset veryfast -tune zerolatency -f hls -hls_time 1 -hls_list_size 3 -hls_flags delete_segments /dev/shm/stream/stream.m3u8
