/*
 * This files just have a lot of helper definitions for
 * bits and masks which can be used to make calls
 * more readable, but which aren't neccessery.
 */

/**
 * @name NRF register bit numbers
 *
 * Name of particular bits. Used to make code easier to read.
 * See page 54 in datasheet for NRF.
 */
//!@{
#define STATUS_MAX_RT 4
#define STATUS_TX_DS 5
#define STATUS_RX_DR 6
#define FEATURE_EN_DYN_ACK 0
#define FEATURE_EN_ACK_PAY 1
#define FEATURE_EN_DPL 2
#define DYNPD_DPL_P0 0
//!@}

/**
 * @name NRF register masks
 *
 * Are used to mask out wanted bits from registers in the device. Makes code easier to read.
 */
//!@{
#define STATUS_MASK_RX_DR 0x40 /**< Data ready in RX FIFO interrupt. */
#define STATUS_MASK_TX_DS 0x20
#define STATUS_MASK_MAX_RT 0x10
#define STATUS_MASK_RX_P_NO 0x0E /**< Data pipe number for payload available for readin from RX_FIFO. */
//!@}
