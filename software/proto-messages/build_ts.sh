#!/bin/bash
# Regenerates GameViewer's protobuf code from the .proto files here:
#   - src/proto/ssl_vision/*.ts      ts-proto classes for the React frontend
#   - src/backend/proto/tracked.json protobufjs bundle for trackedVision.cjs
# Needs protoc, node and npm. The generator versions are pinned below and
# installed into .ts-tools/ (gitignored) on first run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
VIEWER_DIR="${SCRIPT_DIR}/../GameViewer"
TOOLS_DIR="${SCRIPT_DIR}/.ts-tools"
TS_PROTO_VERSION=1.181.1
PROTOBUFJS_VERSION=7.4.0 # keep in sync with GameViewer/package.json

if [ ! -x "${TOOLS_DIR}/node_modules/.bin/protoc-gen-ts_proto" ] ||
	! grep -q "\"version\": \"${TS_PROTO_VERSION}\"" "${TOOLS_DIR}/node_modules/ts-proto/package.json"; then
	mkdir -p "${TOOLS_DIR}"
	npm install --prefix "${TOOLS_DIR}" --no-audit --no-fund --silent \
		"ts-proto@${TS_PROTO_VERSION}" "protobufjs@${PROTOBUFJS_VERSION}"
fi

cd "${SCRIPT_DIR}"

# Frontend: raw vision packets (SSL_WrapperPacket and what it contains).
TS_OUT="${VIEWER_DIR}/src/proto"
rm -rf "${TS_OUT}/ssl_vision"
protoc --plugin=protoc-gen-ts_proto="${TOOLS_DIR}/node_modules/.bin/protoc-gen-ts_proto" \
	--proto_path="${SCRIPT_DIR}" --ts_proto_out="${TS_OUT}" \
	ssl_vision/ssl_wrapper.proto ssl_vision/ssl_vision_detection.proto ssl_vision/ssl_vision_geometry.proto

# Backend: tracked vision packets, resolved with their gc/ imports.
NODE_PATH="${TOOLS_DIR}/node_modules" node - "${SCRIPT_DIR}" "${VIEWER_DIR}/src/backend/proto/tracked.json" <<'EOF'
const fs = require('fs');
const path = require('path');
const protobuf = require('protobufjs');
const [protoDir, outFile] = process.argv.slice(2);
const root = new protobuf.Root();
root.resolvePath = (_origin, target) => path.join(protoDir, target);
root.loadSync('ssl_vision/ssl_wrapper_tracked.proto');
fs.mkdirSync(path.dirname(outFile), { recursive: true });
fs.writeFileSync(outFile,
  '{"//": "Generated from software/proto-messages by build_ts.sh. DO NOT EDIT.",\n' +
  ' "root": ' + JSON.stringify(root.toJSON()) + '}\n');
EOF

echo "GameViewer proto code built in ${TS_OUT}/ssl_vision and src/backend/proto/tracked.json"
