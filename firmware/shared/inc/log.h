#ifndef LOG_H
#define LOG_H

/* Public includes */
#if defined STM32H563xx
#include "stm32h5xx_hal.h"
#elif defined STM32H755xx
#include "stm32h7xx_hal.h"
#endif

/* Public defines */
#define LOG_BUFFER_SIZE 5
#define LOG_MSG_SIZE 100

#define LOG_PRINTF(level, fmt, ...) LOG_Printf(&internal_log_mod, level, fmt, ##__VA_ARGS__)
#define LOG_UI(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_UI, fmt, ##__VA_ARGS__)
#define LOG_TRACE(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_TRACE, fmt, ##__VA_ARGS__)
#define LOG_DEBUG(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_DEBUG, fmt, ##__VA_ARGS__)
#define LOG_INFO(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_INFO, fmt, ##__VA_ARGS__)
#define LOG_WARNING(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_WARNING, fmt, ##__VA_ARGS__)
#define LOG_ERROR(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_ERROR, fmt, ##__VA_ARGS__)
#define LOG_BASESTATION(fmt, ...) LOG_Printf(&internal_log_mod, LOG_LEVEL_BASESTATION, fmt, ##__VA_ARGS__)

/* Public enums */

/**
 * @brief The available log levels, from lowest to highest severity.
 *
 * Descriptions from ChatGPT, thanks!
 */
typedef enum
{
    /*
     * Output from the CLI UI which is reachable through UART.
     */
    LOG_LEVEL_UI,

    /**
     * Very detailed information, useful for developers to trace program
     * execution in a granular way. Typically disabled in production due
     * to the high volume of logs generated.
     *
     * Use case: Function entry and exit, detailed state machine behaviors.
     */
    LOG_LEVEL_TRACE,

    /*
     * Informational events useful during development and debugging.
     * Provides insights into the application's behavior under normal
     * operation.
     *
     * Use case: Initialization routines, status of hardware peripherals after
     * setup, algorithmic steps.
     */
    LOG_LEVEL_DEBUG,

    /*
     * General operational information about the system's state. It
     * should not be overly verbose and provide a clear overview of the
     * application's status.
     *
     * Use case: System startup, network status, device or sensor readings at
     * a regular interval.
     */
    LOG_LEVEL_INFO,

    /**
     * Indications of potential issues that are not immediate problems
     * but might require attention or could lead to errors if ignored.
     *
     * Use case: Memory usage nearing high watermark, retrying a failed operation,
     * deprecated API usage.
     */
    LOG_LEVEL_WARNING,

    /**
     * Error conditions that are not fatal but indicate failure in specific
     * operations or inability to perform a requested action.
     *
     * Use case: Failed to read from a sensor, communication timeouts, hardware
     * peripheral failures.
     */
    LOG_LEVEL_ERROR,

    /**
     * Logs to basestation. Since it's costly, it's the highest level.
     */
    LOG_LEVEL_BASESTATION,
} LOG_Level;

/* Public structs */
typedef struct {
    LOG_Level level;
    const char* name;
    const char* short_name;
} LOG_Level_Info;

/* Public variables */
static LOG_Level_Info LOG_LEVEL[LOG_LEVEL_BASESTATION + 1] = {{
                                                                  .level = LOG_LEVEL_UI,
                                                                  .name = "User Interface",
                                                                  .short_name = "UI",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_TRACE,
                                                                  .name = "Trace",
                                                                  .short_name = "T",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_DEBUG,
                                                                  .name = "Debug",
                                                                  .short_name = "D",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_INFO,
                                                                  .name = "Info",
                                                                  .short_name = "I",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_WARNING,
                                                                  .name = "Warning",
                                                                  .short_name = "W",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_ERROR,
                                                                  .name = "Error",
                                                                  .short_name = "E",
                                                              },
                                                              {
                                                                  .level = LOG_LEVEL_BASESTATION,
                                                                  .name = "Basestation",
                                                                  .short_name = "B",
                                                              }};

/**
 * Every submodule of the project should have a LOG_Module which give us finer
 * logging controls.
 */
typedef struct {
    LOG_Level min_output_level;
    const char* name;
    uint8_t muted;
} LOG_Module;

/**
 * A backend is someting outputting log messages to somewhere. Examples are
 * through UART or to a local log buffer.
 */
typedef struct {
    LOG_Level min_output_level;
    const char* name;
    uint8_t muted;
} LOG_Backend;

/* Public function declarations */
void LOG_Init(UART_HandleTypeDef* handle);
void LOG_InitModule(LOG_Module* mod, const char* name, LOG_Level min_out_level, uint8_t muted);
void LOG_Printf(LOG_Module* mod, LOG_Level level, const char* format, ...);
void LOG_PrintLogBuffer(int start, int end);
void LOG_ToggleMuteAll();
LOG_Module** LOG_GetModules(int* len);
LOG_Module* LOG_GetModule(int index);
LOG_Backend* LOG_GetBackends(int* len);
LOG_Backend* LOG_GetBackend(int index);
char (*LOG_GetBuffer())[LOG_MSG_SIZE];

#endif /* LOG_H */
