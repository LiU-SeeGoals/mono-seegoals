#!/usr/bin/env python3
"""Tune an Axis camera for SSL robot detection.

The tuner changes only parameters reported by the camera as available and
writable.  It scores each setting using SSL detection packets for one camera.
Run it from the ssl-vision-processor directory, for example:

    python3 python/axis_autotune.py --config config-camera-2.local.yml \
        --expected-robots y0,y1,b0,b1 --duration 12 --settle 2

The robots must move through the camera's field of view during each trial.
Use --probe first to inspect the camera capabilities without changing them.
"""

from __future__ import annotations

import argparse
import getpass
import itertools
import json
import logging
import os
from pathlib import Path
import socket
import struct
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from urllib.parse import urlencode, unquote, urlsplit, urlunsplit
from urllib.request import (
    HTTPBasicAuthHandler,
    HTTPDigestAuthHandler,
    HTTPPasswordMgrWithDefaultRealm,
    Request,
    build_opener,
)
from urllib.error import HTTPError, URLError
from xml.etree import ElementTree

import yaml


LOG = logging.getLogger("axis_autotune")
ROOT = Path(__file__).resolve().parents[1]


def _strip_namespace(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


@dataclass(frozen=True)
class AxisParameter:
    """One parameter returned by VAPIX listdefinitions."""

    path: str
    value: str
    writable: bool
    type_name: str = ""
    minimum: float | None = None
    maximum: float | None = None
    enum_values: tuple[str, ...] = ()


class AxisClient:
    """Small VAPIX client using HTTP Basic or Digest authentication."""

    def __init__(self, stream_url: str, username: str | None, password: str | None, timeout: float) -> None:
        parsed = urlsplit(stream_url)
        if parsed.scheme not in {"http", "https"} or not parsed.hostname:
            raise ValueError("camera.path must be an http:// or https:// URL")
        self.host = parsed.hostname
        self.base_url = urlunsplit((parsed.scheme, parsed.netloc.rsplit("@", 1)[-1], "", "", ""))
        self.username = unquote(username or parsed.username or "")
        self.password = unquote(password if password is not None else (parsed.password or ""))
        self.timeout = timeout
        self._param_endpoint: str | None = None

        if not self.username:
            self.username = input(f"Axis username for {self.host}: ").strip()
        if password is None and not parsed.password:
            self.password = getpass.getpass(f"Axis password for {self.username}@{self.host}: ")

        manager = HTTPPasswordMgrWithDefaultRealm()
        manager.add_password(None, self.base_url, self.username, self.password)
        self._opener = build_opener(
            HTTPDigestAuthHandler(manager),
            HTTPBasicAuthHandler(manager),
        )

    def _request(self, path: str, query: dict[str, str]) -> tuple[int, str]:
        url = self.base_url + path + "?" + urlencode(query)
        request = Request(url, headers={"Accept": "text/plain, text/xml"})
        try:
            with self._opener.open(request, timeout=self.timeout) as response:
                return response.status, response.read().decode("utf-8", errors="replace")
        except HTTPError as error:
            body = error.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"Axis API returned HTTP {error.code}: {body[:240]}") from error
        except URLError as error:
            raise RuntimeError(f"cannot reach Axis camera {self.host}: {error.reason}") from error

    def _endpoint(self) -> str:
        if self._param_endpoint is not None:
            return self._param_endpoint
        errors: list[str] = []
        for endpoint in ("/axis-cgi/param.cgi", "/axis-cgi/admin/param.cgi"):
            for group in ("Properties.API.HTTP.Version", "Properties.System"):
                try:
                    status, body = self._request(endpoint, {"action": "list", "group": group})
                    if status == 200 and not body.lstrip().startswith("# Error"):
                        self._param_endpoint = endpoint
                        return endpoint
                    errors.append(f"{endpoint} ({group}): {body[:120]}")
                except RuntimeError as error:
                    errors.append(f"{endpoint} ({group}): {error}")
        raise RuntimeError("Axis parameter API is unavailable; " + "; ".join(errors))

    @staticmethod
    def _normalise_path(path: str) -> str:
        return path.removeprefix("root.")

    def list_parameters(self, group: str) -> dict[str, AxisParameter]:
        endpoint = self._endpoint()
        status, body = self._request(
            endpoint,
            {"action": "listdefinitions", "listformat": "xmlschema", "group": group},
        )
        if status != 200 or body.lstrip().startswith("# Error"):
            raise RuntimeError(f"Axis parameter definition request failed: {body[:300]}")
        try:
            document = ElementTree.fromstring(body)
        except ElementTree.ParseError as error:
            raise RuntimeError(f"Axis returned non-XML parameter definitions: {body[:300]}") from error

        parameters: dict[str, AxisParameter] = {}

        def walk(node: ElementTree.Element, groups: list[str]) -> None:
            group_name = node.attrib.get("name") if _strip_namespace(node.tag) == "group" else None
            next_groups = groups + ([group_name] if group_name else [])
            if _strip_namespace(node.tag) == "parameter":
                name = node.attrib.get("name", "")
                path = self._normalise_path(".".join(next_groups + [name]))
                raw_value = node.attrib.get("value", "")
                security = node.attrib.get("securityLevel", "")
                writable = len(security) >= 1 and security[-1] in {"4", "6", "7"}
                type_node = next((child for child in node if _strip_namespace(child.tag) == "type"), None)
                type_name = ""
                minimum = maximum = None
                enum_values: list[str] = []
                if type_node is not None:
                    for child in type_node.iter():
                        child_name = _strip_namespace(child.tag)
                        if child_name in {"int", "float"}:
                            type_name = child_name
                            try:
                                minimum = float(child.attrib["min"]) if "min" in child.attrib else None
                                maximum = float(child.attrib["max"]) if "max" in child.attrib else None
                            except ValueError:
                                minimum = maximum = None
                        elif child_name == "enum":
                            type_name = "enum"
                            enum_values.extend(
                                entry.attrib["value"]
                                for entry in child
                                if _strip_namespace(entry.tag) == "entry" and "value" in entry.attrib
                            )
                        elif child_name in {"bool", "string"}:
                            type_name = child_name
                parameters[path] = AxisParameter(
                    path=path,
                    value=raw_value,
                    writable=writable,
                    type_name=type_name,
                    minimum=minimum,
                    maximum=maximum,
                    enum_values=tuple(enum_values),
                )
                return
            for child in node:
                walk(child, next_groups)

        walk(document, [])
        return parameters

    def read(self, group: str) -> dict[str, str]:
        status, body = self._request(self._endpoint(), {"action": "list", "group": group})
        if status != 200 or body.lstrip().startswith("# Error"):
            raise RuntimeError(f"Axis parameter read failed: {body[:300]}")
        result: dict[str, str] = {}
        for line in body.splitlines():
            if "=" not in line or line.startswith("#"):
                continue
            key, value = line.split("=", 1)
            result[self._normalise_path(key.strip())] = value.strip()
        return result

    def update(self, values: dict[str, str]) -> None:
        if not values:
            return
        query = {"action": "update", **values}
        status, body = self._request(self._endpoint(), query)
        if status != 200 or body.strip().startswith("# Error") or "OK" not in body:
            raise RuntimeError(f"Axis parameter update failed: {body[:300]}")


