#include "state_estimator.h"
#include "arm_mat_util_f32.h"
#include "arm_math.h"
#include "imu.h"
#include "nav.h"
#include "lag_element.h"
#include "data_logging.h"

#include "log.h"
#include <string.h>

static const float STATE_DELTA_T  = 0.001;

static LOG_Module internal_log_mod;

// NOTE: Assume 1000hz loop

FusionEKF fusionEKF;

void release_ekf_lock() { fusionEKF.ekf_lock = FREE; }

void set_ekf_lock() { fusionEKF.ekf_lock = LOCKED; }

uint8_t get_ekf_lock() { return fusionEKF.ekf_lock; }

// clang-format off
const float32_t A_f32[16] =
{
  /* Const,   numTaps,   blockSize,   numTaps*blockSize */
  1.0,     0.0,     0.0,     1.0,
  0.0,     1.0,     0.0,     0.0,
  0.0,     0.0,     1.0,     0.0,
  0.0,     0.0,     0.0,     1.0,
};

const float32_t B_f32[16] =
{
  1.0,     0.0,     0.0,     1.0,
  0.0,     1.0,     0.0,     1.0,
  0.0,     0.0,     1.0,     1.0,
  0.0,     0.0,     0.0,     1.0,
};
// clang-format on

// Improvements: Detect when vision loses us and rely on encoders

/*
 *  Init logging for state
 */

void EKFPredict(EKF* pKF)
{
    // >>> State prediction
    // A = jacobian(x+1,x)
    (*pKF->pStateJacobian)(&pKF->x, &pKF->u, &pKF->A);
    (*pKF->pState)(&pKF->x, &pKF->u);

    // >>> Sigma = A*Sigma*A^T + Ex
    // tmp1 = A*Sigma (f x f)
    pKF->tmp1.numRows = pKF->f;
    pKF->tmp1.numCols = pKF->f;
    arm_mat_mult_f32(&pKF->A, &pKF->Sigma, &pKF->tmp1);
    // tmp2 = A^T (f x f)
    pKF->tmp2.numRows = pKF->f;
    pKF->tmp2.numCols = pKF->f;
    arm_mat_trans_f32(&pKF->A, &pKF->tmp2);
    // Sigma = tmp1*tmp2 (f x f)
    arm_mat_mult_f32(&pKF->tmp1, &pKF->tmp2, &pKF->Sigma);
    // Sigma = Sigma + Ex (f x f)
    arm_mat_add_f32(&pKF->Sigma, &pKF->Ex, &pKF->Sigma);
}

