/**
 * @file robot_command.c
 * @brief Custom binary protocol implementation for robot command encoding/decoding
 */

#include "robot_command.h"
#include <string.h>

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

    if (cmd->action_type > 5) {
        return false;
    }

    if (cmd->robot_id < 1 || cmd->robot_id > 7) {
        return false;
    }

    memset(buf, 0, ROBOT_COMMAND_SIZE);

    buf[0] = cmd->action_type;
    buf[1] = cmd->robot_id;
    write_int16_le(&buf[2], cmd->kick_speed);
    write_int16_le(&buf[4], cmd->pos_x);
    write_int16_le(&buf[6], cmd->pos_y);
    write_int16_le(&buf[8], cmd->dest_x);
    write_int16_le(&buf[10], cmd->dest_y);
    write_int16_le(&buf[12], cmd->direction_x);
    write_int16_le(&buf[14], cmd->direction_y);
    write_int16_le(&buf[16], cmd->angular_vel);
    write_int16_le(&buf[18], cmd->angle);

    return true;
}

bool robot_command_decode(const uint8_t* buf, RobotCommand* cmd)
{
    if (buf == NULL || cmd == NULL) {
        return false;
    }

    cmd->action_type = buf[0];
    cmd->robot_id = buf[1];
    cmd->kick_speed = read_int16_le(&buf[2]);
    cmd->pos_x = read_int16_le(&buf[4]);
    cmd->pos_y = read_int16_le(&buf[6]);
    cmd->dest_x = read_int16_le(&buf[8]);
    cmd->dest_y = read_int16_le(&buf[10]);
    cmd->direction_x = read_int16_le(&buf[12]);
    cmd->direction_y = read_int16_le(&buf[14]);
    cmd->angular_vel = read_int16_le(&buf[16]);
    cmd->angle = read_int16_le(&buf[18]);

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
    LOG_DEBUG("  Angle: %d (%.3f rad)\r\n", cmd->angle, cmd->angle / 1000.0);
}
#else
void robot_command_print(const RobotCommand* cmd)
{
    (void)cmd; // Suppress unused parameter warning
}
#endif
