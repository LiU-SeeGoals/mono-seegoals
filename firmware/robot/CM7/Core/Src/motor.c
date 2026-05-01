#include "motor.h"

/* Private includes */
#include "data_logging.h"
#include "log.h"
#include "stm32h7xx_hal_def.h"
#include "stm32h7xx_hal_i2c.h"
#include <stdint.h>

/* Private variables */
static LOG_Module internal_log_mod;
extern float CONTROL_FREQ;

// Lookup table for the crc impl
static uint8_t CRC_TABLE[256] = {0,   94,  188, 226, 97,  63,  221, 131, 194, 156, 126, 32,  163, 253, 31,  65,  157, 195, 33,  127, 252, 162, 64,  30,  95,  1,   227, 189, 62,  96,  130, 220,
                                 35,  125, 159, 193, 66,  28,  254, 160, 225, 191, 93,  3,   128, 222, 60,  98,  190, 224, 2,   92,  223, 129, 99,  61,  124, 34,  192, 158, 29,  67,  161, 255,
                                 70,  24,  250, 164, 39,  121, 155, 197, 132, 218, 56,  102, 229, 187, 89,  7,   219, 133, 103, 57,  186, 228, 6,   88,  25,  71,  165, 251, 120, 38,  196, 154,
                                 101, 59,  217, 135, 4,   90,  184, 230, 167, 249, 27,  69,  198, 152, 122, 36,  248, 166, 68,  26,  153, 199, 37,  123, 58,  100, 134, 216, 91,  5,   231, 185,
                                 140, 210, 48,  110, 237, 179, 81,  15,  78,  16,  242, 172, 47,  113, 147, 205, 17,  79,  173, 243, 112, 46,  204, 146, 211, 141, 111, 49,  178, 236, 14,  80,
                                 175, 241, 19,  77,  206, 144, 114, 44,  109, 51,  209, 143, 12,  82,  176, 238, 50,  108, 142, 208, 83,  13,  239, 177, 240, 174, 76,  18,  145, 207, 45,  115,
                                 202, 148, 118, 40,  171, 245, 23,  73,  8,   86,  180, 234, 105, 55,  213, 139, 87,  9,   235, 181, 54,  104, 138, 212, 149, 203, 41,  119, 244, 170, 72,  22,
                                 233, 183, 85,  11,  136, 214, 52,  106, 43,  117, 151, 201, 74,  20,  246, 168, 116, 42,  200, 150, 21,  75,  169, 247, 182, 232, 10,  84,  215, 137, 107, 53};

/* Handle for encoder i2c bus */
static I2C_HandleTypeDef* enc_i2c_handle;

/*========== Private Functions ==========*/

uint8_t calc_crc(uint8_t* buf, uint8_t start_index, uint8_t end_index)
{
    uint8_t crc = 0;
    uint8_t i = 0;
    for (i = start_index; i < end_index; i++) {
        crc = CRC_TABLE[(crc ^ buf[i])];
    }
    return crc;
}

int32_t deserialize_int32(uint8_t* buffer, uint32_t start_index)
{
    uint32_t value = 0;

    value |= buffer[start_index + 0] << 24;
    value |= buffer[start_index + 1] << 16;
    value |= buffer[start_index + 2] << 8;
    value |= buffer[start_index + 3];
    return (int32_t)value;
}

/** Decodes the velocity data comming from the encoder counters.
 *  @param speeds the array of speeds that the result is written to
 *  @param speeds_offset the offset to begin writing the speeds
 *  @param buffer the buffer containing the data to be decoded
 **/
HAL_StatusTypeDef deserialize_speeds(int32_t* speeds, uint32_t speeds_offset, uint8_t* buffer)
{
    // TODO: Handle errors
    // STX Check
    if (buffer[0] != 0x02) {
        LOG_ERROR("Message does not start with STX\r\n");
        return HAL_ERROR;
    }
    // ETX Check
    if (buffer[10] != 0x03) {
        LOG_ERROR("Message does not contain ETX\r\n");
        return HAL_ERROR;
    }
    // CRC Check only the data
    uint8_t crc = calc_crc(buffer, 1, 9);
    if (buffer[9] != crc) {
        LOG_ERROR("CRC incorrect, expected: %u, got: %u\r\n", crc, buffer[9]);
        return HAL_ERROR;
    }

    speeds[speeds_offset + 0] = deserialize_int32(buffer, 1);
    speeds[speeds_offset + 1] = deserialize_int32(buffer, 5);

    return HAL_OK;
}

