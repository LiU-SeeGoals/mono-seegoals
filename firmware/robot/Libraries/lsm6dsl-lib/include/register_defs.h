/* Henrik Lindgren 2024
 *
 * Enum values are already shifted to their respective positions in the register
 */

#ifndef REGISTER_DEFS_H
#define REGISTER_DEFS_H


#define LSM6DSL_REG_FUNC_CFG_ACCESS          0x01

#define LSM6DSL_REG_SENSOR_SYNC_TIME_FRAME   0x04
#define LSM6DSL_REG_SENSOR_SYNC_RES_RATIO    0x05

#define LSM6DSL_REG_FIFO_CTRL1               0x06
#define LSM6DSL_REG_FIFO_CTRL2               0x07
#define LSM6DSL_REG_FIFO_CTRL3               0x08
#define LSM6DSL_REG_FIFO_CTRL4               0x09
#define LSM6DSL_REG_FIFO_CTRL5               0x0A

#define LSM6DSL_REG_DRDY_PULSE_CFG_G         0x0B

#define LSM6DSL_REG_INT1_CTRL                0x0D
#define LSM6DSL_REG_INT2_CTRL                0x0E

#define LSM6DSL_REG_WHO_AM_I                 0x0F

#define LSM6DSL_REG_CTRL1_XL                 0x10
#define LSM6DSL_REG_CTRL2_G                  0x11
#define LSM6DSL_REG_CTRL3_C                  0x12
#define LSM6DSL_REG_CTRL4_C                  0x13
#define LSM6DSL_REG_CTRL5_C                  0x14
#define LSM6DSL_REG_CTRL6_C                  0x15
#define LSM6DSL_REG_CTRL7_G                  0x16
#define LSM6DSL_REG_CTRL8_XL                 0x17
#define LSM6DSL_REG_CTRL9_XL                 0x18
#define LSM6DSL_REG_CTRL10_C                 0x19

#define LSM6DSL_REG_MASTER_CONFIG            0x1A
#define LSM6DSL_REG_WAKE_UP_SRC              0x1B
#define LSM6DSL_REG_TAP_SRC                  0x1C
#define LSM6DSL_REG_D6D_SRC                  0x1D
#define LSM6DSL_REG_STATUS_REG               0x1E
#define LSM6DSL_REG_OUT_TEMP_L               0x20
#define LSM6DSL_REG_OUT_TEMP_H               0x21
#define LSM6DSL_REG_OUTX_L_G                 0x22
#define LSM6DSL_REG_OUTX_H_G                 0x23
#define LSM6DSL_REG_OUTY_L_G                 0x24
#define LSM6DSL_REG_OUTY_H_G                 0x25
#define LSM6DSL_REG_OUTZ_L_G                 0x26
#define LSM6DSL_REG_OUTZ_H_G                 0x27
#define LSM6DSL_REG_OUTX_L_XL                0x28
#define LSM6DSL_REG_OUTX_H_XL                0x29
#define LSM6DSL_REG_OUTY_L_XL                0x2A
#define LSM6DSL_REG_OUTY_H_XL                0x2B
#define LSM6DSL_REG_OUTZ_L_XL                0x2C
#define LSM6DSL_REG_OUTZ_H_XL                0x2D
#define LSM6DSL_REG_SENSORHUB1_REG           0x2E
#define LSM6DSL_REG_SENSORHUB2_REG           0x2F
#define LSM6DSL_REG_SENSORHUB3_REG           0x30
#define LSM6DSL_REG_SENSORHUB4_REG           0x31
#define LSM6DSL_REG_SENSORHUB5_REG           0x32
#define LSM6DSL_REG_SENSORHUB6_REG           0x33
#define LSM6DSL_REG_SENSORHUB7_REG           0x34
#define LSM6DSL_REG_SENSORHUB8_REG           0x35
#define LSM6DSL_REG_SENSORHUB9_REG           0x36
#define LSM6DSL_REG_SENSORHUB10_REG          0x37
#define LSM6DSL_REG_SENSORHUB11_REG          0x38
#define LSM6DSL_REG_SENSORHUB12_REG          0x39