def load_stream_url(config_path: Path) -> tuple[str, int, str, int]:
    try:
        config = yaml.safe_load(config_path.read_text()) or {}
        camera = config["camera"]
        stream_url = str(camera["path"])
        camera_id = int(config.get("cam_id", 0))
        network = config.get("network", {})
        vision_ip = str(network.get("vision_ip", "224.5.23.2"))
        vision_port = int(network.get("vision_port", 10006))
    except (OSError, KeyError, TypeError, ValueError, yaml.YAMLError) as error:
        raise RuntimeError(f"cannot read camera config {config_path}: {error}") from error
    return stream_url, camera_id, vision_ip, vision_port


def ensure_proto(root: Path) -> None:
    generated = root / "python/proto/ssl_vision_wrapper_pb2.py"
    if generated.exists():
        return
    protoc = shutil_which("protoc")
    if protoc is None:
        raise RuntimeError("generated protobuf modules are missing and protoc is not installed")
    generated.parent.mkdir(parents=True, exist_ok=True)
    proto_files = [str(path.relative_to(root)) for path in sorted((root / "proto").glob("*.proto"))]
    command = [protoc, f"--proto_path={root}", f"--python_out={root / 'python'}", *proto_files]
    result = subprocess.run(command, cwd=root, capture_output=True, text=True, check=False)
    if result.returncode != 0:
        raise RuntimeError(f"protoc failed: {result.stderr.strip()}")


