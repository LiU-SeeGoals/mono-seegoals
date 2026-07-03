#ifndef KICKER_H
#define KICKER_H

#include "main.h"

typedef struct {
    int max_charges_per_kick;
    int charge_wait_us;
    int discharge_wait_us;
    int safe_discharge_wait_us;
    int charges_since_last_kick;
} KICKER_Settings;

typedef enum
{
    KICKER_STRAIGHT,
    KICKER_CHIPPER,
} KickerMode;

typedef enum
{
    KICKER_SPEED_DEFAULT = 0,
    KICKER_SPEED_STRAIGHT_PASS = 1,
    KICKER_SPEED_STRAIGHT_GOAL = 2,
    KICKER_SPEED_CHIP_PASS = 3,
} KickerSpeed;

/**
 * Initalize the kicker subsystem.
 * Curently initializes the log module
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick);

/**
 *
 */
void KICKER_ChargeStart(KickerSpeed kickSpeed);

/**
 *
 */
void KICKER_ChargeStop();

/**
 *
 */
void KICKER_KickStart(KickerSpeed kickSpeed);

/**
 *
 */
void KICKER_KickStop();

void KICKER_KickSafe();

void KICKER_SetKickerMode(KickerMode mode);

/**
 * Returns a reference to the current kicker settings.
 */
KICKER_Settings* KICKER_GetSettings();

#endif /* KICKER_H */