#define LSM6DSL_REG_FIFO_STATUS1             0x3A
#define LSM6DSL_REG_FIFO_STATUS2             0x3B
#define LSM6DSL_REG_FIFO_STATUS3             0x3C
#define LSM6DSL_REG_FIFO_STATUS4             0x3D
#define LSM6DSL_REG_FIFO_DATA_OUT_L          0x3E
#define LSM6DSL_REG_FIFO_DATA_OUT_H          0x3F
#define LSM6DSL_REG_TIMESTAMP0_REG           0x40
#define LSM6DSL_REG_TIMESTAMP1_REG           0x41
#define LSM6DSL_REG_TIMESTAMP2_REG           0x42
#define LSM6DSL_REG_STEP_TIMESTAMP_L         0x49
#define LSM6DSL_REG_STEP_TIMESTAMP_H         0x4A
#define LSM6DSL_REG_STEP_COUNTER_L           0x4B
#define LSM6DSL_REG_STEP_COUNTER_H           0x4C
#define LSM6DSL_REG_SENSORHUB13_REG          0x4D
#define LSM6DSL_REG_SENSORHUB14_REG          0x4E
#define LSM6DSL_REG_SENSORHUB15_REG          0x4F
#define LSM6DSL_REG_SENSORHUB16_REG          0x50
#define LSM6DSL_REG_SENSORHUB17_REG          0x51
#define LSM6DSL_REG_SENSORHUB18_REG          0x52
#define LSM6DSL_REG_FUNC_SRC1                0x53
#define LSM6DSL_REG_FUNC_SRC2                0x54
#define LSM6DSL_REG_WRIST_TILT_IA            0x55
#define LSM6DSL_REG_TAP_CFG                  0x58
#define LSM6DSL_REG_TAP_THS_6D               0x59
#define LSM6DSL_REG_INT_DUR2                 0x5A
#define LSM6DSL_REG_WAKE_UP_THS              0x5B
#define LSM6DSL_REG_WAKE_UP_DUR              0x5C
#define LSM6DSL_REG_FREE_FALL                0x5D
#define LSM6DSL_REG_MD1_CFG                  0x5E
#define LSM6DSL_REG_MD2_CFG                  0x5F
#define LSM6DSL_REG_MASTER_CMD_CODE          0x60


#define LSM6DSL_REG_SENS_SYNC_SPI_ERROR_CODE 0x61
#define LSM6DSL_REG_OUT_MAG_RAW_X_L          0x66
#define LSM6DSL_REG_OUT_MAG_RAW_X_H          0x67
#define LSM6DSL_REG_OUT_MAG_RAW_Y_L          0x68
#define LSM6DSL_REG_OUT_MAG_RAW_Y_H          0x69
#define LSM6DSL_REG_OUT_MAG_RAW_Z_L          0x6A
#define LSM6DSL_REG_OUT_MAG_RAW_Z_H          0x6B
#define LSM6DSL_REG_X_OFS_USR                0x73
#define LSM6DSL_REG_Y_OFS_USR                0x74
#define LSM6DSL_REG_Z_OFS_USR                0x75


/* FUNC_CFG_ACCESS
 ******************************************************************************/
#define LSM6DSL_MASK_FUNC_CFG_EN               0b10000000
#define LSM6DSL_MASK_FUNC_CFG_EN_B             0b00100000

/******************************************************************************/


/* SENSOR_SYNC_TIME_FRAME
 * Unsigned 8-bit value between 0-10 with 500 ms step size
 * 0 = sensor sync disabled
 * */

#define LSM6DSL_TPH_MAX                        10  // 5000 ms
#define LSM6DSL_TPH_STEP                       500 // ms

/******************************************************************************/


/* SENSOR_SYNC_RES_RATIO
 ******************************************************************************/

// TODO: use enum
#define LSM6DSL_SENSOR_SYNC_RES_RATIO_2_11 0b00
#define LSM6DSL_SENSOR_SYNC_RES_RATIO_2_12 0b01
#define LSM6DSL_SENSOR_SYNC_RES_RATIO_2_13 0b10
#define LSM6DSL_SENSOR_SYNC_RES_RATIO_2_14 0b11

