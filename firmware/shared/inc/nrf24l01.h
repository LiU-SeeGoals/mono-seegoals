#ifndef NRF24L01_H
#define NRF24L01_H

#include <stdint.h>

#if defined STM32H563xx
#include "stm32h5xx_hal.h"
#elif defined STM32H755xx
#include "stm32h7xx_hal.h"
#endif

/**
 * @name Defines
 * @anchor defines
 *
 * Ungrouped defines
 */
//!@{

//! Timeout for all SPI communication attempts.
#define NRF_SPI_TIMEOUT_DUR 10
//!@}

/**
 * @name Commands
 * @anchor commands
 * @anchor command
 *
 * Available commands for the device.
 *
 * See table 19 on page 48 in @datasheet.
 */
//!@{
#define NRF_CMD_R_REGISTER 0x00
#define NRF_CMD_W_REGISTER 0x20
#define NRF_CMD_R_RX_PAYLOAD 0x61
#define NRF_CMD_W_TX_PAYLOAD 0xA0
#define NRF_CMD_FLUSH_TX 0xE1
#define NRF_CMD_FLUSH_RX 0xE2
#define NRF_CMD_REUSE_TX_PL 0xE3
#define NRF_CMD_R_RX_PL_WID 0x60
#define NRF_CMD_W_ACK_PAYLOAD 0xA8
#define NRF_CMD_W_TX_PAYLOAD_NO_ACK 0xB0
#define NRF_CMD_NOP 0xFF
//!@}

/**
 * @name Registers
 * @anchor registers
 * @anchor register
 *
 * Available registers on the device.
 *
 * See table 27 on page 54 in @datasheet.
 */
//!@{
#define NRF_REG_CONFIG 0x00
#define NRF_REG_EN_AA 0x01
#define NRF_REG_EN_RXADDR 0x02
#define NRF_REG_SETUP_AW 0x03
#define NRF_REG_SETUP_RETR 0x04
#define NRF_REG_RF_CH 0x05
#define NRF_REG_RF_SETUP 0x06
#define NRF_REG_STATUS 0x07
#define NRF_REG_OBSERVE_TX 0x08
#define NRF_REG_RPD 0x09
#define NRF_REG_RX_ADDR_P0 0x0A
#define NRF_REG_RX_ADDR_P1 0x0B
#define NRF_REG_RX_ADDR_P2 0x0C
#define NRF_REG_RX_ADDR_P3 0x0D
#define NRF_REG_RX_ADDR_P4 0x0E
#define NRF_REG_RX_ADDR_P5 0x0F
#define NRF_REG_TX_ADDR 0x10
#define NRF_REG_RX_PW_P0 0x11
#define NRF_REG_RX_PW_P1 0x12
#define NRF_REG_RX_PW_P2 0x13
#define NRF_REG_RX_PW_P3 0x14
#define NRF_REG_RX_PW_P4 0x15
#define NRF_REG_RX_PW_P5 0x16
#define NRF_REG_FIFO_STATUS 0x17
#define NRF_REG_DYNPD 0x1C
#define NRF_REG_FEATURE 0x1D
//!@}

/**
 * @name Modes
 * @anchor mode
 *
 * Available modes on the device which we easily can reach.
 *
 * See figure 3 (page 21) and table 15 (page 23) in @datasheet.
 */
//!@{
#define NRF_MODE_POWERDOWN 0x0
#define NRF_MODE_STANDBY1 0x1
#define NRF_MODE_RX 0x2
#define NRF_MODE_TX 0x3
//!@}

/**
 * \brief Basically HAL_StatusTypeDef but with a few additions.
 */
typedef enum
{
    NRF_OK = 0x00,            //!< All went well
    NRF_SPI_ERROR = 0x01,     //!< Unspecified SPI error
    NRF_SPI_BUSY = 0x02,      //!< SPI device busy
    NRF_SPI_TIMEOUT = 0x03,   //!< SPI communication timed out
    NRF_ERROR = 0x04,         //!< Unspecified general error
    NRF_MAX_RT = 0x05,        //!< Max retries on packet transmission
    NRF_BAD_TRANSITION = 0x06 //!< When an unallowed mode transition is tried
} NRF_Status;