def shutil_which(command: str) -> str | None:
    """Local wrapper keeps imports small and makes the error easy to test."""
    for directory in os.environ.get("PATH", "").split(os.pathsep):
        candidate = Path(directory) / command
        if candidate.is_file() and os.access(candidate, os.X_OK):
            return str(candidate)
    return None


class DetectionListener:
    def __init__(self, group: str, port: int, camera_id: int, expected: set[str], interface: str | None) -> None:
        self.group = group
        self.port = port
        self.camera_id = camera_id
        self.expected = expected
        self.interface = interface
        self.sock: socket.socket | None = None
        self._packet_type = None

    def __enter__(self) -> DetectionListener:
        if str(ROOT / "python") not in sys.path:
            sys.path.insert(0, str(ROOT / "python"))
        ensure_proto(ROOT)
        from proto.ssl_vision_wrapper_pb2 import SSL_WrapperPacket

        self._packet_type = SSL_WrapperPacket
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.bind(("", self.port))
        interface_ip = self.interface or "0.0.0.0"
        membership = struct.pack("4s4s", socket.inet_aton(self.group), socket.inet_aton(interface_ip))
        sock.setsockopt(socket.IPPROTO_IP, socket.IP_ADD_MEMBERSHIP, membership)
        sock.settimeout(0.25)
        self.sock = sock
        return self

    def __exit__(self, exc_type, exc_value, traceback) -> None:
        if self.sock is not None:
            self.sock.close()
            self.sock = None

    def collect(self, duration: float) -> DetectionStats:
        if self.sock is None or self._packet_type is None:
            raise RuntimeError("detection listener is not open")
        stats = DetectionStats(expected=self.expected)
        deadline = time.monotonic() + duration
        while time.monotonic() < deadline:
            try:
                payload, _address = self.sock.recvfrom(65536)
            except socket.timeout:
                continue
            packet = self._packet_type()
            try:
                packet.ParseFromString(payload)
            except Exception:
                continue
            if not packet.HasField("detection") or packet.detection.camera_id != self.camera_id:
                continue
            stats.add(packet.detection)
        return stats


