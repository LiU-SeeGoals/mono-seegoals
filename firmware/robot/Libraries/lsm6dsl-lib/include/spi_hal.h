#ifndef SPI_HAL_H
#define SPI_HAL_H

#include <stdint.h>

#ifdef __cplusplus
extern "C"
{
#endif /* __cplusplus */

typedef void Lsm6dsl_SpiPortHandle_t;

void lsm6dsl_spi_master_init(Lsm6dsl_SpiPortHandle_t *port_handle);

void lsm6dsl_spi_read_mem(Lsm6dsl_SpiPortHandle_t *port_handle, uint8_t reg_address, uint8_t *inbuf, uint8_t size);
void lsm6dsl_spi_write_mem(Lsm6dsl_SpiPortHandle_t *port_handle, uint8_t reg_address, uint8_t *outbuf, uint8_t size);

#ifdef __cplusplus
}
#endif /* __cplusplus */

#endif // SPI_HAL_H

