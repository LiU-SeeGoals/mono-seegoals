#include "imu.pb.h"
#include "pb_encode.h"
#include "log.h"
#include <string.h>
#include "data_logging.h"
#include "log.h"


static LOG_Module internal_log_mod;
static DataLog data;
static SPI_HandleTypeDef *HSPI;

void DATA_log_imu_data(float x, float y, float z)
{
    uint32_t timestamp = HAL_GetTick();
    data.imu[data.imu_write_idx].x = x;
    data.imu[data.imu_write_idx].y = y;
    data.imu[data.imu_write_idx].z = z;
    data.imu[data.imu_write_idx].timestamp = timestamp;
    if (!data.mutex)
    {
        data.imu_write_idx = (1 + data.imu_write_idx) % 2;
    }
}

void DATA_uart_read(){
    data.mutex = true;
    int read_idx = (1 + data.imu_write_idx) % 2;
    uint32_t ts = data.imu[read_idx].timestamp;
    float x = data.imu[read_idx].x;
    float y = data.imu[read_idx].y;
    float z = data.imu[read_idx].z;
    data.mutex = false;
    LOG_INFO("t%dx%fy%fz%f_\r\n", ts,x,y,z, read_idx);
}

void DATA_Init(SPI_HandleTypeDef *hspi){
    // data.imu_write_idx = 0;
    // data.mutex = 0;
    // data.imu[0].timestamp = 0;
    // data.imu[1].timestamp = 0;
    HSPI = hspi;
    memset(&data,0,sizeof(data));
    LOG_InitModule(&internal_log_mod, "DATA", LOG_LEVEL_INFO, 0);
}

#define PROTO_BUFFER_SIZE 64
static uint8_t protobuf_buf[PROTO_BUFFER_SIZE];
static uint8_t protobuf_len;

bool pack_spi_packet()
{
    for (int i = 0; i < PROTO_BUFFER_SIZE; i++)
    {
        protobuf_buf[i] = 0;
    }

    ImuSample msg = ImuSample_init_zero;

    msg.x = 1;
    msg.y = 2;
    msg.z = 3;
    // memcpy(packet.payload.bytes, data, data_len);

    pb_ostream_t stream = pb_ostream_from_buffer(protobuf_buf, sizeof(protobuf_buf)*sizeof(uint8_t));

    if (!pb_encode(&stream, ImuSample_fields, &msg))
    {
        LOG_INFO("Protobuf packet failed to encode");
        return false;
    }

    if (stream.bytes_written > UINT8_MAX)
    {
        LOG_INFO("Protobuf packet to large");
        return false;
    }

    protobuf_len = stream.bytes_written;
    return true;
}

void DATA_spi_read(){

    HAL_StatusTypeDef status;
    data.mutex = true;
    bool pack_status = pack_spi_packet();
    data.mutex = false;

    if (pack_status == true)
    {
        // for (int i = 0; i < PROTO_BUFFER_SIZE - 1; i++)
        // {
        //     protobuf_buf[i] = protobuf_buf[i+1];
        // }

        // protobuf_buf[0] = protobuf_len;

        status = HAL_SPI_Transmit(HSPI, protobuf_buf, PROTO_BUFFER_SIZE, 1000);
    }
    data.mutex = false;
}
