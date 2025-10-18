#include "HLindgren_LSM6DSL.h"

void lsm6dsl_open_i2c_slave_device(Lsm6dsl_I2cPortHandle_t *port_handle,
                                   uint8_t device_address,
                                   Lsm6dsl_Device_t *device) {
  lsm6dsl_i2c_master_init(port_handle);

  device->port_handle = port_handle;
  device->device_address = device_address;
}

void lsm6dsl_open_spi_slave_device(Lsm6dsl_SpiPortHandle_t *port_handle,
                                   Lsm6dsl_Device_t *device) {
  lsm6dsl_spi_master_init(port_handle);
  device->port_handle = port_handle;
}

void lsm6dsl_read_register(Lsm6dsl_Device_t *device, uint8_t reg_address,
                           uint8_t *inbuf, uint8_t size) {
#ifdef LSM6DSL_USE_SPI
  lsm6dsl_spi_read_mem(device->port_handle, device->device_address, reg_address,
                       inbuf, size);
#elif defined LSM6DSL_USE_I2C_MEM_READ_AND_WRITE
  lsm6dsl_i2c_master_read_mem(device->port_handle, device->device_address,
                              reg_address, inbuf, size);
  // TODO: error check
#else
  lsm6dsl_i2c_master_transmit_first_frame(
      device->port_handle, device->device_address, &reg_address, 1);
  lsm6dsl_i2c_master_receive_last_frame(device->port_handle,
                                        device->device_address, inbuf, size);
#endif
}

void lsm6dsl_write_register(Lsm6dsl_Device_t *device, uint8_t reg_address,
                            uint8_t *outbuf, uint8_t size) {
#ifdef LSM6DSL_USE_SPI
  lsm6dsl_spi_write_mem(device->port_handle, device->device_address,
                        reg_address, outbuf, size);
#elif defined LSM6DSL_USE_I2C_MEM_READ_AND_WRITE
  lsm6dsl_i2c_master_write_mem(device->port_handle, device->device_address,
                               reg_address, outbuf, size);
#else
  lsm6dsl_i2c_master_transmit_first_frame(
      device->port_handle, device->device_address, &reg_address, 1);
  lsm6dsl_i2c_master_transmit_last_frame(device->port_handle,
                                         device->device_address, outbuf, size);
#endif
}

static uint8_t data_rate_to_accel_odr_hp[] = {
    Lsm6dsl_OdrXlHp_12_5Hz,  Lsm6dsl_OdrXlHp_26Hz,    Lsm6dsl_OdrXlHp_52Hz,
    Lsm6dsl_OdrXlHp_104Hz,   Lsm6dsl_OdrXlHp_208Hz,   Lsm6dsl_OdrXlHp_416Hz,
    Lsm6dsl_OdrXlHp_833Hz,   Lsm6dsl_OdrXlHp_1_66kHz, Lsm6dsl_OdrXlHp_3_33kHz,
    Lsm6dsl_OdrXlHp_6_66kHz,
};

static uint8_t data_rate_to_gyro_odr[] = {
    Lsm6dsl_OdrGyro_12_5Hz,  Lsm6dsl_OdrGyro_26Hz,    Lsm6dsl_OdrGyro_52Hz,
    Lsm6dsl_OdrGyro_104Hz,   Lsm6dsl_OdrGyro_208Hz,   Lsm6dsl_OdrGyro_416Hz,
    Lsm6dsl_OdrGyro_833Hz,   Lsm6dsl_OdrGyro_1_66kHz, Lsm6dsl_OdrGyro_3_33kHz,
    Lsm6dsl_OdrGyro_6_66kHz,
};

static uint8_t data_rate_to_fifo_odr[] = {
    Lsm6dsl_OdrFifo_12_5Hz,  Lsm6dsl_OdrFifo_26Hz,    Lsm6dsl_OdrFifo_52Hz,
    Lsm6dsl_OdrFifo_104Hz,   Lsm6dsl_OdrFifo_208Hz,   Lsm6dsl_OdrFifo_416Hz,
    Lsm6dsl_OdrFifo_833Hz,   Lsm6dsl_OdrFifo_1_66kHz, Lsm6dsl_OdrFifo_3_33kHz,
    Lsm6dsl_OdrFifo_6_66kHz,
};

void lsm6dsl_accel_init(Lsm6dsl_Device_t *device,
                        Lsm6dsl_DataRate_e accel_data_rate,
                        Lsm6dsl_FullScaleXl_e full_scale_g) {
  // Note assumes high performance mode (XL_HM_MODE = 0)
  uint8_t accel_odr = data_rate_to_accel_odr_hp[accel_data_rate] | full_scale_g;
  lsm6dsl_write_register(device, LSM6DSL_REG_CTRL1_XL, &accel_odr, 1);
}

