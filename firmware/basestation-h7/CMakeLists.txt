cmake_minimum_required(VERSION 3.25)

# Project name and language setup
project(basestation C CXX ASM)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_C_STANDARD 11)

# Specify cross-compilers and tools
set(CMAKE_C_COMPILER arm-none-eabi-gcc)
set(CMAKE_CXX_COMPILER arm-none-eabi-g++)
set(CMAKE_ASM_COMPILER  arm-none-eabi-gcc)
set(CMAKE_AR arm-none-eabi-ar)
set(CMAKE_OBJCOPY arm-none-eabi-objcopy)
set(CMAKE_OBJDUMP arm-none-eabi-objdump)
set(SIZE arm-none-eabi-size)

# Debug and optimization setup
set(CMAKE_BUILD_TYPE Debug)
set(OPTIMIZATION_FLAGS -Og)

# Add compile and link options
add_compile_options(-mcpu=cortex-m7 -mthumb -mfpu=fpv5-d16 -mfloat-abi=hard -ffunction-sections -fdata-sections -Wall ${OPTIMIZATION_FLAGS})
add_link_options(-mcpu=cortex-m7 -mthumb -specs=nano.specs -Wl,-Map=${PROJECT_BINARY_DIR}/basestation_CM7.map,--cref -Wl,--gc-sections)

# Add definitions
add_compile_definitions(DEBUG CORE_CM7 USE_HAL_DRIVER STM32H755xx)

# Include directories
include_directories(
    ${CMAKE_SOURCE_DIR}/CM7/LWIP/App
    ${CMAKE_SOURCE_DIR}/CM7/LWIP/Target
    ${CMAKE_SOURCE_DIR}/CM7/Core/Inc
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/system
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Inc
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Inc/Legacy
    ${CMAKE_SOURCE_DIR}/Drivers/BSP/Components/lan8742
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/netif/ppp
    ${CMAKE_SOURCE_DIR}/Drivers/CMSIS/Device/ST/STM32H7xx/Include
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/lwip
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/lwip/apps
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/lwip/priv
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/lwip/prot
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/netif
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/compat/posix
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/compat/posix/arpa
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/compat/posix/net
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/compat/posix/sys
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/src/include/compat/stdc
    ${CMAKE_SOURCE_DIR}/Middlewares/Third_Party/LwIP/system/arch
    ${CMAKE_SOURCE_DIR}/Drivers/CMSIS/Include
)

# Define source files
file(APPEND CM7_SOURCES
    ${CMAKE_SOURCE_DIR}/CM7/LWIP/App/lwip.c
    ${CMAKE_SOURCE_DIR}/CM7/LWIP/Target/ethernetif.c
    ${CMAKE_SOURCE_DIR}/CM7/Core/Src/main.c
    ${CMAKE_SOURCE_DIR}/CM7/Core/Src/stm32h7xx_it.c
    ${CMAKE_SOURCE_DIR}/CM7/Core/Src/stm32h7xx_hal_msp.c
    ${CMAKE_SOURCE_DIR}/Drivers/BSP/Components/lan8742/lan8742.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_eth.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_eth_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_rcc.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_rcc_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_flash.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_flash_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_gpio.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_hsem.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_dma.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_dma_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_mdma.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_pwr.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_pwr_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_cortex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_i2c.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_i2c_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_exti.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_tim.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_tim_ex.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_uart.c
    ${CMAKE_SOURCE_DIR}/Drivers/STM32H7xx_HAL_Driver/Src/stm32h7xx_hal_uart_ex.c
    ${CMAKE_SOURCE_DIR}/Common/Src/system_stm32h7xx.c
    ${CMAKE_SOURCE_DIR}/CM7/Core/Src/sysmem.c
    ${CMAKE_SOURCE_DIR}/CM7/Core/Src/syscalls.c
)

# Add assembly sources
file(GLOB CM7_ASM_SOURCE ${CMAKE_SOURCE_DIR}/Buildfiles/CM7/startup_stm32h755xx_CM7.s)

# Linker script
set(CM7_LINKER_SCRIPT ${CMAKE_SOURCE_DIR}/Buildfiles/CM7/stm32h755xx_flash_CM7.ld)

# Add executable target
add_executable(basestation_CM7.elf ${CM7_SOURCES} ${CM7_ASM_SOURCE})

# Linker options
target_link_options(basestation_CM7.elf PUBLIC -T ${CM7_LINKER_SCRIPT})

# Link libraries
target_link_libraries(basestation_CM7.elf c m nosys)

# Custom commands for creating hex and bin files
add_custom_command(TARGET basestation_CM7.elf POST_BUILD
    COMMAND ${CMAKE_OBJCOPY} -Oihex $<TARGET_FILE:basestation_CM7.elf> ${CMAKE_BINARY_DIR}/basestation_CM7.hex
    COMMAND ${CMAKE_OBJCOPY} -Obinary $<TARGET_FILE:basestation_CM7.elf> ${CMAKE_BINARY_DIR}/basestation_CM7.bin
    COMMENT "Building HEX and BIN files"
)

# Add custom target to flash the MCU
add_custom_target(flash_cm7 DEPENDS ${CMAKE_BINARY_DIR}/basestation_CM7.bin basestation_CM7.elf
    COMMAND STM32_Programmer_CLI -c port=SWD -w ${CMAKE_BINARY_DIR}/basestation_CM7.bin 0x08000000 -rst
)

# Clean target
add_custom_target(clean_all
    COMMAND rm -rf ${CMAKE_BINARY_DIR}
)
