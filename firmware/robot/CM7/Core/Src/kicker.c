#include "kicker.h"

/* Private includes */
#include "common.h"
#include "log.h"
#include <stdbool.h>

/* Private defines */
// ...

#define KICK_STRAIGHT_PASS_US 250000
#define KICK_STRAIGHT_GOAL_US 450000
#define KICK_CHIP_PASS_US 650000

/* Private variables */
// Kicker tuning Chip discharge: 650000. Kicker tuning straight discharge: 650000
// Kicker tuning straight goal kick: 350000. kicker straight pass: 250000
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {.max_charges_per_kick = 6, .safe_discharge_wait_us=15, .charge_wait_us = 350000, .discharge_wait_us = 300, .charges_since_last_kick = 0};
static volatile bool charging = false;
static volatile bool kicking = false;
static TIM_HandleTypeDef* htim_kicker_charge;
static TIM_HandleTypeDef* htim_kicker_kick;
static volatile KickerMode kickerMode;

/*
 * Private function declaration
 */

int get_charge_us(KickerSpeed kickSpeed);

/*
 * Public functions implementations
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick)
{
    LOG_InitModule(&internal_log_mod, "KICKER", LOG_LEVEL_INFO, 0);
    htim_kicker_charge = htim_charge;
    htim_kicker_kick = htim_kick;

    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, settings.charge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_charge, 0);
    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
}

void KICKER_ChargeStart(KickerSpeed kickSpeed)
{
    if (settings.charges_since_last_kick >= settings.max_charges_per_kick) {
        LOG_DEBUG("Max charges per kick reached.\r\n");
        return;
    }

    LOG_DEBUG("Charging start\r\n");

    if (charging) {
        LOG_DEBUG("Already charging\r\n");
        return;
    }

    int charge_us = get_charge_us(kickSpeed);

    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_SET);
    charging = true;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, charge_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_charge, 0);
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_charge, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_charge);
    settings.charges_since_last_kick++;
}

void KICKER_ChargeStop()
{
    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_RESET);
    settings.charges_since_last_kick++;
    LOG_DEBUG("Charging stopped, charged %d times...\r\n", settings.charges_since_last_kick);
    HAL_TIM_Base_Stop_IT(htim_kicker_charge);
    charging = false;
}

void KICKER_KickStart()
{
    if (kicking) {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }
    kicking = true;

    LOG_DEBUG("Kicking start\r\n");

    // Kicks on low
    if (kickerMode == KICKER_STRAIGHT)
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
    }
    else if (kickerMode == KICKER_CHIPPER)
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_RESET);
    }

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_kick, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_kick);
}

void KICKER_KickStop()
{
    // Stop kick on high, both chipper and straight
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_SET);
    HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
    LOG_DEBUG("Kicking stop\r\n");
    HAL_TIM_Base_Stop_IT(htim_kicker_kick);
    kicking = false;
}

void KICKER_SetKickerMode(KickerMode mode)
{
    kickerMode = mode;
}

void KICKER_KickSafe()
{
    // Kicks on low
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
    COMMON_Wait(settings.safe_discharge_wait_us);
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
}

KICKER_Settings* KICKER_GetSettings() { return &settings; }

/*
   Private function implementation
*/
int get_charge_us(KickerSpeed kickSpeed)
{
    int charge_us = settings.charge_wait_us;

    if (kickSpeed == KICKER_SPEED_STRAIGHT_GOAL)
    {
        charge_us = KICK_STRAIGHT_PASS_US;
    }
    else if (kickSpeed == KICKER_SPEED_STRAIGHT_PASS)
    {
        charge_us = KICK_STRAIGHT_GOAL_US;
    }
    else if (kickSpeed == KICKER_SPEED_CHIP_PASS)
    {
        charge_us = KICK_CHIP_PASS_US;
    }

    return charge_us;
}

