#include "kicker.h"

/* Private includes */
#include "common.h"
#include "log.h"
#include "main.h"
#include "stm32h755xx.h"
#include "stm32h7xx_hal_spi.h"
#include <stdbool.h>

/* Private defines */
// ...

/* Private variables */
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {
    .max_charges_per_kick = 6, .safe_discharge_wait_us = 15, .charge_wait_us = 1500000, .discharge_wait_us_kicker = 300, .discharge_wait_us_chipper = 300, .charges_since_last_kick = 0};
static volatile bool charging = false;
static volatile bool kicking = false;
static TIM_HandleTypeDef* htim_kicker_charge;
static TIM_HandleTypeDef* htim_kicker_kick;

static SPI_HandleTypeDef* HSPI;

/*
 * Public functions implementations
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick)
{
    LOG_InitModule(&internal_log_mod, "KICKER", LOG_LEVEL_INFO, 0);
    htim_kicker_charge = htim_charge;
    htim_kicker_kick = htim_kick;
    // HSPI = hspi;

    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, settings.charge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_charge, 0);
    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us_kicker);
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
}

int16_t KICKER_ReadADC()
{
    uint8_t buffer[2] = {0x00, 0x00};
    HAL_SPI_Receive(HSPI, &buffer[0], 2, 1000);

    uint16_t voltage = (uint16_t)buffer[0] | ((uint16_t)buffer[1] << 8);

    return voltage;
}

void KICKER_ChargeStart()
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

    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_SET);
    charging = true;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, settings.charge_wait_us);
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

void KICKER_KickStart(KICKER_type type)
{
    if (kicking) {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }

    LOG_DEBUG("Kicking start\r\n");

    if (type == KICKER) {
        // Kicker
        // Kicks on low
        HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
        __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us_kicker);
    } else {
        // Chipper
        // Kicks on low
        HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_RESET);
        __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us_chipper);
    }

    kicking = true;

    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_kick, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_kick);
}

void KICKER_KickStop()
{
    // Stop kick on high
    // Set both kickers high for simplicity even tough we only used one.
    HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_SET);
    HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
    LOG_DEBUG("Kicking stop\r\n");
    HAL_TIM_Base_Stop_IT(htim_kicker_kick);
    kicking = false;
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
