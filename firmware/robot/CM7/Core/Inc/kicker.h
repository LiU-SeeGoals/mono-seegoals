#ifndef KICKER_H
#define KICKER_H

#include "main.h"

typedef struct {
    int max_charges_per_kick;
    int charge_wait_us;
    int discharge_wait_us;
    int charges_since_last_kick;
} KICKER_Settings;

/**
 * Initalize the kicker subsystem.
 * Curently initializes the log module
 */
void KICKER_Init();

/**
 * Charges the kicker caps.
 * Curently this is For the time specified in the private settings variable,
 * and frezzes the rest of the program.
 */
void KICKER_Charge();

/**
 * Activates the kicker.
 */
void KICKER_Kick();

/**
 * Returns a reference to the current kicker settings.
 */
KICKER_Settings* KICKER_GetSettings();

#endif /* KICKER_H */
