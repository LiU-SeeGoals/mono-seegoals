#ifndef DATA_LOGGING_H
#define DATA_LOGGING_H

#include "stm32h7xx_hal_spi.h"
#include <stdbool.h>
#include <stdint.h>
#define DOUBLE_BUFFER 2
#define IMU_DATASIZE 4

typedef struct
{
    uint32_t timestamp;
    float x;
    float y;
    float z;
} Imu;

typedef struct
{
    Imu imu[2];
    uint8_t imu_write_idx;
    volatile bool mutex; // volatile bools are atomic
} DataLog;


void DATA_log_imu_data(float x, float y, float z);
void DATA_uart_send();
void DATA_Init(SPI_HandleTypeDef *hspi);
void DATA_spi_send();

#endif
