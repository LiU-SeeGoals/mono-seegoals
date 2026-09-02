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

import argparse
import json
import sys
import threading

import cv2
import yaml

from binary import parser_binary, run_binary
from dataset import threaded_field_iter, parser_test_data
from visionsocket import VisionRecorder


_thread_counter = 1
_thread_ip = threading.local()
_thread_lock = threading.RLock()

def thread_local_ip():
    if not hasattr(_thread_ip, 'ip'):
        global _thread_counter

        with _thread_lock:
            _thread_ip.ip = str(_thread_counter)
            _thread_counter += 1

    return '224.83.83.' + _thread_ip.ip


if __name__ == '__main__':
    parser = parser_test_data(parser_binary(argparse.ArgumentParser(prog='Vision recorder')))
    parser.add_argument('--scenes_per_field', default=None, type=int, help='Amount of scenes per field to process')
    parser.add_argument('--out', default=None, help='Output name')
    args = parser.parse_args()

    if not args.out:
        args.out = args.binary.name

    def consumer(dataset):
        recorder = VisionRecorder(vision_ip=thread_local_ip())

        for video, _ in zip(dataset.images(), range(args.scenes_per_field if args.scenes_per_field else 1000000)):
            print(f"Recording {video}")

            if video.suffix == '.mp4':
                # https://stackoverflow.com/questions/25359288/how-to-know-total-number-of-frame-in-a-file-with-cv2-in-python
                capture = cv2.VideoCapture(str(video))
                frames = int(capture.get(cv2.CAP_PROP_FRAME_COUNT))
                upscale = int(capture.get(cv2.CAP_PROP_FRAME_WIDTH)) == 1224
            else:
                frames = 1
                upscale = False

            detections = []

            while len(detections) != frames:
                run_binary(args.binary, recorder, dataset, video, upscale=upscale)

                detections = recorder.dict_subfield('detection')

                if len(detections) != frames:
                    print(f"{video}: Detection size mismatch: Expected {frames} Vision {len(detections)}, repeating", file=sys.stderr)

            with video.with_suffix('.' + args.out + '.json').open('w') as file:
                json.dump(detections, file)

    threaded_field_iter(args.data_folder, consumer, field_filter=args.field)
