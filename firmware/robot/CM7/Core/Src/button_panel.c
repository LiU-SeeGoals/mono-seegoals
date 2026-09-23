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
static const int PRESS_VALUE_AUX;
static const int PRESS_VALUE_MOTOR_OFF;
static const int PRESS_VALUE_CHIPPER;
static const int PRESS_VALUE_KICKER;
static const int PRESS_VALUE_DRIBBLER;
static const int PRESS_VALUE_RANGE;

ADC_HandleTypeDef hadc2;

static LOG_Module internal_log_mod;


/* Private functions declarations */
void Button_AUX();
void Button_Motor_Off();
void Button_Chipper();
void Button_Kicker(); // Lowest power possible
void Button_Dribbler();

/* Public functions implementations */


/**
  * @brief  Start the ADC of the button panel
  * @note   Interruptions enabled in this function: None.
  */
void Button_Panel_ADC() 
{
    HAL_StatusTypeDef status = HAL_ERROR;
    status = HAL_ADC_Start(&hadc2);
    if (status != HAL_OK) {
        LOG_ERROR("IR sensor ADC failed start.\r\n");
    }

    // Wait for conversion to complete, timeout 20ms
    status = HAL_ADC_PollForConversion(&hadc2, 20);
    if (status != HAL_OK) {
        LOG_ERROR("ADC poll wait failed.\r\n");
    }
    
    uint32_t raw = HAL_ADC_GetValue(&hadc2);
    
    status = HAL_ADC_Stop(&hadc2);
    
    if (status != HAL_OK) {
        LOG_ERROR("ADC stop failed.\r\n");
    }
    LOG_INFO("ADC2 VALUE IN: %u\r\n", raw);

    // Now that we have the raw ADC value we check it against the press_values to see which was pressed

    if (PRESS_VALUE_AUX - PRESS_VALUE_RANGE < raw || raw < PRESS_VALUE_AUX + PRESS_VALUE_RANGE) 
    {
        Button_AUX();
    }
    else if (PRESS_VALUE_MOTOR_OFF - PRESS_VALUE_RANGE < raw || raw < PRESS_VALUE_MOTOR_OFF + PRESS_VALUE_RANGE)
    {
        Button_Motor_Off();
    }
    else if (PRESS_VALUE_CHIPPER - PRESS_VALUE_RANGE < raw || raw < PRESS_VALUE_MOTOR_OFF + PRESS_VALUE_RANGE)
    {
        Button_Chipper();
    }
    else if (PRESS_VALUE_KICKER - PRESS_VALUE_RANGE < raw || raw < PRESS_VALUE_KICKER + PRESS_VALUE_RANGE)
    {
        Button_Kicker();
    }
    else if (PRESS_VALUE_DRIBBLER - PRESS_VALUE_RANGE < raw || raw < PRESS_VALUE_DRIBBLER + PRESS_VALUE_RANGE)
    {
        Button_Dribbler();
    }
    else
    {
        LOG_ERROR("The button pressed could not be determined: %u\r\n", raw);
    }
}


/* Private functions implementations */