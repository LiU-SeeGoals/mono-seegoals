/* USER CODE BEGIN Header */
/**
 ******************************************************************************
 * @file           : main.h
 * @brief          : Header for main.c file.
 *                   This file contains the common defines of the application.
 ******************************************************************************
 * @attention
 *
 * Copyright (c) 2023 STMicroelectronics.
 * All rights reserved.
 *
 * This software is licensed under terms that can be found in the LICENSE file
 * in the root directory of this software component.
 * If no LICENSE file comes with this software, it is provided AS-IS.
 *
 ******************************************************************************
 */
/* USER CODE END Header */

/* Define to prevent recursive inclusion -------------------------------------*/
#ifndef __MAIN_H
#define __MAIN_H

#ifdef __cplusplus
extern "C" {
#endif

/* Includes ------------------------------------------------------------------*/
#include "stm32h7xx_hal.h"

/* Private includes ----------------------------------------------------------*/
/* USER CODE BEGIN Includes */
#include <stdatomic.h>
/* USER CODE END Includes */

/* Exported types ------------------------------------------------------------*/
/* USER CODE BEGIN ET */

/* USER CODE END ET */

/* Exported constants --------------------------------------------------------*/
/* USER CODE BEGIN EC */

/* USER CODE END EC */

/* Exported macro ------------------------------------------------------------*/
/* USER CODE BEGIN EM */

/* USER CODE END EM */

void HAL_TIM_MspPostInit(TIM_HandleTypeDef *htim);

/* Exported functions prototypes ---------------------------------------------*/
void Error_Handler(void);

/* USER CODE BEGIN EFP */
#define TASK_PING 1
extern volatile atomic_uint main_tasks;
/* USER CODE END EFP */

/* Private defines -----------------------------------------------------------*/
#define MOTOR4_BREAK_Pin GPIO_PIN_2
#define MOTOR4_BREAK_GPIO_Port GPIOE
#define MOTOR2_REVERSE_Pin GPIO_PIN_4
#define MOTOR2_REVERSE_GPIO_Port GPIOE
#define MOTOR2_PWM_Pin GPIO_PIN_5
#define MOTOR2_PWM_GPIO_Port GPIOE
#define MOTOR2_BREAK_Pin GPIO_PIN_6
#define MOTOR2_BREAK_GPIO_Port GPIOE
#define BTN_USER_Pin GPIO_PIN_13
#define BTN_USER_GPIO_Port GPIOC
#define BTN_USER_EXTI_IRQn EXTI15_10_IRQn
#define IMU_CS_Pin GPIO_PIN_1
#define IMU_CS_GPIO_Port GPIOA
#define IMU_INT1_Pin GPIO_PIN_2
#define IMU_INT1_GPIO_Port GPIOA
#define IMU_INT2_Pin GPIO_PIN_3
#define IMU_INT2_GPIO_Port GPIOA
#define NRF_SCK_Pin GPIO_PIN_5
#define NRF_SCK_GPIO_Port GPIOA
#define NRF_MISO_Pin GPIO_PIN_6
#define NRF_MISO_GPIO_Port GPIOA
#define LED_GREEN_Pin GPIO_PIN_0
#define LED_GREEN_GPIO_Port GPIOB
#define MOTOR1_BREAK_Pin GPIO_PIN_2
#define MOTOR1_BREAK_GPIO_Port GPIOB
#define MOTOR3_REVERSE_Pin GPIO_PIN_7
#define MOTOR3_REVERSE_GPIO_Port GPIOE
#define OLD_MOTOR2_REVERSE_Pin GPIO_PIN_8
#define OLD_MOTOR2_REVERSE_GPIO_Port GPIOE
#define MOTOR1_PWM_Pin GPIO_PIN_9
#define MOTOR1_PWM_GPIO_Port GPIOE
#define OLD_MOTOR2_PWM_Pin GPIO_PIN_11
#define OLD_MOTOR2_PWM_GPIO_Port GPIOE
#define KICKER_CS_Pin GPIO_PIN_12
#define KICKER_CS_GPIO_Port GPIOE
#define MOTOR3_PWM_Pin GPIO_PIN_13
#define MOTOR3_PWM_GPIO_Port GPIOE
#define MOTOR4_PWM_Pin GPIO_PIN_14
#define MOTOR4_PWM_GPIO_Port GPIOE
#define MOTOR4_REVERSE_Pin GPIO_PIN_15
#define MOTOR4_REVERSE_GPIO_Port GPIOE
#define KICKER_DISCHARGE2_Pin GPIO_PIN_10
#define KICKER_DISCHARGE2_GPIO_Port GPIOB
#define KICKER_CHARGE_Pin GPIO_PIN_11
#define KICKER_CHARGE_GPIO_Port GPIOB
#define KICKER_DISCHARGE1_Pin GPIO_PIN_12
#define KICKER_DISCHARGE1_GPIO_Port GPIOB
#define KICKER_ERR_Pin GPIO_PIN_13
#define KICKER_ERR_GPIO_Port GPIOB
#define LED_RED_Pin GPIO_PIN_14
#define LED_RED_GPIO_Port GPIOB
#define NRF_IRQ_Pin GPIO_PIN_15
#define NRF_IRQ_GPIO_Port GPIOB
#define NRF_IRQ_EXTI_IRQn EXTI15_10_IRQn
#define MOTOR3_BREAK_Pin GPIO_PIN_12
#define MOTOR3_BREAK_GPIO_Port GPIOD
#define OLD_MOTOR2_BREAK_Pin GPIO_PIN_13
#define OLD_MOTOR2_BREAK_GPIO_Port GPIOD
#define NRF_CE_Pin GPIO_PIN_6
#define NRF_CE_GPIO_Port GPIOC
#define IR_IN_Pin GPIO_PIN_11
#define IR_IN_GPIO_Port GPIOA
#define IR_OUT_Pin GPIO_PIN_12
#define IR_OUT_GPIO_Port GPIOA
#define EXT_LED1_RED_Pin GPIO_PIN_1
#define EXT_LED1_RED_GPIO_Port GPIOD
#define EXT_LED4_BLUE_Pin GPIO_PIN_3
#define EXT_LED4_BLUE_GPIO_Port GPIOD
#define EXT_LED2_BLUE_Pin GPIO_PIN_4
#define EXT_LED2_BLUE_GPIO_Port GPIOD
#define DRIBBLER_Pin GPIO_PIN_5
#define DRIBBLER_GPIO_Port GPIOD
#define EXT_LED3_WHITE_Pin GPIO_PIN_6
#define EXT_LED3_WHITE_GPIO_Port GPIOD
#define BUZZER_Pin GPIO_PIN_7
#define BUZZER_GPIO_Port GPIOD
#define SPI_MOSI_Pin GPIO_PIN_5
#define SPI_MOSI_GPIO_Port GPIOB
#define MOTOR1_REVERSE_Pin GPIO_PIN_6
#define MOTOR1_REVERSE_GPIO_Port GPIOB
#define NRF_CSN_Pin GPIO_PIN_8
#define NRF_CSN_GPIO_Port GPIOB
#define LED_YELLOW_Pin GPIO_PIN_1
#define LED_YELLOW_GPIO_Port GPIOE

/* USER CODE BEGIN Private defines */
#define IS_ROBOT 1
#define IMU_SCL_Pin GPIO_PIN_14
#define IMU_SCL_GPIO_Port GPIOF
#define IMU_SDA_Pin GPIO_PIN_15
#define IMU_SDA_GPIO_Port GPIOF
#define MOTOR1_TIM_CHANNEL TIM_CHANNEL_1
#define MOTOR2_TIM_CHANNEL TIM_CHANNEL_1
#define MOTOR3_TIM_CHANNEL TIM_CHANNEL_3
#define MOTOR4_TIM_CHANNEL TIM_CHANNEL_4
/* USER CODE END Private defines */

#ifdef __cplusplus
}
#endif

#endif /* __MAIN_H */