/******************************************************************************/


/* FIFO_CTRL1
 * FIFO control register
 * 
 * uint FTH [7:0] FIFO threshold setting, continues in FIFO_CTRL2
 ******************************************************************************/

#define LSM6DSL_MASK_FTH_7_DOWNTO_0       0b11111111

/******************************************************************************/


/* FIFO_CTRL2
 * FIFO control register
 *
 * bool TIMER_PEDO_FIFO_EN   Store pedometer, step counter and timestamp in FIFO
 * bool TIMER_PEDO_FIFO_DRDY FIFO write mode
 * bool FIFO_TEMP_EN         Store temperature data in FIFO
 * uint FTH [10:8]           Continuation of FIFO threshold setting
 ******************************************************************************/

#define LSM6DSL_MASK_TIMER_PEDO_FIFO_EN   0b10000000
#define LSM6DSL_MASK_TIMER_PEDO_FIFO_DRDY 0b01000000
#define LSM6DSL_MASK_FIFO_TEMP_EN         0b00001000
#define LSM6DSL_MASK_FTH_10_DOWNTO_8      0b00000111

/******************************************************************************/


/* FIFO_CTRL3
 * Controls FIFO decimation for input from gyroscope and accelerometer
 *
 * enum DEC_FIFO_GYRO [2:0] Gyroscope decimation setting
 * enum DEC_FIFO_XL   [2:0] Accelerometer decimation setting
 ******************************************************************************/

#define LSM6DSL_MASK_DEC_FIFO_GYRO        0b00111000
typedef enum
{
    Lsm6dsl_DecFifoGyro_NotInFifo       = 0b00000000,
    Lsm6dsl_DecFifoGyro_NoDec           = 0b00001000,
    Lsm6dsl_DecFifoGyro_2x              = 0b00010000,
    Lsm6dsl_DecFifoGyro_3x              = 0b00011000,
    Lsm6dsl_DecFifoGyro_4x              = 0b00100000,
    Lsm6dsl_DecFifoGyro_8x              = 0b00101000,
    Lsm6dsl_DecFifoGyro_16x             = 0b00110000,
    Lsm6dsl_DecFifoGyro_32x             = 0b00111000,
} Lsm6dsl_DecFifoGyro_e;


#define LSM6DSL_MASK_DEC_FIFO_XL          0b00000111
typedef enum
{
    Lsm6dsl_DecFifoXl_NotInFifo         = 0b00000000,
    Lsm6dsl_DecFifoXl_NoDec             = 0b00000001,
    Lsm6dsl_DecFifoXl_2x                = 0b00000010,
    Lsm6dsl_DecFifoXl_3x                = 0b00000011,
    Lsm6dsl_DecFifoXl_4x                = 0b00000100,
    Lsm6dsl_DecFifoXl_8x                = 0b00000101,
    Lsm6dsl_DecFifoXl_16x               = 0b00000110,
    Lsm6dsl_DecFifoXl_32x               = 0b00000111,
} Lsm6dsl_DecFifoXl_e;

/******************************************************************************/


/* FIFO_CTRL4
 ******************************************************************************/
#define LSM6DSL_MASK_STOP_ON_FTH          0b10000000
#define LSM6DSL_MASK_ONLY_HIGH_DATA       0b01000000
#define LSM6DSL_MASK_DEC_DS4_FIFO         0b00111000
#define LSM6DSL_MASK_DEC_DS3_FIFO         0b00000111
// TODO

/******************************************************************************/


/* FIFO_CTRL5
 * Control register for FIFO
 *
 * enum ODR_FIFO  [3:0] Output data rate selection
 * enum FIFO_MODE [2:0] Mode selection
 ******************************************************************************/

