#!/bin/bash

# Define the root directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

# Function to generate Go code from proto files in a directory
generate_go_code() {
	local dir_name="$1"
	local dir_path="${SCRIPT_DIR}/${dir_name}" # Corrected to include subdirectory
	local output_dir="${SCRIPT_DIR}/proto_go"  # Output in subdirectories

	# Ensure the output directory exists
	mkdir -p "${output_dir}"

	# Check if the source directory exists
	if [ -d "${dir_path}" ]; then
		# Find all .proto files and generate Go code
		find "${dir_path}" -name '*.proto' -exec protoc --proto_path="${SCRIPT_DIR}" --go_out="${output_dir}" --go_opt=paths=source_relative {} +
	else
		echo "Warning: Directory ${dir_path} does not exist."
	fi
}

# Generate Go code for proto files in the grsim, robot_action, and ssl_vision directories
generate_go_code "simulation"
generate_go_code "robot_action"
generate_go_code "parsed_vision"
generate_go_code "ssl_vision"
generate_go_code "grsim"

echo "Proto files built in ${SCRIPT_DIR}/proto_go"
