#include "button_panel.h"

/* Private includes */
#include "com.h"
#include "common.h"
#include "kicker.h"
#include "log.h"
#include "nav.h"
#include "pos_follow.h"
#include "state_estimator.h"
#include <stdlib.h>
#include <string.h>


/* Private defines */

/* Private enums/structs */

/* Private variables */
static const int NOISE_FILTER = 1200;

static const int AUX_LOWER = 2600;
static const int AUX_UPPER = 2800;

static const int MOTOR_OFF_LOWER = 1900;
static const int MOTOR_OFF_UPPER = 2100;

static const int CHIPPER_LOWER = 1630;
static const int CHIPPER_UPPER = 1780;

static const int KICKER_LOWER = 1530;
static const int KICKER_UPPER = 1600;

static const int DRIBBLER_LOWER = 1300;
static const int DRIBBLER_UPPER = 1490;

static const int PRESS_VALUE_RANGE = 40;

static ADC_HandleTypeDef *hadc2;
static LOG_Module internal_log_mod;

bool button_handled = false;

// AUX variables

// Motor_Off variables

// Chipper variables

// Kicker variables

//Dribbler variables


/* Private functions declarations */
void Button_AUX();
void Button_Motor_Off();
void Button_Chipper();
void Button_Kicker(); 
void Button_Dribbler();
void Kick();

/* Public functions implementations */

/**
  * @brief  Initialise the button panel system
  */
void Button_Panel_INIT(ADC_HandleTypeDef* handle)
{
    hadc2 = handle;
    NAV_EnableMovement(); // Incase it does not enable by default
    LOG_InitModule(&internal_log_mod, "Button_Panel", LOG_LEVEL_UI, 0);
}


/**
  * @brief  Start the ADC of the button panel
  * @note   Interruptions enabled in this function: None.
  * @note   Reached via timer interrupt ???
  */
void Button_Panel_ADC() 
{
    HAL_StatusTypeDef status = HAL_ERROR;
    status = HAL_ADC_Start(hadc2);
    if (status != HAL_OK) {
        LOG_ERROR("IR sensor ADC failed start.\r\n");
    }

    // Wait for conversion to complete, timeout 20ms
    status = HAL_ADC_PollForConversion(hadc2, 20);
    if (status != HAL_OK) {
        LOG_ERROR("ADC poll wait failed.\r\n");
    }
    
    uint32_t raw = HAL_ADC_GetValue(hadc2);
    
    status = HAL_ADC_Stop(hadc2);
    
    if (status != HAL_OK) {
        LOG_ERROR("ADC stop failed.\r\n");
    }
    LOG_INFO("ADC2 VALUE IN: %u\r\n", raw);

    // Now that we have the raw ADC value we check it against the press_values to see which was pressed

    if (raw < NOISE_FILTER)
    {
        button_handled = false;
        return;
    }

    if (button_handled)
    {
        return;
    }
    else if (AUX_LOWER < raw && raw < AUX_UPPER) 
    {
        Button_AUX();
    }
    else if (MOTOR_OFF_LOWER < raw && raw < MOTOR_OFF_UPPER)
    {
        Button_Motor_Off();
    }
    else if (CHIPPER_LOWER < raw && raw < CHIPPER_UPPER)
    {
        Button_Chipper();
    }
    else if (KICKER_LOWER < raw && raw < KICKER_UPPER)
    {
        Button_Kicker();
    }
    else if (DRIBBLER_LOWER < raw && raw < DRIBBLER_UPPER)
    {
        Button_Dribbler();
    }
    else
    {
        LOG_ERROR("The button pressed could not be determined: %u\r\n", raw);
        button_handled = false;
        return;
    }
    button_handled = true;
    return;
}


/* Private functions implementations */

/**
  * @brief  Set the kicker to straight and straight pass
  */
void Button_AUX() 
{
    KICKER_SetKickerMode(KICKER_STRAIGHT );
    KICKER_ChargeStart(KICKER_SPEED_STRAIGHT_PASS);
    return;
}

/**
  * @brief  Run the NAV_TEST_TireTest
  */
void Button_Motor_Off() 
{
    // If NAV is by default enabled then all we should need is to run the tire test

    STATE_disable_calibration();
    NAV_TEST_TireTest();

    // STATE_enable_calibration


    return;
}

/**
  * @brief  Set the kicker to chipper and chip pass
  */
void Button_Chipper()
{
    KICKER_SetKickerMode(KICKER_CHIPPER);
    KICKER_ChargeStart(KICKER_SPEED_CHIP_PASS);
    return;
}


/**
  * @brief  Set the kicker to chipper and straight shoot
  */
void Button_Kicker()
{
    KICKER_SetKickerMode(KICKER_STRAIGHT);
    KICKER_ChargeStart(KICKER_SPEED_STRAIGHT_GOAL);
    return;
}

/**
  * @brief  Start dribbler test
  */
void Button_Dribbler()
{
    NAV_TestDribbler();
    return;
}