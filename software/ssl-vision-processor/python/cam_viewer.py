#!/usr/bin/env python3

#    Copyright 2024 Felix Weinmann
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

import os
import argparse
import threading
import subprocess


directory = os.path.dirname(__file__)


def mpv(path):
    while True:
        subprocess.run([
            "mpv",
            path,
            "--profile=low-latency",
            "--untimed",
            "--no-cache-pause",
            "--no-cache",
            "--demuxer-lavf-o=reorder_queue_size=0",
            f"--input-conf={directory}/camviewer.conf",
            "--no-osc"
        ])


if __name__ == '__main__':
    parser = argparse.ArgumentParser(prog='Cam Viewer')
    parser.add_argument('--stream_base_ip', default='224.5.23.100', help='Multicast IP address of the vision')
    parser.add_argument('--stream_port', type=int, default=10100, help='Multicast port of the vision')
    parser.add_argument('--cameras', type=int, default=1, help='Amount of cameras')
    args = parser.parse_args()

    for cam_id in range(args.cameras):
        ip = [int(segment) for segment in args.stream_base_ip.split('.')]
        ip[3] += cam_id
        ip = '.'.join([str(segment) for segment in ip])
        sdp_filename = f"{directory}/cam{cam_id}.sdp"
        with open(sdp_filename, "wt") as sdpfile:
            # o=[...] vision_ip  technically not correct (originator)
            sdpfile.write(f"""v=0
o=- 0 0 IN IP4 {ip}
s=Cam{cam_id}
c=IN IP4 {ip}
t=0 0
a=tool:libavformat 60.3.100
m=video {args.stream_port} RTP/AVP 96
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1""")
        threading.Thread(target=mpv, name=sdp_filename, args=(sdp_filename, )).start()
