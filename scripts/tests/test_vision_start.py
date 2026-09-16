"""Run with: python3 -m unittest discover -s scripts/tests -v."""

import importlib.machinery
import importlib.util
import os
from pathlib import Path
import runpy
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]


def load(name, path):
    loader = importlib.machinery.SourceFileLoader(name, str(path))
    spec = importlib.util.spec_from_loader(name, loader)
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


vision = load("vision_start", ROOT / "software/ssl-vision-processor/start.py")
sg = load("sg_start", ROOT / "scripts/bin/sg-start")
CHOICE = next(i for i, config in enumerate(sg.choices) if config.get("vision_processor"))


class VisionLauncherTests(unittest.TestCase):
    def test_local_override_only_replaces_its_camera(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "config-camera-1.local.yml").touch()
            jobs = vision.commands(root)
            self.assertEqual(jobs[0][1][-1], "geometry-seagoals.yml")
            self.assertEqual([argv[-1] for _, argv in jobs[1:]], [
                "config-camera-0.yml", "config-camera-1.local.yml", "config-camera-2.yml"
            ])

    def test_missing_build_has_actionable_error(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(RuntimeError, r"Run ./setup.sh"):
                vision.check(Path(directory))

    def test_camera_failure_stops_remaining_processes(self):
        children = []
        real_popen = subprocess.Popen

        def spawn(*args, **kwargs):
            child = real_popen(*args, **kwargs)
            children.append(child)
            return child

        jobs = [
            ("publisher", [sys.executable, "-c", "import time; time.sleep(30)"]),
            ("camera 1", [sys.executable, "-c", "import time; time.sleep(0.2); raise SystemExit(7)"]),
        ]
        with mock.patch.object(vision.subprocess, "Popen", side_effect=spawn):
            with self.assertRaisesRegex(RuntimeError, "camera 1 exited with status 7"):
                vision.supervise(jobs, startup_grace=10)
        self.assertTrue(all(child.poll() is not None for child in children))

    def test_sigterm_reaps_all_children(self):
        with tempfile.TemporaryDirectory() as directory:
            pidfile = Path(directory) / "child.pid"
            worker = (
                "import os, pathlib, time; "
                f"pathlib.Path({str(pidfile)!r}).write_text(str(os.getpid())); time.sleep(30)"
            )
            script = (
                f"import runpy; v = runpy.run_path({str(ROOT / 'software/ssl-vision-processor/start.py')!r}); "
                f"v['supervise']([('camera', [{sys.executable!r}, '-c', {worker!r}])], startup_grace=10)"
            )
            supervisor = subprocess.Popen([sys.executable, "-c", script], stdout=subprocess.DEVNULL)
            try:
                deadline = time.monotonic() + 5
                while not pidfile.exists() and time.monotonic() < deadline:
                    time.sleep(0.05)
                self.assertTrue(pidfile.exists(), "Worker did not start")
                child_pid = int(pidfile.read_text())
                supervisor.send_signal(signal.SIGTERM)
                self.assertEqual(supervisor.wait(timeout=8), 0)
                with self.assertRaises(ProcessLookupError):
                    os.kill(child_pid, 0)
            finally:
                if supervisor.poll() is None:
                    supervisor.kill()
                    supervisor.wait()


class StartupIntegrationTests(unittest.TestCase):
    def setUp(self):
        sg.network = False

    def test_new_option_starts_service_then_lab_containers(self):
        with mock.patch.object(sg.subprocess, "run") as run, \
                mock.patch.object(sg.subprocess, "Popen") as popen:
            sg.execute_choice(CHOICE, False)
        calls = [call.args[0] for call in run.call_args_list]
        self.assertEqual(calls[0][0], "systemd-run")
        self.assertIn("--property=Type=notify", calls[0])
        self.assertEqual(calls[1][:4], ["docker", "compose", "--env-file", sg.env_path])
        self.assertIn(f"{sg.docker_dir}/docker-compose.fetdatorn.yml", calls[1])
        self.assertNotIn(f"{sg.docker_dir}/docker-compose.autoref.yml", calls[1])
        self.assertEqual(calls[2][:2], ["docker", "exec"])
        popen.assert_not_called()  # In particular, never start legacy ssl-vision.

    def test_compose_failure_stops_vision_and_does_not_enter_ai(self):
        def run(argv, **kwargs):
            if argv[:2] == ["docker", "compose"]:
                raise subprocess.CalledProcessError(1, argv)
            return subprocess.CompletedProcess(argv, 0)

        with mock.patch.object(sg.subprocess, "run", side_effect=run) as calls:
            with self.assertRaises(subprocess.CalledProcessError):
                sg.execute_choice(CHOICE, False)
        commands = [call.args[0] for call in calls.call_args_list]
        self.assertIn(["systemctl", "--user", "stop", sg.vision_processor_unit], commands)
        self.assertFalse(any(command[:2] == ["docker", "exec"] for command in commands))

    def test_failed_preflight_does_not_stop_existing_stack(self):
        def run(argv, **kwargs):
            if argv[-1] == "--check":
                raise subprocess.CalledProcessError(1, argv)
            return subprocess.CompletedProcess(argv, 1 if argv[0] == "pgrep" else 0)

        with mock.patch.object(sys, "argv", ["sg-start", "--vision-processor"]), \
                mock.patch("subprocess.run", side_effect=run) as calls, \
                mock.patch("shutil.which", return_value="/usr/bin/tool"), \
                mock.patch("os.system") as system:
            with self.assertRaises(SystemExit) as exit_status:
                runpy.run_path(str(ROOT / "scripts/bin/sg-start"), run_name="__main__")
        self.assertEqual(exit_status.exception.code, 1)
        commands = [call.args[0] for call in calls.call_args_list]
        self.assertFalse(any(command[0].endswith("sg-kill") for command in commands))
        self.assertFalse(any(command[:2] == ["docker", "rm"] for command in commands))
        system.assert_not_called()


if __name__ == "__main__":
    unittest.main()
