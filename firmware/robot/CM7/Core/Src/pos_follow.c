#include "pos_follow.h"
#include "log.h"
#include "math.h"
#include "nav.h"
#include "state_estimator.h"

// Each state has an integration part in the pid controller
float dist_I = 0.01;
float angle_I = 0.01;

control_params params_dist;
control_params params_angle;

static robot_nav_command robot_nav;

const float DELTA_T = 0.001;

static LOG_Module internal_log_mod;

void POS_Init() { LOG_InitModule(&internal_log_mod, "POS", LOG_LEVEL_ERROR, 0); }

float angle_error(float angle, float desired)
{
    // TODO make sure returned sign is correct for the desired direction

    return desired - angle;
}

float standard_error(float current, float desired) { return desired - current; }

void set_params()
{
    params_angle.umin = -100.0;
    params_angle.umax = 100.0;
    params_angle.Ts = DELTA_T;
    params_angle.Ti = 1000000000000;
    params_angle.Td = 0.1;
    params_angle.K = 30 * 1.1;

    params_dist.umin = -100.0;
    params_dist.umax = 100.0;
    params_dist.Ts = DELTA_T;
    params_dist.Ti = 0.0015;
    params_dist.Td = 0.1;
    params_dist.K = 500.0f * 1000.50;
}

void TEST_vy(float ref_angle, float speed)
{
    float control_w = PID_p(STATE_get_robot_angle(), ref_angle, angle_error, &params_angle);
    NAV_steer(0, speed, control_w);
}

void TEST_vx(float ref_angle, float speed)
{
    float control_w = PID_p(STATE_get_robot_angle(), ref_angle, angle_error, &params_angle);
    NAV_steer(speed, 0.0f, control_w);
}

void TEST_angle_control(float ref_angle)
{
    float control_w = PID_p(STATE_get_robot_angle(), ref_angle, angle_error, &params_angle);
    NAV_steer(0, 0, control_w);
}

int log_num = 0;
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

    float cur_x = STATE_get_posx();
    float cur_y = STATE_get_posy();
    float angle = STATE_get_robot_angle();

    float rel_x = dest_x - cur_x;
    float rel_y = dest_y - cur_y;
    float euclidian_distance = sqrt(rel_x * rel_x + rel_y * rel_y);

    // Control on global frame coordinates
    float distance_control_signal = 300.f;
    float control_w = PID_p(STATE_get_robot_angle(), wantw, angle_error, &params_angle);

    // Rotate from world to robot frame (inverse the robot angle)
    float x = distance_control_signal * ((rel_x * arm_cos_f32(-angle)) - (rel_y * arm_sin_f32(-angle)));
    float y = distance_control_signal * ((rel_x * arm_sin_f32(-angle)) + (rel_y * arm_cos_f32(-angle)));

    NAV_steer(x, y, control_w);
}

float PID_p(float current, float desired, float (*error_func)(float, float), control_params* param)
{
    set_params(); // TODO: Do we need to set this every time?

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
    set_params(); // TODO: Do we need to set this every time

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
