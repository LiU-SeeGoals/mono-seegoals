#ifndef TELEM_H
#define TELEM_H

#include "stm32h7xx_hal.h"
#include "stm32h7xx_hal_spi.h"

/** Initialization method for the telemetry subsystem.
 * Takes the given spi handle and uses it to transmit telemetry messages.
 *
 * @param hspi The spi handle to be used for telemetry communication
 */
void TELEM_Init(SPI_HandleTypeDef* hspi);

#endif
