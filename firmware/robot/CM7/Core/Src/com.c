#include "com.h"

/* Private includes */
#include "log.h"
#include "main.h"
#include "nav.h"
#include <nrf24l01.h>
#include <nrf_helper_defines.h>
#include <robot_action.pb-c.h>
#include <stdint.h>
#include <stdio.h>

/* Private defines */
#define MSG_PING 0
#define MSG_ACTION 1
#define CONNECT_MAGIC 0x4d, 0xf8, 0x42, 0x79
#define CONTROLLER_ADDR {2, 255, 255, 255, 255}
#define ROBOT_ADDR(id) {1, 255, 255, id, 255}
#define RF_CONTROLLER_PIPE 0

/* Private functions declarations */
static void parse_controller_packet(uint8_t* payload, uint8_t len);
static char* ping_ack_to_string(uint8_t ack);

/* Private variables */
static LOG_Module internal_log_mod;
static int nRFon = 0;
static volatile uint8_t ping_ack;
volatile uint8_t last_rec_id = 0xff;
volatile uint32_t last_rec_time = 0;

/*
 * Public functions implementations
 */

void COM_Init(SPI_HandleTypeDef* hspi, uint8_t* nrf_available)
{
    LOG_InitModule(&internal_log_mod, "COM", LOG_LEVEL_INFO, 0);

    // Initialize and enter standby-I mode
    NRF_Init(hspi, NRF_CSN_GPIO_Port, NRF_CSN_Pin, NRF_CE_GPIO_Port, NRF_CE_Pin);
    if (NRF_VerifySPI() != NRF_OK) {
        LOG_ERROR("Couldn't verify nRF24 SPI communication...\r\n");
        return;
    } else {
        *nrf_available = 1;
    }

    nRFon = 1;

    // Resets all registers but keeps the device in standby-I mode
    NRF_Reset();

    // Setup the nRF registers for rx purpose
    COM_RF_Init();
}

void COM_RF_Init()
{
    NRF_Reset();

    uint8_t address[5] = ROBOT_ADDR(COM_Get_ID());
    NRF_WriteRegister(NRF_REG_RX_ADDR_P0, address, 5);

    // Channel 2.525 GHz
    NRF_WriteRegisterByte(NRF_REG_RF_CH, 0x7d);

    // No retransmissions
    NRF_WriteRegisterByte(NRF_REG_SETUP_RETR, 0x00);

    // No auto-acknowledgement
    NRF_WriteRegisterByte(NRF_REG_EN_AA, 0x00);

    // Channel rf 2525
    NRF_WriteRegisterByte(NRF_REG_RF_CH, 0x7d);

    // Dynamic data length
    NRF_WriteRegisterByte(NRF_REG_DYNPD, 0x01);
    NRF_WriteRegisterByte(NRF_REG_FEATURE, 0x04);

    // Enter RX mode
    NRF_EnterMode(NRF_MODE_RX);

    LOG_DEBUG("Initialized RF...\r\n");
}

void COM_RF_HandleIRQ()
{
    uint8_t status = NRF_ReadStatus();

    if (status & STATUS_MASK_RX_DR) {
        // Received message
        const uint8_t pipe = (NRF_ReadStatus() & STATUS_MASK_RX_P_NO) >> 1;
        COM_RF_Receive(pipe);
    }

    if (status & STATUS_MASK_TX_DS) {
        // ACK received
        NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_TX_DS);
    }

    if (status & STATUS_MASK_MAX_RT) {
        // Max retries while sending.
        NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_MAX_RT);
    }

    if (!status) {
        LOG_ERROR("RF_HandleIRQ called but status empty...\r\n");
    }
}

