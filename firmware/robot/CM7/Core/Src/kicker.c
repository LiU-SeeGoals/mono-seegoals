#include "kicker.h"

/* Private includes */
#include "common.h"
#include "log.h"
#include <stdbool.h>

/* Private defines */
// The kicker is always charged to full voltage and the kick strength is
// controlled by how long the capacitor bank is discharged into the solenoid.
// The values below are in timer ticks and should be tuned per strength.
#define KICK_DISCHARGE_DEFAULT_US 300
#define KICK_DISCHARGE_PASS_US 250
#define KICK_DISCHARGE_GOAL_US 450
#define KICK_DISCHARGE_CHIP_US 400

/* Private variables */
// Kicker tuning Chip discharge: 650000. Kicker tuning straight discharge: 650000
// Kicker tuning straight goal kick: 350000. kicker straight pass: 250000
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {.max_charges_per_kick = 6, .safe_discharge_wait_us=15, .charge_wait_us = 350000, .discharge_wait_us = KICK_DISCHARGE_DEFAULT_US, .charges_since_last_kick = 0};
static volatile bool charging = false;
static volatile bool charged = false;
static volatile bool kicking = false;
static TIM_HandleTypeDef* htim_kicker_charge;
static TIM_HandleTypeDef* htim_kicker_kick;
static volatile KickerMode kickerMode = KICKER_STRAIGHT;
static volatile KickerSpeed kickerSpeed = KICKER_SPEED_DEFAULT;
static volatile bool useIrSensor = false;

/*
 * Private function declaration
 */

int get_discharge_us(KickerSpeed kickSpeed);

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

void KICKER_SetKickConfig(KickerMode mode, KickerSpeed speed, bool use_ir_sensor)
{
    kickerMode = mode;
    kickerSpeed = speed;
    useIrSensor = use_ir_sensor;
    LOG_DEBUG("Kick config: mode=%i speed=%i use_ir=%i\r\n", mode, speed, use_ir_sensor);
}

void KICKER_ChargeStart(void)
{
    if (kicking) {
        LOG_DEBUG("Cannot charge while kicking\r\n");
        return;
    }

    if (settings.charges_since_last_kick >= settings.max_charges_per_kick) {
        LOG_DEBUG("Max charges per kick reached.\r\n");
        return;
    }

    if (charging) {
        LOG_DEBUG("Already charging\r\n");
        return;
    }

    LOG_DEBUG("Charging start\r\n");

    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_SET);
    charging = true;
    charged = false;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, settings.charge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_charge, 0);
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_charge, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_charge);
    settings.charges_since_last_kick++;
}

void KICKER_ChargeComplete(void)
{
    HAL_TIM_Base_Stop_IT(htim_kicker_charge);
    charging = false;
    charged = true;
    // Keep the charge input asserted so the capacitor bank stays topped up.
    LOG_DEBUG("Charge complete\r\n");
}

KICKER_Status KICKER_KickStart(void)
{
    if (kicking) {
        LOG_DEBUG("Already kicking\r\n");
        return KICKER_STATUS_BUSY;
    }

    if (!charged) {
        LOG_DEBUG("Not charged, rejecting kick\r\n");
        return KICKER_STATUS_NOT_READY;
    }

    LOG_DEBUG("Kicking start\r\n");

    // Stop topping up the capacitor bank while it is discharged into the solenoid.
    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_RESET);
    charging = false;
    charged = false;

    // Kicks on low
    if (kickerMode == KICKER_STRAIGHT)
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
    }
    else
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_RESET);
    }

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, get_discharge_us(kickerSpeed));
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_kick, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_kick);
    kicking = true;

    return KICKER_STATUS_OK;
}

void KICKER_KickStop(void)
{
    // Stop kick on high, both chipper and straight
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_SET);
    HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
    LOG_DEBUG("Kicking stop\r\n");
    HAL_TIM_Base_Stop_IT(htim_kicker_kick);
    kicking = false;

    // Start charging again so the kicker is ready for the next kick.
    KICKER_ChargeStart();
}

void KICKER_KickSafe()
{
    // Kicks on low
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
    COMMON_Wait(settings.safe_discharge_wait_us);
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
}

KICKER_Status KICKER_GetStatus(void)
{
    if (kicking) {
        return KICKER_STATUS_BUSY;
    }
    if (!charged) {
        return KICKER_STATUS_NOT_READY;
    }
    return KICKER_STATUS_OK;
}

bool KICKER_IsCharged(void)
{
    return charged && !kicking;
}

KICKER_Settings* KICKER_GetSettings() { return &settings; }

/*
   Private function implementation
*/
int get_discharge_us(KickerSpeed kickSpeed)
{
    switch (kickSpeed) {
    case KICKER_SPEED_STRAIGHT_GOAL:
        return KICK_DISCHARGE_GOAL_US;
    case KICKER_SPEED_STRAIGHT_PASS:
        return KICK_DISCHARGE_PASS_US;
    case KICKER_SPEED_CHIP_PASS:
        return KICK_DISCHARGE_CHIP_US;
    default:
        return KICK_DISCHARGE_DEFAULT_US;
    }
}