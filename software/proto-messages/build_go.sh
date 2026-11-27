#!/bin/bash
# Regenerates proto_go/ (Go module github.com/LiU-SeeGoals/proto_go) from the
# .proto files in this directory. Needs protoc and protoc-gen-go v1.34.2.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
OUT_DIR="${SCRIPT_DIR}/proto_go"
DIRS=(gc grsim parsed_vision robot_action simulation ssl_vision)

cd "${SCRIPT_DIR}"
for dir in "${DIRS[@]}"; do
	rm -f "${OUT_DIR}/${dir}"/*.pb.go
done

# One protoc call so imports across directories (e.g. ssl_vision -> gc) resolve.
protos=()
for dir in "${DIRS[@]}"; do
	protos+=("${dir}"/*.proto)
done
protoc --proto_path="${SCRIPT_DIR}" --go_out="${OUT_DIR}" --go_opt=paths=source_relative "${protos[@]}"

echo "Proto files built in ${OUT_DIR}"
