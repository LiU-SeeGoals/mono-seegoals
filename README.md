# SeeGoals
This is the main repository used for [SeeGoals](https://fiarobotics.se/seegoals/) development, it contains everything to get started developing!

## Systems
The SeeGoals project consists of different systems needed to make everything to work. Each system exists within this repository and (should) include their own `README.md` inside their folder, check them out for more details about the system.

Here follows a short description of the systems:
```
./docker/                   - Contains general docker information and Docker Compose files used to startup the different systems.
./firmware/                 - All the firmware used on the robots and basestation microcontrollers.
./scripts/                  - All scripts used to make development easier in the project.
./software/                 - All the software used within the project, i.e. the non-firmware stuff.
./software/GameController/  - No code, only stores temporary data that the GameController needs.
./software/GameViewer/      - A website that displays the current gamestate.
./software/controller/      - The software that plans/execute plans for the robots.
./sotware/proto-messages/   - The definition of our protbuf messages.
./software/ssl-vision/      - Forked software from the SSL league, it parses camera data which our controller uses to plan.
```

## Quickstart Guide
Follow these steps to get started developing!

### Dependencies
1. Make sure you have git installed by running `git -v` in your terminal. You'll see something similar to `git version 2.52.0`.

2. Setup SSH keys on your system so that you're allowed to clone this repository, follow [this guide](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account).

3. You'll need to [install Docker](https://docs.docker.com/engine/install/). After installation is done, run `docker -v` to confirm everything is working, you should see something similar to `Docker version 28.5.2, build ecc694264d`. 

4. You also need to make sure that a *normal* user can run docker, the scripts can break otherwise. Test by running `docker ps` (**not using sudo**), if it doesn't work, try following [this guide](https://docs.docker.com/engine/install/linux-postinstall/).

5. Install [Docker Compose](https://docs.docker.com/compose/install/linux/). Verify that it works by running `docker compose version`, you should see something similar to `Docker Compose version 2.40.3`.

### Installing the repository
1. Start by cloning this repository to the computer you decide to work from, i.e. by using `git clone git@github.com:LiU-SeeGoals/mono-seegoals.git`.

2. Installing the scripts.

We're using some scripts to make development easier, these scripts should be added to your PATH environment variable so you can run them from anywhere.

The scripts are located at [./scripts/bin/](./scripts/bin/), the full path should be used, for me that is `/home/re/Dev/mono-seegoals/scripts/bin/` since I cloned the repository to `/home/re/Dev/`.

If you're using bash, then update your `~/.bashrc` as follows:
```
export PATH="$PATH:PATH_TO_MONO_SEEGOALS_REPOSITORY"

# for me it is the path below, for you it is not
# export PATH="$PATH:/home/re/Dev/mono-seegoals/scripts/bin"
```

If you're not using bash, then update the corresponding *rc* file for your shell (if you're unsure what you're using, you're using bash).

3. Then restart your shell and you should have all commands available needed to continue, test by writing `sg-start --help` from your home folder, you should see the text `as if...` printed in your terminal.

### Running the project (simulation)
The command `sg-start` is used to start different environments for developing, you can see all available setups by simply running `sg-start` from anywhere.

Once you've started the correct environment, you can edit the corresponding systems files and test it out live.

The command `sg-enter` can be used to enter a specific systems docker container. When you've entered a SeeGoals docker container, you should be able to press the up arrow to get the command needed to start that system.

Please see the `README.md` files in the different system folders for more information on how to develop for that system.

### Running the project live
If you want to try out the project on the field in the lab, then:

1. Make sure the basestation is powered up and connected to the switch that fetdatorn is connected to.
2. From fetdatorn, run the fetdatorn configuration available on `sg-start`, press up arrow and run the controller program.
3. Go to our [GameViewer](http://localhost:5173) from fetdatorn to see that everything seems to work.

Debugging:
- Check that the `BASESTATION_ADDR` within `./environment.ini` is the same as the address the basesation reports `sg-fw serial /dev/serial/by-id/BASESATION_ID`.
- Check that the `ENVIRONMENT` within `./environment.ini` is set to `competition` and not `simulation`.
