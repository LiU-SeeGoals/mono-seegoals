set(DRIVERS_ROOT robot/Drivers/STM32H7xx_HAL_Driver/Src)
set(DSP_ROOT robot/Drivers/CMSIS/DSP/Source)
file(GLOB ROBOT_DRIVERS_SOURCE
    ${DRIVERS_ROOT}/stm32h7xx_hal_cortex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_tim.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_tim_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_rcc.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_rcc_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_flash.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_flash_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_gpio.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_hsem.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_dma.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_dma_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_mdma.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_pwr.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_pwr_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_i2c.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_i2c_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_exti.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_uart.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_uart_ex.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_spi.c
    ${DRIVERS_ROOT}/stm32h7xx_hal_spi_ex.c
    ${DSP_ROOT}/ControllerFunctions/arm_sin_cos_f32.c
    ${DSP_ROOT}/CommonTables/arm_common_tables.c
    ${DSP_ROOT}/MatrixFunctions/MatrixFunctions.c
    ${DSP_ROOT}/FilteringFunctions/FilteringFunctions.c
    ${DSP_ROOT}/FastMathFunctions/FastMathFunctions.c
)

file(GLOB_RECURSE ROBOT_COMMON_SOURCES "robot/Common/*.*")
file(GLOB ROBOT_LSM6DSL_SOURCE "robot/Libraries/lsm6dsl-lib/src/HLindgren_LSM6DSL.c")
file(GLOB_RECURSE ROBOT_SHARED_SOURCES "shared/src/*.*")

set(ROBOT_COMMON_INCLUDES
    robot/Drivers/STM32H7xx_HAL_Driver/Inc
    robot/Drivers/STM32H7xx_HAL_Driver/Inc/Legacy
    robot/Drivers/CMSIS/Device/ST/STM32H7xx/Include
    robot/Drivers/CMSIS/Include
    robot/Drivers/CMSIS/DSP/DSP_Lib_TestSuite/RefLibs/inc
    robot/Drivers/CMSIS/DSP/Include
    robot/Libraries/lsm6dsl-lib/include
    shared/inc/
)

set(ROBOT_COMMON_DEFINITIONS
    DEBUG
    USE_HAL_DRIVER
    STM32H755xx
    ARM_MATH_CM4
    ARM_MATH_MATRIX_CHECK
    ARM_MATH_ROUNDING
    LSM6DSL_USE_I2C_MEM_READ_AND_WRITE
)

set(ROBOT_COMMON_COMPILE_OPTIONS
    -mcpu=cortex-m7
    -mfloat-abi=hard
    -mfpu=fpv4-sp-d16
)

set(ROBOT_COMMON_LINK_OPTIONS
    -mcpu=cortex-m7
    -mthumb
    -mfloat-abi=hard
    -mfpu=fpv4-sp-d16
    -specs=nano.specs
    -u_printf_float
)

file(GLOB_RECURSE ROBOT_CM7_SOURCES "robot/CM7/Core/Src/*.*")
file(GLOB ROBOT_CM7_ASM_SOURCE "robot/Buildfiles/CM7/*.s")
set(ROBOT_CM7_LINKER_SCRIPT ${CMAKE_SOURCE_DIR}/robot/Buildfiles/CM7/stm32h755xx_flash_CM7.ld)

add_executable(robot_CM7.elf EXCLUDE_FROM_ALL
    ${ROBOT_CM7_SOURCES}
    ${ROBOT_COMMON_SOURCES}
    ${ROBOT_DRIVERS_SOURCE}
    ${ROBOT_LSM6DSL_SOURCE}
    ${ROBOT_SHARED_SOURCES}
    ${ROBOT_CM7_ASM_SOURCE}
    ${ROBOT_CM7_LINKER_SCRIPT}
)

target_compile_options(robot_CM7.elf PRIVATE ${ROBOT_COMMON_COMPILE_OPTIONS})
target_include_directories(robot_CM7.elf PRIVATE robot/CM7/Core/Inc ${ROBOT_COMMON_INCLUDES})
target_compile_definitions(robot_CM7.elf PRIVATE CORE_CM7 ${ROBOT_COMMON_DEFINITIONS})
target_link_options(robot_CM7.elf PRIVATE
    ${ROBOT_COMMON_LINK_OPTIONS}
    -T${ROBOT_CM7_LINKER_SCRIPT}
    -Wl,-gc-sections,--print-memory-usage,-Map=${PROJECT_BINARY_DIR}/robot_CM7.map
)
target_link_libraries(robot_CM7.elf c m nosys)

# Robot CM4 Target
file(GLOB_RECURSE ROBOT_CM4_SOURCES "robot/CM4/Core/*.*")
file(GLOB ROBOT_CM4_ASM_SOURCE "robot/Buildfiles/CM4/*.s")
set(ROBOT_CM4_LINKER_SCRIPT ${CMAKE_SOURCE_DIR}/robot/Buildfiles/CM4/stm32h755xx_flash_CM4.ld)

add_executable(robot_CM4.elf EXCLUDE_FROM_ALL
    ${ROBOT_CM4_SOURCES}
    ${ROBOT_COMMON_SOURCES}
    ${ROBOT_DRIVERS_SOURCE}
    ${ROBOT_CM4_ASM_SOURCE}
    ${ROBOT_CM4_LINKER_SCRIPT}
)

target_compile_options(robot_CM4.elf PRIVATE ${ROBOT_COMMON_COMPILE_OPTIONS})
target_include_directories(robot_CM4.elf PRIVATE robot/CM4/Core/Inc ${ROBOT_COMMON_INCLUDES})
target_compile_definitions(robot_CM4.elf PRIVATE CORE_CM4 ${ROBOT_COMMON_DEFINITIONS})
target_link_options(robot_CM4.elf PRIVATE
    ${ROBOT_COMMON_LINK_OPTIONS}
    -T${ROBOT_CM4_LINKER_SCRIPT}
    -Wl,-gc-sections,--print-memory-usage,-Map=${PROJECT_BINARY_DIR}/robot_CM4.map
)
target_link_libraries(robot_CM4.elf c m nosys)

# Post-build for both cores
foreach(CORE CM7 CM4)
    add_custom_command(TARGET robot_${CORE}.elf POST_BUILD
        COMMAND ${CMAKE_OBJCOPY} -Oihex $<TARGET_FILE:robot_${CORE}.elf> ${PROJECT_BINARY_DIR}/robot_${CORE}.hex
        COMMAND ${CMAKE_OBJCOPY} -Obinary $<TARGET_FILE:robot_${CORE}.elf> ${PROJECT_BINARY_DIR}/robot_${CORE}.bin
        COMMENT "Building robot_${CORE}.hex and robot_${CORE}.bin"
    )
endforeach()

# Robot convenience target (builds both cores)
add_custom_target(robot DEPENDS robot_CM7.elf robot_CM4.elf)

# Flash targets
add_custom_target(flash_robot DEPENDS robot_CM7.elf
    COMMAND st-flash --reset write ${PROJECT_BINARY_DIR}/robot_CM7.bin 0x08000000
)

add_custom_target(flash_robot_stm32 DEPENDS robot_CM7.elf
    COMMAND STM32_Programmer_CLI -c port=SWD -w ${PROJECT_BINARY_DIR}/robot_CM7.bin 0x08000000 -rst sn 002B00473132511438363431
)