void EKFUpdate(EKF* pKF)
{
    // >>> H = jacobian(z,x)
    (*pKF->pMeasJacobian)(&pKF->x, &pKF->C);

    // >>> K = Sigma*C^T*(C*Sigma*C^T+Ez)^-1
    // tmp1 = C^T (f x h)
    pKF->tmp1.numRows = pKF->f;
    pKF->tmp1.numCols = pKF->h;
    arm_mat_trans_f32(&pKF->C, &pKF->tmp1);
    // tmp2 = Sigma*tmp1 (f x h)
    pKF->tmp2.numRows = pKF->f;
    pKF->tmp2.numCols = pKF->h;
    arm_mat_mult_f32(&pKF->Sigma, &pKF->tmp1, &pKF->tmp2);
    // tmp3 = C*tmp2 (h x h)
    pKF->tmp3.numRows = pKF->h;
    pKF->tmp3.numCols = pKF->h;
    arm_mat_mult_f32(&pKF->C, &pKF->tmp2, &pKF->tmp3);
    // tmp3 = tmp3+Ez (h x h)
    arm_mat_add_f32(&pKF->tmp3, &pKF->Ez, &pKF->tmp3);
    // tmp1 = tmp3^-1 (h x h)
    pKF->tmp1.numRows = pKF->h;
    pKF->tmp1.numCols = pKF->h;
    arm_status invStat;
    switch (pKF->h) {
    case 1:
        if (pKF->tmp3.pData[0] == 0.0f) {
            invStat = ARM_MATH_SINGULAR;
        } else {
            pKF->tmp1.pData[0] = 1.0f / pKF->tmp3.pData[0];
            invStat = ARM_MATH_SUCCESS;
        }
        break;
    case 2:
        invStat = arm_mat_inv_2x2_f32(&pKF->tmp3, &pKF->tmp1);
        break;
    case 3:
        invStat = arm_mat_inv_3x3_f32(&pKF->tmp3, &pKF->tmp1);
        break;
    default:
        invStat = arm_mat_inverse_f32(&pKF->tmp3, &pKF->tmp1);
        break;
    }
    /*if(invStat != ARM_MATH_SUCCESS)*/
    /*	LogErrorC("Matrix inverse error", invStat);*/
    // K = tmp2*tmp1 (f x h)
    arm_mat_mult_f32(&pKF->tmp2, &pKF->tmp1, &pKF->K);

    // >>> x = x + K*(z - pMeas(x))
    // tmp1 = pMeas(x) (h x 1)
    pKF->tmp1.numRows = pKF->h;
    pKF->tmp1.numCols = 1;
    (*pKF->pMeas)(&pKF->x, &pKF->tmp1);
    // tmp2 = z - tmp1 (h x 1)
    pKF->tmp2.numRows = pKF->h;
    pKF->tmp2.numCols = 1;
    arm_mat_sub_f32(&pKF->z, &pKF->tmp1, &pKF->tmp2);
    // Wrap angle innovation to [-pi, pi] to avoid 2pi discontinuity issues
    if (pKF->h >= 3) {
        float angle_diff = pKF->tmp2.pData[2];
        pKF->tmp2.pData[2] = atan2f(sinf(angle_diff), cosf(angle_diff));
    }
    // tmp3 = K*tmp2 (f x 1)
    pKF->tmp3.numRows = pKF->f;
    pKF->tmp3.numCols = 1;
    arm_mat_mult_f32(&pKF->K, &pKF->tmp2, &pKF->tmp3);
    // x = x + tmp3 (f x 1)
    arm_mat_add_f32(&pKF->x, &pKF->tmp3, &pKF->x);
    // Wrap state angle to [-pi, pi] after update
    if (pKF->f >= 3) {
        float angle = pKF->x.pData[2];
        pKF->x.pData[2] = atan2f(sinf(angle), cosf(angle));
    }

    // >>> Sigma = (I - K*C)*Sigma
    // tmp1 = K*C (f x f)
    pKF->tmp1.numRows = pKF->f;
    pKF->tmp1.numCols = pKF->f;
    arm_mat_mult_f32(&pKF->K, &pKF->C, &pKF->tmp1);
    // tmp2 = I (f x f)
    pKF->tmp2.numRows = pKF->f;
    pKF->tmp2.numCols = pKF->f;
    arm_mat_identity_f32(&pKF->tmp2);
    // tmp2 = tmp2 - tmp1 (f x f)
    arm_mat_sub_f32(&pKF->tmp2, &pKF->tmp1, &pKF->tmp2);
    // tmp3 = Sigma (f x f)
    arm_mat_copy_f32(&pKF->Sigma, &pKF->tmp3);
    // Sigma = tmp2*tmp3 (f x f)
    arm_mat_mult_f32(&pKF->tmp2, &pKF->tmp3, &pKF->Sigma);
}

void EKFInit(EKF* pKF, uint16_t numStates, uint16_t numCtrl, uint16_t numSensors, float* pData)
{
    uint16_t f = numStates;
    uint16_t g = numCtrl;
    uint16_t h = numSensors;

    uint16_t max = f;
    if (g > max)
        max = g;
    if (h > max)
        max = h;

    memset(pData, 0, EKF_DATA_SIZE(f, g, h, max) * sizeof(float));

    pKF->f = f;
    pKF->g = g;
    pKF->h = h;

    pKF->A.numRows = f;
    pKF->A.numCols = f;
    pKF->A.pData = pData;
    pData += EKF_SIZE_A(f);

    pKF->C.numRows = h;
    pKF->C.numCols = f;
    pKF->C.pData = pData;
    pData += EKF_SIZE_C(h, f);

    pKF->Ex.numRows = f;
    pKF->Ex.numCols = f;
    pKF->Ex.pData = pData;
    pData += EKF_SIZE_EX(f);

    pKF->Ez.numRows = h;
    pKF->Ez.numCols = h;
    pKF->Ez.pData = pData;
    pData += EKF_SIZE_EZ(h);

    pKF->x.numRows = f;
    pKF->x.numCols = 1;
    pKF->x.pData = pData;
    pData += EKF_SIZE_X(f);

    pKF->Sigma.numRows = f;
    pKF->Sigma.numCols = f;
    pKF->Sigma.pData = pData;
    pData += EKF_SIZE_SIGMA(f);

    pKF->K.numRows = f;
    pKF->K.numCols = h;
    pKF->K.pData = pData;
    pData += EKF_SIZE_K(f, h);

    pKF->u.numRows = g;
    pKF->u.numCols = 1;
    pKF->u.pData = pData;
    pData += EKF_SIZE_U(g);

    pKF->z.numRows = h;
    pKF->z.numCols = 1;
    pKF->z.pData = pData;
    pData += EKF_SIZE_Z(h);

    pKF->tmp1.numRows = max;
    pKF->tmp1.numCols = max;
    pKF->tmp1.pData = pData;
    pData += EKF_SIZE_MAX(max);

    pKF->tmp2.numRows = max;
    pKF->tmp2.numCols = max;
    pKF->tmp2.pData = pData;
    pData += EKF_SIZE_MAX(max);

    pKF->tmp3.numRows = max;
    pKF->tmp3.numCols = max;
    pKF->tmp3.pData = pData;
    pData += EKF_SIZE_MAX(max);

    pData += EKF_SIZE_X(f);

    arm_mat_identity_f32(&pKF->Sigma);
}

