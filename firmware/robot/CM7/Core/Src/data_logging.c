#include "imu.pb.h"
#include "pb_encode.h"
#include "log.h"
#include <string.h>
#include "data_logging.h"
#include "log.h"

#define SPI_BUFFER_SIZE 129

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

    ImuSample msg = ImuSample_init_zero;

    int idx = get_read_idx(&data.imu_write_idx);
    msg.imu_x = data.imu[idx].x;
    msg.imu_y = data.imu[idx].y;
    msg.imu_z = data.imu[idx].z;
    msg.imu_ts = data.imu[idx].timestamp;

    // Skip the first byte to place message length there
    pb_ostream_t stream = pb_ostream_from_buffer(protobuf_buf + 1, sizeof(protobuf_buf) - 1);

    if (!pb_encode(&stream, ImuSample_fields, &msg))
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

    HAL_StatusTypeDef status;
    bool pack_status = pack_spi_packet();

    if (pack_status == true)
    {
        status = HAL_SPI_Transmit_IT(HSPI, protobuf_buf, SPI_BUFFER_SIZE);
    }
    data.mutex = false;
}
