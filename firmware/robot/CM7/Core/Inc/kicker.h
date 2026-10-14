#ifndef KICKER_H
#define KICKER_H

#include "main.h"
#include <stdbool.h>

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
 * Result of a kick request.
 */
typedef enum
{
    KICKER_STATUS_OK = 0,
    KICKER_STATUS_NOT_READY,
    KICKER_STATUS_BUSY,
} KICKER_Status;

/**
 * Initialize the kicker subsystem.
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick);

/**
 * Configure how the next kick should behave. The parameters are only
 * applied when KICKER_KickStart() is called, keeping the charge and
 * kick routines separate.
 *
 * @param mode         KICKER_STRAIGHT or KICKER_CHIPPER.
 * @param speed        Requested kick strength.
 * @param use_ir_sensor Enable IR-sensor triggered firing (reserved).
 */
void KICKER_SetKickConfig(KickerMode mode, KickerSpeed speed, bool use_ir_sensor);

/**
 * Begin charging the kicker to full voltage. Charging runs in the
 * background so the capacitor bank is kept ready to kick at request.
 */
void KICKER_ChargeStart(void);

/**
 * Called by the charge timer interrupt when the capacitor bank is charged.
 */
void KICKER_ChargeComplete(void);

/**
 * Fire a kick with the currently configured mode and speed.
 *
 * The kick is fired immediately if the kicker is charged. If it is not,
 * the kick is rejected and KICKER_STATUS_NOT_READY is returned. Charging
 * continues so a later call can succeed.
 *
 * @return KICKER_STATUS_OK if the kick was fired, otherwise an error code.
 */
KICKER_Status KICKER_KickStart(void);

/**
 * Called by the discharge timer interrupt when the kick is done. Restarts
 * charging so the kicker is ready for the next kick.
 */
void KICKER_KickStop(void);

/**
 * Safely dump the capacitor charge without kicking.
 */
void KICKER_KickSafe();

/**
 * @return The current state of the kicker.
 */
KICKER_Status KICKER_GetStatus(void);

/**
 * @return True if the kicker is charged and able to kick.
 */
bool KICKER_IsCharged(void);

/**
 * Returns a reference to the current kicker settings.
 */
KICKER_Settings* KICKER_GetSettings();

#endif /* KICKER_H */
