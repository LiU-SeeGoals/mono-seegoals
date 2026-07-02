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

// Weight up angle alot, it is important to have the correct angle
static float K[3][3] = {
    {4.0f * 150.f, 0.0f, 0.0f},
    {0.0f, 4.0f * 150.f, 0.0f},
    {0.0f, 0.0f, 3.0f * 50.f}
};

static float vel_max_xy = 50.0f;
static float vel_max_w = 400.0f;
// Damping gain for angular velocity (using gyro feedback)
static float Kd_omega = 2.0f;

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

void POS_Init()
{
    set_params();
    LOG_InitModule(&internal_log_mod, "POS", LOG_LEVEL_ERROR, 0);
}

float angle_error(float angle, float desired)
{
    // TODO make sure returned sign is correct for the desired direction

    float delta = desired - angle;
    return atan2f(sinf(delta), cosf(delta));

}

float standard_error(float current, float desired) { return desired - current; }

void POS_go_to_position_lqr(float dest_x, float dest_y, float dest_w)
{
    ControlSignal sigx;
    ControlSignal sigy;
    ControlSignal sigw;

    static float i_prev_y = 0;
    static float i_prev_x = 0;
    // Get current state from EKF
    const float cur_x = STATE_get_posx();
    const float cur_y = STATE_get_posy();
    const float cur_w = STATE_get_robot_angle();

    // Compute errors in world frame

    float ex = dest_x - cur_x;
    float ey = dest_y - cur_y;
    float ew = angle_error(cur_w, dest_w);  // Wrapped to [-pi, pi]
    float Ix = i_prev_x + (DELTA_T / 0.0004) * ex;
    float Iy = i_prev_y + (DELTA_T / 0.0004) * ey;
    Ix = 0;
    Iy = 0;

    float vx_world = K[0][0] * (ex);
    float vy_world = K[1][1] * (ey);

    // Transform world frame velocity to robot frame
    float cos_w = arm_cos_f32(cur_w);
    float sin_w = arm_sin_f32(cur_w);

    float vel_xy = sqrtf(vx_world * vx_world + vy_world * vy_world);

    if (vel_xy < vel_max_xy) {
        i_prev_x = Ix;
        i_prev_y = Iy;
    }
    else {
        // Anti integrator windup, do nothing
    }

    float vx_robot = vx_world * cos_w + vy_world * sin_w;
    float vy_robot = -vx_world * sin_w + vy_world * cos_w;

    float cmd_x = vx_robot;
    float cmd_y = vy_robot;

    POS_velocity_control(vx_robot, vy_world, dest_w);
}

void POS_velocity_control(float vel_x, float vel_y, float dest_w)
{
    const float cur_w = STATE_get_robot_angle();

    float ew = angle_error(cur_w, dest_w);

    float omega  = K[2][2] * ew;

    // Add gyro-based damping to reduce angular oscillations
    float gyro_z = STATE_get_gyro_z();
    omega = omega - Kd_omega * gyro_z;

    float cmd_w = omega;

    NAV_steer(vel_x, vel_y, cmd_w);
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
