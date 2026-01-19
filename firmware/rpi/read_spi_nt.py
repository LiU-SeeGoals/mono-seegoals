import ctypes
import os
import sys
import numpy as np
import ntcore
from pathlib import Path
from google.protobuf.message import DecodeError

file_dir = Path(__file__).parent
proto_dir = os.path.abspath(file_dir / "build/generated")
sys.path.append(proto_dir)

import imu_pb2


def readSpi():
    lib_path = os.path.abspath(file_dir / "build/libspi.so")

    inst = ntcore.NetworkTableInstance.getDefault()
    inst.setServer("")

    table = inst.getTable("Robot 1")

    x_pub = table.getFloatTopic("x").publish()
    y_pub = table.getFloatTopic("y").publish()
    z_pub = table.getFloatTopic("z").publish()

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
        msg = imu_pb2.data_sample()
        try:
            msg.ParseFromString(bytes(spiBytes)[1:])
        except DecodeError as e:
            failed += 1
            continue

        dt = msg.gyro.timestamp - prev_ts
        if msg.gyro.timestamp < 1 or dt < 1:
            failed += 1
            continue

        prev_ts = msg.gyro.timestamp

        if dt < 1 or msg.gyro.timestamp < 1:
            failed += 1
            continue

        x_pub.set(msg.gyro.x)
        y_pub.set(msg.gyro.y)
        z_pub.set(msg.gyro.z)

        print(1000 / dt)

    spi.spiClose()


def readFile():
    failed = 0
    success = 0
    prev_ts = 0

    with open(file_dir / "build/output.txt") as f:
        for protoBits in f.readlines():
            msg = imu_pb2.data_sample()
            b = bytes(
                int(protoBits[i : i + 8], 2)
                for i in range(0, len(protoBits.strip()), 8)
            )

            try:
                msg.ParseFromString(b)
            except DecodeError as e:
                failed += 1
                continue

            success += 1
            yield msg

    print(f"Failed {failed} out of {failed + success} packets")


if __name__ == "__main__":
    readSpi()
    # readFile()
