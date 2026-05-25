#include "pos_follow.h"
#include "log.h"
#include "data_logging.h"
#include "math.h"
#include "nav.h"
#include "state_estimator.h"

static control_params params_dist;
static control_params params_angle;

static float DELTA_T = 0.001;

static LOG_Module internal_log_mod;

// Constant derived from infinite horizon LQR
// run lqr.py to create new constants based on tuning
const float k11 = 0.009993;
const float k12 = 0.000000;
const float k13 = 0.000000;
const float k14 = 0.142126;
const float k15 = 0.000000;
const float k16 = 0.000000;
const float k21 = 0.000000;
const float k22 = 0.009993;
const float k23 = 0.000000;
const float k24 = 0.000000;
const float k25 = 0.142126;
const float k26 = 0.000000;
const float k31 = 0.000000;
const float k32 = 0.000000;
const float k33 = 0.444880;
const float k34 = 0.000000;
const float k35 = 0.000000;
const float k36 = 1.045146;

static float vel_max_xy = 1.0f;
static float vel_max_w = 4.0f;
// Damping gain for angular velocity (using gyro feedback)
static float Kd_omega = 0.5f;

void set_params()
{
    params_angle.umin = -200.0;
    params_angle.umax = 200.0;
    params_angle.Ts = DELTA_T;
    params_angle.Ti = 100000000;
    params_angle.Td = 0.02;
    params_angle.K = 50.0;

    params_dist.umin = -100.0;
    params_dist.umax = 100.0;
    params_dist.Ts = DELTA_T;
    params_dist.Ti = 0.0015;
    params_dist.Td = 0.1;
    params_dist.K = 500.0f * 1000.50;
}

void POS_Init() {
    set_params();
    LOG_InitModule(&internal_log_mod, "POS", LOG_LEVEL_ERROR, 0);
}

float angle_error(float a, float b)
{
    // TODO make sure returned sign is correct for the desired direction

    float delta = a - b;
    return atan2f(sinf(delta), cosf(delta));
}

float standard_error(float current, float desired) { return desired - current; }

void POS_go_to_position_lqr(float dest_x, float dest_y, float dest_w)
{
    ControlSignal sigx;
    ControlSignal sigy;
    ControlSignal sigw;

    static float i_dist_y = 0;
    static float i_dist_x = 0;
    // Get current state from EKF
    const float cur_x = STATE_get_posx();
    const float cur_y = STATE_get_posy();
    const float cur_w = STATE_get_robot_angle();
    const float cur_vx = STATE_get_vx();
    const float cur_vy = STATE_get_vy();
    const float cur_omega = STATE_get_gyro_z();

    float ex_world = cur_x - dest_x;
    float ey_world = cur_y - dest_y;
    float ew = angle_error(cur_w, dest_w);

    float cos_w = arm_cos_f32(cur_w);
    float sin_w = arm_sin_f32(cur_w);

    float ex = ex_world * cos_w + ey_world * sin_w;
    float ey = -ex_world * sin_w + ey_world * cos_w;

    float evx = cur_vx * cos_w + cur_vy * sin_w;
    float evy = -cur_vx * sin_w + cur_vy * cos_w;
    float eomega = cur_omega;

    float vx_robot = -(
        k11 * ex +
        k12 * ey +
        k13 * ew +
        k14 * evx +
        k15 * evy +
        k16 * eomega
    );

    float vy_robot = -(
        k21 * ex +
        k22 * ey +
        k23 * ew +
        k24 * evx +
        k25 * evy +
        k26 * eomega
    );

    float omega = -(
        k31 * ex +
        k32 * ey +
        k33 * ew +
        k34 * evx +
        k35 * evy +
        k36 * eomega
    );

    // Add gyro-based damping to reduce angular oscillations
    // Essentialy makes it PD loop
    float gyro_z = STATE_get_gyro_z();
    omega = omega - Kd_omega * gyro_z;

    float vel_xy = sqrtf(vx_robot * vx_robot + vx_robot * vx_robot);

    if (vel_xy > vel_max_xy) {
        float scale = vel_max_xy / vel_xy;
        vx_robot *= scale;
        vy_robot *= scale;
    }

    if (omega > vel_max_w) omega = vel_max_w;
    if (omega < -vel_max_w) omega = -vel_max_w;

    const float vel_to_motor_scale = 250.0f;
    float cmd_x = vx_robot * vel_to_motor_scale;
    float cmd_y = vy_robot * vel_to_motor_scale;
    float cmd_w = omega * 50.0f;  // Angular scaling

    float deadzone = 5;
    if (cmd_w < deadzone && cmd_w > -deadzone){
        cmd_w = 0;
    }
    if (cmd_x < deadzone && cmd_x > -deadzone){
        cmd_x = 0;
    }
    if (cmd_y < deadzone && cmd_y > -deadzone){
        cmd_y = 0;
    }

    NAV_steer(cmd_x, cmd_y, cmd_w);

    sigx.u = cmd_x;
    sigx.r = dest_x;
    sigx.e = ex;

    sigy.u = cmd_y;
    sigy.r = dest_y;
    sigy.e = ey;

    sigw.u = cmd_w;
    sigw.r = dest_w;
    sigw.e = ew;
    DATA_log_pos(sigx, sigy, sigw);
}

float PID_p(float current, float desired, float (*error_func)(float, float), control_params* param)
{
    float error = error_func(current, desired);

    float v = param->K * (error);
    float u = 0;

    if (v > param->umax) {
        u = param->umax;
    }

    else if (v < param->umin) {
        u = param->umin;
    }

    else {
        u = v;
    }

    return u;
}

float PID_pi(float current, float desired, float* I_prev, float (*error_func)(float, float), control_params* param)
{
    float error = error_func(current, desired);
    float I = *I_prev + (param->Ts / param->Ti) * error;

    float feed_forward = 0.0;
    float v = param->K * (error + I);
    float u = 0;
    // integrator windup fix
    if (v < param->umin || v > param->umax) {
        I = *I_prev;
    }

    if (v > param->umax) {
        u = param->umax;
    }

    else if (v < param->umin) {
        u = param->umin;
    }

    else {
        u = v;
    }

    *I_prev = I;

    return u;
}
