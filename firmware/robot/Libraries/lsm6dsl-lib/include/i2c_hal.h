#ifndef I2C_HAL_H
#define I2C_HAL_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif /* __cplusplus */

typedef void Lsm6dsl_I2cPortHandle_t;

void lsm6dsl_i2c_master_init(Lsm6dsl_I2cPortHandle_t *port_handle);

#ifdef LSM6DSL_USE_I2C_MEM_READ_AND_WRITE
void lsm6dsl_i2c_master_read_mem(
    Lsm6dsl_I2cPortHandle_t *port_handle, uint8_t device_address,
    uint8_t mem_address, // Sub addresss within device
    uint8_t *inbuf, uint8_t size);

void lsm6dsl_i2c_master_write_mem(
    Lsm6dsl_I2cPortHandle_t *port_handle, uint8_t device_address,
    uint8_t mem_address, // Sub address within device
    uint8_t *outbuf, uint8_t size);
#else

/* void lsm6dsl_i2c_master_transmit(Lsm6dsl_I2cPortHandle_t *port_handle, */
/*                                  uint8_t device_address, */
/*                                  uint8_t *outbuf, */
/*                                  uint8_t size); */

/* void lsm6dsl_i2c_master_receive(Lsm6dsl_I2cPortHandle_t *port_handle, */
/*                                 uint8_t device_address, */
/*                                 uint8_t *inbuf, */
/*                                 uint8_t size); */

void lsm6dsl_i2c_master_transmit_first_frame(
    Lsm6dsl_I2cPortHandle_t *port_handle, uint8_t device_address,
    uint8_t *outbuf, uint8_t size);

/* void lsm6dsl_i2c_master_transmit_next_frame(Lsm6dsl_I2cPortHandle_t
 * *port_handle, uint8_t device_address, uint8_t *outbuf, uint8_t size); */

void lsm6dsl_i2c_master_transmit_last_frame(
    Lsm6dsl_I2cPortHandle_t *port_handle, uint8_t device_address,
    uint8_t *outbuf, uint8_t size);

/* void lsm6dsl_i2c_master_receive_first_frame(Lsm6dsl_I2cPortHandle_t
 * *port_handle, uint8_t device_address, uint8_t *inbuf, uint8_t size); */

/* void lsm6dsl_i2c_master_receive_next_frame(Lsm6dsl_I2cPortHandle_t
 * *port_handle, uint8_t device_address, uint8_t *inbuf, uint8_t size); */

void lsm6dsl_i2c_master_receive_last_frame(Lsm6dsl_I2cPortHandle_t *port_handle,
                                           uint8_t device_address,
                                           uint8_t *inbuf, uint8_t size);
#endif

#ifdef __cplusplus
}
#endif /* __cplusplus */

#endif // I2C_HAL_H
