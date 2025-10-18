#include "nrf24l01.h"
#include <stdio.h>

/*
 * Local defines
 */

// Config bits
#define CFG_BIT_EN_CRC 0x3
#define CFG_BIT_PWR_UP 0x1
#define CFG_BIT_PRIM_RX 0x0

// Status bits
#define STATUS_BIT_RX_DR 0x6
#define STATUS_BIT_TX_DS 0x5
#define STATUS_BIT_MAX_RT 0x4
#define STATUS_BIT_RX_P_NO 0x1 // consists of bits 1-3
#define STATUS_BIT_TX_FULL 0x0

// FIFO status bits
#define FIFO_STATUS_BIT_TX_REUSE 0x6
#define FIFO_STATUS_BIT_TX_FULL 0x5
#define FIFO_STATUS_BIT_TX_EMPTY 0x4
#define FIFO_STATUS_BIT_RX_FULL 0x1
#define FIFO_STATUS_BIT_RX_EMPTY 0x0

/* Local globals */
SPI_HandleTypeDef* HSPI;
GPIO_TypeDef* NRF_CSN_Port;
uint16_t NRF_CSN_Pin;
GPIO_TypeDef* NRF_CE_Port;
uint16_t NRF_CE_Pin;
uint32_t CPU_Freq = 0x00;
int current_mode = NRF_MODE_POWERDOWN;

/*
 * Private functions
 */

void csn_set() { HAL_GPIO_WritePin(NRF_CSN_Port, NRF_CSN_Pin, GPIO_PIN_SET); }

void csn_reset() { HAL_GPIO_WritePin(NRF_CSN_Port, NRF_CSN_Pin, GPIO_PIN_RESET); }

void ce_set() { HAL_GPIO_WritePin(NRF_CE_Port, NRF_CE_Pin, GPIO_PIN_SET); }

void ce_reset() { HAL_GPIO_WritePin(NRF_CE_Port, NRF_CE_Pin, GPIO_PIN_RESET); }

uint8_t read_csn() { return HAL_GPIO_ReadPin(NRF_CSN_Port, NRF_CSN_Pin); }

uint8_t read_ce() { return HAL_GPIO_ReadPin(NRF_CE_Port, NRF_CE_Pin); }

void wait(uint64_t us)
{
    uint32_t volatile cycles = CPU_Freq * us / 1000000;
    uint32_t volatile current = 0;
    while (current <= cycles) {
        current++;
    }
}

/*
 *
 * Main functions
 *
 */

NRF_Status NRF_Init(SPI_HandleTypeDef* handle, GPIO_TypeDef* PortCSN, uint16_t PinCSN, GPIO_TypeDef* PortCE, uint16_t PinCE)
{
    HSPI = handle;
    NRF_CSN_Port = PortCSN;
    NRF_CSN_Pin = PinCSN;
    NRF_CE_Port = PortCE;
    NRF_CE_Pin = PinCE;

    CPU_Freq = HAL_RCC_GetSysClockFreq();
    if (CPU_Freq == 0x00) {
        return NRF_ERROR;
    }

    // Make sure CSN is pulled high
    csn_set();

    // Takes ~100ms from power on to start up
    HAL_Delay(100);

    return NRF_EnterMode(NRF_MODE_STANDBY1);
}

NRF_Status NRF_SendCommand(uint8_t cmd)
{
    NRF_Status ret = NRF_OK;
    uint8_t status;

    csn_reset();
    ret = (NRF_Status)HAL_SPI_TransmitReceive(HSPI, &cmd, &status, 1, NRF_SPI_TIMEOUT_DUR);
    if (ret != NRF_OK) {
        return ret;
    }
    csn_set();

    return ret;
}

NRF_Status NRF_SendWriteCommand(uint8_t cmd, uint8_t* write, uint8_t length)
{
    NRF_Status ret = NRF_OK;
    uint8_t status;

    csn_reset();
    ret = (NRF_Status)HAL_SPI_TransmitReceive(HSPI, &cmd, &status, 1, NRF_SPI_TIMEOUT_DUR);
    if (ret != NRF_OK) {
        return ret;
    }
    ret = (NRF_Status)HAL_SPI_Transmit(HSPI, write, length, NRF_SPI_TIMEOUT_DUR);
    if (ret != NRF_OK) {
        return ret;
    }
    csn_set();

    return ret;
}

