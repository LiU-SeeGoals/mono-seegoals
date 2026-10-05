#include "telem.h"
#include "stm32h7xx_hal_spi.h"

static SPI_HandleTypeDef* spi_handle;

void TELEM_Init(SPI_HandleTypeDef* hspi) {
    spi_handle = hspi;
}

