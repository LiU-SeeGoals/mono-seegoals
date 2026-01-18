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

void set_params()
{
    params_angle.umin = -200.0;
    params_angle.umax = 200.0;
    params_angle.Ts = DELTA_T;
    params_angle.Ti = 100000000;
    params_angle.Td = 0.02;  // Reduced for better damping without overshoot
    params_angle.K = 50.0;   // Slightly reduced for stability with derivative

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

    float delta = desired - angle;
    return atan2f(sinf(delta), cosf(delta));

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

    ControlSignal sigx;
    ControlSignal sigy;
    ControlSignal sigw;

    const float cur_x = STATE_get_posx();
    const float cur_y = STATE_get_posy();
    const float angle = STATE_get_robot_angle();

    float rel_x = dest_x - cur_x;
    float rel_y = dest_y - cur_y;

    // Compute angle error first for priority-based control
    float ang_err = angle_error(angle, wantw);
    float abs_ang_err = fabsf(ang_err);

    // Angular control with derivative term for better damping
    static float prev_ang_err = 0.0f;
    float ang_err_derivative = (ang_err - prev_ang_err) / DELTA_T;
    prev_ang_err = ang_err;

    // PD control for angle: u = K * (error + Td * derivative)
    float control_w = params_angle.K * (ang_err + params_angle.Td * ang_err_derivative);

    // Clamp angular control signal
    if (control_w > params_angle.umax) control_w = params_angle.umax;
    if (control_w < params_angle.umin) control_w = params_angle.umin;

    // Coordinated control: reduce translation speed when angle error is large
    // This gives priority to angular correction
    // Scale factor: 1.0 when angle error is 0, reduced when error is large
    // Using cos^2 for smooth falloff: full speed at 0 error, ~50% at 45deg, 0 at 90deg
    float angle_priority_scale = 1.0f;
    if (abs_ang_err > 0.1f) {  // ~6 degrees threshold
        // Smooth scaling based on angle error (max error we care about is ~PI/2)
        float normalized_err = abs_ang_err / (PI / 2.0f);
        if (normalized_err > 1.0f) normalized_err = 1.0f;
        angle_priority_scale = 1.0f - 0.7f * normalized_err;  // Scale down to 30% at 90deg error
    }

    // Control on global frame coordinates
    float distance_control_signal = 200.f * angle_priority_scale;

    // Rotate from world to robot frame (inverse the robot angle)
    float x = distance_control_signal * ((rel_x * arm_cos_f32(-angle)) - (rel_y * arm_sin_f32(-angle)));
    float y = distance_control_signal * ((rel_x * arm_sin_f32(-angle)) + (rel_y * arm_cos_f32(-angle)));

    const float magnitude = sqrtf(x * x + y * y);

    // Reduced max velocity to leave headroom for angular control
    const float umax = 250.0f;

    if (magnitude > umax)
    {
        x = x * umax/magnitude;
        y = y * umax/magnitude;
    }

    NAV_steer(x, y, control_w);

    sigx.u = x;
    sigx.r = dest_x;
    sigx.e = rel_x;

    sigy.u = y;
    sigy.r = dest_y;
    sigy.e = rel_y;

    sigw.u = control_w;
    sigw.r = wantw;
    sigw.e = angle_error(angle, wantw);
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
