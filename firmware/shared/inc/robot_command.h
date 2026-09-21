/**
 * @file robot_command.h
 * @brief Custom binary protocol for robot command encoding/decoding
 *
 * Protocol: 32 bytes total (fills the NRF24 32-byte limit)
 *
 * Byte Layout:
 * [0]     : Action Type (uint8) - enum 0-5 (KICK, STOP, MOVE_TO, INIT, MOVE, ROTATE)
 * [1]     : Robot ID (uint8) - 0-15
 * [2-3]   : Kick Speed (int16, little-endian) - mm/s
 * [4-5]   : Pos X (int16, little-endian) - mm
 * [6-7]   : Pos Y (int16, little-endian) - mm
 * [8-9]   : Dest X (int16, little-endian) - mm
 * [10-11] : Dest Y (int16, little-endian) - mm
 * [12-13] : Direction X (int16, little-endian) - raw value
 * [14-15] : Direction Y (int16, little-endian) - raw value
 * [16-17] : Angular Vel (int16, little-endian) - degrees/sec
 * [18-19] : Angle (int16, little-endian) - radians * 1000 (milliradians)
 *
 * Total: 20 bytes of fields, 12 bytes reserved (zero-filled)
 */

#ifndef ROBOT_COMMAND_H
#define ROBOT_COMMAND_H

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

#define ROBOT_COMMAND_SIZE 32
#define ACTION_TYPE_KICK 0
#define ACTION_TYPE_STOP 1
#define ACTION_TYPE_MOVE_TO 2
#define ACTION_TYPE_INIT 3
#define ACTION_TYPE_MOVE 4
#define ACTION_TYPE_ROTATE 5

/**
 * @struct RobotCommand
 * @brief Decoded robot command structure
 */
typedef struct {
    uint8_t action_type;
    uint8_t robot_id;
    int16_t kick_speed;
    int16_t pos_x;
    int16_t pos_y;
    int16_t dest_x;
    int16_t dest_y;
    int16_t direction_x;
    int16_t direction_y;
    int16_t angular_vel;
    int16_t angle;
} RobotCommand;

/**
 * @brief Encode a RobotCommand to 32-byte binary buffer
 *
 * @param cmd Pointer to RobotCommand struct to encode
 * @param buf Output buffer (must be at least ROBOT_COMMAND_SIZE bytes)
 * @return true if encoding succeeded, false on error
 *
 */
bool robot_command_encode(const RobotCommand* cmd, uint8_t* buf);

/**
 * @brief Decode a 32-byte binary buffer to RobotCommand
 *
 * @param buf Input buffer (must be exactly ROBOT_COMMAND_SIZE bytes)
 * @param cmd Pointer to RobotCommand struct to populate
 * @return true if decoding succeeded, false on error (e.g., invalid buffer size)
 */
bool robot_command_decode(const uint8_t* buf, RobotCommand* cmd);

void robot_command_print(const RobotCommand* cmd);

#ifdef __cplusplus
}
#endif

#endif // ROBOT_COMMAND_H
