#include "kicker.h"

/* Private includes */
#include "common.h"
#include "log.h"
#include "stm32h7xx_hal_spi.h"
#include <stdbool.h>
#include <stdint.h>

/* Private defines */
// ...

#define KICK_STRAIGHT_PASS_US 250000
#define KICK_STRAIGHT_GOAL_US 450000
#define KICK_CHIP_PASS_US 650000

#define DISCHARGE_STRAIGHT_PASS_US 65
#define DISCHARGE_STRAIGHT_GOAL_US 65
#define DISCHARGE_CHIP_PASS_US 90

#define KICKER_ADC_CHARGE_VALUE 1200 // Random value for now. The ADC is 12 bit long so this is like one quarter charge
#define KICKER_BOARD_ADC_CODE 0x08
#define KICKER_BOARD_SPI_SIZE 2

/* Private variables */
// Kicker tuning Chip discharge: 650000. Kicker tuning straight discharge: 650000
// Kicker tuning straight goal kick: 350000. kicker straight pass: 250000
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {.max_charges_per_kick = 6, .safe_discharge_wait_us=25, .charge_wait_us = 350000, .discharge_wait_us = 300, .charges_since_last_kick = 0};
static volatile bool charging = false;
static volatile bool charged = false;
static volatile bool kicking = false;
static TIM_HandleTypeDef* htim_kicker_charge;
static TIM_HandleTypeDef* htim_kicker_kick;     // One shot timer
static SPI_HandleTypeDef* hspi5_kicker;
static volatile KickerMode kickerMode = NONE;

static uint8_t kicker_spi_tx[KICKER_BOARD_SPI_SIZE] = {KICKER_BOARD_ADC_CODE, 0x00};
static uint8_t kicker_spi_rx[KICKER_BOARD_SPI_SIZE];

static volatile bool kicker_spi_busy = false;
static volatile bool kicker_adc_ready = false;

static uint16_t kicker_current_charge_adc = 0;
static volatile uint16_t kicker_adc_value = 0;
static volatile float kicker_voltage = 0.0f;

/*
 * Private function declaration
 */

int get_charge_us(KickerSpeed kickSpeed);
int get_discharge_us(KickerSpeed kickSpeed);
float KICKER_ADC_To_Voltage_Complex(uint16_t adc);
float KICKER_ADC_To_Voltage_Simple(uint16_t adc);

/*
 * Public functions implementations
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick, SPI_HandleTypeDef* hspi5)
{
    LOG_InitModule(&internal_log_mod, "KICKER", LOG_LEVEL_INFO, 0);
    htim_kicker_charge = htim_charge;
    htim_kicker_kick = htim_kick;
    hspi5_kicker = hspi5;

    kicker_spi_busy = false;
    kicker_adc_ready = false;
    kicker_adc_value = 0;
    kicker_voltage = 0.0f;

    // CS actie low start at high
    HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_SET);

    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;
}

void KICKER_ChargeStart()
{
    /*
    if (settings.charges_since_last_kick >= settings.max_charges_per_kick) {
        LOG_DEBUG("Max charges per kick reached.\r\n");
        return;
    }
    */
    if (kicking)
    {
        LOG_DEBUG("Currently kicking");
        return;
    }
    if (charging) {
        LOG_DEBUG("Already charging\r\n");
        return;
    }

    LOG_INFO("Charging start\r\n");

    // int charge_us = get_charge_us(kickSpeed);

    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_SET);
    charging = true;
    HAL_TIM_Base_Start_IT(htim_kicker_charge);
    
    settings.charges_since_last_kick++;
    KICKER_StartVoltageMeasurement(); // Start getting ADC values
}

void KICKER_ChargeStop()
{
    if (!charging)
    {
        LOG_DEBUG("Not charging\r\n");
        return;
    }
    if (!KICKER_TryGetADC(&kicker_current_charge_adc))
    {
        LOG_DEBUG("Could not get adc\r\n");
        return;
    }

    if (kicker_current_charge_adc < KICKER_ADC_CHARGE_VALUE)
    {
        int status = KICKER_StartVoltageMeasurement();

        if (status != 0)
        {
            KICKER_ChargeStopForced(status);
        }
        return;
    }
    // Literally identical to what we want
    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_RESET);
    settings.charges_since_last_kick++;
    HAL_TIM_Base_Stop_IT(htim_kicker_charge);
    charging = false;
    charged = true;
    LOG_INFO("Charging stopped, charged to %d or %fV\r\n", kicker_current_charge_adc, KICKER_ADC_To_Voltage_Simple(kicker_current_charge_adc));
}

