#ifndef UI_H
#define UI_H

#include "main.h"

/* Public variables */
// ...

/*
 * Public function
 */

void UI_Init(UART_HandleTypeDef* huart);
void UI_RxCallback();

#endif /* UI_H */