@dataclass
class DetectionStats:
    expected: set[str]
    frames: int = 0
    frames_with_bots: int = 0
    bot_detections: int = 0
    confidence_sum: float = 0.0
    presence: dict[str, int] = field(default_factory=dict)
    first_capture: float | None = None
    last_capture: float | None = None

    @staticmethod
    def _key(team: str, robot_id: int) -> str:
        return f"{team[0]}{robot_id}"

    def add(self, detection) -> None:
        self.frames += 1
        if self.first_capture is None:
            self.first_capture = detection.t_capture
        self.last_capture = detection.t_capture
        frame_keys: set[str] = set()
        for team, robots in (("yellow", detection.robots_yellow), ("blue", detection.robots_blue)):
            for bot in robots:
                key = self._key(team, bot.robot_id)
                frame_keys.add(key)
                self.presence[key] = self.presence.get(key, 0) + 1
                self.bot_detections += 1
                self.confidence_sum += bot.confidence
        if frame_keys:
            self.frames_with_bots += 1

    def result(self) -> dict[str, float | int | dict[str, int]]:
        if self.frames == 0:
            return {
                "frames": 0,
                "frames_with_bots": 0,
                "robot_detections": 0,
                "mean_confidence": 0.0,
                "coverage": 0.0,
                "drop_rate": 1.0,
                "score": 0.0,
                "presence": self.presence,
            }
        confidence = self.confidence_sum / max(1, self.bot_detections)
        if self.expected:
            coverage = sum(self.presence.get(key, 0) for key in self.expected) / (self.frames * len(self.expected))
            expected_present = sum(self.presence.get(key, 0) for key in self.expected)
            extras = max(0, self.bot_detections - expected_present)
            extra_rate = extras / max(1, self.frames)
        else:
            coverage = self.frames_with_bots / self.frames
            extra_rate = 0.0
        drop_rate = 1.0 - coverage
        score = 100.0 * coverage + 20.0 * confidence - 10.0 * extra_rate
        fps = 0.0
        if self.first_capture is not None and self.last_capture is not None and self.last_capture > self.first_capture:
            fps = (self.frames - 1) / (self.last_capture - self.first_capture)
        return {
            "frames": self.frames,
            "frames_with_bots": self.frames_with_bots,
            "robot_detections": self.bot_detections,
            "mean_confidence": round(confidence, 5),
            "coverage": round(coverage, 5),
            "drop_rate": round(drop_rate, 5),
            "fps": round(fps, 3),
            "score": round(score, 5),
            "presence": self.presence,
        }


def first_parameter(parameters: dict[str, AxisParameter], *names: str) -> AxisParameter | None:
    wanted = {name.lower() for name in names}
    candidates = [parameter for path, parameter in parameters.items() if path.rsplit(".", 1)[-1].lower() in wanted]
    writable = [parameter for parameter in candidates if parameter.writable]
    return sorted(writable or candidates, key=lambda parameter: parameter.path)[0] if (writable or candidates) else None


def parse_values(text: str, cast) -> list:
    values = []
    for item in text.split(","):
        item = item.strip()
        if item:
            values.append(cast(item))
    if not values:
        raise argparse.ArgumentTypeError("must contain at least one value")
    return values


def parse_expected(text: str) -> set[str]:
    result = {item.strip().lower() for item in text.split(",") if item.strip()}
    for item in result:
        if len(item) < 2 or item[0] not in {"y", "b"} or not item[1:].isdigit():
            raise argparse.ArgumentTypeError("expected robots use IDs such as y0,y1,b0")
    return result


def format_value(value: float) -> str:
    return str(int(value)) if float(value).is_integer() else str(value)


def discover(client: AxisClient) -> dict[str, AxisParameter]:
    # Fail clearly when the host or credentials are wrong. Individual groups
    # can legitimately be absent on older Axis firmware, so only those errors
    # are ignored below.
    client._endpoint()
    groups = ("ImageSource.*.Sensor", "Image.*.Sensor", "Image.*.Appearance", "Image.*.Stream", "Properties.Image")
    parameters: dict[str, AxisParameter] = {}
    for group in groups:
        try:
            parameters.update(client.list_parameters(group))
        except RuntimeError as error:
            LOG.debug("parameter group %s unavailable: %s", group, error)
    return parameters


