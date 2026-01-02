#ifndef DATA_LOGGING_H
#define DATA_LOGGING_H

#include "stm32h7xx_hal_spi.h"
#include <stdbool.h>
#include <stdint.h>

typedef struct
{
    float u;
    float r;
    float y;
    float e;
} ControlSignal;

void DATA_log_imu_data(float x, float y, float z);
void DATA_log_state(float x, float y, float w);
void DATA_log_motor(ControlSignal m1, ControlSignal m2, ControlSignal m3, ControlSignal m4);
void DATA_log_pos(ControlSignal x, ControlSignal y, ControlSignal angle);
void DATA_log_vision(float x, float y, float w);
void DATA_uart_send();
void DATA_Init(SPI_HandleTypeDef *hspi);
void DATA_spi_send();

#endif
