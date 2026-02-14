#include "nav.h"
#include "common.h"
#include "kicker.h"
#include "data_logging.h"
#include "pos_follow.h"
#include "state_estimator.h"

/*
 * Private includes
 */
#include "arm_math.h"
#include "stm32h7xx_it.h"
#include "log.h"
#include "motor.h"
#include <stdlib.h>

/*
 * Private variables
 */
static LOG_Module internal_log_mod;
static MotorPWM motors[4];
static robot_nav_command robot_cmd;
static float I_prevs[4]; // PI control I-parts
const float CLOCK_FREQ = 400000000;
const float CONTROL_TIM_FREQ = 200000000;
float CONTROL_FREQ; // set in init
static int queued = 0;
static int is_remote_controlled = 0;

/* Private functions declarations */
void set_motors(float m1, float m2, float m3, float m4);

/*
 * Public function implementations
 */

void NAV_Init(TIM_HandleTypeDef* motor_tick_itr,
              TIM_HandleTypeDef* pwm_htim,
              TIM_HandleTypeDef* pwm15_htim)
{
    LOG_InitModule(&internal_log_mod, "NAV", LOG_LEVEL_INFO, 0);
    HAL_TIM_Base_Start(pwm_htim);
    HAL_TIM_Base_Start(pwm15_htim);

    motors[0].pwm_htim = pwm_htim;
    motors[0].ticks = 0;
    motors[0].speed = 0.f;
    motors[0].prev_tick = 0;
    motors[0].channel = TIM_CHANNEL_1;
    motors[0].breakPinPort = MOTOR1_BREAK_GPIO_Port;
    motors[0].breakPin = MOTOR1_BREAK_Pin;
    motors[0].reversePinPort = MOTOR1_REVERSE_GPIO_Port;
    motors[0].reversePin = MOTOR1_REVERSE_Pin;
    motors[0].dir = 1;

    motors[1].pwm_htim = pwm_htim;
    motors[1].ticks = 0;
    motors[1].speed = 0.f;
    motors[1].prev_tick = 0;
    motors[1].channel = TIM_CHANNEL_2;
    motors[2].breakPinPort = MOTOR2_BREAK_GPIO_Port;
    motors[2].breakPin = MOTOR2_BREAK_Pin;
    motors[1].reversePinPort = MOTOR2_REVERSE_GPIO_Port;
    motors[1].reversePin = MOTOR2_REVERSE_Pin;
    motors[1].dir = 1;

    motors[2].pwm_htim = pwm_htim;
    motors[2].ticks = 0;
    motors[2].speed = 0.f;
    motors[2].prev_tick = 0;
    motors[2].channel = TIM_CHANNEL_3;
    motors[2].breakPinPort = MOTOR3_BREAK_GPIO_Port;
    motors[2].breakPin = MOTOR3_BREAK_Pin;
    motors[2].reversePinPort = MOTOR3_REVERSE_GPIO_Port;
    motors[2].reversePin = MOTOR3_REVERSE_Pin;
    motors[2].dir = 1;

    motors[3].pwm_htim = pwm_htim;
    motors[3].ticks = 0;
    motors[3].speed = 0.f;
    motors[3].prev_tick = 0;
    motors[3].channel = TIM_CHANNEL_4;
    motors[3].breakPinPort = MOTOR4_BREAK_GPIO_Port;
    motors[3].breakPin = MOTOR4_BREAK_Pin;
    motors[3].reversePinPort = MOTOR4_REVERSE_GPIO_Port;
    motors[3].reversePin = MOTOR4_REVERSE_Pin;
    motors[3].dir = 1;

    MOTOR_PWMStart(&motors[0]);
    MOTOR_PWMStart(&motors[1]);
    MOTOR_PWMStart(&motors[2]);
    MOTOR_PWMStart(&motors[3]);

    // memset did not work, idc
    for (int i = 0; i < 4; i++) {
        motors[i].cur_tick_idx = 0;
        motors[i].cur_tick_idx = 0;
        I_prevs[i] = 0.0f;
        for (int j = 0; j < MOTOR_TICK_BUF_SIZE; j++) {
            motors[i].motor_ticks[j] = 0;
        }
    }

    robot_cmd.x = 0;
    robot_cmd.y = 0;
    robot_cmd.w = 0;

    NAV_EnableMovement();
    float control_clock_prescaler = motor_tick_itr->Init.Prescaler + 1;
    float control_clock_period = motor_tick_itr->Init.Period + 1;
    CONTROL_FREQ = CONTROL_TIM_FREQ / (control_clock_prescaler * control_clock_period);
    HAL_TIM_Base_Start_IT(motor_tick_itr);
}