def print_probe(parameters: dict[str, AxisParameter], client: AxisClient) -> None:
    print(f"Axis camera: {client.host}")
    if not parameters:
        print("No parameter definitions were returned.")
        return
    names = {
        "maxexposuretime", "maxgain", "exposurepriority", "exposure", "whitebalance",
        "tnf", "snf", "wdr", "sharpness", "compression", "fps",
    }
    for parameter in sorted(parameters.values(), key=lambda item: item.path):
        if parameter.path.rsplit(".", 1)[-1].lower() not in names:
            continue
        limits = ""
        if parameter.minimum is not None or parameter.maximum is not None:
            limits = f" range={parameter.minimum:g}..{parameter.maximum:g}"
        if parameter.enum_values:
            limits = " values=" + ",".join(parameter.enum_values)
        access = "rw" if parameter.writable else "ro"
        print(f"  {parameter.path}={parameter.value!r} [{access}{limits}]")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    default_config = ROOT / "config-camera-2.local.yml"
    if not default_config.exists():
        default_config = ROOT / "config-camera-2.yml"
    parser.add_argument("--config", type=Path, default=default_config, help="camera YAML (default: %(default)s)")
    parser.add_argument("--username", help="Axis username; otherwise use the URL or prompt")
    parser.add_argument("--password", help="Axis password; otherwise use the URL or AXIS_PASSWORD")
    parser.add_argument("--probe", action="store_true", help="list camera parameters without changing anything")
    parser.add_argument("--dry-run", action="store_true", help="show planned trials without changing camera settings")
    parser.add_argument("--no-apply", action="store_true", help="restore the original settings after the trials")
    parser.add_argument(
        "--duration", type=float, default=10.0,
        help="seconds of detection data per trial (default: %(default)s)",
    )
    parser.add_argument(
        "--settle", type=float, default=2.0,
        help="seconds to wait after changing settings (default: %(default)s)",
    )
    parser.add_argument(
        "--exposure-us", default="2000,4000,6000,8000",
        help="MaxExposureTime candidates in microseconds",
    )
    parser.add_argument("--max-gain", default="25,50,75,100", help="MaxGain candidates in Axis percent")
    parser.add_argument(
        "--exposure-priority", default="0,50,100",
        help="ExposurePriority candidates when shutter limits are unavailable",
    )
    parser.add_argument("--max-trials", type=int, default=0, help="limit candidates; 0 means all combinations")
    parser.add_argument(
        "--expected-robots", type=parse_expected, default=set(),
        help="comma-separated expected IDs, e.g. y0,y1,b0",
    )
    parser.add_argument("--vision-ip", help="override multicast group from YAML")
    parser.add_argument("--vision-port", type=int, help="override multicast port from YAML")
    parser.add_argument("--interface", help="local interface IP for multicast membership")
    parser.add_argument("--report", type=Path, default=Path("axis-autotune-result.json"), help="JSON report path")
    parser.add_argument("--timeout", type=float, default=5.0, help="Axis HTTP timeout in seconds")
    parser.add_argument("--verbose", action="store_true")
    return parser


