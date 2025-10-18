# Project Proto-Messages
**IMPORTANT**: This directory contains protocol buffer definitions that are shared across multiple repositories in the project. Making changes to the files in this directory may **break the communication** between different modules of the project.

## Protocol Buffer Definitions
The proto files for `ssl_vision` and `grsim` are external dependencies and should **not be altered**. These files are based on specifications provided by third-party modules that our project interfaces with. Modifying these proto files would lead to compatibility issues, as changes on our end would not be reflected in the external modules.

## Generating Proto Code
To generate the necessary code from the proto files, you should use the provided scripts.

### For Go
To build the Go code, run the following script from the root of this directory:

```bash
./build_go.sh
```

### For C
To build the C code, run the following script from the root of this directory:

```bash
./build_c.sh
```

The C code uses the [nanopb](https://github.com/nanopb/nanopb) implementation since it's running on embedded systems.

### For Python
To build the Python code, run the following script from the root of this directory:

```bash
./build_python.sh
```
