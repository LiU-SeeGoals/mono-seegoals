#ifndef __MOTOR_H
#define __MOTOR_H

#include "stm32h7xx_hal.h"
// Tick buffer of 100 with 1000hz update is 0.1 second filter on motor
#define motor_tick_buf_size 10
typedef struct {
    TIM_HandleTypeDef* encoder_htim;
    TIM_HandleTypeDef* pwm_htim;
    uint32_t channel;
    int ticks;
    float speed;
    int prev_tick;
    GPIO_TypeDef* breakPinPort;
    uint16_t breakPin;
    GPIO_TypeDef* reversePinPort;
    uint16_t reversePin;
    uint16_t dir;
    float motor_ticks[motor_tick_buf_size];
    int cur_tick_idx;
} MotorPWM;

/**
 * Initilaize motor
 */
void MOTOR_Init(TIM_HandleTypeDef* htim);

/**
 * Returns the average motor ticks per iteration
 */
float MOTOR_get_motor_ticks_per_iteration(MotorPWM* motor);

/**
 * Update motor tick buffer
 * @param val float of motor ticks for this iteration.
 */
void MOTOR_update_motor_ticks(MotorPWM* motor, float val);

/**
 * Enables the PWM signal
 */
void MOTOR_PWMStart(MotorPWM* motor);

/**
 * Disables the PWM signal
 * @param motor Pointer to motor.
 */
void MOTOR_PWMStop(MotorPWM* motor);

/**
 * Set speed of motor in percent 0 - 1
 * TODO: Negative values are interpreted as reverse
 */
void MOTOR_SendPWM(MotorPWM* motor, float pulse_width);

/**
 * Runs one iteration of a PI-loop with setpoint speed to controll motor speed.
 * @param speed Motor speed setpoint in ticks per second
 * @param I_prev A pointer to store the previous I value, should be initialized as zero
 */
void MOTOR_SetSpeed(MotorPWM* motor, float speed, float* I_prev);

/**
 * Sets the break pin HIGH, engaging break
 */
void MOTOR_Break(MotorPWM* motor);

/**
 * Sets the break pin LOW, disengaging break
 */
void MOTOR_StopBreak(MotorPWM* motor);

/**
 * Sets the direction of the motor depending on the sign of the speed.
 * @param speed The speed to take direction from
 */
int setDirection(MotorPWM* motor, float speed);

/**
 * Returns the speed of the motor in ticks per second.
 *
 * @returns float of speed
 */
float MOTOR_ReadSpeed(MotorPWM* motor);

#endif
