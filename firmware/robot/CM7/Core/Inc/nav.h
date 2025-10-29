#ifndef NAV_H
#define NAV_H

/*
 * Public includes
 */
#include "main.h"
#include <robot_action.pb-c.h>

// Which positions robot should go to, sent by AI dudes

typedef struct {
    float x;
    float y;
    float w;
    uint8_t movement_enabled;
    uint8_t panic;
} robot_nav_command;

typedef enum
{
    UP,
    DOWN,
    LEFT,
    RIGHT
} DIRECTION;

/**
 * Initialises navigation system by creating motor interfaces.
 *
 * @param htim Timer handle for motors.
 */
void NAV_Init(TIM_HandleTypeDef* motor_tick_itr,
              TIM_HandleTypeDef* pwm_htim,
              TIM_HandleTypeDef* pwm15_htim);

/**
 * Updates the encoder count, and motor control for all four motors.
 */
void NAV_update_motor_state();

/**
 * Set robot wheel speed from velocity vector in robot coordinate space.
 * @param vx, Velocity in the x direction towards the dribbler
 * @param vy, Velocity in the y direction positive left of x-axis
 * @param w,  rotational velocity
 */
void NAV_steer(float vx, float vy, float w);

/**
 * Stops all movement
 */
void NAV_Stop();

/**
 * Execute command depending on type
 *
 * @param cmd Command to be executed
 */
void NAV_HandleCommand(Command* cmd);

/**
 * Disables the pwm output to the motors.
 */
void NAV_StopMovement();

/**
 * Set robot speed to one unit forward
 */
void NAV_TestMovement();

/**
 * Write stop signal on dribbler pin
 */
void NAV_StopDribbler();

/**
 * Write stop signal on dribbler pin
 */
void NAV_RunDribbler();

/**
 * Start dribbler, wait 2 seconds, stop dribbler
 */
void NAV_TestDribbler();

/**
 * Log motor speeds in m/s
 */
void NAV_log_speed();

/**
 * Run one wheel at a time from first to last first forward in two seconds
 * then backwards for two more.
 */
void NAV_TireTest();

/**
 * Sets robot_cmd disabled
 */
void NAV_DisableMovement();

/**
 * Sets robot_cmd disabled
 */
void NAV_EnableMovement();

/**
 * Inserts the robot velocity in robot coordiante frame from wheel speeds.
 *
 * @param res a 3x1 vector with layout [u, v, w]
 */
void NAV_wheelToBody(float* res);

/**
 * Sets the robot_cmd to move to the desired position
 */
void NAV_GoToAction(Command* command);

/**
 * Returns flag for nav panic, 1 if paniced 0 if not.
 */
uint8_t NAV_IsPanic();

/**
 * Sets the panic flag, this should stop the robot although this is not
 * implemented.
 */
void NAV_SetRobotPanic();

/**
 * Resets the robot after a panic,
 * this is curently not implemented.
 */
void NAV_ClearRobotPanic();

/**
 * Sets the goal position of the nav program in robot centric coordinates.
 * @param x, y, z
 */
void NAV_TEST_Set_robot_cmd(float x, float y, float w);

/**
 * Get the goal x position
 */
float NAV_GetNavX();

/**
 * Get the goal y position
 */
float NAV_GetNavY();

/**
 * Get the goal w rotation
 */
float NAV_GetNavW();
void NAV_SetCommandPosition(float nav_x, float nav_y, float nav_z);
#endif /* NAV_H */