/*======== End Private Functions ========*/

/*
 Averages motor speed ticks
*/
void MOTOR_update_motor_ticks(MotorPWM* motor, float val)
{
    motor->cur_tick_idx = (1 + motor->cur_tick_idx) % (MOTOR_TICK_BUF_SIZE);
    motor->motor_ticks[motor->cur_tick_idx] = val;
}

/*
 Get motor speed
*/
float MOTOR_get_motor_ticks_per_iteration(MotorPWM* motor)
{
    // Can be made faster keeping a moving average and removing the last adding the new?
    float cur = 0;
    for (int i = 0; i < MOTOR_TICK_BUF_SIZE; i++) {
        cur += motor->motor_ticks[i];
    }
    return cur / ((float)MOTOR_TICK_BUF_SIZE);
}

HAL_StatusTypeDef MOTOR_get_encoder_rpms()
{
    // The device adress has to be left shifted by one as the 7 bit adress is in
    // bits 1..7 not 0..6
    uint8_t devaddr_0 = 0x5;
    uint8_t devaddr_1 = 0x6;
    int32_t speeds[4] = { 0 };

    // uint8_t message[10] = {0x03, 0x01};
    // HAL_StatusTypeDef tx_status = HAL_I2C_Master_Transmit(enc_i2c_handle, (devaddr << 1), message, 2, HAL_MAX_DELAY);
    // if (tx_status != HAL_OK) {
    //     LOG_ERROR("Failed to write to encoder counter\r\n");
    //     return HAL_ERROR;
    // }

    uint8_t counter_0_buf[11] = {0};
    HAL_StatusTypeDef rx_status = HAL_I2C_Master_Receive(enc_i2c_handle, (devaddr_0 << 1), counter_0_buf, 0x11, 1000);
    if (rx_status != HAL_OK) {
        LOG_ERROR("Failed to read from encoder counter 0 \r\n");
        return HAL_ERROR;
    }

    uint32_t i;
    for (i = 0; i < 11; i++) {
        LOG_INFO("%u: %u\r\n", i, counter_0_buf[i]);
    }

    uint8_t crc = calc_crc(counter_0_buf, 1, 10);
    LOG_INFO("crc: %u\r\n", crc);

    deserialize_speeds(speeds, 0, counter_0_buf);

    uint8_t counter_1_buf[11] = {0};
    rx_status = HAL_I2C_Master_Receive(enc_i2c_handle, (devaddr_1 << 1), counter_1_buf, 0x11, 1000);
    if (rx_status != HAL_OK) {
        LOG_ERROR("Failed to read from encoder counter 1\r\n");
        return HAL_ERROR;
    }

    for (i = 0; i < 11; i++) {
        LOG_INFO("%u: %u\r\n", i, counter_1_buf[i]);
    }

    crc = calc_crc(counter_1_buf, 1, 10);
    LOG_INFO("crc: %u\r\n", crc);

    deserialize_speeds(speeds, 2, counter_1_buf);

    LOG_INFO("speed0: %d, speed1: %d, speed2: %d, speed3: %d\r\n", speeds[0], speeds[1], speeds[2], speeds[3]);
    return HAL_OK;
}

void MOTOR_Init(TIM_HandleTypeDef* pwm_htim, I2C_HandleTypeDef* enc_i2c)
{
    enc_i2c_handle = enc_i2c;
    LOG_InitModule(&internal_log_mod, "MOTOR", LOG_LEVEL_TRACE, 0);
    HAL_TIM_Base_Start(pwm_htim);
}

void MOTOR_PWMStop(MotorPWM* motor)
{
    // TODO: This might disable the timer for all channels, not sure.
    HAL_TIM_PWM_Stop(motor->pwm_htim, motor->channel);
}

