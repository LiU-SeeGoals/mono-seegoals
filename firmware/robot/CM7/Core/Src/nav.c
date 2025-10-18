#include "nav.h"
#include "kicker.h"
#include "pos_follow.h"
#include "state_estimator.h"

/*
 * Private includes
 */
#include "arm_math.h"
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
float CONTROL_FREQ; // set in init
static int queued = 0;

/* Private functions declarations */
void set_motors(float m1, float m2, float m3, float m4);

/*
 * Public function implementations
 */

void NAV_Init(TIM_HandleTypeDef* motor_tick_itr,
              TIM_HandleTypeDef* pwm_htim,
              TIM_HandleTypeDef* pwm15_htim,
              TIM_HandleTypeDef* encoder1_htim,
              TIM_HandleTypeDef* encoder2_htim,
              TIM_HandleTypeDef* encoder3_htim,
              TIM_HandleTypeDef* encoder4_htim)
{
    LOG_InitModule(&internal_log_mod, "NAV", LOG_LEVEL_DEBUG, 0);
    HAL_TIM_Base_Start(pwm_htim);
    HAL_TIM_Base_Start(pwm15_htim);
    HAL_TIM_Base_Start(encoder1_htim);
    HAL_TIM_Base_Start(encoder2_htim);
    HAL_TIM_Base_Start(encoder3_htim);
    HAL_TIM_Base_Start(encoder4_htim);

    motors[0].pwm_htim = pwm_htim;
    motors[0].ticks = 0;
    motors[0].speed = 0.f;
    motors[0].prev_tick = 0;
    motors[0].encoder_htim = encoder1_htim;
    motors[0].channel = MOTOR1_TIM_CHANNEL;
    motors[0].breakPinPort = MOTOR1_BREAK_GPIO_Port;
    motors[0].breakPin = MOTOR1_BREAK_Pin;
    motors[0].reversePinPort = MOTOR1_REVERSE_GPIO_Port;
    motors[0].reversePin = MOTOR1_REVERSE_Pin;
    motors[0].encoderPinPort = MOTOR1_ENCODER_GPIO_Port;
    motors[0].encoderPin = MOTOR1_ENCODER_Pin;
    motors[0].dir = 1;

#ifdef PCB_MOTOR
    motors[1].pwm_htim = pwm15_htim;
    motors[1].ticks = 0;
    motors[1].speed = 0.f;
    motors[1].prev_tick = 0;
    motors[1].encoder_htim = encoder2_htim;
    motors[1].channel = MOTOR2_TIM_CHANNEL;
    motors[1].breakPinPort = MOTOR2_BREAK_GPIO_Port;
    motors[1].breakPin = MOTOR2_BREAK_Pin;
    motors[1].reversePinPort = MOTOR2_REVERSE_GPIO_Port;
    motors[1].reversePin = MOTOR2_REVERSE_Pin;
    motors[1].dir = 1;
#else
    motors[1].pwm_htim = pwm_htim;
    motors[1].ticks = 0;
    motors[1].speed = 0.f;
    motors[1].prev_tick = 0;
    motors[1].encoder_htim = encoder2_htim;
    motors[1].channel = TIM_CHANNEL_2;
    // motors[1].breakPinPort      = OLD_MOTOR2_BREAK_GPIO_Port;
    // motors[1].breakPin          = OLD_MOTOR2_BREAK_Pin;
    motors[1].reversePinPort = OLD_MOTOR2_REVERSE_GPIO_Port;
    motors[1].reversePin = OLD_MOTOR2_REVERSE_Pin;
    motors[1].dir = 1;
#endif

    motors[2].pwm_htim = pwm_htim;
    motors[2].ticks = 0;
    motors[2].speed = 0.f;
    motors[2].prev_tick = 0;
    motors[2].encoder_htim = encoder3_htim;
    motors[2].channel = MOTOR3_TIM_CHANNEL;
    motors[2].breakPinPort = MOTOR3_BREAK_GPIO_Port;
    motors[2].breakPin = MOTOR3_BREAK_Pin;
    motors[2].reversePinPort = MOTOR3_REVERSE_GPIO_Port;
    motors[2].reversePin = MOTOR3_REVERSE_Pin;
    motors[2].dir = 1;

    motors[3].pwm_htim = pwm_htim;
    motors[3].ticks = 0;
    motors[3].speed = 0.f;
    motors[3].prev_tick = 0;
    motors[3].encoder_htim = encoder4_htim;
    motors[3].channel = MOTOR4_TIM_CHANNEL;
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
        for (int j = 0; j < motor_tick_buf_size; j++) {
            motors[i].motor_ticks[j] = 0;
        }
    }

    robot_cmd.x = 0;
    robot_cmd.y = 0;
    robot_cmd.w = 0;

    NAV_EnableMovement();
    float control_clock_prescaler = motor_tick_itr->Init.Prescaler + 1;
    float control_clock_period = motor_tick_itr->Init.Period + 1;
    CONTROL_FREQ = CLOCK_FREQ / (control_clock_prescaler * control_clock_period);
    HAL_TIM_Base_Start_IT(motor_tick_itr);
}

