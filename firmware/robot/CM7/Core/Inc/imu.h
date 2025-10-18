#ifndef INC_H
#define INC_H

/* Public includes */
#include "arm_math.h"
#include "stm32h7xx_hal.h"
#include <HLindgren_LSM6DSL.h>
#include <stdint.h>

/* Public types */
typedef struct {
    float x;
    float y;
    float z;

} IMU_AccelVec3;

typedef IMU_AccelVec3 IMU_GyroVec3;

/* Public constants */
#define IMU_I2C_DEV_ADDR 0x6B
#define IMU_DATA_RATE Lsm6dsl_DataRate_1_66kHz
#define IMU_ACCEL_FULL_SCALE Lsm6dsl_FullScaleXl_16g
#define IMU_ACCEL_FULL_SCALE_VAL 16.0f
#define IMU_GYRO_FULL_SCALE Lsm6dsl_FullScaleGyro_2000dps
#define IMU_GYRO_FULL_SCALE_VAL 2000.0f
#define IMU_GRAVITY_CONSTANT 9.820665f

/* Public macros */
#define IMU_RAW_TO_RADPS(raw_gyro_val) (PI * ((float)raw_gyro_val * IMU_GYRO_FULL_SCALE_VAL / (float)INT16_MAX) / 180.0f)
#define IMU_RAW_TO_DPS(raw_gyro_val) ((float)raw_gyro_val * IMU_GYRO_FULL_SCALE_VAL / (float)INT16_MAX)
#define IMU_RAW_TO_G(raw_accel_val) ((float)raw_accel_val * IMU_ACCEL_FULL_SCALE_VAL / (float)INT16_MAX)
#define IMU_RAW_TO_MPS2(raw_accel_val) ((float)raw_accel_val * IMU_ACCEL_FULL_SCALE_VAL * IMU_GRAVITY_CONSTANT / (float)INT16_MAX)

/* Public function declarations */

/**
 * Initalize the IMU at the passed I2C handle
 *
 */
void IMU_Init(I2C_HandleTypeDef* hi2c);
int32_t IMU_read_fifo_raw(Lsm6dsl_Data_t* buf, uint16_t n_blocks_to_read);
Lsm6dsl_AccelData_t IMU_read_accel_raw();
Lsm6dsl_GyroData_t IMU_read_gyro_raw();

/**
 * Retruns a vector IMU acceleration scaled to g-force
 * @return A vector of accelerations [ax, ay, az]
 */
IMU_AccelVec3 IMU_read_accel_g(); // Scaled to g-force units

/**
 * Retruns a vector IMU acceleration scaled in m/s^2
 * @return A vector of accelerations [ax, ay, az]
 */
IMU_AccelVec3 IMU_read_accel_mps2(); // Scaled to m/s^2

/**
 * Returns a vector of angular velocity in radians per second for each axis
 * @return Vector of anglular velocities [vx, vy, vz]
 */
IMU_GyroVec3 IMU_read_gyro_radps();

/**
 * Returns a vector of angular velocity in degrees per second for each axis
 * @return Vector of anglular velocities [vx, vy, vz]
 */
IMU_GyroVec3 IMU_read_gyro_dps();

/**
 * Output pretty complete IMU data to the log.
 */
void IMU_test();

#endif /* INC_H */