#define LSM6DSL_MASK_ODR_FIFO             0b01111000
typedef enum
{
    Lsm6dsl_OdrFifo_Disabled            = 0b00000000,
    Lsm6dsl_OdrFifo_12_5Hz              = 0b00001000,
    Lsm6dsl_OdrFifo_26Hz                = 0b00010000,
    Lsm6dsl_OdrFifo_52Hz                = 0b00011000,
    Lsm6dsl_OdrFifo_104Hz               = 0b00100000,
    Lsm6dsl_OdrFifo_208Hz               = 0b00101000,
    Lsm6dsl_OdrFifo_416Hz               = 0b00110000,
    Lsm6dsl_OdrFifo_833Hz               = 0b00111000,
    Lsm6dsl_OdrFifo_1_66kHz             = 0b01000000,
    Lsm6dsl_OdrFifo_3_33kHz             = 0b01001000,
    Lsm6dsl_OdrFifo_6_66kHz             = 0b01010000,
} Lsm6dsl_OdrFifo_e;


#define LSM6DSL_MASK_FIFO_MODE            0b00000111
typedef enum
{
    Lsm6dsl_FifoMode_Bypass             = 0b00000000,
    Lsm6dsl_FifoMode_Fifo               = 0b00000001,
    Lsm6dsl_FifoMode_ContinuousToFifo   = 0b00000011,
    Lsm6dsl_FifoMode_BypassToContinuous = 0b00000100,
    Lsm6dsl_FifoMode_Continuous         = 0b00000110,
} Lsm6dsl_FifoMode_e;

/******************************************************************************/


/* DRDY_PULSE_CFG_G
 ******************************************************************************/

#define LSM5DSL_MASK_DRDY_PULSED        0b10000000
#define LSM5DSL_MASK_INT2_WRIST_TILT    0b00000001

/******************************************************************************/


/* INT1_CTRL
 * INT1 pad control register
 *
 * bool INT1_STEP_DETECTOR Interrupt on pedometer step detection
 * bool INT1_SIGN_MOT      Interrupt on significant motion
 * bool INT1_FULL_FLAG     Interrupt on FIFO full
 * bool INT1_FIFO_OVR      Interrupt on FIFO overrun
 * bool INT1_FTH           Interrupt on FIFO threshold
 * bool INT1_BOOT          Interrupt on boot status available
 * bool INT1_DRDY_G        Interrupt on gyroscope data ready
 * bool INT1_DRDY_XL       Interrupt on accelerometer data ready
 ******************************************************************************/

#define LSM6DSL_MASK_INT1_STEP_DETECTOR 0b10000000
#define LSM6DSL_MASK_INT1_SIGN_MOT      0b01000000
#define LSM6DSL_MASK_INT1_FULL_FLAG     0b00100000
#define LSM6DSL_MASK_INT1_FIFO_OVR      0b00010000
#define LSM6DSL_MASK_INT1_FTH           0b00001000
#define LSM6DSL_MASK_INT1_BOOT          0b00000100
#define LSM6DSL_MASK_INT1_DRDY_G        0b00000010
#define LSM6DSL_MASK_INT1_DRDY_XL       0b00000001

/******************************************************************************/


/* INT2_CTRL
 * INT2 pad control register
 *
 * INT2_STEP_DELTA    Interrupt on pedometer step detection delta time
 * INT2_STEP_COUNT_OV Interrupt on step counter overflow
 * INT2_FULL_FLAG     Interrupt on FIFO full
 * INT2_FIFO_OVR      Interrupt on FIFO overrun
 * INT2_FTH           Interrupt on FIFO threshold
 * INT2_DRDY_TEMP     Interrupt on temperature data ready
 * INT2_DRDY_G        Interrupt on gyroscope data ready
 * INT2_DRDY_XL       Interrupt on accelerometer data ready
 ******************************************************************************/

#define LSM6DSL_MASK_INT2_STEP_DELTA    0b10000000
#define LSM6DSL_MASK_INT2_STEP_COUNT_OV 0b01000000
#define LSM6DSL_MASK_INT2_FULL_FLAG     0b00100000
#define LSM6DSL_MASK_INT2_FIFO_OVR      0b00010000
#define LSM6DSL_MASK_INT2_FTH           0b00001000
#define LSM6DSL_MASK_INT2_DRDY_TEMP     0b00000100
#define LSM6DSL_MASK_INT2_DRDY_G        0b00000010
#define LSM6DSL_MASK_INT2_DRDY_XL       0b00000001

