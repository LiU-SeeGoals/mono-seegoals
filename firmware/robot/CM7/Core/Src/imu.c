/* Private includes */
#include "imu.h"
#include "arm_math.h"

#include <stdint.h>

#include "log.h"

#include "register_defs.h"
#include "stm32h7xx_hal.h"
#include "stm32h7xx_hal_i2c.h"

#include <HLindgren_LSM6DSL.h>

/* Private variables */
static LOG_Module internal_log_mod;
static Lsm6dsl_Device_t imu_device_handle;

/* HAL wrapper functions for driver LSM6DSL */

void lsm6dsl_i2c_master_init(Lsm6dsl_I2cPortHandle_t* port_handle)
{
    // Port Initialization is handled in main.c
}

void lsm6dsl_i2c_master_read_mem(Lsm6dsl_I2cPortHandle_t* port_handle,
                                 uint8_t device_address,
                                 uint8_t mem_address, // Sub addresss within device
                                 uint8_t* inbuf,
                                 uint8_t size)
{

    HAL_StatusTypeDef status = HAL_I2C_Mem_Read(port_handle, device_address << 1, mem_address, I2C_MEMADD_SIZE_8BIT, inbuf, size, 500);

    if (status != HAL_OK) {
        LOG_DEBUG("Failed to read %u byte(s) from IMU register address 0x%X\r\n", size, mem_address);

    } else {
        LOG_TRACE("Read %u byte(s) from IMU register address 0x%X\r\n", size, mem_address);
    }
}

void lsm6dsl_i2c_master_write_mem(Lsm6dsl_I2cPortHandle_t* port_handle,
                                  uint8_t device_address,
                                  uint8_t mem_address, // Sub address within device
                                  uint8_t* outbuf,
                                  uint8_t size)
{

    HAL_StatusTypeDef status = HAL_I2C_Mem_Write(port_handle, device_address << 1, mem_address, I2C_MEMADD_SIZE_8BIT, outbuf, size, 500);

    if (status != HAL_OK) {
        LOG_ERROR("Failed to write %u byte(s) to IMU register address 0x%X\r\n", size, mem_address);

    } else {
        LOG_TRACE("Wrote %u byte(s) from IMU register address 0x%X\r\n", size, mem_address);
    }
}

/* Private defines */
#define IMU_MAX_INIT_RETRIES 20

/* Private functions declarations */

/*
 * Public functions implementations
 */
void IMU_Init(I2C_HandleTypeDef* hi2c)
{
    LOG_InitModule(&internal_log_mod, "IMU", LOG_LEVEL_INFO, 0);

    LOG_INFO("Initializing IMU...\r\n");

    lsm6dsl_open_i2c_slave_device(hi2c, IMU_I2C_DEV_ADDR, &imu_device_handle);

    for (int retries = 0; retries < IMU_MAX_INIT_RETRIES; retries++) {
        uint8_t who_am_i = 0;

        LOG_DEBUG("Polling WHO_AM_I register...\r\n");
        lsm6dsl_read_register(&imu_device_handle, LSM6DSL_REG_WHO_AM_I, &who_am_i, 1);

        LOG_DEBUG("WHO_AM_I: 0x%X\r\n", who_am_i);

        if (who_am_i == 0x6A) {
            LOG_INFO("IMU is responding\r\n");
            break;

        } else {
            LOG_DEBUG("No or incorrect response from IMU, retrying in 1s...\r\n");
            HAL_Delay(1000);
        }
    }

    LOG_INFO("Configuring IMU...\r\n");
    lsm6dsl_accel_init(&imu_device_handle, IMU_DATA_RATE, IMU_ACCEL_FULL_SCALE);
    lsm6dsl_gyro_init(&imu_device_handle, IMU_DATA_RATE, IMU_GYRO_FULL_SCALE);
    lsm6dsl_fifo_init(&imu_device_handle, Lsm6dsl_FifoMode_Continuous, IMU_DATA_RATE);
    lsm6dsl_fifo_set_decimation(&imu_device_handle, Lsm6dsl_DecFifoGyro_NoDec, Lsm6dsl_DecFifoXl_NoDec);

    LOG_INFO("Done initializing IMU\r\n");
}

