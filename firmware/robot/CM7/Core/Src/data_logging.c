#include "imu.pb.h"
#include "pb_encode.h"
#include "log.h"
#include <string.h>
#include "data_logging.h"
#include "log.h"

#define SPI_BUFFER_SIZE 257
#define DOUBLE_BUFFER 2

typedef struct
{
    uint32_t timestamp;
    float x;
    float y;
    float z;
} Imu;

typedef struct
{
    uint32_t timestamp;
    float u;
    float v;
    float w;
} Odometry;

typedef struct
{
    float px;
    float py;
    float pw;
    float vx;
    float vy;
    float vw;
    uint32_t timestamp;
} State;

typedef struct
{
    ControlSignal m1;
    ControlSignal m2;
    ControlSignal m3;
    ControlSignal m4;
    uint32_t timestamp;
} Motor;

typedef struct
{
    ControlSignal x;
    ControlSignal y;
    ControlSignal angle;
    uint32_t timestamp;
} Pos;

typedef struct
{
    float x;
    float y;
    float w;
    uint32_t timestamp;
} Vision;


typedef struct
{
    Imu imu[DOUBLE_BUFFER];
    uint8_t imu_write_idx;

    State state[DOUBLE_BUFFER];
    uint8_t state_write_idx;

    Motor motor[DOUBLE_BUFFER];
    uint8_t motor_write_idx;

    Pos pos[DOUBLE_BUFFER];
    uint8_t pos_write_idx;

    Vision vision[DOUBLE_BUFFER];
    uint8_t vision_write_idx;

    Odometry odometry[DOUBLE_BUFFER];
    uint8_t odometry_write_idx;

    volatile bool mutex; // volatile bools are atomic
} DataLog;


static LOG_Module internal_log_mod;
static DataLog data;
static SPI_HandleTypeDef *HSPI;
static pb_byte_t protobuf_buf[SPI_BUFFER_SIZE];
static uint8_t protobuf_len;

static int get_read_idx(uint8_t* write_idx)
{
    return (1 + *write_idx) % 2;
}

static void buffer_swap(uint8_t* write_idx)
{
    // Assuming the writer is not preempted by reader
    // I.e. currently interrupts have priority over where reading happens (main.c)
    if (!data.mutex)
    {
        *write_idx = (1 + *write_idx) % 2;
    }
}

static bool pack_spi_packet()
{
    // Data layout
    // [uint8 msg length][variable length protobuf]
    data.mutex = true;
    for (int i = 0; i < SPI_BUFFER_SIZE; i++)
    {
        protobuf_buf[i] = 0;
    }

    data_sample msg = data_sample_init_zero;

    int imu_idx = get_read_idx(&data.imu_write_idx);
    // gyro x and y unncessesary
    // msg.imu_x = data.imu[imu_idx].x;
    // msg.imu_y = data.imu[imu_idx].y;
    msg.gyro.z = data.imu[imu_idx].z;
    msg.gyro.timestamp = data.imu[imu_idx].timestamp;

    int state_idx = get_read_idx(&data.state_write_idx);
    msg.state.x = data.state[state_idx].px;
    msg.state.y = data.state[state_idx].py;
    msg.state.z = data.state[state_idx].pw;
    msg.state.timestamp = data.state[state_idx].timestamp;

    int motor_idx = get_read_idx(&data.motor_write_idx);
    msg.m_timestamp = data.motor[motor_idx].timestamp;

    msg.m1.ref = data.motor[motor_idx].m1.r;
    msg.m1.control = data.motor[motor_idx].m1.u;
    msg.m1.output = data.motor[motor_idx].m1.y;
    msg.m1.error = data.motor[motor_idx].m1.e;

    msg.m2.ref = data.motor[motor_idx].m2.r;
    msg.m2.control = data.motor[motor_idx].m2.u;
    msg.m2.output = data.motor[motor_idx].m2.y;
    msg.m2.error = data.motor[motor_idx].m2.e;

    msg.m3.ref = data.motor[motor_idx].m3.r;
    msg.m3.control = data.motor[motor_idx].m3.u;
    msg.m3.output = data.motor[motor_idx].m3.y;
    msg.m3.error = data.motor[motor_idx].m3.e;

    msg.m4.ref = data.motor[motor_idx].m4.r;
    msg.m4.control = data.motor[motor_idx].m4.u;
    msg.m4.output = data.motor[motor_idx].m4.y;
    msg.m4.error = data.motor[motor_idx].m4.e;

    int pos_idx = get_read_idx(&data.pos_write_idx);
    msg.pos_timestamp = data.pos[pos_idx].timestamp;

    msg.pos_x.ref = data.pos[pos_idx].x.r;
    msg.pos_x.control = data.pos[pos_idx].x.u;
    msg.pos_x.output = data.pos[pos_idx].x.y;
    msg.pos_x.error = data.pos[pos_idx].x.e;

    msg.pos_y.ref = data.pos[pos_idx].y.r;
    msg.pos_y.control = data.pos[pos_idx].y.u;
    msg.pos_y.output = data.pos[pos_idx].y.y;
    msg.pos_y.error = data.pos[pos_idx].y.e;

    msg.pos_angle.ref = data.pos[pos_idx].angle.r;
    msg.pos_angle.control = data.pos[pos_idx].angle.u;
    msg.pos_angle.output = data.pos[pos_idx].angle.y;
    msg.pos_angle.error = data.pos[pos_idx].angle.e;

    int vision_idx = get_read_idx(&data.vision_write_idx);
    msg.vision.x = data.vision[vision_idx].x;
    msg.vision.y = data.vision[vision_idx].y;
    msg.vision.z = data.vision[vision_idx].w;
    msg.vision.timestamp = data.vision[vision_idx].timestamp;

    int odometry_idx = get_read_idx(&data.odometry_write_idx);
    msg.odometry.x = data.odometry[odometry_idx].u;
    msg.odometry.y = data.odometry[odometry_idx].v;
    msg.odometry.z = data.odometry[odometry_idx].w;
    msg.odometry.timestamp = data.odometry[odometry_idx].timestamp;

    msg.has_gyro = true;
    msg.has_state = true;
    msg.has_vision = true;
    msg.has_m1 = true;
    msg.has_m2 = true;
    msg.has_m3 = true;
    msg.has_m4 = true;
    msg.has_pos_x = true;
    msg.has_pos_y = true;
    msg.has_pos_angle = true;
    msg.has_vision = true;
    msg.has_odometry = true;

    // Skip the first byte to place message length there
    pb_ostream_t stream = pb_ostream_from_buffer(protobuf_buf + 1, sizeof(protobuf_buf) - 1);

    if (!pb_encode(&stream, data_sample_fields, &msg))
    {
        LOG_DEBUG("Protobuf packet failed to encode");
        return false;
    }

    if (stream.bytes_written > UINT8_MAX || stream.bytes_written > SPI_BUFFER_SIZE)
    {
        LOG_ERROR("Protobuf packet to large");
        return false;
    }

    protobuf_buf[0] = stream.bytes_written;

    data.mutex = false;
    return true;
}