NRF_Status NRF_SendReadCommand(uint8_t cmd, uint8_t* read, uint8_t length)
{
    NRF_Status ret = NRF_OK;
    uint8_t status;

    csn_reset();
    ret = (NRF_Status)HAL_SPI_TransmitReceive(HSPI, &cmd, &status, 1, NRF_SPI_TIMEOUT_DUR);
    if (ret != NRF_OK) {
        return ret;
    }
    ret = (NRF_Status)HAL_SPI_Receive(HSPI, read, length, NRF_SPI_TIMEOUT_DUR);
    if (ret != NRF_OK) {
        return ret;
    }
    csn_set();

    return ret;
}

/*
 *
 * Device control
 *
 */

NRF_Status NRF_EnterMode(uint8_t mode)
{
    NRF_Status ret = NRF_OK;

    switch (mode) {
    case NRF_MODE_POWERDOWN:
        csn_set();
        ce_reset();
        ret = NRF_ResetRegisterBit(NRF_REG_CONFIG, CFG_BIT_PWR_UP);
        break;
    case NRF_MODE_STANDBY1:
        if (current_mode == NRF_MODE_POWERDOWN) {
            ret = NRF_SetRegisterBit(NRF_REG_CONFIG, CFG_BIT_PWR_UP);
            wait(1500);
        } else if (current_mode == NRF_MODE_RX) {
            ret = NRF_ResetRegisterBit(NRF_REG_CONFIG, CFG_BIT_PRIM_RX);
            ce_reset();
        } else if (current_mode == NRF_MODE_TX) {
            ce_reset();
        }
        break;
    case NRF_MODE_RX:
        if (current_mode != NRF_MODE_STANDBY1) {
            return NRF_BAD_TRANSITION;
        }
        ret = NRF_SetRegisterBit(NRF_REG_CONFIG, CFG_BIT_PRIM_RX);
        ce_set();
        break;
    case NRF_MODE_TX:
        if (current_mode != NRF_MODE_STANDBY1) {
            return NRF_BAD_TRANSITION;
        }
        ret = NRF_ResetRegisterBit(NRF_REG_CONFIG, CFG_BIT_PRIM_RX);
        ce_set();
        break;
    default:
        ret = NRF_ERROR;
        break;
    }

    if (ret == NRF_OK) {
        current_mode = mode;
    }

    return ret;
}

NRF_Status NRF_WritePayload(uint8_t* payload, uint8_t length) { return NRF_SendWriteCommand(NRF_CMD_W_TX_PAYLOAD, payload, length); }

NRF_Status NRF_ReadPayload(uint8_t* read, uint8_t length) { return NRF_SendReadCommand(NRF_CMD_R_RX_PAYLOAD, read, length); }

NRF_Status NRF_Transmit(uint8_t* payload, uint8_t length)
{
    NRF_Status ret = NRF_OK;
    ret = NRF_WritePayload(payload, length);
    if (ret != NRF_OK) {
        return ret;
    }

    ce_set();
    wait(30);
    ce_reset();

    return ret;
}

NRF_Status NRF_TransmitNoAck(uint8_t* payload, uint8_t length)
{
    NRF_Status ret = NRF_OK;
    ret = NRF_WritePayloadNoAck(payload, length);
    if (ret != NRF_OK) {
        return ret;
    }

    ce_set();
    wait(30);
    ce_reset();

    return ret;
}

void NRF_ReTransmit()
{
    ce_set();
    wait(30);
    ce_reset();
}

NRF_Status NRF_TransmitAndWait(uint8_t* payload, uint8_t length)
{
    NRF_Status ret = NRF_OK;

    ret = NRF_WritePayload(payload, length);
    if (ret != NRF_OK) {
        return ret;
    }

    // Transmit
    ce_set();

    // Wait for status update
    uint8_t status;
    for (;;) {
        status = NRF_ReadStatus();
        if (status & (1 << STATUS_BIT_TX_DS)) {
            // Packet transmitted
            ret = NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_BIT_TX_DS); // clear flag
            break;
        } else if (status & (1 << STATUS_BIT_MAX_RT)) {
            // Max retransmits reached
            NRF_SetRegisterBit(NRF_REG_STATUS, STATUS_BIT_MAX_RT); // clear flag
            ret = NRF_MAX_RT;
            break;
        }
    }
    ce_reset();

    return ret;
}