void STATE_Init()
{
    LOG_InitModule(&internal_log_mod, "STATE", LOG_LEVEL_TRACE, 0);
    initEKF();

    /*LagElementPT1Init(&fusionEKF.lagAccel[0], 1.0f, fusionEKF.pConfig->emaAccelT, STATE_DELTA_T);*/
    /*LagElementPT1Init(&fusionEKF.lagAccel[1], 1.0f, fusionEKF.pConfig->emaAccelT, STATE_DELTA_T);*/

    LagElementPT1Init(&fusionEKF.lagAccel[0], 1.0f, 0.01, STATE_DELTA_T);
    LagElementPT1Init(&fusionEKF.lagAccel[1], 1.0f, 0.01, STATE_DELTA_T);

    fusionEKF.ekf_lock = 0;
    fusionEKF.bias.is_calibrated = -1;
    /*LagElementPT1Init(&fusionEKF.dribbler.lagCurrent, 1.0f, 0.005f, STATE_DELTA_T);*/
}

/*
 *  Print a 4x4 matrix
 */
void STATE_logm44(const float32_t* m44)
{
    // clang-format off
  LOG_DEBUG("Matrix 4x4 \r\n%0.2f %0.2f %0.2f %0.2f\r\n%0.2f %0.2f %0.2f %0.2f\r\n%0.2f %0.2f %0.2f %0.2f\r\n%0.2f %0.2f %0.2f %0.2f end\r\n",
      m44[0*4], m44[0*4 + 1], m44[0*4 + 2], m44[0*4 + 3],
      m44[1*4], m44[1*4 + 1], m44[1*4 + 2], m44[1*4 + 3],
      m44[2*4], m44[2*4 + 1], m44[2*4 + 2], m44[2*4 + 3],
      m44[3*4], m44[3*4 + 1], m44[3*4 + 2], m44[3*4 + 3]
      );
    // clang-format on
}

/*
   Calculate the jacobian, mostly used to update covariance.
*/

static void ekfStateJacobianFunc(const arm_matrix_instance_f32* pX, const arm_matrix_instance_f32* pU, arm_matrix_instance_f32* pF)
{
    // Motion model
    // xk+1 = xk + (vx*cos(a) - vy*sin(a)) * dt
    // yk+1 = yk + (vx*sin(a) + vy*cos(a)) * dt
    // angle = angle + omega * dt

    // Gives the jacobian

    // [dx/dx     dx/dy     dx/domenga]
    // [dy/dx     dy/dy     dy/domenga]
    // [domega/dx domega/dy domega/domenga]
    const float dt = STATE_DELTA_T;

    float gyr_w = MAT_ELEMENT(*pU, 0, 0);
    float vx = MAT_ELEMENT(*pU, 1, 0);
    float vy = MAT_ELEMENT(*pU, 2, 0);
    // vx = 0;
    // vy = 0;

    float px = MAT_ELEMENT(*pX, 0, 0);
    float py = MAT_ELEMENT(*pX, 1, 0);
    float pw = MAT_ELEMENT(*pX, 2, 0);

    arm_mat_identity_f32(pF);

    MAT_ELEMENT(*pF, 0, 2) = (-vx*sin(pw) - vy*cos(pw)) * dt;
    MAT_ELEMENT(*pF, 1, 2) = (vx*cos(pw)-vy*sin(pw))*dt;
}

