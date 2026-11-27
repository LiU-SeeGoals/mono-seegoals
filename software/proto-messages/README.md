# Project Proto-Messages
**IMPORTANT**: This directory contains protocol buffer definitions that are shared across multiple repositories in the project. Making changes to the files in this directory may **break the communication** between different modules of the project.

## Protocol Buffer Definitions
The proto files for `ssl_vision`, `gc` (SSL game controller, vendored from RoboCup-SSL/ssl-game-controller), `simulation` and `grsim` are external dependencies and should **not be altered**. These files are based on specifications provided by third-party modules that our project interfaces with. Modifying these proto files would lead to compatibility issues, as changes on our end would not be reflected in the external modules.

## Generating Proto Code
To generate the necessary code from the proto files, you should use the provided scripts.

### For Go
To build the Go code, run the following script from the root of this directory:

```bash
./build_go.sh
```

This needs `protoc` and `protoc-gen-go` v1.34.2 on your PATH. The output in
`proto_go/` is committed, and the controller uses it directly through a
`replace` directive in `software/controller/go.mod`, so rebuild the controller
after regenerating.

### For the GameViewer (TypeScript / JavaScript)
```bash
./build_ts.sh
```

Writes ts-proto classes for raw vision to `GameViewer/src/proto/ssl_vision/`
and a protobufjs JSON bundle for tracked vision to
`GameViewer/src/backend/proto/tracked.json`. Needs `protoc`, `node` and `npm`;
the generator versions are pinned in the script. The output is committed.

### For C
To build the C code, run the following script from the root of this directory:

First we need to build proto-c locally
Download from their release page: https://github.com/protobuf-c/protobuf-c

Install

```
sudo apt install libprotobuf-dev libprotoc-dev protobuf-compiler
```

then build proto c with

```
./configure && make && sudo make install
```

Now you can build proto c files

```bash
./build_c.sh
```

The C code uses the [nanopb](https://github.com/nanopb/nanopb) implementation since it's running on embedded systems.

### For Python
To build the Python code, run the following script from the root of this directory:

```bash
./build_py.sh
```