/**
 * @name Main
 * @anchor mains
 * @anchor main
 *
 * These functions are the main way to communicate directly
 * with the device.
 */
//!@{
/**
 * @anchor init
 * \brief Initalises the library and make the device enter standby-I mode.
 * \note Is expected to be the first interaction with the device after powerup.
 *
 * \param handle A pointer at the SPI_HandleTypeDef which HAL creates for the
 *          SPI communication.
 * \param PortCSN A pointer at the GPIO_TypeDef for the port that the device CSN
 *          pin is on.
 * \param PinCSN The pin that the devices CSN is connected to.
 * \param PortCE A pointer at the GPIO_TypeDef far the port that the device CE
 *          pin is on.
 * \param PinCE The pin that the devices CE is connected to.
 * \return NRF_Status
 */
NRF_Status NRF_Init(SPI_HandleTypeDef* handle, GPIO_TypeDef* PortCSN, uint16_t PinCSN, GPIO_TypeDef* PortCE, uint16_t PinCE);

/**
 * \brief Sends a @ref command to the device.
 *
 * \param cmd A @ref command.
 * \return NRF_Status
 */
NRF_Status NRF_SendCommand(uint8_t cmd);

/**
 * \brief Sends a write @ref command to the device along with
 * a buffer to write.
 *
 * \param cmd A write @ref command.
 * \param write Buffer to write.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_SendWriteCommand(uint8_t cmd, uint8_t* write, uint8_t length);

/**
 * \brief Sends a read @ref command to the device along with
 * a buffer to read to.
 *
 * \param cmd A read @ref command.
 * \param write Buffer to read to.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_SendReadCommand(uint8_t cmd, uint8_t* read, uint8_t length);
//!@}

/**
 * @name Device control
 *
 * These functions are used to control the device and
 * transmit/recieve data.
 *
 */
//!@{
/**
 * \brief Enter a specific @ref mode. See details.
 * \warning We're doing some handholding, but we don't keep track of entering
 * Standby-II mode. If that happens, the behaviour is undefind. Please look at figure 3
 * page 21 in @datasheet to see what transitions make sense. And table 15
 * on page 23 for what register values means which mode.
 *
 * \note The device is in Standby-I from running @ref init.
 * \note If you enter TX mode you'll end up in Standby-II mode
 * unless you reset CE pin manually. Use @ref transmit function
 * if you only want to send one packet.
 *
 * Expected transitions: \n
 * Any -> Powerdown \n
 * Powerdown -> Standby-I \n
 * Standby-I > RX/TX \n
 *
 * \param mode A @ref mode.
 * \return NRF_Status
 */
NRF_Status NRF_EnterMode(uint8_t mode);

/**
 * \brief Write payload (1-32 bytes).
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_WritePayload(uint8_t* payload, uint8_t length);

/**
 * \brief Read payload (1-32 bytes).
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_ReadPayload(uint8_t* payload, uint8_t length);

/**
 * @anchor transmit
 * \brief Transmit the specified payload from standby-I mode.
 *
 * \note Expects device to be in standby-I mode (which it should enter from
 * @ref init) and be configured as transmitter (PRIM_RX is 0 in CONFIG
 * register, default).
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return HAL_OK on success, HAL_ERROR else.
 */
NRF_Status NRF_Transmit(uint8_t* payload, uint8_t length);

/**
 *\brief Transmit the specified payload from standby-I mode,
 * with NO_ACK bit set.
 *
 * \note Expects device to be in standby-I mode (which it should enter from
 * @ref init) and be configured as transmitter (PRIM_RX is 0 in CONFIG
 * register, default).
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return HAL_OK on success, HAL_ERROR else.
 */
NRF_Status NRF_TransmitNoAck(uint8_t* payload, uint8_t length);

/**
 * \brief Transmit the payload already in the TX buffer. When
 * a message is sent but no ack is received, it stays in the TX buffer.
 */
void NRF_ReTransmit();

/**
 * \brief Transmit the specified payload from standby-I mode and wait for
 * an ACK or max retries. Usually this is handled through interrupts.
 *
 * \note Expects device to be in standby-I mode (which it should enter from
 * @ref init) and be configured as transmitter (PRIM_RX is 0 in CONFIG
 * register, default).
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return HAL_OK on success, HAL_MAX_RT if max retransmits reached, or a SPI error.
 */