void COM_RF_Receive(uint8_t pipe)
{
    HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_SET);
    uint8_t len = 0;
    NRF_Status status;
    status = NRF_SendReadCommand(NRF_CMD_R_RX_PL_WID, &len, 1);
    if (status != NRF_OK) {
        LOG_ERROR("Couldn't read length of RF packet...\r\n");
    }

    uint8_t payload[len];
    status = NRF_ReadPayload(payload, len);
    if (status != NRF_OK) {
        LOG_ERROR("Couldn't read RF packet payload...\r\n");
    }

    NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_RX_DR);

    // Timestamp of last received message
    last_rec_time = HAL_GetTick();

    parse_controller_packet(payload + 1, len - 1);

    HAL_GPIO_WritePin(LED_YELLOW_GPIO_Port, LED_YELLOW_Pin, GPIO_PIN_RESET);
    NRF_SendCommand(NRF_CMD_FLUSH_RX);
}

void COM_RF_PrintInfo(void)
{
    uint8_t ret = NRF_ReadStatus();

    if (!nRFon) {
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
}

void COM_RF_Reset()
{
    // Initialize and enter standby-I mode
    if (NRF_VerifySPI() != NRF_OK) {
        LOG_ERROR("Couldn't verify nRF24 SPI communication...\r\n");
        return;
    } else {
        LOG_INFO("Could communicate with nRF24.\r\n");
    }

    // Resets all registers but keeps the device in standby-I mode
    NRF_Reset();

    LOG_INFO("nRF24 resetted.\r\n");

    COM_RF_Init();
}

bool COM_Update()
{
    if (HAL_GetTick() - last_rec_time < COM_BASESTATION_TIMEOUT_MS) {
        return true;
    }
    if (last_rec_id != 0xfe) {
        last_rec_id = 0xfe;
        LOG_INFO("Basestation connection timed out\r\n");
    }
    return false;
}

void COM_RF_Send(uint8_t* msg, uint8_t length)
{
    NRF_EnterMode(NRF_MODE_STANDBY1);

    if (NRF_TransmitAndWait(msg, length) != NRF_OK) {
        LOG_ERROR("Failed sending message...\r\n");
    }

    NRF_EnterMode(NRF_MODE_RX);
}

uint8_t COM_Get_ID()
{
    uint32_t w0 = HAL_GetUIDw0();
    uint32_t w1 = HAL_GetUIDw1();
    uint32_t w2 = HAL_GetUIDw2();

    if (w0 == 2162755 && w1 == 875712788 && w2 == 926168633) {
        return 3;
    }
    if (w0 == 3932237 && w1 == 892490001 && w2 == 842217265) {
        return 1;
    }
    if (w0 == 2687023 && w1 == 858935561 && w2 == 808727605) {
        return 1;
    }
    if (w0 == 3080253 && w1 == 892490001 && w2 == 842217265) {
        return 2;
    }
    if (w0 == 1572912 && w1 == 892490001 && w2 == 842217265) {
        return 3;
    }
    if (w0 == 2490418 && w1 == 858935561 && w2 == 808727605) {
        return 3;
    }
    if (w0 == 4259883 && w1 == 892490001 && w2 == 842217265) {
        return 4;
    }
    if (w0 == 4522020 && w1 == 892490001 && w2 == 842217265) {
        return 5;
    }
    if (w0 == 4522048 && w1 == 892490001 && w2 == 842217265) {
        return 6;
    }
    if (w0 == 3801132 && w1 == 892490001 && w2 == 842217265) {
        return 7;
    }
    if (w0 == 2293800 && w1 == 858935561 && w2 == 808727605) {
        return 8;
    }
    if (w0 == 2162755 && w1 == 875712788 && w2 == 926168633) {
        return 0;
    }
    LOG_ERROR("Failed ID lookup for robot ID: %d %d %d\r\n", w0, w1, w2);
    return 255;
}

/*
 * Private function implementations
 */

static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    Command* cmd = NULL;
    cmd = command__unpack(NULL, len, payload);

    if (!cmd) {
        LOG_WARNING("Decoding PB failed\r\n");
    } else {
        NAV_HandleCommand(cmd);
    }

    protobuf_c_message_free_unpacked((ProtobufCMessage*)cmd, NULL);
}

static char* ping_ack_to_string(uint8_t ack)
{
    switch (ack) {
    case 0:
        return "Unknown transmission error";
    case 1:
        return "ACK received";
    case 2:
        return "Max retries reached while sending";
    default:
        return "Invalid ACK code";
    }
}
