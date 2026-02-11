#ifndef POSFOLLOW_H
#define POSFOLLOW_H

typedef struct {
    float umin;
    float umax;
    float Ts;
    float Ti;
    float Td;
    float K;
} control_params;

/**
 * Returns the error between two angles.
 */
float angle_error(float angle, float desired);

/**
 * Standard error function for comparing the error between two values.
 */
float standard_error(float current, float desired);

/**
 * Initialize the param structs with preset values.
 */
void POS_set_params();

/**
 * Request robot to move to desired position in field coordinates.
 * @param dest_x, desired field x position
 * @param dest_y, desired field y position
 * @param wantw, desired rotation
 */
void POS_go_to_position(float dest_x, float dest_y, float wantw);

void POS_velocity_control(float vel_u, float vel_v, float dest_w);

/**
 * LQR controller: unified position and angle control in world frame.
 * Computes optimal velocity commands considering both position and angle errors.
 * @param dest_x, desired field x position (meters)
 * @param dest_y, desired field y position (meters)
 * @param dest_w, desired rotation (radians)
 */
void POS_go_to_position_lqr(float dest_x, float dest_y, float dest_w);

/**
 * Initializes the POS module.
 */
void POS_Init();

float PID_pi(float current, float desired, float* I_prev, float (*error_func)(float, float), control_params* param);

/**
 * Run one iteration of a P loop
 */
float PID_p(float current, float desired, float (*error_func)(float, float), control_params* param);
#endif /* COM_H */