NRF_Status NRF_TransmitAndWait(uint8_t* payload, uint8_t length);
//!@}

/**
 * @name Advanced device control
 *
 * These functions require you to enable certain features on the device
 * for them to work
 *
 */
//!@{
/**
 * \brief Write payload that tells receiver to not send an ACK as response,
 * see section 7.4.2.3 on page 29 in @datasheet for details.
 *
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_WritePayloadNoAck(uint8_t* payload, uint8_t length);

/**
 * \brief Write payload (1-32 bytes) to send from the receiver to the
 * transmitter in the next ACK package (1-32 bytes) for specified pipe.
 * See details in section 7.5.1 on page 31 in @datasheet.
 *
 * \param pipe The pipe to
 * \param payload Buffer with payload.
 * \param length Length of buffer.
 * \return NRF_Status
 */
NRF_Status NRF_WriteAckPayload(uint8_t pipe, uint8_t* payload, uint8_t length);
//!@}

/**
 * @name Register helpers
 * @anchor helpers
 *
 * These are abstracted functions to make working with
 * registers nicer.
 *
 */
//!@{
/**
 * \brief Write buffer to @ref register.
 *
 * \param reg One of the available @ref registers.
 * \param write A pointer to the buffer to write.
 * \param length The length of the buffer.
 * \return NRF_Status
 */
NRF_Status NRF_WriteRegister(uint8_t reg, uint8_t* write, uint8_t length);

/**
 * \brief Write byte to @ref register.
 *
 * \param reg A @ref register.
 * \param byte The byte to write to the register.
 * \return NRF_Status
 */
NRF_Status NRF_WriteRegisterByte(uint8_t reg, uint8_t byte);

/**
 * \brief Set bit to 1 in @ref register.
 *
 * \param reg A @ref register.
 * \param bit A particular bit within the register.
 * \return NRF_Status
 */
NRF_Status NRF_SetRegisterBit(uint8_t reg, uint8_t bit);

/**
 * \brief Set bit to 0 in @ref register.
 *
 * \param reg A @ref register.
 * \param bit A particular bit within the register.
 * \return NRF_Status
 */
NRF_Status NRF_ResetRegisterBit(uint8_t reg, uint8_t bit);

/**
 * \brief Read register to buffer.
 *
 * \param reg A @ref register.
 * \param read A pointer to the buffer to read to.
 * \param length The length of the buffer.
 * \return NRF_Status
 */
NRF_Status NRF_ReadRegister(uint8_t reg, uint8_t* read, uint8_t length);

/**
 * \brief Read byte from register.
 *
 * \param reg A @ref register.
 * \return The byte in the register.
 */
uint8_t NRF_ReadRegisterByte(uint8_t reg);

/**
 * \brief Read status by issuing a NOP.
 *
 * More efficient since only a single SPI transmit/recieve is issued.
 *
 * \return The status register byte.
 */
uint8_t NRF_ReadStatus();
//!@}

/**
 * @name Debugging
 * @anchor debugging
 *
 * These can be useful for debugging.
 */
//!@{

/**
 * \brief Returns which mode we're believed to be in.
 *
 * \warning We don't keep track of entering Standby-II mode.
 *
 * \return see @ref mode
 */
int NRF_CurrentMode();

/**
 * \brief Verifies that SPI communication with the device works.
 *
 * We write a predetermined value to the TX_ADDR @ref register.
 * Then we read the value of the register.
 * If the two values are the same then SPI is working.
 *
 * \return HAL_OK on success.
 */
NRF_Status NRF_VerifySPI();

/**
 * \brief Reset all registers/FIFOs to initial values
 * specified in @datasheet table 27 on page 59.
 *
 * \note The device will leave in Standby-I mode.
 *
 * This is pretty overkill but useful for testing since the device
 * doesn't reset all registers on a fast enough power reset.
 */
void NRF_Reset();

/**
 * \brief printf:s the bits from the status register.
 */
void NRF_PrintStatus();

/**
 * \brief printf:s the bits from the FIFO status register.
 */
void NRF_PrintFIFOStatus();

/**
 * \brief printf:s the config register.
 */
void NRF_PrintConfig();
//!@}

#endif /* NRF24L01_H */