/******************************************************************************/


/* CTRL1_XL
 * Control register for accelerometer
 *
 * enum ODR_XL      [3:0] Output data rate and power mode
 * enum FS_XL       [1:0] Full-scale selection
 * bool LPF1_BW_SEL       Digital LPF1 bandwidth selection
 * bool BW0_XL            Analog chain bandwidth selection
 ******************************************************************************/

#define LSM6DSL_MASK_ODR_XL     0b11110000

/* In high performance mode (XL_HM_MODE = 0) */
typedef enum
{
    Lsm6dsl_OdrXlHp_PowerDown = 0b00000000,
    Lsm6dsl_OdrXlHp_12_5Hz    = 0b00010000,
    Lsm6dsl_OdrXlHp_26Hz      = 0b00100000,
    Lsm6dsl_OdrXlHp_52Hz      = 0b00110000,
    Lsm6dsl_OdrXlHp_104Hz     = 0b01000000,
    Lsm6dsl_OdrXlHp_208Hz     = 0b01010000,
    Lsm6dsl_OdrXlHp_416Hz     = 0b01100000,
    Lsm6dsl_OdrXlHp_833Hz     = 0b01110000,
    Lsm6dsl_OdrXlHp_1_66kHz   = 0b10000000,
    Lsm6dsl_OdrXlHp_3_33kHz   = 0b10010000,
    Lsm6dsl_OdrXlHp_6_66kHz   = 0b10100000,
} Lsm6dsl_OdrXlHp_e;

/* In normal or low power mode (XL_HM_MODE = 1) */
typedef enum
{
    Lsm6dsl_OdrXl_PowerDown   = 0b00000000,
    Lsm6dsl_OdrXl_1_6Hz       = 0b10110000,
    Lsm6dsl_OdrXl_12_5Hz      = 0b00010000,
    Lsm6dsl_OdrXl_26Hz        = 0b00100000,
    Lsm6dsl_OdrXl_52Hz        = 0b00110000,
    Lsm6dsl_OdrXl_104Hz       = 0b01000000,
    Lsm6dsl_OdrXl_208Hz       = 0b01010000,
    Lsm6dsl_OdrXl_416Hz       = 0b01100000,
    Lsm6dsl_OdrXl_833Hz       = 0b01110000,
    Lsm6dsl_OdrXl_1_66kHz     = 0b10000000,
    Lsm6dsl_OdrXl_3_33kHz     = 0b10010000,
    Lsm6dsl_OdrXl_6_66kHz     = 0b10100000,
} Lsm6dsl_OdrXl_e;


#define LSM6DSL_MASK_FS_XL      0b00001100
typedef enum {
    Lsm6dsl_FullScaleXl_2g    = 0b00000000,
    Lsm6dsl_FullScaleXl_16g   = 0b00000100,
    Lsm6dsl_FullScaleXl_4g    = 0b00001000,
    Lsm6dsl_FullScaleXl_8g    = 0b00001100,
} Lsm6dsl_FullScaleXl_e;


#define LSM6DSL_MASK_LPF1_BW_SEL 0b00000010
#define LSM6DSL_MASK_BW0_XL      0b00000001

/******************************************************************************/


/* CTRL2_G 
 * Control register for gyroscope
 *
 * enum ODR_G  [3:0] Output data rate and power mode
 * enum FS_G   [1:0] Full-scale selection
 * bool FS_125       Full-scale at 125 dps
 ******************************************************************************/

/* Same values whether in high performance mode (G_HM_MODE = 0)
 * or low power mode (G_HM_MODE = 1) */

