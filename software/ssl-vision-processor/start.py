#!/usr/bin/env python3
"""Run the SeeGoals geometry publisher and three camera processors together."""

import argparse
import ctypes
import ctypes.util
import os
from pathlib import Path
import shutil
import signal
import socket
import subprocess
import sys
import time


ROOT = Path(__file__).resolve().parent


def commands(root=ROOT):
    jobs = [("geometry publisher", [sys.executable, "-u", "python/geom_publisher.py",
                                     "geometry-seagoals.yml"])]
    for camera in range(3):
        config = root / f"config-camera-{camera}.local.yml"
        if not config.is_file():
            config = root / f"config-camera-{camera}.yml"
        jobs.append((f"camera {camera}", [str(root / "build/vision_processor"), config.name]))
    return jobs


def check(root=ROOT):
    binary = root / "build/vision_processor"
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise RuntimeError(
            f"Missing executable: {binary}\nRun ./setup.sh in {root} first; "
            "answer n to installing systemd services."
        )
    for path in [root / "python/geom_publisher.py", root / "geometry-seagoals.yml",
                 root / "robot-heights.yml", *(root / job[1][-1] for job in commands(root)[1:])]:
        if not path.is_file():
            raise RuntimeError(f"Missing configuration or script: {path}")
    if not shutil.which("protoc"):
        raise RuntimeError("Missing protoc. Install the dependencies with ./setup.sh.")
    try:
        import yaml  # noqa: F401
        import google.protobuf  # noqa: F401
    except ImportError as error:
        raise RuntimeError("Install python3-yaml and python3-protobuf with ./setup.sh.") from error

    library = ctypes.util.find_library("OpenCL")
    if not library:
        raise RuntimeError("No OpenCL runtime found. Install an OpenCL driver with ./setup.sh.")
    opencl = ctypes.CDLL(library)
    count = ctypes.c_uint()
    if opencl.clGetPlatformIDs(0, None, ctypes.byref(count)) != 0 or count.value == 0:
        raise RuntimeError("No OpenCL platform found. Check the OpenCL driver and GPU permissions.")


def notify_ready():
    address = os.environ.get("NOTIFY_SOCKET")
    if address:
        if address.startswith("@"):
            address = "\0" + address[1:]
        with socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM) as sock:
            sock.connect(address)
            sock.sendall(b"READY=1\nSTATUS=Geometry publisher and three camera processes started")


def supervise(jobs, cwd=ROOT, startup_grace=2.0):
    stopping = False

    def stop(_signum, _frame):
        nonlocal stopping
        stopping = True

    previous = {sig: signal.signal(sig, stop) for sig in (signal.SIGINT, signal.SIGTERM)}
    children = []
    try:
        for name, argv in jobs:
            if stopping:
                break
            print(f"Starting {name}", flush=True)
            children.append((name, subprocess.Popen(argv, cwd=cwd)))

        started = time.monotonic()
        ready = False
        while not stopping:
            for name, child in children:
                status = child.poll()
                if status is not None:
                    raise RuntimeError(f"{name} exited with status {status}; stopping vision.")
            if not ready and time.monotonic() - started >= startup_grace:
                notify_ready()
                print("All vision processes started.", flush=True)
                ready = True
            time.sleep(0.1)
        return 0
    finally:
        for _, child in children:
            if child.poll() is None:
                child.terminate()
        deadline = time.monotonic() + 5
        for _, child in children:
            try:
                child.wait(timeout=max(0, deadline - time.monotonic()))
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
        for sig, handler in previous.items():
            signal.signal(sig, handler)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check prerequisites without starting vision")
    args = parser.parse_args()
    try:
        check()
        if args.check:
            print("Vision processor prerequisites are available.")
            return 0
        return supervise(commands())
    except (OSError, RuntimeError) as error:
        print(f"Vision processor: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
