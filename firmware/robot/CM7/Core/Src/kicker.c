#include "kicker.h"

/* Private includes */
#include "log.h"

/* Private defines */
// ...

/* Private variables */
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {.max_charges_per_kick = 6, .charge_wait_us = 25000, .discharge_wait_us = 130, .charges_since_last_kick = 0};

/* Private functions declarations */
static void wait(uint64_t us);

/*
 * Public functions implementations
 */
void KICKER_Init() { LOG_InitModule(&internal_log_mod, "KICKER", LOG_LEVEL_DEBUG, 0); }

void KICKER_Charge()
{
    if (settings.charges_since_last_kick >= settings.max_charges_per_kick) {
        LOG_INFO("Max charges per kick reached.\r\n");
        return;
    }

    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_SET);
    wait(settings.charge_wait_us); // TODO replace with hardware timer
    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_RESET);
    settings.charges_since_last_kick++;
    LOG_DEBUG("Charged %d times...\r\n", settings.charges_since_last_kick);
}

void KICKER_Kick()
{
    // Kicks on low
    HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_RESET);
    wait(settings.discharge_wait_us); // TODO replace with hardware timer
    HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_SET);
    settings.charges_since_last_kick = 0;
    LOG_DEBUG("Kicking!\r\n");
}

KICKER_Settings* KICKER_GetSettings() { return &settings; }

/*
 * Private functions implementations
 */
void wait(uint64_t us)
{
    uint32_t volatile cycles = HAL_RCC_GetSysClockFreq() * us / 1000000;
    uint32_t volatile current = 0;
    while (current <= cycles) {
        current++;
    }
}
