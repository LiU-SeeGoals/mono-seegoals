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
endforeach()
