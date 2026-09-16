# Scripts

Add `scripts/bin` to your PATH as described in the [main README](../README.md).
Run `sg-start --help` to list startup options, or `sg-start` for the menu.

## New vision processor on the lab field

One-time setup on the camera-connected Linux machine:

```bash
cd software/ssl-vision-processor
./setup.sh
```

Answer **n** to the setup script's systemd-service prompt. `sg-start` manages
the three-camera service itself. The Axis cameras use the OpenCV backend;
Spinnaker and Bluefox SDKs are not required. A working OpenCL runtime and a
systemd user session are required.

Start everything with:

```bash
sg-start --vision-processor
# Equivalent: sg-start 8, or select the new vision processor in the menu.
```

This starts the geometry publisher with `geometry-seagoals.yml` and one
processor for each of cameras 0, 1 and 2. A `config-camera-N.local.yml` file
takes precedence over `config-camera-N.yml`, so camera credentials and local
tuning can remain in ignored files. The launcher's working directory is always
`software/ssl-vision-processor`, even when `sg-start` runs from elsewhere.

The lab Docker stack starts alongside vision, including AutoRef, GameViewer,
and an AI development shell. Vision publishes raw packets on
`224.5.23.2:10006`; AutoRef supplies tracked packets on port `10010`.
The AI program is started manually from the shell as in the existing lab option.
The new browser wrapper is not required for this launch path.

The processes run in the transient user service
`seegoals-vision-processor.service`. Leaving the AI shell keeps vision running.
If any vision process exits, the service stops its remaining processes.
`sg-kill` or starting another `sg-start` configuration stops the entire service.
The existing startup choices keep their menu numbers and still use the old
vision program. Stop any old vision instance before switching to the new one;
the launcher refuses to start while a process named `vision` is running.

```bash
journalctl --user -u seegoals-vision-processor.service -f  # logs
systemctl --user status seegoals-vision-processor.service
sg-kill
```

To view the camera streams, install `mpv` and run
`python3 python/cam_viewer.py --cameras 3` from `software/ssl-vision-processor`
before starting vision, so the viewer receives the initial H.264 parameters.
