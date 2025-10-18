#include <math.h>
#include <stdio.h>

#include "../CM7/Core/Inc/HandmadeMath.h"
#include "../CM7/Core/Inc/pos_follow.h"
/*#include "pos_follow.h"*/

// === "mocked" functions ====
Vec2 current;
float curangle;

float get_posx() {
  return current.X;
}

float get_posy() {
  return current.Y;
}

float get_robot_angle() {
  return curangle;
};
float DELTA_T = 1;
void steer(float vx, float vy, float vw) {
  current.X += vx * DELTA_T;
  current.Y += vy * DELTA_T;
  curangle += vw * DELTA_T;
};

// === start real code ===
// Each state has an integration part in the pid controller
float dist_I = 0;
float angle_I = 0;

control_params params_dist;
control_params params_angle;

const float CLOCK_FREQ = 400000000;

float angle_error(float angle, float desired){

  // TODO make sure returned sign is correct for the desired direction
  float left_error = desired - angle;
  float right_error = angle - desired;
  if (right_error < 0){
    right_error += 2 * PI;
  }
  if (left_error < 0){
    left_error += 2 * PI;
  }
  if (right_error < left_error){
    return right_error;
  }
  else {
    return -left_error;
  }
}

float standard_error(float current, float desired) {
  return desired - current;
}

void POS_set_params() {

  params_angle.umin = -10.0;
  params_angle.umax = 10.0;
  params_angle.Ts = DELTA_T;
  params_angle.Ti = 0.02;
  params_angle.K = 0.0015;

  params_dist.umin = -10.0;
  params_dist.umax = 10.0;
  params_dist.Ts = DELTA_T;
  params_dist.Ti = 0.02;
  params_dist.K = 0.0015;
}

void go_to_position(Vec2 desired_pos, float wantw) {

  Vec2 current_pos = {get_posx(), get_posy()};

  float angle = get_robot_angle();
  Vec2 relative_pos = SubV2(current_pos, desired_pos); 
  float euclidian_distance = sqrt(relative_pos.X * relative_pos.X + relative_pos.Y * relative_pos.Y);
  // We want the distance to be zero.
  float distance_control_signal = PID_it(euclidian_distance, 0.0, &dist_I, standard_error, params_dist);
  // We want the angle to be the desired
  float control_w = PID_it(get_robot_angle(), wantw, &angle_I, angle_error, params_angle);

  // The steering signal is a velocity, so calculate how much of each component we need
  /*printf("== control signals ===\n");*/
  /*printf("%f\n", distance_control_signal * cos(angle));*/
  /*printf("%f\n", distance_control_signal * sin(angle));*/
  /*printf("%f\n", control_w);*/

  Vec2 r = {cos(angle), sin(angle)};
  // Project the local coordinate vector unto the relative vector to get the desiered scaled contribute of each x and y axis
  Vec2 projected =  MulV2F(relative_pos, distance_control_signal * DotV2(r, relative_pos) / euclidian_distance);

  // Dont know why minus lol
  steer(projected.X, projected.Y, -control_w);
}


float PID_it(float current, float desired, float* I_prev, float (*error_func)(float,float), control_params param){

  float error = error_func(current, desired);
  float I = *I_prev + param.Ts / param.Ti * error;
  float v = param.K * (error + I);
  float u = 0;
  // integrator windup fix
  if (v < param.umin || v > param.umin){
    I = *I_prev;
  }

  if (v > param.umax) {
    u = param.umax;
  }

  else if (v < param.umin) {
    u = param.umin;
  }

  else {
    u = v;
  }

  /*LOG_INFO("DATAu:%f;\r\n", u);*/
  // HAL_Delay(1);
  *I_prev = I;
  return u;
  // for some time
}

// === end real code ===

int main(){
  Vec2 want = {0,0};
  current.X = 10;
  current.Y = 10;
  curangle = PI/2;

  set_params();

  for (int i = 0; i < 100000; i++){
    go_to_position(want, 0);
    printf("== State ==\n");
    printf("%f\n", current.X);
    printf("%f\n", current.Y);
    printf("%f\n", curangle);
  }

  return 0;
}