NRF_Status NRF_WritePayloadNoAck(uint8_t* payload, uint8_t length) { return NRF_SendWriteCommand(NRF_CMD_W_TX_PAYLOAD_NO_ACK, payload, length); }

NRF_Status NRF_WriteAckPayload(uint8_t pipe, uint8_t* payload, uint8_t length) { return NRF_SendWriteCommand(NRF_CMD_W_ACK_PAYLOAD | pipe, payload, length); }

/*
 *
 * Register helpers
 *
 */

/* Writing */
NRF_Status NRF_WriteRegister(uint8_t reg, uint8_t* write, uint8_t length) { return NRF_SendWriteCommand(NRF_CMD_W_REGISTER | reg, write, length); }

NRF_Status NRF_WriteRegisterByte(uint8_t reg, uint8_t byte)
{
    uint8_t write = byte;
    return NRF_WriteRegister(reg, &write, 1);
}

NRF_Status NRF_SetRegisterBit(uint8_t reg, uint8_t bit)
{
    NRF_Status ret = NRF_OK;
    uint8_t cfg = 0x00;

    ret = NRF_ReadRegister(reg, &cfg, 1);
    if (ret != NRF_OK) {
        return ret;
    }

    cfg = cfg | (1 << bit);
    return NRF_WriteRegister(reg, &cfg, 1);
}

NRF_Status NRF_ResetRegisterBit(uint8_t reg, uint8_t bit)
{
    NRF_Status ret = NRF_OK;
    uint8_t cfg = 0x00;

    ret = NRF_ReadRegister(reg, &cfg, 1);
    if (ret != NRF_OK) {
        return ret;
    }

    cfg = cfg & ~(1 << bit);
    return NRF_WriteRegister(reg, &cfg, 1);
}

/* Reading */
NRF_Status NRF_ReadRegister(uint8_t reg, uint8_t* read, uint8_t length) { return NRF_SendReadCommand(NRF_CMD_R_REGISTER | reg, read, length); }

uint8_t NRF_ReadRegisterByte(uint8_t reg)
{
    uint8_t read;
    NRF_SendReadCommand(NRF_CMD_R_REGISTER | reg, &read, 1);
    return read;
}

uint8_t NRF_ReadStatus()
{
    uint8_t status = 0x00;
    uint8_t cmd = NRF_CMD_NOP;

    csn_reset();
    HAL_SPI_TransmitReceive(HSPI, &cmd, &status, 1, NRF_SPI_TIMEOUT_DUR);
    csn_set();

    return status;
}

/*
 *
 * Debugging
 *
 */

int NRF_CurrentMode() { return current_mode; }

NRF_Status NRF_VerifySPI()
{
    NRF_Status ret = NRF_OK;
    uint8_t write[5] = "0x57!";
    uint8_t read[5];

    ret = NRF_WriteRegister(NRF_REG_TX_ADDR, write, 5);
    if (ret != NRF_OK) {
        return ret;
    }
    ret = NRF_ReadRegister(NRF_REG_TX_ADDR, read, 5);
    if (ret != NRF_OK) {
        return ret;
    }

    for (int i = 0; i < 5; i++) {
        if (write[i] != read[i]) {
            return NRF_SPI_ERROR;
        }
    }

    return ret;
}

