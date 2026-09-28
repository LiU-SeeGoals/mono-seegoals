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
bool dribbling = false;


/* Private functions declarations */
void Button_AUX();
void Button_Motor_Off();
void Button_Chipper();
void Button_Kicker(); // Lowest power possible
void Button_Dribbler();
void Kick();

/* Public functions implementations */


void Button_Panel_INIT(ADC_HandleTypeDef* handle)
{
    hadc2 = handle;
    NAV_EnableMovement(); // Incase it does not enable by default
    LOG_InitModule(&internal_log_mod, "Button_Panel", LOG_LEVEL_UI, 0);
}


/**
  * @brief  Start the ADC of the button panel
  * @note   Interruptions enabled in this function: None.
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
  * @brief  Toggle the motor
  * @note   Ensure that there is an upper limit so that it can't run forever (maybe)
  */
void Button_AUX() 
{
    KICKER_SetKickerMode(KICKER_STRAIGHT );
    KICKER_ChargeStart(KICKER_SPEED_STRAIGHT_PASS);
    return;
}

/**
  * @brief  Toggle the motor
  * @note   Ensure that there is an upper limit so that it can't run forever (maybe)
  */
void Button_Motor_Off() 
{
    /*
    } else if (current_state == state_motors) {
        static int dribble_cur = 0;

        switch (key) {
        case 'S': // Steer
            current_state = state_motors_steer;
            print_help();
            break;
        case 'T': // Toggle movement
            LOG_INFO("Movement toggled\r\n");
            if (moving) {
                NAV_DisableMovement();
                moving = 0;
            } else {
                NAV_EnableMovement();
                moving = 1;
            }
            break;
        case 'G': // Go tire test
            STATE_disable_calibration();
            NAV_TEST_TireTest();
            break;
        
    */

    // If NAV is by default enabled then all we should need is to run the tire test

    STATE_disable_calibration();
    NAV_TEST_TireTest();


    return;
}

void Button_Chipper()
{
    KICKER_SetKickerMode(KICKER_CHIPPER);
    KICKER_ChargeStart(KICKER_SPEED_CHIP_PASS);

    // When to discharge?
    
    /*
    LOG_UI("Discharging\r\n");
            KICKER_KickStart();

    case 'C':
            LOG_UI("Chipper kicking\r\n");
            KICKER_SetKickerMode(KICKER_CHIPPER);
            KICKER_ChargeStart(KICKER_SPEED_CHIP_PASS);
            break;
    */

    

    return;
}

void Button_Kicker()
{
    /*
    else if (current_state == state_kicker) {
        switch (key) {
        case 'D':
            LOG_UI("Discharging\r\n");
            KICKER_KickStart();
            break;
        case 'S':
            LOG_UI("Straight kicking\r\n");
            KICKER_SetKickerMode(KICKER_STRAIGHT);
            KICKER_ChargeStart(KICKER_SPEED_DEFAULT);
            break;
        
        case 'P': // Print vars
        {
            KICKER_Settings* set = KICKER_GetSettings();
            LOG_UI("Max charges per kick: %i\r\nCharge wait (us): %i\r\nDischarge wait (us): %i\r\n", set->max_charges_per_kick, set->charge_wait_us, set->discharge_wait_us);
        } break;
        case 'E': // Edit vars
            current_state = state_kicker_edit;
            print_help();
            break;
        }
    */

    KICKER_SetKickerMode(KICKER_STRAIGHT);
    KICKER_ChargeStart(KICKER_SPEED_STRAIGHT_GOAL);
}

/**
  * @brief  Toggle the dribbler
  * @note   Ensure that there is an upper limit so that it can't run forever (maybe)
  */
void Button_Dribbler()
{
    if (dribbling)
    {
        NAV_StopDribbler();
        dribbling = false;
    }
    else 
    {
        NAV_RunDribbler();
        dribbling = true;
    }

    return;
}

void Kick() 
{

}