void lsm6dsl_gyro_init(Lsm6dsl_Device_t *device,
                       Lsm6dsl_DataRate_e gyro_data_rate,
                       Lsm6dsl_FullScaleGyro_e full_scale_dps) {
  uint8_t gyro_odr = data_rate_to_gyro_odr[gyro_data_rate] | full_scale_dps;
  lsm6dsl_write_register(device, LSM6DSL_REG_CTRL2_G, &gyro_odr, 1);
}

void lsm6dsl_fifo_init(Lsm6dsl_Device_t *device, Lsm6dsl_FifoMode_e fifo_mode,
                       Lsm6dsl_DataRate_e data_rate) {
  uint8_t fifo_ctrl5 = data_rate_to_fifo_odr[data_rate] | fifo_mode;
  lsm6dsl_write_register(device, LSM6DSL_REG_FIFO_CTRL5, &fifo_ctrl5, 1);

  // Enable Block Data Update (and keep address auto increment on)
  uint8_t ctrl3_c = LSM6DSL_MASK_BDU | LSM6DSL_MASK_IF_INC;
  lsm6dsl_write_register(device, LSM6DSL_REG_CTRL3_C, &ctrl3_c, 1);
}

void lsm6dsl_fifo_set_decimation(Lsm6dsl_Device_t *device,
                                 Lsm6dsl_DecFifoGyro_e gyro_dec,
                                 Lsm6dsl_DecFifoXl_e accel_dec) {
  uint8_t fifo_ctrl3 = gyro_dec | accel_dec;
  lsm6dsl_write_register(device, LSM6DSL_REG_FIFO_CTRL3, &fifo_ctrl3, 1);
}

Lsm6dsl_GyroData_t lsm6dsl_gyro_read(Lsm6dsl_Device_t *device) {
  uint8_t inbuf[LSM6DSL_GYRO_DATA_SIZE] = {0};
  lsm6dsl_read_register(device, LSM6DSL_REG_OUTX_L_G, inbuf,
                        LSM6DSL_GYRO_DATA_SIZE);

  uint16_t *buf16 = (uint16_t *)&inbuf;

  return (Lsm6dsl_GyroData_t){.x = buf16[0], .y = buf16[1], .z = buf16[2]};
}

Lsm6dsl_AccelData_t lsm6dsl_accel_read(Lsm6dsl_Device_t *device) {
  uint8_t inbuf[LSM6DSL_ACCEL_DATA_SIZE] = {0};
  lsm6dsl_read_register(device, LSM6DSL_REG_OUTX_L_XL, inbuf,
                        LSM6DSL_ACCEL_DATA_SIZE);

  uint16_t *buf16 = (uint16_t *)&inbuf;

  return (Lsm6dsl_AccelData_t){.x = buf16[0], .y = buf16[1], .z = buf16[2]};
}

#define LSM6DSL_ERR_FIFO_DATABUF_RAN_OUT -2
// Error: Buffer was filled before the FIFO was emptied
// Either the provided buffer too small to fit all data in FIFO or
// the FIFO is being filled faster than it is being read.

int32_t lsm6dsl_fifo_read_to_end(Lsm6dsl_Device_t *device,
                                 Lsm6dsl_Data_t *databuf, uint16_t nmemb) {
  for (uint16_t databuf_i = 0; databuf_i < nmemb; databuf_i++) {
    uint16_t data_words[LSM6DSL_FIFO_DATA_WORDS] = {0};

    for (int i = 0; i < LSM6DSL_FIFO_DATA_WORDS; i++) {
      // Read 4 status registers and 16-bit data out word from FIFO (= 6 bytes)
      uint8_t inbuf[6] = {0};
      lsm6dsl_read_register(device, LSM6DSL_REG_FIFO_STATUS1, inbuf, 6);

      uint8_t fifo_status2 = inbuf[1];

      bool empty = (fifo_status2 & LSM6DSL_MASK_FIFO_EMPTY) > 0;

      if (empty) {
        // Return number of datablocks written
        return databuf_i;
      }

      data_words[i] = ((uint16_t *)&inbuf)[2];
    }

    databuf[databuf_i] = (Lsm6dsl_Data_t){
        .gyro =
            {
                .x = data_words[0],
                .y = data_words[1],
                .z = data_words[2],
            },

        .accel =
            {
                .x = data_words[3],
                .y = data_words[4],
                .z = data_words[5],
            },

        .sensorhub_1_to_6 =
            {
                data_words[6],
                data_words[7],
                data_words[8],
            },

        .sensorhub_7_to_12 =
            {
                data_words[9],
                data_words[10],
                data_words[11],
            },

        // TODO: Other options for 4th data set
    };
  }

  // Data buffer filled, FIFO may not have been completely read
  return LSM6DSL_ERR_FIFO_DATABUF_RAN_OUT;
}