static void ekfMeasFunc(const arm_matrix_instance_f32* pX, arm_matrix_instance_f32* pY)
{
    memcpy(pY->pData, pX->pData, sizeof(float) * 3);
}

void ekfMeasJacobianFunc(const arm_matrix_instance_f32* pX, arm_matrix_instance_f32* pH)
{
    (void)pX;

    arm_mat_zero_f32(pH);
    MAT_ELEMENT(*pH, 0, 0) = 1.0f;
    MAT_ELEMENT(*pH, 1, 1) = 1.0f;
    MAT_ELEMENT(*pH, 2, 2) = 1.0f;
}

// static inline float wrapToPi(float angle)
// {
//     while (angle > M_PI)
//         angle -= 2.0f * M_PI;
//     while (angle < -M_PI)
//         angle += 2.0f * M_PI;
//     return angle;
// }

static void ekfStateFunc(arm_matrix_instance_f32* pX, const arm_matrix_instance_f32* pU)
{
    /*const float dt = 0.001f;*/
    const float dt = STATE_DELTA_T;

    float gyr_w = MAT_ELEMENT(*pU, 0, 0);
    float vx = MAT_ELEMENT(*pU, 1, 0);
    float vy = MAT_ELEMENT(*pU, 2, 0);
    // vx = 0;
    // vy = 0;

    float p_x = MAT_ELEMENT(*pX, 0, 0);
    float p_y = MAT_ELEMENT(*pX, 1, 0);
    float p_w = MAT_ELEMENT(*pX, 2, 0);

    // Robot to global from robot frame
    float px1 = p_x + (vx * cosf(p_w) - vy * sinf(p_w)) * dt;
    float py1 = p_y + (vx * sinf(p_w) + vy * cosf(p_w)) * dt;
    // Wrap angle to [-pi, pi] to prevent unbounded growth
    float pw1 = atan2f(sinf(p_w + gyr_w * dt), cosf(p_w + gyr_w * dt));

    MAT_ELEMENT(*pX, 0, 0) = px1;
    MAT_ELEMENT(*pX, 1, 0) = py1;
    MAT_ELEMENT(*pX, 2, 0) = pw1;
    // MAT_ELEMENT(*pX, 3, 0) = vx1;
    // MAT_ELEMENT(*pX, 4, 0) = vy1;
}

static FusionEKFConfig configFusionEKF = {
    .posNoiseXY = 1.f,
    .posNoiseW = 0.001f,
    .velNoiseXY = 0.005f,
    .visNoiseXY = 0.002f,
    .visNoiseW = 0.30f,
    .outlierMaxVelXY = 3.0f,
    .outlierMaxVelW = 3.0f,
    .trackingCoeff = 1.0f,
    .visCaptureDelay = 20,
    .fusionHorizon = 35,
    .visionTimeoutMs = 1000,
    .emaAccelT = 0.005f,
    .ballCenteringFactor = 0.1f,
    .ballCenteringVelLimit = 0.02f,
    .dribblerStrongOn = 70,
    .dribblerStrongOff = 40,
    .ballTimeoutMs = 2000,
    .activeDribblingForce_mN = 500,
};

int STATE_vision_initialized() { return fusionEKF.vision.online; }

static void loadNoiseCovariancesFromConfig()
{
}

