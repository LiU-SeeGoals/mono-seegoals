#include "ui.h"

/* Private includes */
#include "com.h"
#include "kicker.h"
#include "log.h"
#include "nav.h"
#include "pos_follow.h"
#include <stdlib.h>
#include <string.h>

/* Private defines */
#define RX_BUFFER_LEN 20
#define KEY_NEWLINE 0xd
#define KEY_CTRL_C 0x3
#define KEY_ESC 0x1b
#define KEY_BACKSPACE 0x7f

/* Private enums/structs */
typedef struct {
    int key;
    const char* description;
} CommandInfo;

typedef struct {
    const char* name;
    CommandInfo* cmds;
    size_t len_cmds;
    int parent;
} StructInfo;

/**
 * @brief Available states.
 *
 * A state has a set of commands that can be executed in
 * the state.
 */
typedef enum
{
    state_default = 0,
    state_kicker,
    state_kicker_edit,
    state_logs,
    state_logs_mod_configure,
    state_rf,
    state_motors,
    state_motors_steer,
} state;

/**
 * Every state has a set of commands. The commands are
 * then interpreted in the parse_key() function.
 *
 * [H]elp and [B]ack keys can't be used as they are
 * global commands.
 */
//!@{
CommandInfo default_commands[4] = {{'R', "F"}, {'K', "icker"}, {'L', "ogs"}, {'M', "otors"}};

CommandInfo kicker_commands[4] = {
    {'C', "harge"},
    {'K', "ick"},
    {'P', "rint vars"},
    {'E', "dit vars"},
};

CommandInfo kicker_edit_commands[3] = {
    {'M', "ax charges per kick"},
    {'C', "harge wait (us)"},
    {'D', "ischarge wait (us)"},
};

CommandInfo log_commands[4] = {
    {'P', "rint buffer"},
    {'M', "odules"},
    {'S', "how backends"},
    {'T', "oggle all logging"},
};

CommandInfo log_mod_conf_commands[2] = {
    {'M', "ute"},
    {'S', "et minimum output level"},
};

CommandInfo rf_commands[2] = {
    {'S', "tatus"},
    {'R', "eset"},
};

CommandInfo motors_commands[2] = {{'S', "teer"}, {'T', "oggle movement"}};

CommandInfo motors_steer_commands[4] = {
    {'W', ""},
    {'A', ""},
    {'S', ""},
    {'D', ""},
};
//!@}

/**
 * @brief All the available states that the UI can be in.
 *
 * Each state should have a set of commands that can be
 * executed in the state. These will be printed from
 * the print_help() function.
 *
 * The parent is the state which we'll go back to if
 * the user presses B. The name will be printed
 * from the print_help() function.
 */
StructInfo states[8] = {
    {
        .name = "",
        .cmds = default_commands,
        .len_cmds = sizeof(default_commands) / sizeof(CommandInfo),
        .parent = state_default,
    },
    {
        .name = "Kicker",
        .cmds = kicker_commands,
        .len_cmds = sizeof(kicker_commands) / sizeof(CommandInfo),
        .parent = state_default,
    },
    {
        .name = "Kicker->Edit",
        .cmds = kicker_edit_commands,
        .len_cmds = sizeof(kicker_edit_commands) / sizeof(CommandInfo),
        .parent = state_kicker,
    },
    {
        .name = "Log",
        .cmds = log_commands,
        .len_cmds = sizeof(log_commands) / sizeof(CommandInfo),
        .parent = state_default,
    },
    {
        .name = "Log->Module",
        .cmds = log_mod_conf_commands,
        .len_cmds = sizeof(log_mod_conf_commands) / sizeof(CommandInfo),
        .parent = state_logs,
    },
    {
        .name = "RF",
        .cmds = rf_commands,
        .len_cmds = sizeof(rf_commands) / sizeof(CommandInfo),
        .parent = state_default,
    },
    {
        .name = "Motors",
        .cmds = motors_commands,
        .len_cmds = sizeof(motors_commands) / sizeof(CommandInfo),
        .parent = state_default,
    },
    {
        .name = "Motors->Steer",
        .cmds = motors_steer_commands,
        .len_cmds = sizeof(motors_steer_commands) / sizeof(CommandInfo),
        .parent = state_motors,
    },
};

/* Private variables */
static UART_HandleTypeDef* huart;
static state current_state = state_default;
static int current_command = -1;
static uint8_t key;
static uint8_t reading_to_buffer = 0;
static uint8_t memory; // remember things between states
static uint8_t rx_buffer[RX_BUFFER_LEN];
static char* input_text;
static size_t rx_buffer_loc = 0;
static LOG_Module internal_log_mod;
static uint8_t moving = 0;

/* Private functions declarations */
void print_help();
void read_to_buffer();
void finished_reading_to_buffer();
void stop_reading_to_buffer();
void parse_key();

/*
 * Public functions implementations
 */

void UI_Init(UART_HandleTypeDef* handle)
{
    huart = handle;
    HAL_UART_Receive_IT(huart, &key, 1);
    LOG_InitModule(&internal_log_mod, "UI", LOG_LEVEL_UI, 0);
    print_help();
}

