#!/bin/sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cleanup() {
  kill $VID1_PID $VID2_PID $VID3_PID 2>/dev/null
}

trap cleanup INT TERM EXIT

cd $SCRIPT_DIR

"$SCRIPT_DIR/bin/vision" -s -c 3 &
VISION_PID=$!

"$SCRIPT_DIR/video-stream-to-file.py" 10 &
VID1_PID=$!

"$SCRIPT_DIR/video-stream-to-file.py" 11 &
VID2_PID=$!

"$SCRIPT_DIR/video-stream-to-file.py" 12 &
VID3_PID=$!

wait $VISION_PID
