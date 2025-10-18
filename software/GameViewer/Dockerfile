# Use an official Node.js runtime as a parent image
FROM node:18

# Set the working directory in the container
WORKDIR /var/GameViewer

# pnpm requires git in some cases, depending on your dependencies
RUN apt-get update && apt-get install -y git

# Install pnpm
RUN npm install -g pnpm

COPY .bash_history /root/.bash_history
# Copy package.json and pnpm-lock.yaml to the working directory
# This can help to cache the installed dependencies if these files don't change
COPY package.json pnpm-lock.yaml ./

# Install project dependencies
RUN pnpm install

# Copy the rest of the application's code to the container
COPY . .

# The port your app will run on
EXPOSE 5173

# Command to run your app using pnpm
CMD ["pnpm", "run", "dev"]
