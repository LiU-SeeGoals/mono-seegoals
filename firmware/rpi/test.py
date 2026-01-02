import ctypes
import os
import sys
import numpy as np
from pathlib import Path
from google.protobuf.message import DecodeError

file_dir = Path(__file__).parent
proto_dir = os.path.abspath(file_dir / "build/generated")
lib_path = os.path.abspath(file_dir / "build/libspi.so")
sys.path.append(proto_dir)

import imu_pb2

def readSpi():
    spi = ctypes.CDLL(lib_path)
    dataSize = spi.getDataSize()
    failed = 0
    success = 0

    spi.spiOpen()
    prev_ts = 0

    while True:
        spiBytes = (ctypes.c_uint8 * dataSize)()
        spi.spiRead(spiBytes)
        msg_size = int(bytes(spiBytes)[0])
        msg = imu_pb2.ImuSample()
        try:
            msg.ParseFromString(bytes(spiBytes)[1:])
        except DecodeError as e:
            failed += 1
            continue

        dt = msg.imu_ts - prev_ts
        if msg.imu_ts < 1 or dt < 1:
            failed += 1
            continue

        prev_ts = msg.imu_ts

        if dt < 1 or msg.imu_ts < 1:
            failed += 1
            continue

        print(1000/dt)

    spi.spiClose()

def readFile():
    failed = 0
    success = 0
    prev_ts = 0

    with open(file_dir/"build/output.txt") as f:
        for protoBits in f.readlines():
            msg = imu_pb2.ImuSample()
            b = bytes(int(protoBits[i:i+8], 2) for i in range(0, len(protoBits.strip()), 8))
            print("====")
            print(protoBits)
            print("".join(f"{byte:08b}" for byte in b))
            print("".join(f"{byte} " for byte in b))

            try:
                msg.ParseFromString(b)
            except DecodeError as e:
                failed += 1
                print("Failed decoding byte 1")
                continue

            print(msg.imu_z)

            dt = msg.imu_ts - prev_ts
            if msg.imu_ts < 1 or dt < 1:
                failed += 1
                #print("Failed decoding byte 2")
                continue

            prev_ts = msg.imu_ts

            if dt < 1 or msg.imu_ts < 1:
                failed += 1
                #print("Failed decoding byte 3")
                continue

            print(1000/dt)

if __name__ == "__main__":
    # readSpi()
    readFile()
