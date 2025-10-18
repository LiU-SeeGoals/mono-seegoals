/* Private includes */
#include "com.h"
#include <log.h>
#include <nrf24l01.h>
#include <nrf_helper_defines.h>
#include <parsed_vision.pb-c.h>
#include <protobuf-c.h>
#include <robot_action.pb-c.h>

/* Private defines */
#define PIPE_CONTROLLER 0
#define PIPE_VISION 1
#define CONNECT_MAGIC 0x4d, 0xf8, 0x42, 0x79
#define MAX_NO_RESPONSES 200

/* Private structs */
typedef enum RobotComStatus
{
    COMSTAT_INIT,
    COMSTAT_CONNECT,
    COMSTAT_OK,
    COMSTAT_FAIL,
    COMSTAT_DISCONNECTED,
    COMSTAT_INVALID_PACKET,
} RobotComStatus;

/* Private variables */
TX_SEMAPHORE semaphore;
static LOG_Module internal_log_mod;

/*
 * Public functions implementations
 */

void COM_RF_Init(SPI_HandleTypeDef* hspi)
{
    LOG_InitModule(&internal_log_mod, "RF", LOG_LEVEL_TRACE, 0);

    NRF_Init(hspi, NRF_CSN_GPIO_Port, NRF_CSN_Pin, NRF_CE_GPIO_Port, NRF_CE_Pin);
    if (NRF_VerifySPI() != NRF_OK) {
        LOG_WARNING("Couldn't verify nRF24 SPI...\r\n");
        Error_Handler();
    }

    if (tx_semaphore_create(&semaphore, "NRF-semaphore", 1) != TX_SUCCESS) {
        LOG_ERROR("Failed creating NRF-semaphore\r\n");
    }

    NRF_Reset();

    uint8_t address[5] = {1, 255, 255, 1, 255};
    NRF_WriteRegister(NRF_REG_TX_ADDR, address, 5);
    NRF_WriteRegister(NRF_REG_RX_ADDR_P0, address, 5);

    // Channel 2.525 GHz
    NRF_WriteRegisterByte(NRF_REG_RF_CH, 0x7d);

    // No retransmissions
    NRF_WriteRegisterByte(NRF_REG_SETUP_RETR, 0x00);

    // No auto-acknowledgement
    NRF_WriteRegisterByte(NRF_REG_EN_AA, 0x00);

    // Dynamic data length
    NRF_WriteRegisterByte(NRF_REG_DYNPD, 0x01);
    NRF_WriteRegisterByte(NRF_REG_FEATURE, 0x04);

    //
    // NRF_WriteRegisterByte(NRF_REG_RF_SETUP, 0x00);

    LOG_INFO("Initialized...\r\n");
}

void COM_Test()
{
    LOG_INFO("Sending data...\r\n");

    uint8_t msg[10] = "HelloWorld";
    uint8_t addr[5] = ROBOT_ACTION_ADDR(1);
    NRF_WriteRegister(NRF_REG_TX_ADDR, addr, 5);
    NRF_EnterMode(NRF_MODE_STANDBY1);
    NRF_Status ret = NRF_Transmit(msg, 10);
    NRF_SendCommand(NRF_CMD_FLUSH_TX);
    switch (ret) {
    case NRF_MAX_RT:
        LOG_INFO("Max retransmissions reached, RX device not responding?\r\n");
        break;
    case NRF_SPI_BUSY:
    case NRF_SPI_TIMEOUT:
    case NRF_SPI_ERROR:
        LOG_INFO("SPI error, pins correctly connected?\r\n");
        break;
    case NRF_OK:
        LOG_INFO("Data sent...\r\n");
        break;
    case NRF_ERROR:
        LOG_INFO("Error when sending.\r\n");
        break;
    case NRF_BAD_TRANSITION:
        LOG_INFO("Bad transition.\r\n");
        break;
    }
}

void COM_RF_HandleIRQ()
{
    uint8_t status = NRF_ReadStatus();

    if (status & STATUS_MASK_MAX_RT) {
        // Max retries while sending.
        NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_MAX_RT);
    }

    if (status & STATUS_MASK_TX_DS) {
        // ACK received
        NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_TX_DS);
    }

    if (status & STATUS_MASK_RX_DR) {
        // Received packet
        uint8_t pipe = (status & STATUS_MASK_RX_P_NO) >> 1;
        COM_RF_Receive(pipe);
    }
}

void COM_RF_Transmit(uint8_t robot, uint8_t* data, uint8_t len)
{
    tx_semaphore_get(&semaphore, TX_WAIT_FOREVER);

    uint8_t addr[5] = ROBOT_ACTION_ADDR(robot);
    NRF_WriteRegister(NRF_REG_TX_ADDR, addr, 5);
    NRF_EnterMode(NRF_MODE_STANDBY1);
    NRF_Transmit(data, len);
    NRF_SendCommand(NRF_CMD_FLUSH_TX);

    tx_semaphore_put(&semaphore);
}