void NRF_Reset()
{
    NRF_EnterMode(NRF_MODE_POWERDOWN);
    NRF_EnterMode(NRF_MODE_STANDBY1);

    // Flush FIFOs
    NRF_EnterMode(NRF_MODE_TX);
    NRF_SendCommand(NRF_CMD_FLUSH_TX);
    NRF_EnterMode(NRF_MODE_STANDBY1);
    NRF_EnterMode(NRF_MODE_RX);
    NRF_SendCommand(NRF_CMD_FLUSH_RX);
    NRF_EnterMode(NRF_MODE_STANDBY1);

    // Flush register
    NRF_WriteRegisterByte(NRF_REG_CONFIG, 0x0A);
    NRF_WriteRegisterByte(NRF_REG_EN_AA, 0x3f);
    NRF_WriteRegisterByte(NRF_REG_EN_RXADDR, 0x03);
    NRF_WriteRegisterByte(NRF_REG_SETUP_AW, 0x03);
    NRF_WriteRegisterByte(NRF_REG_SETUP_RETR, 0x03);
    NRF_WriteRegisterByte(NRF_REG_RF_CH, 0x02);
    NRF_WriteRegisterByte(NRF_REG_RF_SETUP, 0x0e);
    NRF_WriteRegisterByte(NRF_REG_STATUS, 0x70); // clear flags

    uint8_t address[5] = {0xE7, 0xE7, 0xE7, 0xE7, 0xE7};
    uint8_t address2[5] = {0xC2, 0xC2, 0xC2, 0xC2, 0xC2};
    NRF_WriteRegister(NRF_REG_RX_ADDR_P0, address, 5);
    NRF_WriteRegister(NRF_REG_RX_ADDR_P1, address2, 5);
    NRF_WriteRegisterByte(NRF_REG_RX_ADDR_P2, 0xC3);
    NRF_WriteRegisterByte(NRF_REG_RX_ADDR_P3, 0xC4);
    NRF_WriteRegisterByte(NRF_REG_RX_ADDR_P4, 0xC5);
    NRF_WriteRegisterByte(NRF_REG_RX_ADDR_P5, 0xC6);
    NRF_WriteRegister(NRF_REG_TX_ADDR, address, 5);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P0, 0x00);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P1, 0x00);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P2, 0x00);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P3, 0x00);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P4, 0x00);
    NRF_WriteRegisterByte(NRF_REG_RX_PW_P5, 0x00);

    NRF_WriteRegisterByte(NRF_REG_FIFO_STATUS, 0x00);
    NRF_WriteRegisterByte(NRF_REG_DYNPD, 0x00);
    NRF_WriteRegisterByte(NRF_REG_FEATURE, 0x00);
}

void NRF_PrintStatus()
{
    uint8_t status = NRF_ReadStatus();

    printf("Status register: %02X\r\n", status);
    printf("CE: %d\r\n", HAL_GPIO_ReadPin(NRF_CE_Port, NRF_CE_Pin));
    printf("CSN: %d\r\n", HAL_GPIO_ReadPin(NRF_CSN_Port, NRF_CSN_Pin));
    printf("TX_FULL:  %1X\r\n", status & (1 << STATUS_BIT_TX_FULL));
    printf("RX_P_NO:  %1X\r\n", (status & (0x3 << STATUS_BIT_RX_P_NO)) >> 1);
    printf("MAX_RT:   %1X\r\n", (status & (1 << STATUS_BIT_MAX_RT)) >> 4);
    printf("TX_DS:    %1X\r\n", (status & (1 << STATUS_BIT_TX_DS)) >> 5);
    printf("RX_DR:    %1X\r\n", (status & (1 << STATUS_BIT_RX_DR)) >> 6);
    printf("\r\n");
}

void NRF_PrintFIFOStatus()
{
    uint8_t reg = NRF_ReadRegisterByte(NRF_REG_FIFO_STATUS);

    printf("FIFO status register: %02X\r\n", reg);
    printf("RX_EMPTY:   %2X\r\n", reg & (1 << FIFO_STATUS_BIT_RX_EMPTY));
    printf("RX_FULL:    %2X\r\n", (reg & (1 << FIFO_STATUS_BIT_RX_FULL)) >> 1);
    printf("TX_EMPTY:   %2X\r\n", (reg & (1 << FIFO_STATUS_BIT_TX_EMPTY)) >> 4);
    printf("TX_FULL:    %2X\r\n", (reg & (1 << FIFO_STATUS_BIT_TX_FULL)) >> 5);
    printf("TX_REUSE:   %2X\r\n", (reg & (1 << FIFO_STATUS_BIT_TX_REUSE)) >> 6);
    printf("\r\n");
}

void NRF_PrintConfig()
{
    uint8_t reg = NRF_ReadRegisterByte(NRF_REG_CONFIG);

    printf("Config register: %02X\r\n", reg);
    printf("PRIM_RX:      %1X\r\n", reg & (1 << 0));
    printf("PWR_UP:       %1X\r\n", reg & (1 << 1));
    printf("CRCO:         %1X\r\n", reg & (1 << 2));
    printf("EN_CRC:       %1X\r\n", reg & (1 << 3));
    printf("MASK_MAX_RT:  %1X\r\n", reg & (1 << 4));
    printf("MASK_TX_DS:   %1X\r\n", reg & (1 << 5));
    printf("MASK_RX_DR:   %1X\r\n", reg & (1 << 6));
    printf("\r\n");
}
