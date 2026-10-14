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
 * Initalize the kicker subsystem.
 * Curently initializes the log module
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick, SPI_HandleTypeDef* hspi5);

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
void KICKER_KickStart();

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

/**
 * @brief Starts an SPI transfer to the KICKER_Board requesting the voltage charge in the Capacitors 
 * @return Success codes:
  0: Transfer started.
  1: No transfer started due to already transferring on SPI5
  -1: Something very bad happened. You are allowed to panic with this return code.
 */
int KICKER_StartVoltageMeasurement();

/**
 * @brief The SPI callback function that the kicker is on which saves the ADC convert value and lowers the pin.
 * @param hspi currently SPI5 only but is the same as the SPI was started on 
 */
void KICKER_SPI_TxRxCpltCallback(SPI_HandleTypeDef *hspi);


/**
 * @brief If the SPI failed for some reason we still have to set the pin to high
 * @param hspi currently SPI5 only but is the same as the SPI was started on 
 */
void KICKER_SPI_ErrorCallback(SPI_HandleTypeDef *hspi);

/**
 * @brief Atomically tries to access the adc convert value.
 * @note After getting the adc value the function cannot be called again until a new value is transferred
 * @note since the value isn't available until the SPI is done using while(TryADC){} is needed immmediately after calling the start function
 * @param adc reference to where the adc value should get placed
 * @return true if an adc value was gotten. false if not
 */
bool KICKER_TryGetADC(uint16_t *adc);

#endif /* KICKER_H */
