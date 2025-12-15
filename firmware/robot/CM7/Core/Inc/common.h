#ifndef COMMON_H
#define COMMON_H

#include "main.h"
#include <stdint.h>

void COMMON_Wait(uint64_t us);
void COMMON_buzzer_warning_with_delay();

void COMMON_buzzer_done();
void COMMON_buzzer_nokia();

#endif /* COMMON_H */