def run(args: argparse.Namespace) -> int:
    if args.duration <= 0 or args.settle < 0 or args.max_trials < 0:
        raise RuntimeError("duration and settle must be non-negative, and max-trials cannot be negative")
    stream_url, camera_id, yaml_group, yaml_port = load_stream_url(args.config)
    vision_group = args.vision_ip or yaml_group
    vision_port = args.vision_port or yaml_port
    password = args.password or os.environ.get("AXIS_PASSWORD")
    client = AxisClient(stream_url, args.username, password, args.timeout)
    parameters = discover(client)
    if args.probe:
        print_probe(parameters, client)
        return 0

    exposure_parameter = first_parameter(parameters, "MaxExposureTime")
    gain_parameter = first_parameter(parameters, "MaxGain")
    priority_parameter = first_parameter(parameters, "ExposurePriority")
    if exposure_parameter is None and gain_parameter is None and priority_parameter is None:
        print_probe(parameters, client)
        raise RuntimeError("camera exposes no writable exposure or gain control supported by this tuner")

    axes: list[tuple[AxisParameter, list[float]]] = []
    if exposure_parameter is not None and exposure_parameter.writable:
        axes.append((exposure_parameter, parse_values(args.exposure_us, float)))
    if gain_parameter is not None and gain_parameter.writable:
        axes.append((gain_parameter, parse_values(args.max_gain, float)))
    if not axes and priority_parameter is not None and priority_parameter.writable:
        axes.append((priority_parameter, parse_values(args.exposure_priority, float)))
    if not axes:
        raise RuntimeError("the discovered exposure controls are read-only for the current account")

    for parameter, values in axes:
        name = parameter.path.rsplit(".", 1)[-1]
        if parameter.minimum is not None and any(value < parameter.minimum for value in values):
            raise RuntimeError(f"{name} candidate is below the camera minimum {parameter.minimum:g}")
        if parameter.maximum is not None and any(value > parameter.maximum for value in values):
            raise RuntimeError(f"{name} candidate is above the camera maximum {parameter.maximum:g}")
    original = {parameter.path: parameter.value for parameter, _values in axes}
    candidates = [
        {
            parameter.path: format_value(value)
            for (parameter, _values), value in zip(axes, values)
        }
        for values in itertools.product(*(values for _parameter, values in axes))
    ]
    if args.max_trials:
        candidates = candidates[: args.max_trials]
    print(f"Camera {camera_id} ({client.host}); testing {len(candidates)} settings on {vision_group}:{vision_port}")
    print(
        "Tuning " + " and ".join(parameter.path for parameter, _values in axes)
        + "; robots should be moving in view."
    )
    if args.dry_run:
        for index, candidate in enumerate(candidates, 1):
            print(f"  {index:02d}: " + ", ".join(f"{key}={value}" for key, value in candidate.items()))
        return 0

    results: list[dict] = []
    best: tuple[float, dict[str, str], dict] | None = None
    try:
        with DetectionListener(vision_group, vision_port, camera_id, args.expected_robots, args.interface) as listener:
            for index, candidate in enumerate(candidates, 1):
                setting_text = ", ".join(
                    f"{key.rsplit('.', 1)[-1]}={value}" for key, value in candidate.items()
                )
                print(f"[{index}/{len(candidates)}] {setting_text}", flush=True)
                client.update(candidate)
                if args.settle:
                    time.sleep(args.settle)
                stats = listener.collect(args.duration)
                metrics = stats.result()
                result = {"settings": candidate, **metrics}
                results.append(result)
                print(
                    f"  score={metrics['score']:.2f} coverage={metrics['coverage']:.3f} "
                    f"confidence={metrics['mean_confidence']:.3f} frames={metrics['frames']}",
                    flush=True,
                )
                score = float(metrics["score"])
                if best is None or score > best[0]:
                    best = (score, candidate, metrics)

        if best is None:
            raise RuntimeError("no tuning trials completed")
        best_score, best_settings, best_metrics = best
        if not any(int(result["frames"]) > 0 for result in results):
            raise RuntimeError(
                f"received no detections for camera {camera_id} on {vision_group}:{vision_port}; "
                "start the vision processor and move robots into view"
            )
        if args.no_apply:
            client.update(original)
            applied = False
        else:
            client.update(best_settings)
            applied = True
        setting_text = ", ".join(
            f"{key.rsplit('.', 1)[-1]}={value}" for key, value in best_settings.items()
        )
        print(f"Best score {best_score:.2f}: {setting_text}")
        print("Best settings applied." if applied else "Original settings restored (--no-apply).")
        report = {
            "created_at": datetime.now(timezone.utc).isoformat(),
            "camera_host": client.host,
            "camera_id": camera_id,
            "vision_group": vision_group,
            "vision_port": vision_port,
            "expected_robots": sorted(args.expected_robots),
            "original_settings": original,
            "best_settings": best_settings,
            "best_metrics": best_metrics,
            "applied": applied,
            "trials": results,
        }
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, indent=2) + "\n")
        print(f"Report written to {args.report}")
        return 0
    except BaseException:
        try:
            client.update(original)
            print("Original Axis settings restored after interrupted or failed tuning.", file=sys.stderr)
        except Exception as restore_error:
            print(f"Could not restore original Axis settings: {restore_error}", file=sys.stderr)
        raise


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    logging.basicConfig(level=logging.DEBUG if args.verbose else logging.WARNING, format="%(levelname)s %(message)s")
    try:
        return run(args)
    except KeyboardInterrupt:
        print("Interrupted.", file=sys.stderr)
        return 130
    except (RuntimeError, ValueError, OSError) as error:
        print(f"axis-autotune: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
