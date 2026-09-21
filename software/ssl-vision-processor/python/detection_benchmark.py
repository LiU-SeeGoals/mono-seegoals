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
import math
from collections import defaultdict
from pathlib import Path
from statistics import fmean

from dataset import parser_test_data, threaded_field_iter, Dataset

if __name__ == '__main__':
    args = parser_test_data(argparse.ArgumentParser(prog='Vision recorder')).parse_args()

    types = {'ball'}
    for i in range(16):
        types.add('y' + str(i))
        types.add('b' + str(i))

    # [binary][dataset]
    frametimes = defaultdict(lambda: defaultdict(lambda: 0))
    frames = defaultdict(lambda: defaultdict(lambda: 0))
    # [binary][dataset][cam][video][type]
    truepositive_rates = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: 0)))))
    falsepositive_rates = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: 0)))))
    falsenegative_rates = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: 0)))))

    def consumer(dataset: Dataset):
        global types, truepositive_rates, falsepositive_rates, falsenegative_rates

        def get_detections(frame, as_set=True):
            objects = []

            if 'balls' in frame:
                objects.append('ball')
            if 'robots_yellow' in frame:
                for bot in frame['robots_yellow']:
                    objects.append('y' + str(bot['robot_id']))
            if 'robots_blue' in frame:
                for bot in frame['robots_blue']:
                    objects.append('b' + str(bot['robot_id']))

            if as_set:
                return set(objects)
            return objects

        for video in dataset.images():
            video: Path = video
            print(f"Processing {video}")

            detections = {}
            for records in video.parent.glob(video.stem + '.*.json'):
                binary = records.stem.split('.')[-1]
                with records.open('r') as file:
                    detections[binary] = json.load(file)

            for binary, d2 in detections.items():
                frames[binary][dataset.folder.parent] += len(d2)
                for detection in d2:
                    frametimes[binary][dataset.folder.parent] += float(detection['t_sent']) - float(detection['t_capture'])

            with_manual = 'manual' in detections.keys()
            video_frames = max(len(detection_list) for detection_list in detections.values())
            binaries = {binary for binary in detections.keys()}

            # Binary, Type
            truepositive = defaultdict(lambda: defaultdict(lambda: 0))
            falsepositive = defaultdict(lambda: defaultdict(lambda: 0))
            falsenegative = defaultdict(lambda: defaultdict(lambda: 0))

            if with_manual:
                for i in range(video_frames):
                    for binary in binaries:
                        visible: list = get_detections(detections['manual'][i], as_set=False)
                        detected: list = get_detections(detections[binary][i], as_set=False)

                        for type in detected:
                            if type in visible:
                                visible.remove(type)
                                truepositive[binary][type] += 1
                            else:
                                falsepositive[binary][type] += 1

                        for type in visible:
                            falsenegative[binary][type] += 1
            else:
                visibility = defaultdict(lambda: 0)
                for i in range(video_frames):
                    visible: set = {key for key, value in visibility.items() if value > 0}
                    detected: set = set()

                    for binary in binaries:
                        objects = get_detections(detections[binary][i])

                        for type in visible & objects:
                            truepositive[binary][type] += 1
                        for type in visible - objects:
                            falsenegative[binary][type] += 1
                        for type in objects - visible:
                            falsepositive[binary][type] += 1

                        detected.update(objects)

                    for type in types:
                        visibility[type] = min(max(visibility[type] + (1 if type in detected else -1), -15), 15)

            for binary in binaries:
                for type in types:
                    if truepositive[binary][type]:
                        truepositive_rates[binary][dataset.folder.parent][dataset.folder.name][video][type] = truepositive[binary][type]
                    if falsepositive[binary][type]:
                        falsepositive_rates[binary][dataset.folder.parent][dataset.folder.name][video][type] = falsepositive[binary][type]
                    if falsenegative[binary][type]:
                        falsenegative_rates[binary][dataset.folder.parent][dataset.folder.name][video][type] = falsenegative[binary][type]

    try:
        threaded_field_iter(args.data_folder, consumer, 1, field_filter=args.field)
    except KeyboardInterrupt:
        pass

    def dsum(d: dict, generator=lambda x: x, filter=None) -> float:
        return sum(generator(value) for key, value in d.items() if filter is None or filter == key)

    def nanmean(x):
        x = [i for i in x if i is not math.nan]
        if not x:
            return math.nan
        return fmean(x)

    def dictmean(d: dict, s: dict, dgenerator=lambda x, y: x / y, filter=None) -> float:
        return nanmean(
            value
            for value in (
                dgenerator(d[key], s[key])
                for key in {*s.keys(), *d.keys()}
                if filter is None or filter == key
            )
            if value is not math.nan
        )

    def detection_rate(s2, binary, dataset_filter=None, video_filter=None, object_filter=None):
        def camsum(cams):
            return dsum(
                cams,
                lambda videos: dsum(
                    videos,
                    lambda objects: dsum(
                        objects,
                        filter=object_filter
                    ),
                    video_filter
                )
            )

        def camavg(x, y):
            try:
                return camsum(x) / (camsum(x) + camsum(y))
            except ZeroDivisionError:
                return math.nan


        return dictmean(
            truepositive_rates[binary], s2[binary],
            camavg,
            dataset_filter
        )

    for binary in truepositive_rates.keys():
        print(f"--- {binary} ---")
        print(f"Total Recall {detection_rate(falsenegative_rates, binary): .4f} Precision {detection_rate(falsepositive_rates, binary): .4f}")

        img_recall = []
        img_precision = []
        video_recall = []
        video_precision = []
        for dataset in truepositive_rates[binary].keys():
            recall = detection_rate(falsenegative_rates, binary, dataset)
            precision = detection_rate(falsepositive_rates, binary, dataset)
            if len(list(dataset.glob('*/*.mp4'))) == 0:
                img_recall.append(recall)
                img_precision.append(precision)
            else:
                video_recall.append(recall)
                video_precision.append(precision)

            try:
                frametime = 1000 * frametimes[binary][dataset] / frames[binary][dataset]
            except:
                frametime = math.nan

            print(f"  Dataset {dataset.name: >11} Recall {recall: .4f} Precision {precision: .4f} Frametime {frametime: .2f}ms")

        print(f"Video Recall {nanmean(video_recall): .4f} Precision {nanmean(video_precision): .4f}")
        print(f"Image Recall {nanmean(img_recall): .4f} Precision {nanmean(img_precision): .4f}")

        min_video = None, 1.0
        for dataset in truepositive_rates[binary].keys():
            for camera in truepositive_rates[binary][dataset].keys():
                for video in truepositive_rates[binary][dataset][camera].keys():
                    rate = detection_rate(falsenegative_rates, binary, dataset, video)
                    if rate < min_video[1]:
                        min_video = video, rate
        print(f"Worst video {min_video[0]} Recall {min_video[1]: .4f}")

        min_type = None, 1.0
        for t in {t for cams in truepositive_rates[binary].values() for videos in cams.values() for types in videos.values() for t in types.keys()}:
            if t != 'false':
                rate = detection_rate(falsenegative_rates, binary, object_filter=t)
                if rate < min_type[1]:
                    min_type = t, rate
        print(f"Worst type {min_type[0]} Recall {min_type[1]: .4f}")

        print()