void NAV_update_motor_state()
{

    for (int i = 0; i < 4; i++) {
        int ticks_before = motors[i].prev_tick;
        int new_ticks = motors[i].encoder_htim->Instance->CNT;
        MOTOR_update_motor_ticks(&motors[i], new_ticks - ticks_before);
        motors[i].ticks = new_ticks - ticks_before;
        motors[i].prev_tick = new_ticks;
    }

    // Dont move this into the other for loop, we want motors to run simultanious!!
    for (int i = 0; i < 4; i++) { // do for all motor
        if (robot_cmd.movement_enabled == 1) {
            MOTOR_SetSpeed(&motors[i], motors[i].speed, &I_prevs[i]);
        } else {
            MOTOR_SetSpeed(&motors[i], 0, &I_prevs[i]);
        }
    }
}

void NAV_log_speed() { LOG_INFO("Got speed m1 %f m2 %f m3 %f m4 %f\r\n", MOTOR_ReadSpeed(&motors[0]), MOTOR_ReadSpeed(&motors[1]), MOTOR_ReadSpeed(&motors[2]), MOTOR_ReadSpeed(&motors[3])); }

// res is a 3x1 vector
void NAV_wheelToBody(float* res)
{

    // wheel to body psudeo inverse https://tdpsearch.com/#/tdp/soccer_smallsize__2020__RoboTeam_Twente__0?ref=list
    // TODO: measure real wheel radius and chasis radius
    float r = 1.f;
    float R = 1.f;

    float psi = PI * 31.f / 180.0f;
    float theta = PI * 45.f / 180.0f;

    float wrf = MOTOR_get_motor_ticks_per_iteration(&motors[0]);
    float wrb = MOTOR_get_motor_ticks_per_iteration(&motors[1]);
    float wlb = MOTOR_get_motor_ticks_per_iteration(&motors[2]);
    float wlf = MOTOR_get_motor_ticks_per_iteration(&motors[3]);

    float cos_psi = arm_cos_f32(psi);
    float cos_theta = arm_cos_f32(theta);
    float sin_psi = arm_sin_f32(psi);
    float sin_theta = arm_sin_f32(theta);

    float m11 = r * (cos_psi / (2.0f * (cos_psi * cos_psi + cos_theta * cos_theta)));
    float m12 = m11;
    float m13 = -m11;
    float m14 = -m11;

    float m21 = r * (1.0 / (2.0f * (sin_psi + sin_theta)));
    float m22 = -m21;
    float m23 = -m21;
    float m24 = m21;

    float m31 = r * (sin_theta / (2.0f * R * (sin_psi + sin_theta)));
    float m32 = m31;
    float m33 = m31;
    float m34 = m31;

    float u = wrf * m11 + wrb * m12 + wlb * m13 + wlf * m14;
    float v = wrf * m21 + wrb * m22 + wlb * m23 + wlf * m24;
    float w = wrf * m31 + wrb * m32 + wlb * m33 + wlf * m34;
    res[0] = u;
    res[1] = v;
    res[2] = w;
}

