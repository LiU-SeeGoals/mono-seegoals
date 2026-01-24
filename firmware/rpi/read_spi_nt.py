# /// script
# dependencies = [
#    "robotpy",
#    "protobuf",
#    "grpcio",
# ]
# ///

import ctypes
import os
import sys
import numpy as np
import ntcore
import time
from pathlib import Path
from google.protobuf.message import DecodeError

file_dir = Path(__file__).parent
proto_dir = os.path.abspath(file_dir / "build/generated")
sys.path.append(proto_dir)

import imu_pb2


def publishvec3(table):
    x = table.getFloatTopic("x").publish()
    y = table.getFloatTopic("y").publish()
    z = table.getFloatTopic("z").publish()
    dt = table.getFloatTopic("dt").publish()
    return (x, y, z, dt)


def setvec3(topics, x, y, z, dt):
    x_pub, y_pub, z_pub, dt_pub = topics

    x_pub.set(x)
    y_pub.set(y)
    z_pub.set(z)
    dt_pub.set(dt)


def publish_control_signal(table):
    ref = table.getFloatTopic("Reference").publish()
    control = table.getFloatTopic("Control").publish()
    error = table.getFloatTopic("Error").publish()
    output = table.getFloatTopic("Output").publish()

    return (ref, control, error, output)


def set_control_signal(topics, ref, control, error, output):
    ref_pub, control_pub, error_pub, output_pub = topics

    ref_pub.set(ref)
    control_pub.set(control)
    error_pub.set(error)
    output_pub.set(output)


def readSpi():
    lib_path = os.path.abspath(file_dir / "build/libspi.so")

    inst = ntcore.NetworkTableInstance.getDefault()
    inst.startClient4("Robot 1")
    inst.setServer("10.158.180.229")

    table = inst.getTable("Robot 1")

    imu_table = table.getSubTable("Imu")
    imu_topics = publishvec3(imu_table)

    state_table = table.getSubTable("State")
    state_topics = publishvec3(state_table)

    vision_table = table.getSubTable("Vision")
    vision_topics = publishvec3(vision_table)

    odom_table = table.getSubTable("Odometry")
    odom_topics = publishvec3(odom_table)

    motor_table = table.getSubTable("Motors")
    m1_table = motor_table.getSubTable("m1")
    m1_topics = publish_control_signal(m1_table)
    m2_table = motor_table.getSubTable("m2")
    m2_topics = publish_control_signal(m2_table)
    m3_table = motor_table.getSubTable("m3")
    m3_topics = publish_control_signal(m3_table)
    m4_table = motor_table.getSubTable("m4")
    m4_topics = publish_control_signal(m4_table)

    pos_table = table.getSubTable("Pos")
    pos_x_table = pos_table.getSubTable("pos_x")
    pos_x_topics = publish_control_signal(pos_x_table)
    pos_y_table = pos_table.getSubTable("pos_y")
    pos_y_topics = publish_control_signal(pos_y_table)
    pos_angle_table = pos_table.getSubTable("pos_angle")
    pos_angle_topics = publish_control_signal(pos_angle_table)

    spi = ctypes.CDLL(lib_path)
    dataSize = spi.getDataSize()
    failed = 0
    success = 0

    spi.spiOpen()
    imu_prev_ts = 0
    state_prev_ts = 0
    vision_prev_ts = 0
    odom_prev_ts = 0

    while True:
        spiBytes = (ctypes.c_uint8 * dataSize)()
        spi.spiRead(spiBytes)
        msg_size = int(bytes(spiBytes)[0])
        msg = imu_pb2.data_sample()
        try:
            msg.ParseFromString(bytes(spiBytes)[1 : msg_size + 1])
        except DecodeError as e:
            # print(f"Decode Error: {e}")
            failed += 1
            continue

        # print("Passed decode check")

        imu_dt = msg.gyro.timestamp - imu_prev_ts
        if msg.gyro.timestamp < 1 or imu_dt < 1:
            failed += 1
            continue

        imu_prev_ts = msg.gyro.timestamp

        setvec3(imu_topics, msg.gyro.x, msg.gyro.y, msg.gyro.z, imu_dt)

        state_dt = msg.state.timestamp - state_prev_ts
        state_prev_ts = msg.state.timestamp

        setvec3(state_topics, msg.state.x, msg.state.y, msg.state.z, state_dt)

        vision_dt = msg.vision.timestamp - vision_prev_ts
        vision_prev_ts = msg.vision.timestamp

        setvec3(vision_topics, msg.vision.x, msg.vision.y, msg.vision.z, vision_dt)

        odom_dt = msg.odometry.timestamp - odom_prev_ts
        odom_prev_ts = msg.odometry.timestamp

        setvec3(odom_topics, msg.odometry.x, msg.odometry.y, msg.odometry.z, odom_dt)

        set_control_signal(
            m1_topics, msg.m1.ref, msg.m1.control, msg.m1.error, msg.m1.output
        )
        set_control_signal(
            m2_topics, msg.m2.ref, msg.m2.control, msg.m2.error, msg.m2.output
        )
        set_control_signal(
            m3_topics, msg.m3.ref, msg.m3.control, msg.m3.error, msg.m3.output
        )
        set_control_signal(
            m4_topics, msg.m4.ref, msg.m4.control, msg.m4.error, msg.m4.output
        )

        set_control_signal(
            pos_x_topics,
            msg.pos_x.ref,
            msg.pos_x.control,
            msg.pos_x.error,
            msg.pos_x.output,
        )
        set_control_signal(
            pos_y_topics,
            msg.pos_y.ref,
            msg.pos_y.control,
            msg.pos_y.error,
            msg.pos_y.output,
        )
        set_control_signal(
            pos_angle_topics,
            msg.pos_angle.ref,
            msg.pos_angle.control,
            msg.pos_angle.error,
            msg.pos_angle.output,
        )

        print(1000 / dt)

    spi.spiClose()


def readFile():
    failed = 0
    success = 0
    prev_ts = 0
    inst = ntcore.NetworkTableInstance.getDefault()
    inst.startClient4("Robot 1")
    inst.setServer("10.158.180.229")

    table = inst.getTable("Robot 1")

    x_pub = table.getFloatTopic("x").publish()
    y_pub = table.getFloatTopic("y").publish()
    z_pub = table.getFloatTopic("z").publish()

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
            x_pub.set(msg.gyro.x)
            y_pub.set(msg.gyro.y)
            z_pub.set(msg.gyro.z)

            time.sleep(0.100)
            success += 1
            yield msg

    print(f"Failed {failed} out of {failed + success} packets")


if __name__ == "__main__":
    readSpi()
    # readFile()