int32_t IMU_read_fifo_raw(Lsm6dsl_Data_t* buf, uint16_t n_blocks_to_read)
{
    LOG_DEBUG("Reading %u blocks from FIFO...\r\n", n_blocks_to_read);
    int32_t blocks_read = lsm6dsl_fifo_read_to_end(&imu_device_handle, buf, n_blocks_to_read);

    if (blocks_read >= 0) {
        LOG_DEBUG("Read %u blocks from FIFO\r\n", blocks_read);

        return blocks_read;

    } else {
        LOG_ERROR("Got error code %d when reading FIFO\r\n", blocks_read);

        return blocks_read;
    }
}

Lsm6dsl_AccelData_t IMU_read_accel_raw() { return lsm6dsl_accel_read(&imu_device_handle); }

Lsm6dsl_GyroData_t IMU_read_gyro_raw() { return lsm6dsl_gyro_read(&imu_device_handle); }

IMU_AccelVec3 IMU_read_accel_g()
{
    Lsm6dsl_AccelData_t accel_data = IMU_read_accel_raw();

    return (IMU_AccelVec3){.x = IMU_RAW_TO_G(accel_data.x), .y = IMU_RAW_TO_G(accel_data.y), .z = IMU_RAW_TO_G(accel_data.z)};
}

IMU_AccelVec3 IMU_read_accel_mps2()
{
    Lsm6dsl_AccelData_t accel_data = IMU_read_accel_raw();

    return (IMU_AccelVec3){.x = IMU_RAW_TO_MPS2(accel_data.x), .y = IMU_RAW_TO_MPS2(accel_data.y), .z = IMU_RAW_TO_MPS2(accel_data.z)};
}

IMU_GyroVec3 IMU_read_gyro_radps()
{
    Lsm6dsl_GyroData_t gyro_data = IMU_read_gyro_raw();

    return (IMU_GyroVec3){.x = IMU_RAW_TO_RADPS(gyro_data.x), .y = IMU_RAW_TO_RADPS(gyro_data.y), .z = IMU_RAW_TO_RADPS(gyro_data.z)};
}

IMU_GyroVec3 IMU_read_gyro_dps()
{
    Lsm6dsl_GyroData_t gyro_data = IMU_read_gyro_raw();

    return (IMU_GyroVec3){.x = IMU_RAW_TO_DPS(gyro_data.x), .y = IMU_RAW_TO_DPS(gyro_data.y), .z = IMU_RAW_TO_DPS(gyro_data.z)};
}

void IMU_test()
{
    LOG_INFO("Reading accelerometer value ...\n");

    IMU_AccelVec3 acc = IMU_read_accel_g();
    LOG_INFO("X=%f Y=%f Z=%f [g]\n", acc.x, acc.y, acc.z);
    acc = IMU_read_accel_mps2();
    LOG_INFO("X=%f Y=%f Z=%f [m/s^2]\n", acc.x, acc.y, acc.z);

    LOG_INFO("Reading gyroscope value ...\n");

    IMU_GyroVec3 gyr = IMU_read_gyro_dps();
    LOG_INFO("X=%f Y=%f Z=%f [dps]\n", acc.x, acc.y, acc.z);

    Lsm6dsl_Data_t imu_buf[200];

    LOG_INFO("Reading FIFO until it is empty...\n");
    int32_t blocks_read = IMU_read_fifo_raw(imu_buf, 200);

    if (blocks_read > 0) {
        for (int32_t i = 0; i < 10 /* blocks_read */; i += 1) {
            LOG_INFO("Gyro X=%f Y=%f Z=%f [dps] \n", IMU_RAW_TO_DPS(imu_buf[i].gyro.x), IMU_RAW_TO_DPS(imu_buf[i].gyro.y), IMU_RAW_TO_DPS(imu_buf[i].gyro.z));

            LOG_INFO("Accel X=%f Y=%f Z=%f [g]\n", IMU_RAW_TO_G(imu_buf[i].accel.x), IMU_RAW_TO_G(imu_buf[i].accel.x), IMU_RAW_TO_G(imu_buf[i].accel.x));
        }

        LOG_INFO("... Truncating after 10 blocks\n");
    }
}
