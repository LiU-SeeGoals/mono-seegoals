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
import sys

import yaml
from google.protobuf.json_format import MessageToDict

from binary import parser_binary, run_binary
from dataset import threaded_field_iter, parser_test_data
from record import thread_local_ip
from visionsocket import VisionRecorder

if __name__ == '__main__':
    parser = parser_test_data(parser_binary(argparse.ArgumentParser(prog='Vision geometry recorder')))
    parser.add_argument('--scenes_per_field', default=None, type=int, help='Amount of scenes per field to process')
    args = parser.parse_args()

    def consumer(dataset):
        print(f"Recording {dataset} geometry")

        recorder = VisionRecorder(vision_ip=thread_local_ip())

        reference_geometry = dataset.reference_geometry
        del reference_geometry.geometry.calib[:]

        run_binary(args.binary, recorder, dataset, dataset.field, geometry=reference_geometry, stdoutconsumer=lambda line: print(line, end=''))

        geometries = recorder.subfield('geometry')
        geometries = [geometry for geometry in geometries if len(geometry.calib) > 0]

        if geometries:
            with (dataset.folder / ('geometry.' + args.binary.name + '.yml')).open('w') as file:
                yaml.dump(MessageToDict(geometries[-1], preserving_proto_field_name=True), file)
        else:
            print(f"No calibration received!", file=sys.stderr)

    threaded_field_iter(args.data_folder, consumer, 1, field_filter=args.field)