void UI_RxCallback()
{
    if (reading_to_buffer) {
        read_to_buffer();
    } else if ((key == 'B' || key == KEY_BACKSPACE) && current_state != state_default) {
        current_state = states[current_state].parent;
        current_command = key;
        print_help();

        if (moving) {
            NAV_DisableMovement();
            moving = 0;
        }
    } else {
        current_command = key;
        if (key == 'H') {
            print_help();
        } else {
            parse_key();
        }
    }

    HAL_UART_Receive_IT(huart, &key, 1);
}

/*
 * Private functions implementations
 */

void print_help()
{
    if (current_state == state_default) {
        LOG_UI("[UI]: [H]elp ");
    } else {
        LOG_UI("[UI->%s]: [B]ack [H]elp\r\n", states[current_state].name);
    }

    for (int i = 0; i < states[current_state].len_cmds; i++) {
        LOG_UI("[%c]%s ", states[current_state].cmds[i].key, states[current_state].cmds[i].description);
    }
    LOG_UI("\r\n");
}

/**
 * We end up here if we've asked for a value by setting
 * reading_to_buffer high.
 */
void read_to_buffer()
{
    if (key == KEY_NEWLINE || rx_buffer_loc == RX_BUFFER_LEN) {
        finished_reading_to_buffer();
        stop_reading_to_buffer();
    } else if (key == KEY_CTRL_C || key == KEY_ESC) {
        LOG_UI("Exiting...\r\n");
        stop_reading_to_buffer();
    } else if (key == KEY_BACKSPACE) {
        if (rx_buffer_loc > 0) {
            rx_buffer_loc--;
            rx_buffer[rx_buffer_loc] = 0;
        }
        LOG_UI("\r\nInput: %s", rx_buffer);
    } else {
        rx_buffer[rx_buffer_loc] = key;
        LOG_UI("%c", rx_buffer[rx_buffer_loc]);
        rx_buffer_loc++;
    }
}

void start_reading_to_buffer()
{
    reading_to_buffer = 1;
    LOG_UI("\r\n"
           "(ctrl-c / esc to exit)\r\n"
           "Input: ");
}

/**
 * Just quits from the user entering a value end clears the buffer.
 */
void stop_reading_to_buffer()
{
    reading_to_buffer = 0;
    rx_buffer_loc = 0;
    memset(rx_buffer, 0, RX_BUFFER_LEN);
    print_help();
}

/**
 * Whenever the user presses a key into the terminal while
 * having an active UART connection we should end up here.
 *
 * Depending on the current_state we'll interpret the keys
 * differently. Each key should correspond to one of the
 * commands of the state and either should put the user
 * into a new state or perform an action.
 */
