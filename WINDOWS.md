# Windows Setup
SeeGoals is developed on Linux. On Windows you run it inside **WSL2 (Ubuntu)** with **Docker Desktop**, and then follow the normal [README](README.md) from inside Ubuntu. This guide covers the Windows-specific parts.

Only the **simulation** works on Windows. Running on the real field (fetdatorn, basestation, cameras) and flashing firmware need a Linux machine.

## 1. Install WSL2 and Ubuntu
Open PowerShell **as Administrator**:
```
wsl --install -d Ubuntu
```
Reboot, open **Ubuntu** from the Start menu and create a Linux username and password.

Make sure WSL is up to date (this also gives you WSLg, which is needed for GUI windows):
```
wsl --update
```

From now on, **every command in this guide runs in the Ubuntu terminal** unless it says PowerShell.

## 2. Install Docker Desktop
1. Install [Docker Desktop for Windows](https://docs.docker.com/desktop/setup/install/windows-install/) with the **WSL2 backend**.
2. In Docker Desktop: **Settings → Resources → WSL Integration → enable Ubuntu**.
3. Keep Docker Desktop running whenever you work on SeeGoals.

Do **not** install Docker inside Ubuntu (`apt install docker.io` or `docker-ce`). Docker Desktop provides it. If you already did, remove it:
```
sudo apt remove -y docker.io docker-ce docker-ce-cli containerd containerd.io
```

### Run docker without sudo
The scripts break if `docker` needs `sudo`. Test with `docker ps`. If it asks for sudo or says permission denied:
```
sudo groupadd docker
sudo usermod -aG docker $USER
```
Then in PowerShell run `wsl --shutdown` and reopen Ubuntu. `docker ps` should now work without sudo.

## 3. Install tools in Ubuntu
```
sudo apt update
sudo apt install -y git python3 python3-pip x11-xserver-utils
```
`x11-xserver-utils` provides `xhost`, which `sg-start` uses to allow containers to open windows.

Check that the display works:
```
echo $DISPLAY
xhost +
```
`$DISPLAY` should print something like `:0`. If it is empty, run `wsl --update` in PowerShell, then `wsl --shutdown`, and reopen Ubuntu.

To stop Git from converting line endings (which breaks the scripts with errors like `python3\r: No such file`):
```
git config --global core.autocrlf input
```

## 4. SSH key for GitHub
Create the key **inside Ubuntu**. Keys stored in Windows are not used:
```
ssh-keygen -t ed25519
cat ~/.ssh/id_ed25519.pub
```
Add the printed key to GitHub ([guide](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account)).

## 5. Clone the repository
Clone into your Linux home folder, **not** into `/mnt/c/...`. The Windows drive is very slow from WSL and breaks file permissions.
```
cd ~
git clone --recurse-submodules git@github.com:LiU-SeeGoals/mono-seegoals.git
cd mono-seegoals
git checkout dev
```

> **Use the `dev` branch.** On `main`, all simulation containers use `network_mode: host`. On Windows that "host" is Docker Desktop's hidden VM, so nothing is reachable from your Windows browser. `dev` publishes the ports properly.

Create the Game Controller folders so Docker doesn't create them owned by root (otherwise `ssl-game-controller` exits with a permission error):
```
mkdir -p software/GameController/config software/GameController/data
sudo chown -R 1000:1000 software/GameController
```

## 6. Add the scripts to PATH
```
echo 'export PATH="$PATH:$HOME/mono-seegoals/scripts/bin"' >> ~/.bashrc
source ~/.bashrc
sg-start --help
```

## 7. Remove the `/dev/input` mapping
WSL has no `/dev/input`, so the `ai` container fails with:
```
error gathering device information while adding custom device "/dev/input": no such file or directory
```
In `docker/docker-compose.yml`, comment out these two lines under the `ai` service:
```yaml
    # devices:
    #   - /dev/input:/dev/input
```
They are only needed for using an Xbox controller. Tell Git to ignore your local change so you don't commit it:
```
git update-index --skip-worktree docker/docker-compose.yml
```
Before switching branches or pulling changes to that file, undo this with `git update-index --no-skip-worktree docker/docker-compose.yml` and stash your edit.

## 8. Start the simulation
```
sg-start 1
```
Then open <http://localhost:5173> in your normal Windows browser.

To run the AI, enter the container with `sg-enter`, choose `ai`, press the **up arrow** to get the start command, and press Enter.

### Which `sg-start` options work on Windows
| Option | Works | Why |
|---|---|---|
| 0 Rebuild images | Yes | |
| 1 Develop AI | Yes | |
| 2 Develop GameViewer | Yes | |
| 3 Develop AI and GameViewer | Yes | |
| 8, 9 vs TIGERs AI | Yes | |
| 4, 5, 10 Fetdatorn | No | Lab machine only, host networking |
| 6 Competition | No | Lab machine only, host networking |
| 7 GameViewer, enter on startup | No | Uses host networking |

## 9. Edit the code with VS Code
VS Code runs on Windows, but with the **WSL extension** it opens the files and runs its terminal inside Ubuntu. That way it is fast and the `sg-*` scripts work in its terminal.

1. Install [VS Code](https://code.visualstudio.com/) on **Windows** (not inside Ubuntu). Keep **"Add to PATH"** checked in the installer.
2. In VS Code, install the **WSL** extension (`ms-vscode-remote.remote-wsl`).
3. Open the repo from the Ubuntu terminal:
   ```
   cd ~/mono-seegoals
   code .
   ```
   The first time, this installs the VS Code server in Ubuntu, which takes a minute.

You can also open it from VS Code: press `Ctrl+Shift+P`, run **WSL: Connect to WSL**, then **File → Open Folder** and choose `/home/<your-user>/mono-seegoals`.

Check that the bottom-left corner of VS Code says **WSL: Ubuntu**. If it doesn't, you're editing the files from Windows. That is slow and can add Windows line endings.

Things to know:
- **The VS Code terminal is Ubuntu.** Open it with ``Ctrl+` `` and run `sg-start 1`, `sg-enter` and so on there.
- **Extensions are installed separately for WSL.** Extensions you already have on Windows show **"Install in WSL: Ubuntu"** in the Extensions panel. Install the ones you need, such as **Go** for the controller.
- **No need to attach to the containers.** The containers mount the repo folders, so changes you save in VS Code show up inside them immediately.
- **`code: command not found` in Ubuntu:** reinstall VS Code with "Add to PATH" checked, run `wsl --shutdown` in PowerShell, and reopen Ubuntu.

## How localhost reaches the containers
```
Windows browser → localhost:5173 → Docker Desktop port forward → game-viewer container
```
Docker Desktop forwards every port in a service's `ports:` list to Windows. After the page loads, the browser also connects to these ports:

| Port | Service |
|---|---|
| 5173 | GameViewer page |
| 3000 | SSL-Vision websocket (game-viewer) |
| 3001 | Game controller websocket (game-viewer) |
| 5174 | AI → GameViewer socket (ai) |
| 8081 | SSL Game Controller UI |
| 8082 | SSL Vision client |

The containers talk to each other over Docker's internal network, which doesn't go through Windows.

## Troubleshooting
**Check container status and ports**
```
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```
`game-viewer` should show `0.0.0.0:5173->5173/tcp`.

| Problem | Fix |
|---|---|
| `docker ps` needs sudo | See [Run docker without sudo](#run-docker-without-sudo). |
| `xhost: command not found` | `sudo apt install -y x11-xserver-utils` |
| `/dev/input` error, or `container ... is not running` | See [step 7](#7-remove-the-devinput-mapping). |
| Ports column is empty | You're on `main`, or you used a host-networking option. Switch to `dev` and use `sg-start 1`. |
| `ssl-game-controller` shows `Exited (2)` | Fix the folder ownership (see [step 5](#5-clone-the-repository)), then check `docker logs ssl-game-controller`. |
| Page doesn't load | Test from Ubuntu with `curl -I http://localhost:5173`. If that works, test from PowerShell with `curl.exe -I http://localhost:5173`. Also try `http://127.0.0.1:5173`. |
| Page loads but shows no data | Open browser DevTools → Network → WS. Failed websockets mean that service isn't running. Start it with `sg-enter`. |
| Port already in use | In PowerShell, run `netstat -ano \| findstr :5173` to find the program using it. |
| `python3\r: No such file` | `git config --global core.autocrlf input`, then clone again. |
| Everything is very slow | The repo is under `/mnt/c`. Clone it into `~` instead. |

### Not supported on Windows
- **Live field / fetdatorn:** multicast from the lab network does not reach the WSL VM.
- **Firmware flashing and serial (`sg-fw`):** USB devices are not visible in WSL by default. [usbipd-win](https://github.com/dorssel/usbipd-win) can pass them through, but this is untested.
- **Vision processor (option 10):** needs systemd and OpenCL. Use a Linux machine.
