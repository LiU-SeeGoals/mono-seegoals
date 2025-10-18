#ifndef LSM6DSL_H
#define LSM6DSL_H

#include <stdbool.h>
#include <stdint.h>

#include "i2c_hal.h"
#include "register_defs.h"
#include "spi_hal.h"

#ifdef __cplusplus
extern "C" {
#endif /* __cplusplus */

typedef struct {
#ifdef USE_SPI
  Lsm6dsl_SpiPortHandle_t *port_handle;
#else
  Lsm6dsl_I2cPortHandle_t *port_handle;
  uint8_t device_address;
#endif
} Lsm6dsl_Device_t;

/**** Low-level API
 * *****************************************************************/

void lsm6dsl_open_i2c_slave_device(Lsm6dsl_I2cPortHandle_t *port_handle,
                                   uint8_t device_address,
                                   Lsm6dsl_Device_t *device);

void lsm6dsl_read_register(Lsm6dsl_Device_t *device, uint8_t reg_address,
                           uint8_t *inbuf, uint8_t size);

void lsm6dsl_write_register(Lsm6dsl_Device_t *device, uint8_t reg_address,
                            uint8_t *outbuf, uint8_t size);

/************************************************************************************/

/**** High-level API
 * ****************************************************************/

#define LSM6DSL_GYRO_DATA_SIZE 6
typedef struct {
  int16_t x;
  int16_t y;
  int16_t z;
} Lsm6dsl_GyroData_t;

#define LSM6DSL_ACCEL_DATA_SIZE 6
typedef struct {
  int16_t x;
  int16_t y;
  int16_t z;
} Lsm6dsl_AccelData_t;

#define LSM6DSL_FIFO_DATA_WORDS 12 // 4 Data sets of 3 16-bit words each
#define LSM6DSL_FIFO_DATA_SIZE (LSM6DSL_FIFO_DATA_WORDS * 2)
typedef struct {
  Lsm6dsl_GyroData_t gyro;
  Lsm6dsl_AccelData_t accel;

  uint8_t sensorhub_1_to_6[6];

  /* union { */
  uint8_t sensorhub_7_to_12[6];
  /* TODO:
   *
   * Include alternative data sets in union
   *
   * Datasheet page 36 (5.5.6):
   * The 4th FIFO data set can be alternately associated to the external
   * sensor data stored in the registers from SENSORHUB7_REG (34h) to
   * SENSORHUB12_REG (39h), to the step counter and timestamp info, or to
   * the temperature sensor data.
   */
  /* }; */

} Lsm6dsl_Data_t;

typedef enum {
  Lsm6dsl_DataRate_12_5Hz,
  Lsm6dsl_DataRate_26Hz,
  Lsm6dsl_DataRate_52Hz,
  Lsm6dsl_DataRate_104Hz,
  Lsm6dsl_DataRate_208Hz,
  Lsm6dsl_DataRate_416Hz,
  Lsm6dsl_DataRate_833Hz,
  Lsm6dsl_DataRate_1_66kHz,
  Lsm6dsl_DataRate_3_33kHz,
  Lsm6dsl_DataRate_6_66kHz,
} Lsm6dsl_DataRate_e;

void lsm6dsl_accel_init(Lsm6dsl_Device_t *device,
                        Lsm6dsl_DataRate_e accel_data_rate,
                        Lsm6dsl_FullScaleXl_e full_scale);

void lsm6dsl_gyro_init(Lsm6dsl_Device_t *device,
                       Lsm6dsl_DataRate_e gyro_data_rate,
                       Lsm6dsl_FullScaleGyro_e full_scale);

Lsm6dsl_GyroData_t lsm6dsl_gyro_read(Lsm6dsl_Device_t *device);
Lsm6dsl_AccelData_t lsm6dsl_accel_read(Lsm6dsl_Device_t *device);

void lsm6dsl_fifo_init(Lsm6dsl_Device_t *device, Lsm6dsl_FifoMode_e fifo_mode,
                       Lsm6dsl_DataRate_e data_rate);

void lsm6dsl_fifo_set_decimation(Lsm6dsl_Device_t *device,
                                 Lsm6dsl_DecFifoGyro_e gyro_dec,
                                 Lsm6dsl_DecFifoXl_e accel_dec);

int32_t lsm6dsl_fifo_read_to_end(Lsm6dsl_Device_t *device,
                                 Lsm6dsl_Data_t *databuf, uint16_t nmemb);
void lsm6dsl_fifo_read_if_watermark(Lsm6dsl_Device_t *device,
                                    Lsm6dsl_Data_t *inbuf, uint16_t size);

void lsm6dsl_fifo_set_threshold(Lsm6dsl_Device_t *device,
                                uint16_t threshold); // TODO: Implement

void lsm6dsl_config_int1(Lsm6dsl_Device_t *device,
                         uint8_t flags); // TODO: Implement
void lsm6dsl_config_int2(Lsm6dsl_Device_t *device,
                         uint8_t flags); // TODO: Implement

#ifdef __cplusplus
}
#endif /* __cplusplus */

#endif // LSM6DSL_H
