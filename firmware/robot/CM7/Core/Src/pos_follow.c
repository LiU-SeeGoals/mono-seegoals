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

// LQR gain matrix K (3x3): maps [ex, ey, e_angle] to [vx_world, vy_world, omega]
// Row 0: vx_world = K[0][0]*ex + K[0][1]*ey + K[0][2]*e_angle
// Row 1: vy_world = K[1][0]*ex + K[1][1]*ey + K[1][2]*e_angle
// Row 2: omega    = K[2][0]*ex + K[2][1]*ey + K[2][2]*e_angle
static float K[3][3] = {
    {2.0f, 0.0f, 0.0f},   // vx_world gains
    {0.0f, 2.0f, 0.0f},   // vy_world gains
    {0.0f, 0.0f, 5.0f}    // omega gains
};

// Velocity limits in world frame
static float vel_max_xy = 1.0f;    // m/s max linear velocity
static float vel_max_w = 4.0f;     // rad/s max angular velocity

// Damping gain for angular velocity (using gyro feedback)
static float Kd_omega = 0.5f;

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

    // Control on global frame coordinates
    float distance_control_signal = 200.f;

    // Rotate from world to robot frame (inverse the robot angle)
    float x = distance_control_signal * ((rel_x * arm_cos_f32(-angle)) - (rel_y * arm_sin_f32(-angle)));
    float y = distance_control_signal * ((rel_x * arm_sin_f32(-angle)) + (rel_y * arm_cos_f32(-angle)));

    const float magnitude = sqrtf(x * x + y * y);

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

// LQR controller: unified position and angle control in world frame
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

    // Compute errors in world frame

    float ex = dest_x - cur_x;
    float ey = dest_y - cur_y;
    float ew = angle_error(cur_w, dest_w);  // Wrapped to [-pi, pi]
    float Ix = i_dist_x + (DELTA_T / 5.5) * ex;
    float Iy = i_dist_y + (DELTA_T / 5.5) * ey;

    float vx_world = K[0][0] * (ex + Ix);
    float vy_world = K[1][1] * (ey + Iy);
    float omega  = K[2][2] * ew;

    // Add gyro-based damping to reduce angular oscillations
    // Essentialy makes it PD loop
    float gyro_z = STATE_get_gyro_z();
    omega = omega - Kd_omega * gyro_z;

    float vel_xy = sqrtf(vx_world * vx_world + vy_world * vy_world);

    if (vel_xy > vel_max_xy) {
        float scale = vel_max_xy / vel_xy;
        vx_world *= scale;
        vy_world *= scale;
    }
    else{
        i_dist_y = Iy;
        i_dist_x = Ix;
    }
    if (omega > vel_max_w) omega = vel_max_w;
    if (omega < -vel_max_w) omega = -vel_max_w;

    // Transform world frame velocity to robot frame
    // v_robot = R(-theta) * v_world
    // [vx_robot]   [cos(theta)   sin(theta)] [vx_world]
    // [vy_robot] = [-sin(theta)  cos(theta)] [vy_world]
    float cos_w = arm_cos_f32(cur_w);
    float sin_w = arm_sin_f32(cur_w);
    float vx_robot = vx_world * cos_w + vy_world * sin_w;
    float vy_robot = -vx_world * sin_w + vy_world * cos_w;

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