void NAV_update_motor_state()
{

    int* motor_ticks = ITR_GetMotorTicks();

    for (int i = 0; i < 4; i++) {
        int ticks_before = motors[i].prev_tick;
        int new_ticks = motor_ticks[i];
        MOTOR_update_motor_ticks(&motors[i], new_ticks - ticks_before);
        motors[i].ticks = new_ticks - ticks_before;
        motors[i].prev_tick = new_ticks;
    }

    // Dont move this into the other for loop, we want motors to run simultanious!!
    ControlSignal sigs[4];
    for (int i = 0; i < 4; i++) { // do for all motor
        if (robot_cmd.movement_enabled == 1) {
            sigs[i] = MOTOR_SetSpeed(&motors[i], motors[i].speed, &I_prevs[i]);
        } else {
            sigs[i] = MOTOR_SetSpeed(&motors[i], 0, &I_prevs[i]);
        }
    }
    DATA_log_motor(sigs[0], sigs[1], sigs[2], sigs[3]);
}

// res is a 3x1 vector
void NAV_wheelToBody(float* res)
{

    // wheel to body psudeo inverse https://tdpsearch.com/#/tdp/soccer_smallsize__2020__RoboTeam_Twente__0?ref=list
    // TODO: measure real wheel radius and chasis radius
    float r = 0.025;
    float R = 0.09;

    float psi = PI * 31.f / 180.0f;
    float theta = PI * 45.f / 180.0f;

    float wrf = MOTOR_GetMotorSign(&motors[0]) * MOTOR_ReadTicksPerSecond(&motors[0]) / 48.f * 2.0f * PI;
    float wlf = MOTOR_GetMotorSign(&motors[1]) * MOTOR_ReadTicksPerSecond(&motors[1]) / 48.f * 2.0f * PI;
    float wlb = MOTOR_GetMotorSign(&motors[2]) * MOTOR_ReadTicksPerSecond(&motors[2]) / 48.f * 2.0f * PI;
    float wrb = MOTOR_GetMotorSign(&motors[3]) * MOTOR_ReadTicksPerSecond(&motors[3]) / 48.f * 2.0f * PI;

    // motors[0].speed = wrf;
    // motors[1].speed = wlf;
    // motors[2].speed = wlb;
    // motors[3].speed = wrb;

    float cos_psi = arm_cos_f32(psi);
    float cos_theta = arm_cos_f32(theta);
    float sin_psi = arm_sin_f32(psi);
    float sin_theta = arm_sin_f32(theta);

    float denom1 = 2.0f * (cos_psi * cos_psi + cos_theta * cos_theta);

    float m11 = r * cos_psi / denom1;
    float m12 = r * cos_theta / denom1;
    float m13 = r * -cos_theta / denom1;
    float m14 = r * -cos_psi / denom1;

    float denom2 = 2.0f * (sin_psi + sin_theta);

    float m21 = r * 1.f / denom2;
    float m22 = r * -1.f / denom2;
    float m23 = r * -1.f / denom2;
    float m24 = r * 1.f / denom2;

    float denom3 = 2 * R * (sin_psi + sin_theta);

    float m31 = sin_theta / denom3;
    float m32 = sin_psi / denom3;
    float m33 = sin_psi / denom3;
    float m34 = sin_theta / denom3;

    float u = wrf * m11 + wrb * m12 + wlb * m13 + wlf * m14;
    float v = wrf * m21 + wrb * m22 + wlb * m23 + wlf * m24;
    float w = wrf * m31 + wrb * m32 + wlb * m33 + wlf * m34;

    DATA_log_odometry(u, v, w);
    res[0] = u;
    res[1] = v;
    res[2] = w;
}

