#include "common.h"

void COMMON_Wait(uint64_t us)
{
    uint64_t cycles = ((uint64_t)HAL_RCC_GetSysClockFreq() * us) / 1000000;
    for (volatile uint64_t i = 0; i < cycles; i++);
}
