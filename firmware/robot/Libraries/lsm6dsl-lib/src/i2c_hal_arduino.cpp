#include <Wire.h>

#include "i2c_hal_arduino.h"


void lsm6dsl_i2c_master_init(Lsm6dsl_I2cPortHandle_t *port_handle)
{
    // The port handle may be used to get information about the
    // I2C port intended to be used. Here we ignore it and don't
    // supply any pins to begin()

    Wire.begin();
}

void lsm6dsl_i2c_master_transmit_first_frame(Lsm6dsl_I2cPortHandle_t *port_handle,
                                             uint8_t device_address, uint8_t *outbuf, uint8_t size)
{
    Wire.beginTransmission(device_address); // Start condition
    
    Wire.write(outbuf, size);
}


void lsm6dsl_i2c_master_transmit_last_frame(Lsm6dsl_I2cPortHandle_t *port_handle,
                                            uint8_t device_address, uint8_t *outbuf, uint8_t size)
{
    Wire.write(outbuf, size);
    
    Wire.endTransmission(true); // Stop condition
}


void lsm6dsl_i2c_master_receive_last_frame(Lsm6dsl_I2cPortHandle_t *port_handle,
                                           uint8_t device_address, uint8_t *inbuf, uint8_t size)
{
    Wire.endTransmission(false); // Repeated start condition
    
    uint32_t bytes_received = Wire.requestFrom(device_address, size);
    Wire.readBytes(inbuf, bytes_received > size ? size : bytes_received);
    
    Wire.endTransmission(true);  // Stop condition
}

