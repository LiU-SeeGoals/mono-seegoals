file(GLOB_RECURSE BASESTATION_SOURCES
    "basestation/AZURE_RTOS/*.*"
    "basestation/Core/*.*"
    "basestation/Drivers/STM32H5xx_HAL_Driver/*.*"
    "basestation/Drivers/BSP/Components/lan8742/*.c"
    "basestation/Middlewares/*.*"
    "basestation/NetXDuo/*.*"
    "shared/src/*.*"
)

set(BASESTATION_LINKER_SCRIPT ${CMAKE_SOURCE_DIR}/basestation/STM32H563ZITX_FLASH.ld)

add_executable(basestation.elf EXCLUDE_FROM_ALL
    ${BASESTATION_SOURCES}
    ${BASESTATION_LINKER_SCRIPT}
)

target_compile_options(basestation.elf PRIVATE
    -mcpu=cortex-m33
)

target_include_directories(basestation.elf PRIVATE
    basestation/Core/Inc
    basestation/AZURE_RTOS/App
    basestation/Drivers/BSP/Components/lan8742
    basestation/Drivers/STM32H5xx_HAL_Driver/Inc
    basestation/Drivers/STM32H5xx_HAL_Driver/Inc/Legacy
    basestation/Drivers/CMSIS/Include
    basestation/Drivers/CMSIS/Device/ST/STM32H5xx/Include
    basestation/NetXDuo/App
    basestation/NetXDuo/Target
    basestation/Middlewares/ST/netxduo/common/drivers/ethernet
    basestation/Middlewares/ST/netxduo/addons/dhcp
    basestation/Middlewares/ST/threadx/common/inc
    basestation/Middlewares/ST/netxduo/common/inc
    basestation/Middlewares/ST/netxduo/ports/cortex_m33/gnu/inc
    basestation/Middlewares/ST/threadx/ports/cortex_m33/gnu/inc
    shared/inc
)

target_compile_definitions(basestation.elf PRIVATE
    DEBUG
    USE_HAL_DRIVER
    STM32H563xx
    NX_INCLUDE_USER_DEFINE_FILE
    TX_INCLUDE_USER_DEFINE_FILE
    TX_SINGLE_MODE_NON_SECURE=1
)

target_link_options(basestation.elf PRIVATE
    -mcpu=cortex-m33
    -mthumb
    -T ${BASESTATION_LINKER_SCRIPT}
    -Wl,-gc-sections,--print-memory-usage,-Map=${PROJECT_BINARY_DIR}/basestation.map
)

add_custom_command(TARGET basestation.elf POST_BUILD
    COMMAND ${CMAKE_OBJCOPY} -Oihex $<TARGET_FILE:basestation.elf> ${PROJECT_BINARY_DIR}/basestation.hex
    COMMAND ${CMAKE_OBJCOPY} -Obinary $<TARGET_FILE:basestation.elf> ${PROJECT_BINARY_DIR}/basestation.bin
    COMMENT "Building basestation.hex and basestation.bin"
)

add_custom_target(basestation DEPENDS basestation.elf)

add_custom_target(flash_basestation DEPENDS basestation.elf
    COMMAND STM32_Programmer_CLI -c port=SWD ap=1 -w ${PROJECT_BINARY_DIR}/basestation.bin 0x08000000 -rst
)
