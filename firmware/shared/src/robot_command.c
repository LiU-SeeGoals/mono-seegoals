/**
 * @file robot_command.c
 * @brief Custom binary protocol implementation for robot command encoding/decoding
 */

#include "robot_command.h"
#include <string.h>

// Helper functions for little-endian conversions
static inline void write_int16_le(uint8_t* buf, int16_t val) {
    buf[0] = (uint8_t)(val & 0xFF);
    buf[1] = (uint8_t)((val >> 8) & 0xFF);
}

static inline int16_t read_int16_le(const uint8_t* buf) {
    return (int16_t)((buf[0]) | ((int16_t)buf[1] << 8));
}

static inline void write_int32_le(uint8_t* buf, int32_t val) {
    buf[0] = (uint8_t)(val & 0xFF);
    buf[1] = (uint8_t)((val >> 8) & 0xFF);
    buf[2] = (uint8_t)((val >> 16) & 0xFF);
    buf[3] = (uint8_t)((val >> 24) & 0xFF);
}

static inline int32_t read_int32_le(const uint8_t* buf) {
    return (int32_t)((buf[0]) |
                     ((int32_t)buf[1] << 8) |
                     ((int32_t)buf[2] << 16) |
                     ((int32_t)buf[3] << 24));
}

bool robot_command_encode(const RobotCommand* cmd, uint8_t* buf)
{
    if (cmd == NULL || buf == NULL) {
        return false;
    }

    // Validate action type
    if (cmd->action_type > 5) {
        return false;
    }

    // Validate robot ID
    if (cmd->robot_id < 1 || cmd->robot_id > 7) {
        return false;
    }

    // Initialize buffer to zeros
    memset(buf, 0, ROBOT_COMMAND_SIZE);

    // [0] Action Type
    buf[0] = cmd->action_type;

    // [1] Robot ID
    buf[1] = cmd->robot_id;

    // [2-3] Kick Speed (int16, little-endian)
    write_int16_le(&buf[2], cmd->kick_speed);

    // [4-5] Pos X (int16, little-endian)
    write_int16_le(&buf[4], cmd->pos_x);

    // [6-7] Pos Y (int16, little-endian)
    write_int16_le(&buf[6], cmd->pos_y);

    // [8-9] Dest X (int16, little-endian)
    write_int16_le(&buf[8], cmd->dest_x);

    // [10-11] Dest Y (int16, little-endian)
    write_int16_le(&buf[10], cmd->dest_y);

    // [12-13] Direction X (int16, little-endian)
    write_int16_le(&buf[12], cmd->direction_x);

    // [14-15] Direction Y (int16, little-endian)
    write_int16_le(&buf[14], cmd->direction_y);

    // [16-17] Angular Vel (int16, little-endian)
    write_int16_le(&buf[16], cmd->angular_vel);

    // [18-19] Orientation W (int16, little-endian, [0, 3600) in tenths of degrees)
    int16_t orient_w = cmd->orientation_w;
    if (orient_w < 0) orient_w = 0;
    if (orient_w >= 3600) orient_w = 3599;
    write_int16_le(&buf[18], orient_w);

    return true;
}

bool robot_command_decode(const uint8_t* buf, RobotCommand* cmd)
{
    if (buf == NULL || cmd == NULL) {
        return false;
    }

    // Note: In C, we can't validate buffer size here without additional parameter
    // The caller is responsible for ensuring buf is ROBOT_COMMAND_SIZE bytes

    // [0] Action Type
    cmd->action_type = buf[0];

    // [1] Robot ID
    cmd->robot_id = buf[1];

    // [2-3] Kick Speed (int16, little-endian)
    cmd->kick_speed = read_int16_le(&buf[2]);

    // [4-5] Pos X (int16, little-endian)
    cmd->pos_x = read_int16_le(&buf[4]);

    // [6-7] Pos Y (int16, little-endian)
    cmd->pos_y = read_int16_le(&buf[6]);

    // [8-9] Dest X (int16, little-endian)
    cmd->dest_x = read_int16_le(&buf[8]);

    // [10-11] Dest Y (int16, little-endian)
    cmd->dest_y = read_int16_le(&buf[10]);

    // [12-13] Direction X (int16, little-endian)
    cmd->direction_x = read_int16_le(&buf[12]);

    // [14-15] Direction Y (int16, little-endian)
    cmd->direction_y = read_int16_le(&buf[14]);

    // [16-17] Angular Vel (int16, little-endian)
    cmd->angular_vel = read_int16_le(&buf[16]);

    // [18-19] Orientation W (int16, little-endian)
    cmd->orientation_w = read_int16_le(&buf[18]);

    return true;
}

#ifdef LOG_DEBUG
void robot_command_print(const RobotCommand* cmd)
{
    if (cmd == NULL) {
        LOG_ERROR("robot_command_print: cmd is NULL\r\n");
        return;
    }

    LOG_DEBUG("RobotCommand:\r\n");
    LOG_DEBUG("  Action Type: %d\r\n", cmd->action_type);
    LOG_DEBUG("  Robot ID: %d\r\n", cmd->robot_id);
    LOG_DEBUG("  Kick Speed: %d\r\n", cmd->kick_speed);
    LOG_DEBUG("  Pos: (%d, %d)\r\n", cmd->pos_x, cmd->pos_y);
    LOG_DEBUG("  Dest: (%d, %d)\r\n", cmd->dest_x, cmd->dest_y);
    LOG_DEBUG("  Direction: (%d, %d)\r\n", cmd->direction_x, cmd->direction_y);
    LOG_DEBUG("  Angular Vel: %d\r\n", cmd->angular_vel);
    LOG_DEBUG("  Orientation W: %d (%.1f degrees)\r\n", cmd->orientation_w, cmd->orientation_w / 10.0);
}
#else
void robot_command_print(const RobotCommand* cmd)
{
    (void)cmd; // Suppress unused parameter warning
}
#endif
