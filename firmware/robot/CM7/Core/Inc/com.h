#ifndef COM_H
#define COM_H

/* Public includes */
#include "main.h"
#include <stdbool.h>

/* Public constants */
#define COM_BASESTATION_TIMEOUT_MS 3000

/* Public function declarations */

/**
 * Configures the NRF device according to this robots serial number.
 *
 * @param hspi The handle for the SPI communication.
 */
void COM_Init(SPI_HandleTypeDef* hspi, uint8_t* nrf_available);

/**
 * Checks if a message was received within COM_BASESTATION_TIMEOUT_MS.
 * Call often.
 */
bool COM_Update();

/**
 * Initialize the nRF card.
 */
void COM_RF_Init();

/**
 * Parse the received message and handle it correctly.
 *
 * @param pipe What pipe the message was received on.
 */
void COM_RF_Receive(uint8_t pipe);

/**
 * Verifies nRF communication (through SPI) and resets nRF.
 */
void COM_RF_Reset();

/**
 * Handles interrupts sent from the IRQ pin on the NRF.
 */
void COM_RF_HandleIRQ(void);

/**
 * Printf:s status and FIFO status registers from the NRF.
 */
void COM_RF_PrintInfo(void);

/**
 * Transmits msg with specified length and waits for ACK.
 */
void COM_RF_Send(uint8_t* msg, uint8_t length);

/**
 * Maps the 96 bit Unique device identifier into a robot id 0-15.
 * Returns -1 if this device has no mapping.
 */
uint8_t COM_Get_ID();

#endif /* COM_H */
