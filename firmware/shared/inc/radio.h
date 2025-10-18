#ifndef FW_SM_RADIO_H
#define FW_SM_RADIO_H

#define MAX_ROBOT_COUNT 16
#define ROBOT_ID_BROADCAST 127

#define CONTROLLER_ADDR {2, 255, 255, 255, 255}
#define ROBOT_ADDR(id) {id, 255, 255, 255, 127}

#define CONNECT_MAGIC 0x4df84279
#define CONNECT_MAGIC_BYTES 0x4d, 0xf8, 0x42, 0x79
#define CONNECT_MAGIC_READ(b) ((b[0] << 24) | (b[1] << 16) | (b[2] << 8) | b[3])

#define MESSAGE_ID_PING 0
#define MESSAGE_ID_COMMAND 1
#define MESSAGE_ID_VISION 2

#endif // FW_SM_RADIO_H