#define LSM6DSL_MASK_ODR_G          0b11110000
typedef enum
{
    Lsm6dsl_OdrGyro_PowerDown     = 0b00000000,
    Lsm6dsl_OdrGyro_12_5Hz        = 0b00010000,
    Lsm6dsl_OdrGyro_26Hz          = 0b00100000,
    Lsm6dsl_OdrGyro_52Hz          = 0b00110000,
    Lsm6dsl_OdrGyro_104Hz         = 0b01000000,
    Lsm6dsl_OdrGyro_208Hz         = 0b01010000,
    Lsm6dsl_OdrGyro_416Hz         = 0b01100000,
    Lsm6dsl_OdrGyro_833Hz         = 0b01110000,
    Lsm6dsl_OdrGyro_1_66kHz       = 0b10000000,
    Lsm6dsl_OdrGyro_3_33kHz       = 0b10010000,
    Lsm6dsl_OdrGyro_6_66kHz       = 0b10100000,
} Lsm6dsl_OdrGyro_e;


#define LSM6DSL_MASK_FS_G           0b00001100
#define LSM6DSL_MASK_FS_125         0b00000010
typedef enum {
    Lsm6dsl_FullScaleGyro_125dps  = 0b00000010,
    Lsm6dsl_FullScaleGyro_250dps  = 0b00000000,
    Lsm6dsl_FullScaleGyro_500dps  = 0b00000100,
    Lsm6dsl_FullScaleGyro_1000dps = 0b00001000,
    Lsm6dsl_FullScaleGyro_2000dps = 0b00001100,
} Lsm6dsl_FullScaleGyro_e;




/******************************************************************************/


/* CTRL3_C
 * Control register 3
 *
 * bool BOOT      Reboot memory content
 * bool BDU       Block Data Update
 * bool H_LACTIVE Interrupt activation level
 * bool PP_OD     Push-pull/open-drain selection on INT1 and INT2 pads
 * bool SIM       SPI mode selection
 * bool IF_INC    Register address auto increment
 * bool BLE       Big/Little Endian selection
 * bool SW_RESET  Software reset
 ******************************************************************************/

#define LSM6DSL_MASK_BOOT      0b10000000
#define LSM6DSL_MASK_BDU       0b01000000
#define LSM6DSL_MASK_H_LACTIVE 0b00100000
#define LSM6DSL_MASK_PP_OD     0b00010000
#define LSM6DSL_MASK_SIM       0b00001000
#define LSM6DSL_MASK_IF_INC    0b00000100
#define LSM6DSL_MASK_BLE       0b00000010
#define LSM6DSL_MASK_SW_RESET  0b00000001

/******************************************************************************/


/* CTRL4_C
 ******************************************************************************/

#define LSM6DSL_MASK_DEN_XL_EN                 0b10000000
#define LSM6DSL_MASK_SLEEP                     0b01000000
#define LSM6DSL_MASK_INT2_ON_INT1              0b00100000
#define LSM6DSL_MASK_DEN_DRDY_INT1             0b00010000
#define LSM6DSL_MASK_DRDY_MASK                 0b00001000
#define LSM6DSL_MASK_I2C_DISABLE               0b00000100
#define LSM6DSL_MASK_LPF1_SEL_G                0b00000010

/******************************************************************************/


/* FIFO_STATUS1
 * FIFO status register
 *
 * uint DIFF_FIFO       [7:0] Number of unread 16-bit words (continues in FIFO_STATUS2)
 ******************************************************************************/

#define LSM6DSL_MASK_DIFF_FIFO_7_DOWNTO_0  0b11111111

/******************************************************************************/


/* FIFO_STATUS2
 * FIFO status register
 *
 * bool WaterM                 Watermark status
 * bool OVER_RUN               Overrun status
 * bool FIFO_FULL_SMART        Smart full status
 * bool FIFO_EMPTY             Empty bit
 * uint DIFF_FIFO       [10:8] Number of unread 16-bit words (begins in FIFO_STATUS1)
 ******************************************************************************/

#define LSM6DSL_MASK_WATERM                0b10000000
#define LSM6DSL_MASK_OVER_RUN              0b01000000
#define LSM6DSL_MASK_FIFO_FULL_SMART       0b00100000
#define LSM6DSL_MASK_FIFO_EMPTY            0b00010000
#define LSM6DSL_MASK_DIFF_FIFO_10_DOWNTO_8 0b00000111

/******************************************************************************/

// TODO: FIFO_STATUS3 and 4

#endif // REGISTER_DEFS_H