void NAV_steer(float u, float v, float w)
{
    // Ref: https://tdpsearch.com/#/tdp/soccer_smallsize__2020__RoboTeam_Twente__0?ref=list
    // wheels RF, RB, LB, LF
    // wheel direction is RF forward vector toward dribbler
    // u forward toward dribbler
    // v to the sides
    // w angle from LF to LB to RB to RF

    // u is x in robot frame
    // v is y in robot frame

    float psi = PI * 31.f / 180.0f;
    float theta = PI * 45.f / 180.0f;
    // r is wheel radius, R is chasis radius, currently 1 because idc and
    // our speeds are currently not a real unit i.e. ticks/second and not meter/second
    float r = 1.f;
    float R = 1.f;

    float wrf = 1.0 / r * (u * arm_cos_f32(psi) + v * arm_sin_f32(psi) + w * R);
    float wrb = 1.0 / r * (u * arm_cos_f32(theta) - v * arm_sin_f32(theta) + w * R);
    float wlb = 1.0 / r * (-u * arm_cos_f32(theta) - v * arm_sin_f32(theta) + w * R);
    float wlf = 1.0 / r * (-u * arm_cos_f32(psi) + v * arm_sin_f32(psi) + w * R);

    // Wheel-space saturation: scale all wheels proportionally to preserve velocity ratio
    const float WHEEL_MAX = 400.0f;
    float max_wheel = fabsf(wrf);
    if (fabsf(wrb) > max_wheel) max_wheel = fabsf(wrb);
    if (fabsf(wlb) > max_wheel) max_wheel = fabsf(wlb);
    if (fabsf(wlf) > max_wheel) max_wheel = fabsf(wlf);

    if (max_wheel > WHEEL_MAX) {
        float scale = WHEEL_MAX / max_wheel;
        wrf *= scale;
        wrb *= scale;
        wlb *= scale;
        wlf *= scale;
    }

    motors[0].speed = wrf;
    motors[1].speed = wlf;
    motors[2].speed = wlb;
    motors[3].speed = wrb;
}

void NAV_Direction(DIRECTION dir)
{
    switch (dir) {
    case UP:
        MOTOR_PWMStart(&motors[0]);
        break;
    case DOWN:
        MOTOR_PWMStart(&motors[1]);
        break;
    case LEFT:
        MOTOR_PWMStart(&motors[2]);
        break;
    case RIGHT:
        MOTOR_PWMStart(&motors[3]);
        break;
    }
}

void NAV_Stop()
{
    MOTOR_PWMStop(&motors[0]);
    MOTOR_PWMStop(&motors[1]);
    MOTOR_PWMStop(&motors[2]);
    MOTOR_PWMStop(&motors[3]);
}

float speed = 0;

void NAV_TestMovement() { NAV_steer(1, 0, 0); }

void NAV_DisableMovement() { robot_cmd.movement_enabled = 0; }

void NAV_EnableMovement() { robot_cmd.movement_enabled = 1; }

int NAV_IsRemoteControlled() { return is_remote_controlled; }

void NAV_HandleCommand(Command* cmd)
{
    static int kicks_since_last_kick = 0;

    switch (cmd->command_id) {
    case ACTION_TYPE__STOP_ACTION:
        NAV_DisableMovement();
        break;
    case ACTION_TYPE__MOVE_TO_ACTION: {

        NAV_EnableMovement();
        NAV_GoToAction(cmd);

        if(cmd->angular_vel == 1)
        {
            NAV_RunDribbler();
        }
        else
        {
            NAV_StopDribbler();
        }

    } break;

    case ACTION_TYPE__MOVE_ACTION: {
        const int32_t kickSpeed = cmd->kick_speed;
        const int32_t x = cmd->direction->x;
        const int32_t y = cmd->direction->y;

        const float angle = cmd->angular_vel / 100.f;

        is_remote_controlled = 1;
        if (kickSpeed == 1)
        {
            KICKER_ChargeStart();
        }
        else if (x == 0 && y == 0)
        {
            STATE_set_posx(0);
            STATE_set_posy(0);
            NAV_SetCommandPosition(0, 0, angle);
        }
        else
        {
            NAV_EnableMovement();
            NAV_SetCommandPosition(x*1000, y*1000, angle);
        }

    } break;
    case ACTION_TYPE__ROTATE_ACTION:
        break;
    case ACTION_TYPE__KICK_ACTION:
        NAV_EnableMovement();

        KICKER_ChargeStart();
        NAV_GoToAction(cmd);

        break;
    default:
        LOG_ERROR("Not known command: %i\r\n", cmd->command_id);
        break;
    }
    if (cmd->command_id == ACTION_TYPE__KICK_ACTION)
    {
        kicks_since_last_kick++;
    }
    else
    {
        kicks_since_last_kick = 0;
    }
}

/*
 * Private function implementations
 */