static void initEKF()
{
    EKFInit(&fusionEKF.ekf, 3, 3, 3, fusionEKF.ekfData);
    arm_mat_identity_f32(&fusionEKF.ekf.Sigma);
    arm_mat_scale_f32(&fusionEKF.ekf.Sigma, 0.1f, &fusionEKF.ekf.Sigma);

    fusionEKF.ekf.pState = &ekfStateFunc;
    fusionEKF.ekf.pStateJacobian = &ekfStateJacobianFunc;
    fusionEKF.ekf.pMeas = &ekfMeasFunc;
    fusionEKF.ekf.pMeasJacobian = &ekfMeasJacobianFunc;

    arm_mat_identity_f32(&fusionEKF.ekf.Ex);
    arm_mat_identity_f32(&fusionEKF.ekf.Ez);

    MAT_ELEMENT(fusionEKF.ekf.Ex, 0, 0) = configFusionEKF.posNoiseXY * configFusionEKF.posNoiseXY;
    MAT_ELEMENT(fusionEKF.ekf.Ex, 1, 1) = configFusionEKF.posNoiseXY * configFusionEKF.posNoiseXY;
    MAT_ELEMENT(fusionEKF.ekf.Ex, 2, 2) = configFusionEKF.posNoiseW * configFusionEKF.posNoiseW;
    // MAT_ELEMENT(fusionEKF.ekf.Ex, 3, 3) = fusionEKF.pConfig->velNoiseXY * fusionEKF.pConfig->velNoiseXY;
    // MAT_ELEMENT(fusionEKF.ekf.Ex, 4, 4) = fusionEKF.pConfig->velNoiseXY * fusionEKF.pConfig->velNoiseXY;

    MAT_ELEMENT(fusionEKF.ekf.Ez, 0, 0) = configFusionEKF.visNoiseXY * configFusionEKF.visNoiseXY;
    MAT_ELEMENT(fusionEKF.ekf.Ez, 1, 1) = configFusionEKF.visNoiseXY * configFusionEKF.visNoiseXY;
    MAT_ELEMENT(fusionEKF.ekf.Ez, 2, 2) = configFusionEKF.visNoiseW * configFusionEKF.visNoiseW;

    // TODO: Load some covariance values for process and measurement noise
    // loadNoiseCovariancesFromConfig();
}

void STATE_FusionEKFVisionUpdate(float posx, float posy, float posw)
{
    // VISION
    DATA_log_vision(posx, posy, posw);

    float pos[3] = {posx, posy, posw};

    if (!fusionEKF.vision.online) {
        fusionEKF.vision.online = 1;

        // Make sure EKF jumps immediately to new position in first measurement.

        // TODO: We can be smart here and track when vision loses
        // the robot and rely on encoders instead (as tigers do hehe).
        memcpy(fusionEKF.ekf.x.pData, pos, sizeof(float) * 3);
    } else {
        // Move vision data to ekf measurement vector and do the measurement update
        set_ekf_lock();
        memcpy(fusionEKF.ekf.z.pData, pos, sizeof(float) * 3);
        EKFUpdate(&fusionEKF.ekf);
        release_ekf_lock();
    }
}

/*
   Fuse IMU measurements.
*/

void STATE_FusionEKFIntertialUpdate(IMU_AccelVec3 acc, IMU_GyroVec3 gyr)
{

    // GYRO + ACCELEROMETER
    float linear_acc_x = acc.x - fusionEKF.bias.acc_x;
    float linear_acc_y = acc.y - fusionEKF.bias.acc_y;
    // Turn off acc for now.. to hard to handle
    linear_acc_x = 0;
    linear_acc_y = 0;

    float gyrAcc[3];
    float body_speed[3];
    NAV_wheelToBody(body_speed);
    gyrAcc[0] = gyr.z - fusionEKF.bias.gyr_z;
    gyrAcc[1] = body_speed[0];
    gyrAcc[2] = body_speed[1];
    // TODO: if using accelerometer we can lowpass noisy accelerometer
    /*gyrAcc[1] = LagElementPT1Process(&fusionEKF.lagAccel[0], linear_acc_x);*/
    /*gyrAcc[2] = LagElementPT1Process(&fusionEKF.lagAccel[1], linear_acc_y);*/
    // gyrAcc[1] = 0;
    // gyrAcc[2] = 0;

    // TODO: Add odometry

    // INERTIAL NAVIGATION SYSTEM (INS)
    if (get_ekf_lock() == LOCKED) {
        return;
    }
    memcpy(fusionEKF.ekf.u.pData, gyrAcc, sizeof(float) * 3);
    EKFPredict(&fusionEKF.ekf);
}