void KICKER_ChargeStopForced(int status)
{
    HAL_GPIO_WritePin(KICKER_CHARGE_GPIO_Port, KICKER_CHARGE_Pin, GPIO_PIN_RESET);
    settings.charges_since_last_kick++;
    LOG_ERROR("Something with starting the SPI transfer went wrong and I don't know what %d \r\n", status);
    HAL_TIM_Base_Stop_IT(htim_kicker_charge);
    charging = false;
}

void KICKER_KickStart(KickerSpeed kickspeed)
{
    if (!charged)
    {
        LOG_DEBUG("Not yet fully charged");
        return;
    }

    if (charging)
    {
        LOG_DEBUG("Currently charging");
    }

    if (kicking)
    {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }
    kicking = true;

    LOG_INFO("Kicking start\r\n");

    // Kicks on low
    if (kickerMode == KICKER_STRAIGHT)
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE2_GPIO_Port, KICKER_DISCHARGE2_Pin, GPIO_PIN_RESET);
    }
    else if (kickerMode == KICKER_CHIPPER)
    {
        HAL_GPIO_WritePin(KICKER_DISCHARGE1_GPIO_Port, KICKER_DISCHARGE1_Pin, GPIO_PIN_RESET);
    }

    int discharge_length = get_discharge_us(kickspeed);

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, discharge_length);
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

int KICKER_StartVoltageMeasurement(void)
{
    if (kicker_spi_busy)
    {
        return 1;
    }

    // If the SPI5 is busy then don't try to use it
    if (HAL_SPI_GetState(hspi5_kicker) != HAL_SPI_STATE_READY)
    {
        return 1;
    }

    kicker_spi_busy = true;

    // The message needs to be reset as well since it has gotten whatever the adc is
    kicker_spi_tx[0] = KICKER_BOARD_ADC_CODE;
    kicker_spi_tx[1] = 0x00;

    // Activate the chip by putting the pin low
    HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_RESET);

    // Start the SPI transfer non blocking
    HAL_StatusTypeDef status = HAL_SPI_TransmitReceive_IT(hspi5_kicker, kicker_spi_tx, kicker_spi_rx, KICKER_BOARD_SPI_SIZE);

    if (status != HAL_OK)
    {
        // Transfer didn't start. Put pin back high
        HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_SET);

        kicker_spi_busy = false;

        if (status == HAL_BUSY)
        {
            return 1;
        }

        return -1;
    }

    return 0;
}

void KICKER_SPI_TxRxCpltCallback(SPI_HandleTypeDef *hspi)
{
    if (hspi != hspi5_kicker)
    {
        return;
    }
    LOG_DEBUG("Got an ADC\r\n");

    // SPI done release the pin
    HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_SET);

    uint16_t raw_adc = ((uint16_t)kicker_spi_rx[0] << 8) | (uint16_t)kicker_spi_rx[1];

    raw_adc &= 0x0FFF;

    kicker_adc_value = raw_adc;
    kicker_adc_ready = true;
    kicker_spi_busy = false;
}

void KICKER_SPI_ErrorCallback(SPI_HandleTypeDef *hspi)
{
    if (hspi != hspi5_kicker)
    {
        return;
    }
    LOG_ERROR("FUCK\r\n");


    HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_SET);
    kicker_spi_busy = false;
}

bool KICKER_TryGetADC(uint16_t *adc)
{
    if (adc == NULL)
    {
        LOG_DEBUG("adc is null\r\n");
        return false;
    }

    if (!kicker_adc_ready)
    {
        LOG_DEBUG("adc is not ready\r\n");
        return false;
    }

    // Getting the adc value has to be atomic (No chaging the value while we try to get it)
    uint32_t primask = __get_PRIMASK();
    __disable_irq();

    *adc = kicker_adc_value;
    kicker_adc_ready = false;

    if (primask == 0U)
    {
        __enable_irq();
    }

    return true;
}

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

int get_discharge_us(KickerSpeed kickSpeed)
{
    int discharge_us = settings.discharge_wait_us;

    if (kickSpeed == KICKER_SPEED_STRAIGHT_GOAL)
    {
        discharge_us = DISCHARGE_STRAIGHT_GOAL_US;
    }
    else if (kickSpeed == KICKER_SPEED_STRAIGHT_PASS)
    {
        discharge_us = DISCHARGE_STRAIGHT_PASS_US;
    }
    else if (kickSpeed == KICKER_SPEED_CHIP_PASS)
    {
        discharge_us = DISCHARGE_CHIP_PASS_US;
    }

    return discharge_us;
}

float KICKER_ADC_To_Voltage_Complex(uint16_t adc)
{
    float adc_voltage = ((float) adc * 3.3f) / 4095.0f;

    return adc_voltage * ((10000000.0f + 133000.0f) / 133000.0f);
}

float KICKER_ADC_To_Voltage_Simple(uint16_t adc)
{
    return ((float) adc * 251.0f) / 4095.0f;
}
