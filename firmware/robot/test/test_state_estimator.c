/*#include "../CM7/Core/Inc/state_estimator.h"*/
#include "../CM7/Core/Inc/HandmadeMath.h"


typedef struct {

  int is_initiated;
  Vec2 statex;
  Vec2 statey;
  Vec2 statew;

  Mat2 Px;
  Mat2 Py;
  Mat2 Pw;

} robot_state; 

robot_state robot;

void init(){

  robot.statex.X = 2;
  robot.statex.Y = 1;

  robot.statey.X = 2;
  robot.statey.Y = 1;

  robot.statew.X = 2;
  robot.statew.Y = 1;
  /*robot.statey = statey;*/
  /*robot.statew = statew;*/

  robot.Px.Elements[0][0] = 1;
  robot.Px.Elements[0][1] = 0;
  robot.Px.Elements[1][0] = 0;
  robot.Px.Elements[1][1] = 1;

  robot.Py.Elements[0][0] = 1;
  robot.Py.Elements[1][0] = 0;
  robot.Py.Elements[0][1] = 0;
  robot.Py.Elements[1][1] = 1;

  robot.Pw.Elements[0][0] = 1;
  robot.Pw.Elements[1][0] = 0;
  robot.Pw.Elements[0][1] = 0;
  robot.Pw.Elements[1][1] = 1;

  robot.is_initiated = -1;
}


void measurement_update_vec2_1d(Mat2* P, float R, Vec2 H, float mejure, Vec2* x)
{
  /*Vec2 y = SubV2(mejurement, MulV2(H, *x));*/
  float y = mejure - DotV2(H,*x);

  /*Vec2 Sxy = Add(MulV2(H, MulM2V2(*P, H)), R);*/

  float S = P->Columns[0].X + R;
  Vec2 K = MulV2F(P->Columns[0], 1/S);
  Mat2 KH = {K.X, 0, 
             K.Y, 0};

  KH = TransposeM2(KH);

  *x = AddV2(*x, MulV2F(K, y));
  *P = SubM2(*P, MulM2(KH, *P));
}

void time_update_vec2(Mat2 F,  Mat2 Q, Mat2* P, Vec2* x, Mat2 B, Vec2 u)
{
  *x = AddV2(MulM2V2(F, *x), MulM2V2(B, u));
  *P = AddM2(MulM2(F, MulM2(*P, TransposeM2(F))), Q);
}


void cv_update_vec2(Mat2* P, Vec2* x, Mat2 B, Vec2 u) {
  float T = 1;
  Mat2 F = {1, T,
            0, 1};
  F = TransposeM2(F);

  Mat2 G = {T*T/2, 0,
            0,     T};

  G = TransposeM2(G);

  Mat2 q = {1, 0,
            0, 1};
  q = TransposeM2(q);

  Mat2 Q = MulM2(G, MulM2(q ,TransposeM2(G)));
  time_update_vec2(F, Q, P, x, B, u);
}

void camera_meas(float posx, float posy, float angle){
    float R = 1;
    Vec2 H = {1, 0};
    measurement_update_vec2_1d(&robot.Px, R, H, posx, &robot.statex);
    measurement_update_vec2_1d(&robot.Py, R, H, posy, &robot.statey);
    measurement_update_vec2_1d(&robot.Pw, R, H, angle, &robot.statew);

    /*robot.statex.X = posx;*/
    /*robot.statey.X = posy;*/
    /*robot.statew.X = angle;*/
}

float get_robot_angle() {

  if (robot.statew.X < 0){
    robot.statew.X += 2 * PI;
  }
  if (robot.statew.X > 2 * PI){
    robot.statew.X -= 2 * PI;
  }
  return robot.statew.X;
}

void initialize_kalman(float x, float y, float w){
  robot.statex.X = x;
  robot.statey.X = y;
  robot.statew.X = w;

  robot.statex.Y = 0;
  robot.statey.Y = 0;
  robot.statew.Y = 0;

  robot.is_initiated = 1;
}

/*
  @Return -1 if not inited 1 
*/
int kalman_is_initiated(){
  return robot.is_initiated;
}

float get_angle_vel(){
  return robot.statew.Y;
}

float get_posx(){
  return robot.statex.X;
}

