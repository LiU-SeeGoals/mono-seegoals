#include "pos_follow.h"
#include "log.h"
#include "math.h"
#include "nav.h"
#include "state_estimator.h"

static control_params params_dist;
static control_params params_angle;

static float DELTA_T = 0.001;

static LOG_Module internal_log_mod;

void set_params()
{
    params_angle.umin = -100.0;
    params_angle.umax = 100.0;
    params_angle.Ts = DELTA_T;
    params_angle.Ti = 100000000;
    params_angle.Td = 0.1;
    params_angle.K = 40 * 1.1;

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

float angle_error(float angle, float desired)
{
    // TODO make sure returned sign is correct for the desired direction

    return desired - angle;
}

float standard_error(float current, float desired) { return desired - current; }


void POS_go_to_position(float dest_x, float dest_y, float wantw)
{
    // Robot to world transformation given by
    // [cos(-a) -sin(-a)]
    // [sin(-a) cos(-a)]

    // world to robot transformation given by
    // [cos(a) -sin(a)]
    // [sin(a) cos(a)]

    // Robot to wheel transformation given by
    // [0 1]
    // [1 0]

    const float cur_x = STATE_get_posx();
    const float cur_y = STATE_get_posy();
    const float angle = STATE_get_robot_angle();

    float rel_x = dest_x - cur_x;
    float rel_y = dest_y - cur_y;


    // Control on global frame coordinates
    float distance_control_signal = 200.f;
    float control_w = PID_p(STATE_get_robot_angle(), wantw, angle_error, &params_angle);

    // Rotate from world to robot frame (inverse the robot angle)
    float x = distance_control_signal * ((rel_x * arm_cos_f32(-angle)) - (rel_y * arm_sin_f32(-angle)));
    float y = distance_control_signal * ((rel_x * arm_sin_f32(-angle)) + (rel_y * arm_cos_f32(-angle)));

    const float magnitude = sqrtf(x * x + y * y);

    const float umax = 100;

    if (magnitude > umax)
    {
        x = x * umax/magnitude;
        y = y * umax/magnitude;
    }

    NAV_steer(x, y, control_w);
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
