# What is this repo ❓

This repo and README is the starting point for new memebers!

This repo contains the Docker configuration for SeeGoals. It sets up both the production and development environments.

If you're here lokking for documentation on controlling the robots, visit the [controller repository](https://github.com/LiU-SeeGoals/controller). To learn more about how the visualization works, check out the [gameViewer repository](https://github.com/LiU-SeeGoals/GameViewer).

# How to run 🚀

Note: this project uses a tool called Docker. Docker can be installed on both linux and windows, but is way easier to work with on linux. If you dont have linux on you computer, it's recomended to work from the lab computer "fetdatorn", and then using the "remote explorer" plugin in vscode. Ask around for help with this setup.

### Dependencies
1. You need to make sure you have git installed by running ```git -v``` in terminal. You see something similar to this as output: ```git version 2.43.0```. You also need to fix ssh keys so you are allowed to clone repos. If you don't have it follow this [guide](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account).

2. Then install docker. [Here](https://docs.docker.com/engine/install/ubuntu/) is a guide. Then run ```docker -v``` to confirm installation. You also need to make sure that a "normal" user can run docker. Test by running ```docker ps``` (not using sudo). Otherwise you need to fix docker permissions, (the scripts might break if not fixed).

3. Install docker compose. [Here](https://docs.docker.com/compose/install/linux/) is a installation guide. You should now verify install by running ```docker compose --help```.

### Setup
4. Start with cloning this repo to the computer you decided to work from. Click green button with the text "code" at this [website](https://github.com/LiU-SeeGoals/seegoals). Then make sure "SSH" is selected, and then copy the text. 

5. Then clone the repo by writing `git clone` and copying the command from previous step. It should look similar to this:

```
git clone git@github.com:LiU-SeeGoals/seegoals.git
```

6. Then go inside the new folder (repo) called docker `(cd seegoals)`. To start all the containers, run if you are on linux (recomended):

```
./start.sh
```

Select the option 1 (start base configuration)

All the containers will then start running. (it can take while to download everything)

To see the website, navigate to the URL displayed in the terminal log. You should see a slight vibration in the robots, indicating that the containers have started as expected. Refresh the website if there is no connection to the webserver (sometimes it takes a while for the webserver to start).

# Want to code for the AI or the website?

Run the same command as mentioned above, then select another option depending on where you want to develop. You should select the option so where you want to develop is the local one. For example, if you want to develop on "controller" then select the option "local controller". You will automaticly enter the development environment in this terminal (docker container). You can see a folder that is named controller. This is where you should make the changes. Note: make sure to switch to your own branch to develop on.

# Want to configure Docker?

The images are available in the Packages tab of the repo at ```https://github.com/LiU-SeeGoals```. A GitHub Action builds and updates an image with every change in the main branch, as shown in this video: ```https://www.youtube.com/watch?v=RgZyX-e6W9E&list=LL&index=2&t=308s```.

Some images are sourced from ```https://github.com/RoboCup-SSL```. More RoboCup-SSL images will be added later for different configurations.
