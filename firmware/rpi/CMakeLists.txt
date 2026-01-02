cmake_minimum_required(VERSION 3.16)

project(example C)

# -------------------------------------------------
# Build type (maps DEBUG / RELEASE)
# -------------------------------------------------
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

# -------------------------------------------------
# Compiler
# -------------------------------------------------
set(CMAKE_C_COMPILER gcc)

# -------------------------------------------------
# Compile options (matches CFLAGS)
# -------------------------------------------------
add_compile_options(
    -Wall
    -Winline
    -pipe
)

# Optimization / debug flags
set(CMAKE_C_FLAGS_RELEASE "-O3")
set(CMAKE_C_FLAGS_DEBUG   "-g -O0")

# -------------------------------------------------
# Include paths
# -------------------------------------------------
include_directories(
    /usr/local/include
)

# -------------------------------------------------
# Common functions for executable and dynamic lib
# -------------------------------------------------

function(link_libs target)
    target_link_libraries(${target}
        wiringPi
        wiringPiDev
        pthread
        m
        crypt
        rt
    )
    target_link_libraries(${target} proto_generated nanopb)
endfunction()

set(CMAKE_MODULE_PATH ${CMAKE_CURRENT_SOURCE_DIR}/../nanopb/extra)
find_package(Nanopb REQUIRED)

set(NANOPB_GEN ${CMAKE_SOURCE_DIR}/../nanopb/generator/nanopb_generator.py)
set(PROTO_DIR  ${CMAKE_CURRENT_SOURCE_DIR}/../proto)
set(GEN_DIR    ${CMAKE_BINARY_DIR}/generated)

file(MAKE_DIRECTORY ${GEN_DIR})

# Generate protobuf files

add_custom_command(
    OUTPUT
        ${GEN_DIR}/imu.pb.c
        ${GEN_DIR}/imu.pb.h
        ${GEN_DIR}/imu_pb2.py

    COMMAND python3
        ${NANOPB_GEN}
        --proto-path=${PROTO_DIR}
        --output-dir=${GEN_DIR}
        ${PROTO_DIR}/imu.proto

    COMMAND protoc
        --proto_path=${PROTO_DIR}
        --python_out=${GEN_DIR}
        ${PROTO_DIR}/imu.proto

    DEPENDS
        ${PROTO_DIR}/imu.proto
        ${NANOPB_GEN}

    COMMENT "Generating nanopb C files and Python protobuf"
)

add_library(proto_generated
    ${GEN_DIR}/imu.pb.c
)

target_include_directories(proto_generated PUBLIC
    ${GEN_DIR}
    ${CMAKE_SOURCE_DIR}/../nanopb
)


get_filename_component(bin spiSpeed.c NAME_WE)
add_executable(${bin} spiSpeed.c)
add_library(spi SHARED spiSpeed.c)

link_libs(${bin})
link_libs(spi)