void NAV_steer(float v, float u, float w)
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

    motors[0].speed = wrf;
    motors[1].speed = wrb;
    motors[2].speed = wlb;
    motors[3].speed = wlf;
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

void command_move(Command* cmd)
{

    LOG_INFO("got nav command %d %d %d \r\n", cmd->kick_speed, cmd->command_id, cmd->direction->x, cmd->direction->y);
    if (cmd->command_id == ACTION_TYPE__STOP_ACTION) {
        NAV_steer(0.f, 0.f, 0.f);
        return;
    }

    if (cmd->command_id == ACTION_TYPE__KICK_ACTION) {
        speed = cmd->kick_speed;
        if (speed > 10) {
            speed = 10;
        }
        return;
    }

    if (cmd->command_id == ACTION_TYPE__MOVE_ACTION) {
        NAV_steer(100.f * speed * cmd->direction->x, 100.f * speed * cmd->direction->y, 0.f);
    }
}

void NAV_TestMovement() { NAV_steer(1, 0, 0); }

void NAV_DisableMovement() { robot_cmd.movement_enabled = 0; }

void NAV_EnableMovement() { robot_cmd.movement_enabled = 1; }

void NAV_HandleCommand(Command* cmd)
{
    switch (cmd->command_id) {
    case ACTION_TYPE__STOP_ACTION:
        NAV_DisableMovement();
        LOG_DEBUG("Got stop (id %d)\r\n", cmd->robot_id);
        break;
    case ACTION_TYPE__MOVE_TO_ACTION: {
        NAV_EnableMovement();
        LOG_DEBUG("Got move (id %d)\r\n", cmd->robot_id);
        NAV_GoToAction(cmd);
    } break;

    case ACTION_TYPE__MOVE_ACTION: {
        const int32_t speed = cmd->kick_speed;
        const int32_t x = cmd->direction->x;
        const int32_t y = cmd->direction->y;

        NAV_EnableMovement();

        LOG_DEBUG("keyboard control (x,y,speed): (%i,%i,%i)\r\n", x, y, speed);
        LOG_DEBUG("keyboard control (x,y): (%f,%f)\r\n", 100.f * speed * x, 100.f * speed * y);
        // TODO: Should somehow know that we're in remote control mode
        if (0 <= speed && speed <= 10) {
            NAV_TEST_Set_robot_cmd(x, y, speed);
        }
    } break;
    case ACTION_TYPE__ROTATE_ACTION:
        break;
    case ACTION_TYPE__KICK_ACTION:
        KICKER_Charge();
        KICKER_Charge();
        KICKER_Charge();
        KICKER_Kick();
        break;
    default:
        LOG_ERROR("Not known command: %i\r\n", cmd->command_id);
        break;
    }
}

/*
 * Private function implementations
 */

int32_t prev_nav_x = 2147483647;
int32_t prev_nav_y = 2147483647;
int32_t prev_nav_w = 2147483647;

