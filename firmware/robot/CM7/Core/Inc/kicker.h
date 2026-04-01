#ifndef KICKER_H
#define KICKER_H

#include "main.h"
#include "stm32h7xx_hal_spi.h"

typedef struct {
    int max_charges_per_kick;
    int charge_wait_us;
    int discharge_wait_us_kicker;
    int discharge_wait_us_chipper;
    int safe_discharge_wait_us;
    int charges_since_last_kick;
} KICKER_Settings;

typedef enum {
    KICKER,
    CHIPPER,
} KICKER_type;

/**
 * Initalize the kicker subsystem.
 * Curently initializes the log module
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick);

/**
 *
 */
void KICKER_ChargeStart();

/**
 *
 */
void KICKER_ChargeStop();

/**
 *
 */
void KICKER_KickStart(KICKER_type type);

/**
 *
 */
void KICKER_KickStop();

void KICKER_KickSafe();

/**
 * Returns a reference to the current kicker settings.
 */
KICKER_Settings* KICKER_GetSettings();

#endif /* KICKER_H */