void COM_RF_PrintInfo()
{
    uint8_t ret = NRF_ReadStatus();

    if (!ret) {
        LOG_INFO("nRF24 not running...\r\n");
        return;
    }

    LOG_INFO("Status register: %02X\r\n", ret);
    LOG_INFO("TX_FULL:  %1X\r\n", ret & (1 << 0));
    LOG_INFO("RX_P_NO:  %1X\r\n", (ret & (0x3 << 1)) >> 1);
    LOG_INFO("MAX_RT:   %1X\r\n", (ret & (1 << 4)) >> 4);
    LOG_INFO("TX_DS:    %1X\r\n", (ret & (1 << 5)) >> 5);
    LOG_INFO("RX_DR:    %1X\r\n", (ret & (1 << 6)) >> 6);
    LOG_INFO("\r\n");

    ret = NRF_ReadRegisterByte(NRF_REG_FIFO_STATUS);
    LOG_INFO("FIFO status register: %02X\r\n", ret);
    LOG_INFO("RX_EMPTY:   %2X\r\n", ret & (1 << 0));
    LOG_INFO("RX_FULL:    %2X\r\n", (ret & (1 << 1)) >> 1);
    LOG_INFO("TX_EMPTY:   %2X\r\n", (ret & (1 << 4)) >> 4);
    LOG_INFO("TX_FULL:    %2X\r\n", (ret & (1 << 5)) >> 5);
    LOG_INFO("TX_REUSE:   %2X\r\n", (ret & (1 << 6)) >> 6);
    LOG_INFO("\r\n");

    ret = NRF_ReadRegisterByte(NRF_REG_CONFIG);
    LOG_INFO("Config register: %02X\r\n", ret);
    LOG_INFO("PRIM_RX:      %1X\r\n", ret & (1 << 0));
    LOG_INFO("PWR_UP:       %1X\r\n", ret & (1 << 1));
    LOG_INFO("CRCO:         %1X\r\n", ret & (1 << 2));
    LOG_INFO("EN_CRC:       %1X\r\n", ret & (1 << 3));
    LOG_INFO("MASK_MAX_RT:  %1X\r\n", ret & (1 << 4));
    LOG_INFO("MASK_TX_DS:   %1X\r\n", ret & (1 << 5));
    LOG_INFO("MASK_RX_DR:   %1X\r\n", ret & (1 << 6));
    LOG_INFO("\r\n");

    LOG_INFO("\r\n");
}

void COM_RF_Receive(uint8_t pipe)
{
    uint8_t len = 0;
    NRF_SendReadCommand(NRF_CMD_R_RX_PL_WID, &len, 1);

    uint8_t payload[len];
    NRF_ReadPayload(payload, len);

    if (len > 2 && payload[0] == 0x57 && payload[1] == 0x75) {
        // We received a LOG_BASESTATION(...)
        LOG_INFO("ROBOT %d: %s\r\n", payload[2], payload + 3);
    } else {
        LOG_INFO("Received unknown RF package\r\n");
    }

    NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_TX_DS);
}

UINT COM_ParsePacket(NX_PACKET* packet, PACKET_TYPE packet_type)
{
    UINT ret = NX_SUCCESS;

    switch (packet_type) {
    case ROBOT_COMMAND: {
        int length = packet->nx_packet_append_ptr - packet->nx_packet_prepend_ptr;

        if (length > 32) {
            LOG_ERROR("Robot command packet over 32 bytes (%d bytes)\r\n", length);
            ret = NX_INVALID_PACKET;
            return ret;
        }

        Command* command = NULL;
        command = command__unpack(NULL, length, packet->nx_packet_prepend_ptr);
        if (command == NULL) {
            LOG_ERROR("Invalid ethernet packet\r\n");
            return NX_INVALID_PACKET;
        }

        const ProtobufCEnumValue* enum_value = protobuf_c_enum_descriptor_get_value(&action_type__descriptor, command->command_id);

        uint8_t data[32];
        data[0] = 1;
        memcpy(data + 1, packet->nx_packet_prepend_ptr, length);

        COM_RF_Transmit(command->robot_id, data, length + 1);

        protobuf_c_message_free_unpacked(&command->base, NULL);
    } break;
    default:
        LOG_INFO("Unknown packet type: %d\r\n", packet_type);
        break;
    }

    if (ret != NX_SUCCESS) {
        LOG_WARNING("Failed to parse UDP packet\r\n");
    }

    return ret;
}