float get_posy(){
  return robot.statey.X;
}

float get_vx(){
  return robot.statex.Y;
}

float get_vy(){
  return robot.statey.Y;
}

void printm2(Mat2 a){
  printf("----------------\n");
  printf("%f %f\n", a.Columns[0].X, a.Columns[1].X);
  printf("%f %f\n", a.Columns[0].Y, a.Columns[1].Y);
  printf("----------------\n");

}

void printm3(Mat3 a){
  printf("----------------\n");
  printf("%f %f %f\n", a.Columns[0].X, a.Columns[1].X, a.Columns[2].X);
  printf("%f %f %f\n", a.Columns[0].Y, a.Columns[1].Y, a.Columns[2].Y);
  printf("%f %f %f\n", a.Columns[0].Z, a.Columns[1].Z, a.Columns[2].Z);
  printf("----------------\n");

}

void printv2(Vec2* a){
    printf("x %f y %f\n" , a->X, a->Y);
}

void test_motion_model(){

    init();
    printf("========== testing motion model ============\n");
    float T = 1;

    Mat2 Bnone = {0,0,0,0};
    Mat2 B = {T*T/2, 0,
                T,   0};
    B = TransposeM2(B);

    Vec2 accmejure = {1, 0};

    printf("cv update no acceleration speed %f\n", robot.statex.Y);
    cv_update_vec2(&robot.Px, &robot.statex, Bnone, accmejure);
    printv2(&robot.statex);
    printf("cv update with acceleration 1\n");
    cv_update_vec2(&robot.Px, &robot.statex, B, accmejure);
    printv2(&robot.statex);
    float posmejure = 8;
    printf("covariance before \n");
    printm2(robot.Px);
    float R = 1;
    Vec2 H = {1, 0};
    printf("covariance many updates \n");
    measurement_update_vec2_1d(&robot.Px, R, H, posmejure, &robot.statex);
    cv_update_vec2(&robot.Px, &robot.statex, B, accmejure);
    cv_update_vec2(&robot.Px, &robot.statex, B, accmejure);
    cv_update_vec2(&robot.Px, &robot.statex, B, accmejure);

    printf("cov\n");
    printm2(robot.Px);
    printv2(&robot.statex);
    printm2(robot.Px);
}

void test_camera_meas(){

    printf("========== camera measurement test ==========\n");
    init();
    float x = 1;
    float y = 1;
    float w = 1;
    printf("before camera measurement at pos x:%f y:%f z:%f\n", x, y, w);
    printv2(&robot.statex);
    printv2(&robot.statey);
    printv2(&robot.statew);

    camera_meas(1, 1, 1);

    printf("after camera measurement\n");
    printv2(&robot.statex);
    printv2(&robot.statey);
    printv2(&robot.statew);

}

void test_measurement(){
    printf("========== testing measurements ============\n");
    float T = 1;

    init();

    float posmejure = 8;
    float R = 0.01;

    Vec2 H = {1, 0};
    printf("measurement at %f\n", posmejure);
    measurement_update_vec2_1d(&robot.Px, R, H, posmejure, &robot.statex);
    printv2(&robot.statex);

    printf("measurement at %f\n", posmejure);
    measurement_update_vec2_1d(&robot.Px, R, H, posmejure, &robot.statex);
    printv2(&robot.statex);

    posmejure = 2;
    measurement_update_vec2_1d(&robot.Px, R, H, posmejure, &robot.statex);
    printf("measurement at %f\n", posmejure);
    printv2(&robot.statex);
    Mat2 Bnone = {0,0,0,0};
    Vec2 non = {0,0};
    cv_update_vec2(&robot.Px, &robot.statex, Bnone, non);
    printf("cv2 with speed %f\n", robot.statex.Y);
    printv2(&robot.statex);

    posmejure = 2;
    measurement_update_vec2_1d(&robot.Px, R, H, posmejure, &robot.statex);
    printf("measurement at %f\n", posmejure);
    printv2(&robot.statex);

    printf("cov\n");
    printm2(robot.Px);
    printv2(&robot.statex);
    printm2(robot.Px);

}


int main() {
  test_measurement();
  test_motion_model();
  test_camera_meas();

}
