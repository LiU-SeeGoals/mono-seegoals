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

/**
 * Initializes the POS module.
 */
void POS_Init();

/**
 * Request robot to move sideways at `speed` facing `ref_angle`
 * @param ref_angle, angle that the robot should face
 * @param speed, the speed to move in the y direction
 */
void TEST_vy(float ref_angle, float speed);

/**
 * Request robot to move forward at `speed` facing `ref_angle`
 * @param ref_angle, angle that the robot should face
 * @param speed, the speed to move in the x direction
 */
void TEST_vx(float ref_angle, float speed);

/**
 * Request robot to hold desired angle.
 * @param ref_angle, desired angle
 */
void TEST_angle_control(float ref_angle);

/**
 * Run one iteration of a PI loop
 */
float PID_pi(float current, float desired, float* I_prev, float (*error_func)(float, float), control_params* param);

/**
 * Run one iteration of a P loop
 */
float PID_p(float current, float desired, float (*error_func)(float, float), control_params* param);
#endif /* COM_H */
