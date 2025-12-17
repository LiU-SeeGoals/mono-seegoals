#!/bin/bash

# Function to start the backend services in the background
start_backends() {
    echo "Starting backend proxy services..."
    node src/backend/sslVisionProxy.cjs &
    node src/backend/gameControllerProxy.cjs &
    node src/backend/AIControllerProxy.cjs &
}

# Start backends regardless of frontend mode
start_backends

# Check if the build directory exists and is not empty
if [ -d "./dist" ] && [ "$(ls -A ./dist)" ]; then
    echo "Build files detected. Starting production host..."
    # 'serve' needs to be installed; using npx to ensure it's available
    npx serve -s dist -l 5173
else
    echo "No build files found. Falling back to development mode (Vite)..."
    pnpm install && pnpm run dev
fi

# Keep the script running to prevent the container from exiting
wait -n