void NAV_GoToAction(Command* cmd)
{
    // Only initialised on first run since static
    // Large values to always respect first vision data received
    static int32_t prev_cam_x = 2147483647;
    static int32_t prev_cam_y = 2147483647;
    static int32_t prev_cam_w = 2147483647;

    const int32_t nav_x = cmd->dest->x;
    const int32_t nav_y = cmd->dest->y;
    const int32_t nav_w = cmd->dest->w;

    const int32_t cam_x = cmd->pos->x;
    const int32_t cam_y = cmd->pos->y;
    const int32_t cam_w = cmd->pos->w;

    // Within the robot we work in meters
    // Angle is scaled by 1000 before sent to robot.
    const float f_nav_x = ((float)nav_x) / 1000.f;
    const float f_nav_y = ((float)nav_y) / 1000.f;
    const float f_nav_w = ((float)nav_w) / 1000.f;

    const float f_cam_x = ((float)cam_x) / 1000.f;
    const float f_cam_y = ((float)cam_y) / 1000.f;
    const float f_cam_w = ((float)cam_w) / 1000.f;

    robot_cmd.x = f_nav_x;
    robot_cmd.y = f_nav_y;
    robot_cmd.w = f_nav_w;

    // -- Vision update --

    int32_t diff = prev_cam_x - cam_x + prev_cam_y - cam_y + prev_cam_w - cam_w;

    if (abs(diff) < 2) {
        // If the vision position is exactly the same as last time it is likely not updated information.
        // Ignore old information
        // LOG_INFO("ignoring vision %d\r\n", diff);
        return;
    }


    // LOG_INFO("diff %d\r\n", diff);


    prev_cam_x = cam_x;
    prev_cam_y = cam_y;
    prev_cam_w = cam_w;

    STATE_FusionEKFVisionUpdate(f_cam_x, f_cam_y, f_cam_w);
}

/*
   Set position for robot to move to in (meter)

   Can be used in demos when there is no vision / software updates
*/
void NAV_SetCommandPosition(float nav_x, float nav_y, float nav_z)
{
    robot_cmd.x = nav_x;
    robot_cmd.y = nav_y;
    robot_cmd.w = nav_z;
}

void NAV_TEST_TireTest()
{
    const int us_to_sec = 50000;

    for (int i = 0; i < 4; i++) {
        LOG_INFO("Motor %d clockwise (forward)...\r\n", i);
        float zero = 0;
        setDirection(&motors[i], 80);
        MOTOR_SendPWM(&motors[i], 0.2);
        COMMON_Wait(us_to_sec);
        MOTOR_SendPWM(&motors[i], 0);
    }

    for (int i = 0; i < 4; i++) {
        LOG_INFO("Motor %d counterclockwise (backwards)...\r\n", i);
        float zero = 0;
        setDirection(&motors[i], -80);
        MOTOR_SendPWM(&motors[i], 0.2);
        COMMON_Wait(us_to_sec);
        MOTOR_SendPWM(&motors[i], 0);
    }
}

void set_motors(float m1, float m2, float m3, float m4)
{
    motors[0].speed = m1 * 100.f;
    motors[1].speed = m2 * 100.f;
    motors[2].speed = m3 * 100.f;
    motors[3].speed = m4 * 100.f;
}

void NAV_StopDribbler() {
    // LOG_DEBUG("stop dirbling\r\n");
    HAL_GPIO_WritePin(DRIBBLER_GPIO_Port, DRIBBLER_Pin, GPIO_PIN_RESET); }

void NAV_RunDribbler() {
    // LOG_DEBUG("dirbling\r\n");
    HAL_GPIO_WritePin(DRIBBLER_GPIO_Port, DRIBBLER_Pin, GPIO_PIN_SET); }

void NAV_TestDribbler()
{
    NAV_RunDribbler();
    HAL_Delay(2000);
    NAV_StopDribbler();
}

void NAV_TEST_pwm()
{
    float speed = 0.15;
    MOTOR_SendPWM(&motors[0], speed);
    MOTOR_SendPWM(&motors[1], speed);
    MOTOR_SendPWM(&motors[2], speed);
    MOTOR_SendPWM(&motors[3], speed);
}

void NAV_TEST_Set_robot_cmd(float x, float y, float w)
{
    robot_cmd.x = x;
    robot_cmd.y = y;
    robot_cmd.w = w;
}

float NAV_GetNavX() { return robot_cmd.x; }

float NAV_GetNavY() { return robot_cmd.y; }

float NAV_GetNavW() { return robot_cmd.w; }

