#!/bin/bash
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"

OUTPUT_SRC="${ROOT_DIR}/../../firmware/shared/src"
OUTPUT_INC="${ROOT_DIR}/../../firmware/shared/inc"

generate_c_code() {
	local dir_path="${ROOT_DIR}/$1"
	local temp_dir="${ROOT_DIR}/temp_proto"
	
	mkdir -p "${temp_dir}"
	mkdir -p "${OUTPUT_SRC}"
	mkdir -p "${OUTPUT_INC}"
	
	for PROTO_FILE in "${dir_path}"/*.proto; do
		protoc --c_out="${temp_dir}" --proto_path="${dir_path}" --proto_path="$PWD" "${PROTO_FILE}"
		
		find "${temp_dir}" -name "*.h" -exec sed -i 's|#include "robot_action/robot_action.pb-c.h"|#include "robot_action.pb-c.h"|g' {} \;
		find "${temp_dir}" -name "*.h" -exec sed -i 's|#include <protobuf-c/protobuf-c.h>|#include "protobuf-c.h"|g' {} \;
		find "${temp_dir}" -name "*.c" -exec sed -i 's|#include "robot_action/robot_action.pb-c.h"|#include "robot_action.pb-c.h"|g' {} \;
		find "${temp_dir}" -name "*.c" -exec sed -i 's|#include <protobuf-c/protobuf-c.h>|#include "protobuf-c.h"|g' {} \;
		
		find "${temp_dir}" -name "*.c" -exec mv {} "${OUTPUT_SRC}/" \;
		find "${temp_dir}" -name "*.h" -exec mv {} "${OUTPUT_INC}/" \;
	done
	
	rm -rf "${temp_dir}"
}

echo "Generating C code for SSL Vision proto files..."
generate_c_code "parsed_vision"

echo "Generating C code for Robot Action proto files..."
generate_c_code "robot_action"

echo "Generated files placed in:"
echo "  Headers: ${OUTPUT_INC}"
echo "  Sources: ${OUTPUT_SRC}"