void STATE_Test()
{
    arm_matrix_instance_f32 A;   /* Matrix A Instance */
    arm_matrix_instance_f32 B;   /* Matrix B Instance */
    arm_matrix_instance_f32 AmB; /* Matrix A mutliplied with B */

    const uint32_t srcRows = 4;
    const uint32_t srcColumns = 4;
    arm_status status;
    arm_status test_status = ARM_MATH_SUCCESS;

    // Result buffer
    float32_t AmB_f32[16];

    /* Initialise A Matrix Instance with numRows, numCols and data array(A_f32) */
    arm_mat_init_f32(&A, srcRows, srcColumns, (float32_t*)A_f32);
    arm_mat_init_f32(&B, srcRows, srcColumns, (float32_t*)B_f32);
    arm_mat_init_f32(&AmB, srcRows, srcColumns, (float32_t*)AmB_f32);
    status = arm_mat_mult_f32(&A, &B, &AmB);

    // Sanity test check that we understand how the mult works
    if (status != ARM_MATH_SUCCESS || AmB_f32[0 * 4 + 3] != 2 || AmB_f32[1 * 4 + 3] != 1 || AmB_f32[2 * 4 + 3] != 1 || AmB_f32[3 * 4 + 3] != 1) {
        test_status = ARM_MATH_TEST_FAILURE;
    }

    if (test_status != ARM_MATH_SUCCESS) {
        LOG_DEBUG("Test %d failed glhf\r\n", 1);
    }

    // More tests here.....

    // This test is tricky because of the low pass filter... Seems to work though
    initEKF();
    IMU_AccelVec3 acc = {0, 1, 0};
    IMU_GyroVec3 gyr = {0, 0, 0};
    STATE_FusionEKFIntertialUpdate(acc, gyr);

    float posx = MAT_ELEMENT(fusionEKF.ekf.x, 0, 0);
    float posy = MAT_ELEMENT(fusionEKF.ekf.x, 1, 0);
    float posw = MAT_ELEMENT(fusionEKF.ekf.x, 2, 0);
    float velx = MAT_ELEMENT(fusionEKF.ekf.x, 3, 0);
    float vely = MAT_ELEMENT(fusionEKF.ekf.x, 4, 0);

    LOG_DEBUG("Got from fusion (px,py,pw,vx,vy) %f %f %f %f %f\r\n", posx, posy, posw, velx, vely);

    if (test_status == ARM_MATH_SUCCESS) {
        LOG_DEBUG("Tjoho all test passed\r\n");
    }
}

float STATE_get_posx() { return MAT_ELEMENT(fusionEKF.ekf.x, 0, 0); }

float STATE_get_posy() { return MAT_ELEMENT(fusionEKF.ekf.x, 1, 0); }

float STATE_get_robot_angle()
{

    float angle = MAT_ELEMENT(fusionEKF.ekf.x, 2, 0);
    return angle;
}

float STATE_get_vx() { return MAT_ELEMENT(fusionEKF.ekf.x, 3, 0); }

float STATE_get_vy() { return MAT_ELEMENT(fusionEKF.ekf.x, 4, 0); }

void STATE_log_states() { LOG_DEBUG("px: %f py: %f pw: %f vx: %f vy: %f\r\n", STATE_get_posx(), STATE_get_posy(), STATE_get_robot_angle(), STATE_get_vx(), STATE_get_vy()); }

void STATE_calibrate_imu_gyr()
{
    LOG_INFO("Calibrating imu and gyro\r\n");
    const int calib_size = 5000;

    float acc_bias_x = 0;
    float acc_bias_y = 0;
    float acc_bias_z = 0;

    float gyr_bias_x = 0;
    float gyr_bias_y = 0;
    float gyr_bias_z = 0;

    IMU_AccelVec3 acc;
    IMU_GyroVec3 gyr;
    for (int i = 0; i < calib_size; i++) {
        // Assume imu can handle 1khz update

        HAL_Delay(1);

        gyr = IMU_read_gyro_radps();
        acc = IMU_read_accel_mps2();

        // Read one block and extract gyro and accelerometer
        acc_bias_x += acc.x;
        acc_bias_y += acc.y;
        acc_bias_z += acc.z;

        gyr_bias_x += gyr.x;
        gyr_bias_y += gyr.y;
        gyr_bias_z += gyr.z;
    }

    fusionEKF.bias.acc_x = acc_bias_x / ((float)calib_size);
    fusionEKF.bias.acc_y = acc_bias_y / ((float)calib_size);
    fusionEKF.bias.acc_z = acc_bias_z / ((float)calib_size);

    fusionEKF.bias.gyr_x = gyr_bias_x / ((float)calib_size);
    fusionEKF.bias.gyr_y = gyr_bias_y / ((float)calib_size);
    fusionEKF.bias.gyr_z = gyr_bias_z / ((float)calib_size);

    fusionEKF.bias.is_calibrated = 1;
    LOG_INFO("Done calibrating\r\n");

}

uint16_t STATE_is_calibrated() { return fusionEKF.bias.is_calibrated; }

void STATE_disable_calibration() { fusionEKF.bias.is_calibrated = 0; }
