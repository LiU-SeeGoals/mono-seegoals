#include <HLindgren_LSM6DSL.h>


#define I2C_DEV_ADDR 0x6B

#define FIFO_DATA_ARRAY_NMEMB 4096

static Lsm6dsl_Device_t device_handle;

void setup() {
    Serial.begin(115200);
    Serial.setDebugOutput(true);
    
    Serial.println("Opening device...");
    lsm6dsl_open_i2c_slave_device(NULL, I2C_DEV_ADDR, &device_handle);

    Serial.println("Setting accelerometer data rate...");
    lsm6dsl_accel_init(&device_handle, Lsm6dsl_DataRate_52Hz);

    Serial.println("Setting gyro data rate...");
    lsm6dsl_gyro_init(&device_handle, Lsm6dsl_DataRate_52Hz);

    Serial.println("Initializing FIFO...");
    lsm6dsl_fifo_init(&device_handle, Lsm6dsl_FifoMode_Continuous, Lsm6dsl_DataRate_52Hz);

    Serial.println("Setting decimation...");
    lsm6dsl_fifo_set_decimation(&device_handle, Lsm6dsl_DecFifoGyro_NoDec, Lsm6dsl_DecFifoXl_NoDec);

    Serial.println("Done initializing");
}


static Lsm6dsl_Data_t data[FIFO_DATA_ARRAY_NMEMB] = {0};

void loop() {
    delay(1000);
    

    Serial.println("Reading from FIFO...");
    int32_t blocks_read = lsm6dsl_fifo_read_to_end(&device_handle, data, FIFO_DATA_ARRAY_NMEMB);

    if (blocks_read > 0) {
        Serial.print("Blocks read: ");
        Serial.println(blocks_read);

        for (int32_t i = 0; i < blocks_read; i++) {
            Serial.print("Gyroscope: ");
            Serial.print("X=");
            Serial.print(data[i].gyro.x);
            Serial.print(" Y=");
            Serial.print(data[i].gyro.y);
            Serial.print(" Z=");
            Serial.println(data[i].gyro.z);

            Serial.print("Accelerometer: ");
            Serial.print("X=");
            Serial.print(data[i].accel.x);
            Serial.print(" Y=");
            Serial.print(data[i].accel.y);
            Serial.print(" Z=");
            Serial.println(data[i].accel.z);
        }

    } else {
        Serial.print("Error code: ");
        Serial.println(blocks_read);
    }
}

