#ifndef BUTTON_PANEL_H
#define BUTTON_PANEL_H

// INCLUDES
#include "main.h"
#include "stm32h7xx_hal_adc.h"

/* Public variables */

/* Public function */
void Button_Panel_INIT(ADC_HandleTypeDef *handle);
void Button_Panel_ADC();


#endif /* BUTTON_PANEL_H */