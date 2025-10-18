#ifndef __APP_HANDLE_PACKET_H__
#define __APP_HANDLE_PACKET_H__
#include <nx_api.h>

enum _PACKET_TYPE
{
    SSL_WRAPPER,
    ROBOT_COMMAND
};
typedef enum _PACKET_TYPE PACKET_TYPE;

extern UINT parse_packet(NX_PACKET* packet, PACKET_TYPE packet_type);

#endif // __APP_HANDLE_PACKET_H__s