void MOTOR_PWMStart(MotorPWM* motor) { HAL_TIM_PWM_Start(motor->pwm_htim, motor->channel); }

void MOTOR_Break(MotorPWM* motor) { HAL_GPIO_WritePin(motor->breakPinPort, motor->breakPin, GPIO_PIN_SET); }

void MOTOR_StopBreak(MotorPWM* motor) { HAL_GPIO_WritePin(motor->breakPinPort, motor->breakPin, GPIO_PIN_RESET); }

/*
  Reverses motor direction
*/
int setDirection(MotorPWM* motor, float speed)
{
    // In theory we should have saftey checks to not reverse motor
    // Its hard to do in a good way though...

    // If going backward and speed is positive, change direction
    if (motor->dir == 0 && speed > 0) {
        motor->dir = 1;
        HAL_GPIO_WritePin(motor->reversePinPort, motor->reversePin, GPIO_PIN_RESET);
    }
    // If going forward and speed is negative, change direction
    if (motor->dir == 1 && speed < 0) {
        motor->dir = 0;
        HAL_GPIO_WritePin(motor->reversePinPort, motor->reversePin, GPIO_PIN_SET);
    }
    return HAL_OK;
}

int MOTOR_GetMotorSign(MotorPWM* motor)
{
    if (motor->dir == 1) {
        return 1;
    }
    if (motor->dir == 0) {
        return -1;
    }
}

/*
  PI control loop for motor ticks / second
*/
ControlSignal MOTOR_SetSpeed(MotorPWM* motor, float speed, float* I_prev)
{

    setDirection(motor, speed);

    // if (setDirection(motor, speed) == HAL_BUSY) {
    //     MOTOR_SendPWM(motor, 0);
    // }

    int sign = 1;
    if (speed < 0) {
        speed = -speed;
        sign = -1;
    }
    // PI control loop with integrator windup protection
    ControlSignal sig;

    float umin = 0;
    float umax = 1;
    float Ts = 1.f / CONTROL_FREQ;
    float Ti = 0.02;
    float K = 0.00015;
    float current_speed = (float)MOTOR_ReadTicksPerSecond(motor);
    float error = speed - current_speed;
    float I;
    float v;
    if (fabs(error) < 10.f) {
        error = 0;
        I = *I_prev;
        v = 0;
    } else {
        I = *I_prev + Ts / Ti * error;
        v = K * (error + I);
    }

    float u = 0;
    // integrator windup fix
    if (v < umin || v > umax) {
        I = *I_prev;
    }
    if (v > umax) {
        u = umax;
    } else if (v < umin) {
        u = umin;
    } else {
        u = v;
    }

    sig.u = u * sign;
    sig.e = error * sign;
    sig.y = current_speed * sign;
    sig.r = speed * sign;

    MOTOR_SendPWM(motor, u);
    *I_prev = I;

    return sig;
}

/*
  Send pwm with with pulse width of pulse_width * timer_period
  meaning pulse width is a float between 0 - 1.
*/

void MOTOR_SendPWM(MotorPWM* motor, float pulse_width)
{
    // TODO: How to handle changing directions?

    // Make sure we dont explode the timer limit
    if (pulse_width > 1) {
        pulse_width = 1;
    }
    if (pulse_width < 0) {
        pulse_width = 0;
    }

    float max_scale = 0.1;                 // Use a max scaling for the motor speed
    float scale = max_scale * pulse_width; // make max_scale largest scaling

    // TODO: How to handle rounding errors, do they even matter?
    int pwm_speed = motor->pwm_htim->Init.Period * scale;

    __HAL_TIM_SET_COMPARE(motor->pwm_htim, motor->channel, pwm_speed);
}

float MOTOR_ReadTicksPerSecond(MotorPWM* motor)
{
    // When I write this control freq should be 2000, each iteration is a
    // time-step of 1/2000 so this gives the speed in ticks per second.
    float speed_s = MOTOR_get_motor_ticks_per_iteration(motor) * CONTROL_FREQ;

    return speed_s;
}