void DATA_log_imu_data(float x, float y, float z)
{
    uint32_t timestamp = HAL_GetTick();
    data.imu[data.imu_write_idx].x = x;
    data.imu[data.imu_write_idx].y = y;
    data.imu[data.imu_write_idx].z = z;
    data.imu[data.imu_write_idx].timestamp = timestamp;
    buffer_swap(&data.imu_write_idx);
}

void DATA_log_odometry(float u, float v, float w)
{
    uint32_t timestamp = HAL_GetTick();
    data.odometry[data.odometry_write_idx].u = u;
    data.odometry[data.odometry_write_idx].v = v;
    data.odometry[data.odometry_write_idx].w = w;
    data.odometry[data.odometry_write_idx].timestamp = timestamp;
    buffer_swap(&data.imu_write_idx);
}

void DATA_log_state(float x, float y, float w)
{
    uint32_t timestamp = HAL_GetTick();
    data.state[data.state_write_idx].px = x;
    data.state[data.state_write_idx].py = y;
    data.state[data.state_write_idx].pw = w;
    data.state[data.state_write_idx].timestamp = timestamp;
    buffer_swap(&data.state_write_idx);
}

void DATA_log_vision(float x, float y, float w)
{
    uint32_t timestamp = HAL_GetTick();
    data.vision[data.vision_write_idx].x = x;
    data.vision[data.vision_write_idx].y = y;
    data.vision[data.vision_write_idx].w = w;
    data.vision[data.vision_write_idx].timestamp = timestamp;
    buffer_swap(&data.vision_write_idx);
}

void DATA_log_motor(ControlSignal m1, ControlSignal m2, ControlSignal m3, ControlSignal m4)
{
    uint32_t timestamp = HAL_GetTick();
    data.motor[data.motor_write_idx].m1 = m1;
    data.motor[data.motor_write_idx].m2 = m2;
    data.motor[data.motor_write_idx].m3 = m3;
    data.motor[data.motor_write_idx].m4 = m4;
    data.motor[data.motor_write_idx].timestamp = timestamp;
    buffer_swap(&data.motor_write_idx);
}

void DATA_log_pos(ControlSignal x, ControlSignal y, ControlSignal angle)
{
    uint32_t timestamp = HAL_GetTick();
    data.pos[data.state_write_idx].x = x;
    data.pos[data.state_write_idx].y = y;
    data.pos[data.state_write_idx].angle = angle;
    data.pos[data.state_write_idx].timestamp = timestamp;
    buffer_swap(&data.state_write_idx);
}

void DATA_uart_send(){
    data.mutex = true;
    int read_idx = get_read_idx(&data.imu_write_idx);
    uint32_t ts = data.imu[read_idx].timestamp;
    float x = data.imu[read_idx].x;
    float y = data.imu[read_idx].y;
    float z = data.imu[read_idx].z;
    data.mutex = false;
    LOG_INFO("t%dx%fy%fz%f_\r\n", ts,x,y,z, read_idx);
}

void DATA_Init(SPI_HandleTypeDef *hspi){
    HSPI = hspi;
    DATA_spi_send();
    memset(&data,0,sizeof(data));
    LOG_InitModule(&internal_log_mod, "DATA", LOG_LEVEL_INFO, 0);
}

void DATA_spi_send(){
    // TODO: Using DMA instead of interrupts we can make transfer faster
    // Since there is no overhead with interrupts, and also uses less mcu

    HAL_StatusTypeDef status;
    bool pack_status = pack_spi_packet();

    if (pack_status == true)
    {
        status = HAL_SPI_Transmit_IT(HSPI, protobuf_buf, SPI_BUFFER_SIZE);
    }
    data.mutex = false;
}
