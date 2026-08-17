# Using a USB Xbox Controller to drive robots

This document describes how to run the SeeGoals robot controller using a USB Xbox-style gamepad (the code uses SDL2). It covers running inside the project Docker environment and running locally on the host.

## Overview
- The controller UI that reads an Xbox controller is implemented at `internal/ui/controller_remote_control.go`.
- The dev Docker compose maps `/dev/input` into the container so SDL can access the controller (see `docker/docker-compose.yml`).

## Requirements
- A USB Xbox-compatible controller (wired or USB adapter).
- On the host: `docker` and `docker compose` for the container setup, or `go` and `libsdl2-dev` for running locally.

## Quick (recommended) — run inside the project's Docker environment
1. Plug in the controller.
2. Start the development containers (this maps `/dev/input` into the container and uses host networking):

```bash
./scripts/sg-start
```

Choose the option that starts the `ai` (controller) container.

3. Enter the container (pick the AI/container entry):

```bash
./scripts/sg-enter
```

4. Run the controller UI inside the container (workdir is `/var/controller`):

```bash
cd /var/controller
go run internal/ui/controller_remote_control.go
```

Follow the interactive prompts (enter robot id). The program prints the button and axis mapping.

## What the controls do (as printed by the program)
- Left stick: translation
- Right stick: heading
- LB / RB: kick speeds
- Y: chip kick
- B: stop
- X: toggle dribbler
- LT / RT: decrease / increase speed

(The UI prints the exact mapping when a controller is found.)

## Verification / quick checks
- Ensure Linux sees the controller:

```bash
ls /dev/input
lsusb | grep -i xbox
```

- Test events (optional tools):

```bash
sudo apt install -y evtest joystick
sudo evtest            # choose the controller device
jstest /dev/input/js0
```

## Troubleshooting
- "no game controller detected" (program panics):
  - Confirm the controller appears under `/dev/input`.
  - Check permissions — either add your user to the `input` group or run the program with elevated permissions:

```bash
sudo gpasswd -a $USER input
# then log out/in (or reboot)
```

  - If running in Docker, ensure the container has `/dev/input` mapped (the project's `docker/docker-compose.yml` maps `/dev/input` for the `ai` service).

- Buttons/axes behave unexpectedly:
  - Some controllers have different axis ordering; use `evtest` or `jstest` to inspect the mapping and adjust if needed.

- Commands are not reaching robots:
  - The controller UI connects to a basestation client at `127.0.0.1:20011` by default (see the UI source). When running in the provided Docker compose, containers use `network_mode: host`, so `127.0.0.1` is the host. Ensure the basestation or simulator is running and reachable.

## Where to find code
- Controller UI: `software/controller/internal/ui/controller_remote_control.go`
- Docker compose: `docker/docker-compose.yml`

---
If you want, I can add a short link from `software/controller/README.md` to this file or commit the change for you.