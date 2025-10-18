#!/bin/bash

# Define the root directory where the script is located
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

# Define the directories containing the .proto files
SSL_VISION_DIR="${ROOT_DIR}/ssl_vision"
ACTION_DIR="${ROOT_DIR}/robot_action"

# Create the directories if they don't already exist
mkdir -p "${SSL_VISION_DIR}"
mkdir -p "${ACTION_DIR}"

# Function to generate C code from proto files in a directory
generate_py_code() {
	local dir_path="$1"
	for PROTO_FILE in "${dir_path}"/*.proto; do
		protoc --python_out="${dir_path}" --proto_path="${dir_path}" "${PROTO_FILE}"
	done
}

# Generate Python code for each directory
echo "Generating Python code for SSL Vision proto files..."
generate_py_code "${SSL_VISION_DIR}"

echo "Generating Python code for Robot Action proto files..."
generate_py_code "${ACTION_DIR}"