void parse_key()
{
    if (current_state == state_default) {
        switch (key) {
        case 'R': // RF
            current_state = state_rf;
            print_help();
            break;
        case 'K': // Kicker
            current_state = state_kicker;
            print_help();
            break;
        case 'L': // Logs
            current_state = state_logs;
            int len;
            LOG_Module** modules = LOG_GetModules(&len);
            for (int i = 0; i < len; i++) {
                LOG_UI("%s\r\n"
                       "   Muted: %i\r\n"
                       "   Min level: %s (%i)\r\n",
                       modules[i]->name, modules[i]->muted, LOG_LEVEL[modules[i]->min_output_level].name, modules[i]->min_output_level);
            }
            print_help();
            break;
        case 'M': // Motors
            current_state = state_motors;
            print_help();
            break;
        default:
            LOG_UI("Unknown command: %c (%x)\r\n", key, key);
            print_help();
            break;
        }
    } else if (current_state == state_kicker) {
        switch (key) {
        case 'C': // Charge
            KICKER_Charge();
            break;
        case 'K': // Kick
            KICKER_Kick();
            break;
        case 'P': // Print vars
        {
            KICKER_Settings* set = KICKER_GetSettings();
            LOG_UI("Max charges per kick: %i\r\nCharge wait (us): %i\r\nDischarge wait (us): %i\r\n", set->max_charges_per_kick, set->charge_wait_us, set->discharge_wait_us);
        } break;
        case 'E': // Edit vars
            current_state = state_kicker_edit;
            print_help();
            break;
        }
    } else if (current_state == state_kicker_edit) {
        switch (key) {
        case 'M': // Max charges per kick
        case 'C': // Charge wait
        case 'D': // Discharge wait
        {
            KICKER_Settings* set = KICKER_GetSettings();
            LOG_UI("\r\n---\r\nMax charges per kick: %i\r\nCharge wait (us): %i\r\nDischarge wait (us): %i\r\n", set->max_charges_per_kick, set->charge_wait_us, set->discharge_wait_us);
            start_reading_to_buffer();
        } break;
        }
    } else if (current_state == state_logs) {
        switch (key) {
        case 'P': // Print buffer
        {
            char (*buffer)[LOG_MSG_SIZE] = LOG_GetBuffer();
            LOG_UI("\r\nLog buffer...\r\n");
            for (int i = 0; i < LOG_BUFFER_SIZE; i++) {
                if (buffer[i] == NULL) {
                    return;
                }
                LOG_UI("%s", buffer[i]);
            }
        } break;
        case 'M': // Modules
        {
            int len;
            LOG_Module** modules = LOG_GetModules(&len);
            for (int i = 0; i < len; i++) {
                LOG_UI("[%i] %s\r\n"
                       "   Muted: %i\r\n"
                       "   Min level: %s (%i)\r\n",
                       i, modules[i]->name, modules[i]->muted, LOG_LEVEL[modules[i]->min_output_level].name, modules[i]->min_output_level);
            }
            start_reading_to_buffer();
        } break;
        case 'S': // Show backends
        {
            int len;
            LOG_Backend* backends = LOG_GetBackends(&len);
            for (int i = 0; i < len; i++) {
                LOG_UI(" %s\r\n"
                       "   Muted: %i\r\n"
                       "   Min level: %s (%i)\r\n",
                       backends[i].name, backends[i].muted, LOG_LEVEL[backends[i].min_output_level].name, backends[i].min_output_level);
            }
        } break;
        case 'T': // Toggle all logging
            LOG_ToggleMuteAll();
            LOG_UI("Toggled mute all.\r\n");
            break;
        }
    } else if (current_state == state_logs_mod_configure) {
        switch (key) {
        case 'M': // Mute
        {
            LOG_Module* mod = LOG_GetModule(memory);
            if (mod == NULL) {
                LOG_WARNING("No log module with index %i\r\n", memory);
                return;
            }

            mod->muted = !mod->muted;
            LOG_UI("[%s] Mute toggled (%i)\r\n", mod->name, mod->muted);
        } break;
        case 'S': // Set minimum output level
        {
            LOG_UI("Levels:\r\n");
            for (int i = LOG_LEVEL_TRACE; i < LOG_LEVEL_BASESTATION; i++) {
                LOG_UI("   [%i] %s\r\n", i, LOG_LEVEL[i].name);
            }
            start_reading_to_buffer();
        } break;
        }
    } else if (current_state == state_rf) {
        switch (key) {
        case 'S': // Status
        {
            COM_RF_PrintInfo();
        } break;
        case 'R': // Reset
        {
            COM_RF_Reset();
        } break;
        }
    } else if (current_state == state_motors) {
        switch (key) {
        case 'S': // Steer
            current_state = state_motors_steer;
            print_help();
            break;
        case 'T': // Toggle movement
            LOG_INFO("Movement toggled\r\n");
            if (moving) {
                NAV_DisableMovement();
                moving = 0;
            } else {
                NAV_EnableMovement();
                moving = 1;
            }
            break;
        }
    } else if (current_state == state_motors_steer) {
        switch (key) {
        case 'W':
            NAV_SetCommandPosition(0, 1, 0);
            if (!moving)
                NAV_EnableMovement();
            moving = 1;
            break;
        case 'A':
            NAV_SetCommandPosition(1, 0, 0);
            if (!moving)
                NAV_EnableMovement();
            moving = 1;
            break;
        case 'S':
            NAV_SetCommandPosition(0, -1, 0);
            if (!moving)
                NAV_EnableMovement();
            moving = 1;
            break;
        case 'D':
            NAV_SetCommandPosition(-1, 0, 0);
            if (!moving)
                NAV_EnableMovement();
            moving = 1;
            break;
        }
    }
}

/**
 * After having set reading_to_buffer high and the user then
 * pressed either enter or filled the whole buffer, we'll
 * end up here.
 *
 * Neither current_state or current_command changes, so the
 * newly input value can be used for whatever we wish.
 *
 * See this as a callback for the user having entered a value.
 */
void finished_reading_to_buffer()
{
    if (current_state == state_kicker_edit) {
        KICKER_Settings* set = KICKER_GetSettings();
        switch (current_command) {
        case 'M':
            set->max_charges_per_kick = atoi(rx_buffer);
            break;
        case 'C':
            set->charge_wait_us = atoi(rx_buffer);
            break;
        case 'D':
            set->discharge_wait_us = atoi(rx_buffer);
            break;
        }
        LOG_UI("\r\n---\r\nMax charges per kick: %i\r\nCharge wait (us): %i\r\nDischarge wait (us): %i\r\n", set->max_charges_per_kick, set->charge_wait_us, set->discharge_wait_us);
    } else if (current_state == state_logs) {
        switch (current_command) {
        case 'M':
            current_state = state_logs_mod_configure;
            memory = atoi(rx_buffer);
            LOG_UI("\r\nEditing: %i\r\n", memory);
            break;
        case 'C':
            break;
        }
    } else if (current_state == state_logs_mod_configure) {
        switch (current_command) {
        case 'S': {
            LOG_Module* mod = LOG_GetModule(memory);
            int level = atoi(rx_buffer);
            if (mod == NULL || level < 0 || level > LOG_LEVEL_BASESTATION) {
                LOG_WARNING("No log module with index %i or level %i\r\n", memory, level);
                return;
            }
            mod->min_output_level = level;
            LOG_UI("\r\nSetting to: %i\r\n", level);
        } break;
        }
    } else {
        LOG_UI("[UI] State not configured to set values.\r\n");
    }
}
