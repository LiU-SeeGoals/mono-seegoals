#include "motor.h"

/* Private includes */
#include "log.h"

/* Private variables */
static LOG_Module internal_log_mod;
extern float CONTROL_FREQ;

/*
 Averages motor speed ticks
*/
void MOTOR_update_motor_ticks(MotorPWM* motor, float val)
{
    motor->cur_tick_idx = (1 + motor->cur_tick_idx) % (motor_tick_buf_size);
    motor->motor_ticks[motor->cur_tick_idx] = val;
}

/*
 Get motor speed
*/
float MOTOR_get_motor_ticks_per_iteration(MotorPWM* motor)
{
    // Can be made faster keeping a moving average and removing the last adding the new?
    float cur = 0;
    for (int i = 0; i < motor_tick_buf_size; i++) {
        cur += motor->motor_ticks[i];
    }
    return cur / ((float)motor_tick_buf_size);
}

void MOTOR_Init(TIM_HandleTypeDef* pwm_htim)
{
    LOG_InitModule(&internal_log_mod, "MOTOR", LOG_LEVEL_TRACE, 0);
    HAL_TIM_Base_Start(pwm_htim);
}

void MOTOR_PWMStop(MotorPWM* motor)
{
    // TODO: This might disable the timer for all channels, not sure.
    HAL_TIM_PWM_Stop(motor->pwm_htim, motor->channel);
}

void MOTOR_PWMStart(MotorPWM* motor) { HAL_TIM_PWM_Start(motor->pwm_htim, motor->channel); }

void MOTOR_Break(MotorPWM* motor) { HAL_GPIO_WritePin(motor->breakPinPort, motor->breakPin, GPIO_PIN_SET); }

void MOTOR_StopBreak(MotorPWM* motor) { HAL_GPIO_WritePin(motor->breakPinPort, motor->breakPin, GPIO_PIN_RESET); }

/*
  Reverses motor direction and makes sure that the motor is stopped before reversing
*/
int setDirection(MotorPWM* motor, float speed)
{
    // In theory we should have saftey checks to not reverse motor
    // if it is running... fukit

    // If going backward and speed is positive, change direction
    if (motor->dir == 0 && speed > 0) {
        motor->dir = 1;
        HAL_GPIO_WritePin(motor->reversePinPort, motor->reversePin, GPIO_PIN_RESET);
    }
    // If going forward and speed is negative, change direction
    if (motor->dir == 1 && speed < 0) {
        motor->dir = 0;
        HAL_GPIO_WritePin(motor->reversePinPort, motor->reversePin, GPIO_PIN_SET);
    }
    return HAL_OK;
}

// TODO: How to not have globals? Will cause issues if function is not called
// for some time
/*
  PI controls the motor to the given speed value in hall ticks / second
  Updates I_prev with the previous integrator value

*/
void MOTOR_SetSpeed(MotorPWM* motor, float speed, float* I_prev)
{

    setDirection(motor, speed);
    /*if (setDirection(motor, speed) == HAL_BUSY) {*/
    /*  MOTOR_SendPWM(motor, 0);*/
    /*}*/

    if (speed < 0) {
        speed = -speed;
    }
    // PI control loop with integrator windup protection

    float umin = 0;
    float umax = 1;
    float Ts = 1.f / CONTROL_FREQ;
    float Ti = 0.02;
    float K = 0.00015;
    float current_speed = (float)MOTOR_ReadSpeed(motor);
    float error = speed - current_speed;
    float I = *I_prev + Ts / Ti * error;
    float v = K * (error + I);
    float u = 0;
    // integrator windup fix
    if (v < umin || v > umax) {
        I = *I_prev;
    }
    if (v > umax) {
        u = umax;
    } else if (v < umin) {
        u = umin;
    } else {
        u = v;
    }
    MOTOR_SendPWM(motor, u);
    *I_prev = I;
    // for some time
}

/*
  Send pwm with with pulse width of pulse_width * timer_period
  meaning pulse width is a float between 0 - 1.
*/

void MOTOR_SendPWM(MotorPWM* motor, float pulse_width)
{
    // TODO: How to handle changing directions?

    // Make sure we dont explode the timer limit
    if (pulse_width > 1) {
        pulse_width = 1;
    }
    if (pulse_width < 0) {
        pulse_width = 0;
    }

    float max_scale = 0.5;                 // Use a max scaling for the motor speed
    float scale = max_scale * pulse_width; // make max_scale largest scaling

    // TODO: How to handle rounding errors, do they even matter?
    int pwm_speed = motor->pwm_htim->Init.Period * scale;

    __HAL_TIM_SET_COMPARE(motor->pwm_htim, motor->channel, pwm_speed);
}

float MOTOR_ReadSpeed(MotorPWM* motor)
{
    // When I write this control freq should be 2000, each iteration is a
    // time-step of 1/2000 so this gives the speed in ticks per second.
    float speed_s = MOTOR_get_motor_ticks_per_iteration(motor) * CONTROL_FREQ;

    return speed_s;
}
