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
import math
from collections import defaultdict

from binary import parser_binary, run_binary
from dataset import threaded_field_iter, parser_test_data
from record import thread_local_ip
from visionsocket import VisionRecorder


class AvgValue:

    def __init__(self, fourdigits=False):
        self.value = 0.0
        self.count = 0
        self.fourdigits = fourdigits

    def __iadd__(self, value):
        if math.isnan(value):
            return self

        self.value += value
        self.count += 1
        return self

    def __str__(self):
        try:
            value = self.value / self.count
            return f"{value: .4f}" if self.fourdigits else f"{value: .2f}"
        except ZeroDivisionError:
            return " nan "


if __name__ == '__main__':
    parser = parser_test_data(parser_binary(argparse.ArgumentParser(prog='Vision blob benchmark'), default='blob_benchmark'))
    parser.add_argument('--scenes_per_field', default=None, type=int, help='Amount of scenes per field to process')
    args = parser.parse_args()

    blobs = defaultdict(int)
    errorSum = defaultdict(float)
    sqErrorSum = defaultdict(float)
    fieldScale = defaultdict(float)
    processingTime = defaultdict(float)

    frames = defaultdict(int)
    worstBlobSum = defaultdict(float)
    percentileSum = defaultdict(float)

    balls = defaultdict(int)
    ballErrorSum = defaultdict(float)
    ballSqErrorSum = defaultdict(float)

    bots = defaultdict(int)
    botErrorSum = defaultdict(float)
    botSqErrorSum = defaultdict(float)

    def consumer(dataset):
        print(f"Recording {dataset} blob benchmark")

        recorder = VisionRecorder(vision_ip=thread_local_ip())

        def stdoutprocessor(line: str):
            if not line.startswith("[BlobMachine]"):
                #print(line, end='')
                return

            key = dataset.folder.parent.name
            split = line[:-1].split(' ')
            frames[key] += int(split[1])
            blobs[key] += int(split[2])
            errorSum[key] += float(split[3])
            sqErrorSum[key] += float(split[4])
            worstBlobSum[key] += float(split[5])
            percentileSum[key] += float(split[6])
            balls[key] += int(split[7])
            ballErrorSum[key] += float(split[8])
            ballSqErrorSum[key] += float(split[9])
            bots[key] += int(split[10])
            botErrorSum[key] += float(split[11])
            botSqErrorSum[key] += float(split[12])
            fieldScale[key] += float(split[13])
            processingTime[key] += float(split[14])

        for video, _ in zip(dataset.images(), range(args.scenes_per_field if args.scenes_per_field else 1000000)):
            print(f"Processing {video}")
            run_binary(args.binary, recorder, dataset, video, stdoutconsumer=stdoutprocessor)  # , ground_truth=video.with_suffix('.vision_processor.json')

    threaded_field_iter(args.data_folder, consumer, field_filter=args.field)  #, workers=1

    def errorStddev(error, sqError, amount):
        try:
            return error / amount, math.sqrt(amount * sqError - error ** 2) / amount
        except:
            return math.nan, math.nan

    totalError = AvgValue()
    totalStddev = AvgValue()
    totalBallError = AvgValue()
    totalBallStddev = AvgValue()
    totalBotError = AvgValue()
    totalBotStddev = AvgValue()
    totalPpr = AvgValue(True)
    totalEfsr = AvgValue(True)
    totalFrametime = AvgValue()
    for dataset, b in blobs.items():
        error, stddev = errorStddev(errorSum[dataset], sqErrorSum[dataset], b)
        ballError, ballStddev = errorStddev(ballErrorSum[dataset], ballSqErrorSum[dataset], balls[dataset])
        botError, botStddev = errorStddev(botErrorSum[dataset], botSqErrorSum[dataset], bots[dataset])
        try:
            ppr = worstBlobSum[dataset] / (abs(worstBlobSum[dataset]) + abs(percentileSum[dataset]))
        except ZeroDivisionError:
            ppr = math.nan
        try:
            efsr = errorSum[dataset] / fieldScale[dataset]
        except ZeroDivisionError:
            efsr = math.nan
        frametime = processingTime[dataset] / frames[dataset] * 1000  # ms

        print(f"  {dataset: >11} blobs: {error: .2f}±{stddev: .2f} balls: {ballError: .2f}±{ballStddev: .2f} bots: {botError: .2f}±{botStddev: .2f} PPR {ppr: .4f} EFSR {efsr: .4f} Time {frametime: .2f}")

        totalError += error
        totalStddev += stddev
        totalBallError += ballError
        totalBallStddev += ballStddev
        totalBotError += botError
        totalBotStddev += botStddev
        totalPpr += ppr
        totalEfsr += efsr
        totalFrametime += frametime

    print(f"Total blobs: {totalError}±{totalStddev} balls: {totalBallError}±{totalBallStddev} bots: {totalBotError}±{totalBotStddev} PPR {totalPpr} EFSR {totalEfsr} Time {totalFrametime}")
