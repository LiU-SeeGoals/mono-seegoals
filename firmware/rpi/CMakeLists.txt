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
# Source files (equivalent to SRC)
# -------------------------------------------------
file(GLOB SRC "*.c")

# -------------------------------------------------
# Create one binary per .c (matches BINS logic)
# -------------------------------------------------

set(CMAKE_MODULE_PATH ${CMAKE_CURRENT_SOURCE_DIR}/../nanopb/extra)
find_package(Nanopb REQUIRED)

set(NANOPB_GEN ${CMAKE_SOURCE_DIR}/../nanopb/generator/nanopb_generator.py)
set(PROTO_DIR  ${CMAKE_CURRENT_SOURCE_DIR}/../proto)
set(GEN_DIR    ${CMAKE_BINARY_DIR}/generated)

file(MAKE_DIRECTORY ${GEN_DIR})

add_custom_command(
    OUTPUT
        ${GEN_DIR}/imu.pb.c
        ${GEN_DIR}/imu.pb.h

    COMMAND python3
        ${NANOPB_GEN}
        --proto-path=${PROTO_DIR}
        --output-dir=${GEN_DIR}
        ${PROTO_DIR}/imu.proto

    DEPENDS
        ${PROTO_DIR}/imu.proto
        ${NANOPB_GEN}

    COMMENT "Generating nanopb files (python generator)"
)

add_library(proto_generated
    ${GEN_DIR}/imu.pb.c
)

target_include_directories(proto_generated PUBLIC
    ${GEN_DIR}
    ${CMAKE_SOURCE_DIR}/../nanopb
)


foreach(src ${SRC})
    get_filename_component(bin ${src} NAME_WE)
    add_executable(${bin} ${src})

    target_link_libraries(${bin}
        wiringPi
        wiringPiDev
        pthread
        m
        crypt
        rt
    )
    target_link_libraries(${bin} proto_generated nanopb)
endforeach()
