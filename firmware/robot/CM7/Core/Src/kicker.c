#include "kicker.h"

/* Private includes */
#include "arm_math.h"
#include "common.h"
#include "log.h"
#include "main.h"
#include "stm32h755xx.h"
#include "stm32h7xx_hal.h"
#include "stm32h7xx_hal_adc.h"
#include "stm32h7xx_hal_def.h"
#include "stm32h7xx_hal_gpio.h"
#include "stm32h7xx_hal_spi.h"
#include <stdbool.h>
#include <stdint.h>

/* Private defines */
// ...

/* Private variables */
static LOG_Module internal_log_mod;
static KICKER_Settings settings = {
    .max_charges_per_kick = 6,
    .safe_discharge_wait_us = 15,
    .charge_wait_us = 350000,
    .discharge_wait_us_kicker = 300,
    .discharge_wait_us_chipper = 10,
    .charges_since_last_kick = 0,
    .ir_poll_intervall_us = 20000,
};

static volatile bool charging = false;
static volatile bool kicking = false;
static TIM_HandleTypeDef* htim_kicker_charge;
static TIM_HandleTypeDef* htim_kicker_kick;

static SPI_HandleTypeDef* kicker_hspi;
static ADC_HandleTypeDef* ir_adc;

// Threshold value for the ir sensor, if the reading is bellow this value we
// consider the sensor ocluded.
const uint32_t ir_threshold = 2000;

/*
 * Private functions:
 */

/**
 * Sets the state of the enable pin for the voltage measurement.
 * false => disable
 * true  => enable
 **/
void spi_enable(bool enabled)
{
    if (enabled) {
        // Set pin low to enable the spi connection
        HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_RESET);
    } else {
        // Set pin high to disable the spi connection
        HAL_GPIO_WritePin(KICKER_CS_GPIO_Port, KICKER_CS_Pin, GPIO_PIN_SET);
    }
}

int16_t readADC()
{
    uint8_t tx[2] = {0x08, 0x00};
    uint8_t rx[2] = {0};

    spi_enable(true);

    HAL_SPI_TransmitReceive(kicker_hspi,
                            tx,
                            rx,
                            2,
                            HAL_MAX_DELAY);

    HAL_GPIO_WritePin(KICKER_CS_GPIO_Port,
                      KICKER_CS_Pin,
                      GPIO_PIN_SET);

    uint16_t adc = ((uint16_t)rx[0] << 8) | rx[1];

    LOG_Printf(&internal_log_mod, LOG_LEVEL_INFO, "ADC: %d\r\n", adc);
    return 0;
}

bool ir_sensor_ocluded()
{
    // CoreDebug->DEMCR |= CoreDebug_DEMCR_TRCENA_Msk;
    // DWT->CYCCNT = 0;
    // DWT->CTRL |= DWT_CTRL_CYCCNTENA_Msk;

    // uint32_t start = DWT->CYCCNT;

    HAL_StatusTypeDef status = HAL_ERROR;
    status = HAL_ADC_Start(ir_adc);
    if (status != HAL_OK) {
        LOG_ERROR("IR sensor ADC failed start.\r\n");
    }

    // Wait for conversion to complete, timeout 20ms
    status = HAL_ADC_PollForConversion(ir_adc, 20);
    if (status != HAL_OK) {
        LOG_ERROR("ADC poll wait failed.\r\n");
    }

    uint32_t raw = HAL_ADC_GetValue(ir_adc);

    status = HAL_ADC_Stop(ir_adc);
    if (status != HAL_OK) {
        LOG_ERROR("ADC stop failed.\r\n");
    }
    // LOG_INFO("IR measurement: %u\r\n", raw);

    /*
     * false => adc reading above threshold, the light is reaching the sensor
     * true  => adc reading bellow threshold, less light is reaching the sensor
     *          so it is probablly ocluded, possibly by ball
     */
    bool ball_status;
    if (raw > ir_threshold) {
        ball_status = false;
        // LOG_INFO("Ball?: No\r\n");
    } else {
        ball_status = true;
        // LOG_INFO("Ball?: Yes!\r\n");
    }
    // uint32_t end = DWT->CYCCNT;
    // uint32_t cycles = end - start;

    // LOG_INFO("IR sensor read clk cycles: %u \r\n", cycles);

    return ball_status;
}