void NAV_GoToAction(Command* cmd)
{
    const int32_t nav_x = cmd->dest->x;
    const int32_t nav_y = cmd->dest->y;
    const int32_t nav_w = cmd->dest->w;

    const int32_t cam_x = cmd->pos->x;
    const int32_t cam_y = cmd->pos->y;
    const int32_t cam_w = cmd->pos->w;

    // Hax to cange to to float meter rep just for testing first time... hehe
    // Angle is scaled by 1000 before sent to robot.
    const float f_nav_x = ((float)nav_x) / 1000.f;
    const float f_nav_y = ((float)nav_y) / 1000.f;
    const float f_nav_w = ((float)nav_w) / 1000.f;

    const float f_cam_x = ((float)cam_x) / 1000.f;
    const float f_cam_y = ((float)cam_y) / 1000.f;
    const float f_cam_w = ((float)cam_w) / 1000.f;

    LOG_DEBUG("move to int: %d %d %d:\r\n", nav_x, nav_y, nav_w);
    LOG_DEBUG("Vision int: %d %d %d:\r\n", cam_x, cam_y, cam_w);
    LOG_DEBUG("Vision data: %f %f %f:\r\n", f_cam_x, f_cam_y, f_cam_w);
    LOG_DEBUG("Move to: %f %f %f:\r\n", f_nav_x, f_nav_y, f_nav_w);

    robot_cmd.x = f_nav_x;
    robot_cmd.y = f_nav_y;
    robot_cmd.w = f_nav_w;

    // -- Vision update --

    if (abs(prev_nav_x - nav_x + prev_nav_y - nav_y + prev_nav_w - nav_w) == 0) {
        // If the vision position is exactly the same as last time it is likely not updated information.
        // Ignore old information
        return;
    }

    STATE_FusionEKFVisionUpdate(f_cam_x, f_cam_y, f_cam_w);

    prev_nav_x = f_cam_x;
    prev_nav_y = f_cam_y;
    prev_nav_w = f_cam_w;
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

void NAV_TireTest()
{
    LOG_INFO("Starting tire test...\r\n");

    LOG_INFO("First motor forward...\r\n");
    set_motors(1, 0, 0, 0);
    HAL_Delay(2000);
    LOG_INFO("First motor backwards...\r\n");
    set_motors(-1, 0, 0, 0);
    HAL_Delay(2000);

    LOG_INFO("Second motor forward...\r\n");
    set_motors(0, 1, 0, 0);
    HAL_Delay(2000);
    LOG_INFO("Second motor backwards...\r\n");
    set_motors(0, -1, 0, 0);
    HAL_Delay(2000);

    LOG_INFO("Third motor forward...\r\n");
    set_motors(0, 0, 1, 0);
    HAL_Delay(2000);
    LOG_INFO("Third motor backwards...\r\n");
    set_motors(0, 0, -1, 0);
    HAL_Delay(2000);

    LOG_INFO("Fourth motor forward...\r\n");
    set_motors(0, 0, 0, 1);
    HAL_Delay(2000);
    LOG_INFO("Fourth motor Backwards...\r\n");
    set_motors(0, 0, 0, -1);
    HAL_Delay(2000);

    LOG_INFO("Finished tire test...\r\n");
}

void set_motors(float m1, float m2, float m3, float m4)
{
    motors[0].speed = m1 * 100.f;
    motors[1].speed = m2 * 100.f;
    motors[2].speed = m3 * 100.f;
    motors[3].speed = m4 * 100.f;
}

void NAV_StopDribbler() { HAL_GPIO_WritePin(DRIBBLER_GPIO_Port, DRIBBLER_Pin, GPIO_PIN_RESET); }

uint8_t NAV_IsPanic() { return robot_cmd.panic; }

/*
   Someone thinks something has gone terribly wrong...
   Disable motors and everyhting going forward
   TODO: Actualy implement the behaviour that triggers when the program is set
   to panic.
 */
void NAV_SetRobotPanic() { robot_cmd.panic = 1; }

/*
  Someone solved the panic
  TODO: Implement reset behaviour. I am leaving this as 1 untill the method
  actualy does something.
*/
void NAV_ClearRobotPanic() { robot_cmd.panic = 1; }

void NAV_RunDribbler() { HAL_GPIO_WritePin(DRIBBLER_GPIO_Port, DRIBBLER_Pin, GPIO_PIN_SET); }

void NAV_TestDribbler()
{
    NAV_RunDribbler();
    HAL_Delay(2000);
    NAV_StopDribbler();
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
