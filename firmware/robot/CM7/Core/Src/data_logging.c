#include "log.h"
#include <string.h>
#include "data_logging.h"
#include "log.h"

static LOG_Module internal_log_mod;
static DataLog data;

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
    LOG_INFO("t%dx%fy%fz%f_asdfasdfasfdasfdasdfasdfsadfasdfadsfasfdadsfadsf\r\n", ts,x,y,z, read_idx);
}

void DATA_Init(){
    // data.imu_write_idx = 0;
    // data.mutex = 0;
    // data.imu[0].timestamp = 0;
    // data.imu[1].timestamp = 0;
    memset(&data,0,sizeof(data));
    LOG_InitModule(&internal_log_mod, "DATA", LOG_LEVEL_INFO, 0);
}