/*
 * Public functions implementations
 */
void KICKER_Init(TIM_HandleTypeDef* htim_charge, TIM_HandleTypeDef* htim_kick, SPI_HandleTypeDef* hspi, ADC_HandleTypeDef* adc)
{
    LOG_InitModule(&internal_log_mod, "KICKER", LOG_LEVEL_INFO, 0);
    // Initialize local variables
    htim_kicker_charge = htim_charge;
    htim_kicker_kick = htim_kick;
    kicker_hspi = hspi;
    ir_adc = adc;

    htim_kicker_kick->Instance->EGR = TIM_EGR_UG;
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_charge, settings.charge_wait_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_charge, 0);
    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.discharge_wait_us_kicker);
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);

    HAL_StatusTypeDef status = HAL_ERROR;
    // status = HAL_ADCEx_Calibration_Start(ir_adc, ADC_CALIB_OFFSET, ADC_SINGLE_ENDED);
    // if (status != HAL_OK) {
    // LOG_ERROR("IR sensor ADC failed calibration.\r\n");
    // }

    // Disable the spi connection as default
    spi_enable(false);

    // Testing, reading from the ADC
    while (true) {
        bool ball_status = ir_sensor_ocluded();
        HAL_Delay(500);
    }
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

typedef struct {
    // Indicates that a kick process is in progress, this might be kicking or
    // waiting for the IR sensor.
    bool in_progress;
    // The amout of polls to wait before the kick times out.
    uint32_t timeout;
    // The solenoid we are using, kicker or chipper
    KICKER_type type;
    // If a kick has started
    bool kick_started;
} KICKER_InternalState;

static KICKER_InternalState internal_state = {
    .in_progress = false,
    .timeout = 0,
    .type = KICKER,
    .kick_started = false,
};

void KICKER_KickStartWithIR(KICKER_type type, uint32_t timeout)
{
    // Check if the other kicker function has already been called.
    if (kicking) {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }

    // Check if this kick function has been run before
    if (internal_state.in_progress) {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }

    internal_state.in_progress = true;
    internal_state.type = type;
    internal_state.timeout = timeout;

    // If the timeout is zero we want to kick imediately
    if (timeout == 0) {
        KICKER_KickPoll();

        return;
    }

    __HAL_TIM_SET_AUTORELOAD(htim_kicker_kick, settings.ir_poll_intervall_us);
    __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
    htim_kicker_charge->Instance->EGR = TIM_EGR_UG;
    __HAL_TIM_CLEAR_FLAG(htim_kicker_kick, TIM_FLAG_UPDATE);

    HAL_TIM_Base_Start_IT(htim_kicker_kick);
}

void KICKER_KickPoll()
{

    // If we have started a kick we want to stop
    if (internal_state.kick_started) {
        KICKER_KickStop();

        // Reset internal state
        internal_state.in_progress = false;
        internal_state.kick_started = false;

        return;
    }

    // If the IR sensor is occluded we want to kick the ball and if the timeout
    // is 0 we also want to kick
    if (internal_state.timeout == 0 || ir_sensor_ocluded()) {
        internal_state.kick_started = true;

        LOG_DEBUG("Kicking start\r\n");

        if (internal_state.type == KICKER) {
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

        __HAL_TIM_SET_COUNTER(htim_kicker_kick, 0);
        htim_kicker_charge->Instance->EGR = TIM_EGR_UG;
        __HAL_TIM_CLEAR_FLAG(htim_kicker_kick, TIM_FLAG_UPDATE);

        return;
    }

    // decrease the timeout counter.
    internal_state.timeout -= 1;
}

void KICKER_KickStart(KICKER_type type)
{
    if (kicking) {
        LOG_DEBUG("Already kicking\r\n");
        return;
    }

    // Check if the ir sensor kick function is in progress
    if (internal_state.in_progress) {
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
