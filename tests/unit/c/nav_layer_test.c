/**
 * @file nav_layer_test.c
 * @brief Tests for navigation layer (NAV) command handling
 *
 * Tests the NAV layer with new RobotCommand interface:
 * - NAV_HandleCommand() processing
 * - NAV_SetMovement() with int16 coordinates
 * - NAV_GoToAction() with int16 coordinates
 * - Command type routing
 * - State changes and validation
 *
 * To compile and run:
 *   gcc -I../../../firmware/shared/inc \
 *       ../c/nav_layer_test.c \
 *       ../../../firmware/shared/src/robot_command.c \
 *       -o nav_test && ./nav_test
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>

#include "robot_command.h"

// Color codes
#define RED     "\x1b[31m"
#define GREEN   "\x1b[32m"
#define YELLOW  "\x1b[33m"
#define RESET   "\x1b[0m"

static int tests_passed = 0;
static int tests_failed = 0;

#define TEST(name) printf(YELLOW "[TEST]" RESET " %s\n", name)
#define PASS() printf(GREEN "  ✓ PASS\n" RESET); tests_passed++
#define FAIL(msg) printf(RED "  ✗ FAIL: %s\n" RESET, msg); tests_failed++

/**
 * Simulated robot state and navigation layer
 * In real firmware, this is in firmware/robot/CM7/Core/Src/nav.c
 */
typedef enum {
    STATE_IDLE,
    STATE_MOVING,
    STATE_KICKING,
    STATE_ROTATING,
} RobotState;

typedef struct {
    RobotState current_state;
    int16_t current_x;
    int16_t current_y;
    int16_t target_x;
    int16_t target_y;
    int16_t current_heading;
    int16_t target_heading;
    int16_t movement_speed;
    int16_t angular_velocity;
    uint16_t last_kick_speed;
    
    // Tracking
    int commands_executed;
    int errors;
} NavigationState;

static NavigationState nav_state = {
    .current_state = STATE_IDLE,
    .current_x = 0,
    .current_y = 0,
    .target_x = 0,
    .target_y = 0,
    .current_heading = 0,
    .target_heading = 0,
    .movement_speed = 0,
    .angular_velocity = 0,
    .last_kick_speed = 0,
    .commands_executed = 0,
    .errors = 0,
};

/**
 * Simulates NAV_SetMovement()
 * Real function in firmware/robot/CM7/Core/Src/nav.c
 * Changed signature: moved from Command* to RobotCommand*
 */
bool nav_set_movement(int16_t x, int16_t y, uint16_t speed)
{
    nav_state.target_x = x;
    nav_state.target_y = y;
    nav_state.movement_speed = speed;
    nav_state.current_state = STATE_MOVING;
    return true;
}

/**
 * Simulates NAV_GoToAction()
 * Real function in firmware/robot/CM7/Core/Src/nav.c
 * Changed signature: moved from Command* to RobotCommand*
 */
bool nav_go_to_action(int16_t dest_x, int16_t dest_y, int16_t heading)
{
    nav_state.target_x = dest_x;
    nav_state.target_y = dest_y;
    nav_state.target_heading = heading;
    nav_state.current_state = STATE_MOVING;
    return true;
}

/**
 * Simulates NAV_HandleCommand()
 * Real function in firmware/robot/CM7/Core/Src/nav.c
 * Changed signature: moved from Command* to RobotCommand*
 */
bool nav_handle_command(RobotCommand* cmd)
{
    if (!cmd) {
        nav_state.errors++;
        return false;
    }

    switch (cmd->action_type) {
        case ACTION_TYPE_KICK:
            nav_state.current_state = STATE_KICKING;
            nav_state.last_kick_speed = cmd->kick_speed;
            nav_state.commands_executed++;
            return true;

        case ACTION_TYPE_STOP:
            nav_state.current_state = STATE_IDLE;
            nav_state.movement_speed = 0;
            nav_state.angular_velocity = 0;
            nav_state.commands_executed++;
            return true;

        case ACTION_TYPE_MOVE_TO:
            if (!nav_go_to_action(cmd->dest_x, cmd->dest_y, cmd->orientation_w)) {
                nav_state.errors++;
                return false;
            }
            nav_state.commands_executed++;
            return true;

        case ACTION_TYPE_MOVE:
            if (!nav_set_movement(cmd->pos_x, cmd->pos_y, cmd->kick_speed)) {
                nav_state.errors++;
                return false;
            }
            nav_state.commands_executed++;
            return true;

        case ACTION_TYPE_ROTATE:
            nav_state.current_state = STATE_ROTATING;
            nav_state.target_heading = cmd->orientation_w;
            nav_state.angular_velocity = cmd->angular_vel;
            nav_state.commands_executed++;
            return true;

        case ACTION_TYPE_INIT:
            // Reset to initial state
            nav_state.current_state = STATE_IDLE;
            nav_state.current_x = 0;
            nav_state.current_y = 0;
            nav_state.current_heading = 0;
            nav_state.commands_executed++;
            return true;

        default:
            nav_state.errors++;
            return false;
    }
}

/**
 * Test: NAV_HandleCommand with KICK action
 */
void test_nav_kick_command()
{
    TEST("NAV_HandleCommand - KICK action");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 1,
        .kick_speed = 2000,
        .pos_x = 100,
        .pos_y = 200,
        .dest_x = 300,
        .dest_y = 400,
        .direction_x = 10,
        .direction_y = 20,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle KICK command");
        return;
    }

    if (nav_state.current_state != STATE_KICKING) {
        FAIL("State not set to KICKING");
        return;
    }

    if (nav_state.last_kick_speed != 2000) {
        FAIL("Kick speed not stored");
        return;
    }

    if (nav_state.commands_executed != 1) {
        FAIL("Command count not incremented");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with STOP action
 */
void test_nav_stop_command()
{
    TEST("NAV_HandleCommand - STOP action");

    // First move to a state
    nav_state.current_state = STATE_MOVING;
    nav_state.movement_speed = 100;

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_STOP,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle STOP command");
        return;
    }

    if (nav_state.current_state != STATE_IDLE) {
        FAIL("State not set to IDLE");
        return;
    }

    if (nav_state.movement_speed != 0) {
        FAIL("Movement speed not cleared");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with MOVE_TO action
 */
void test_nav_move_to_command()
{
    TEST("NAV_HandleCommand - MOVE_TO action");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 1000,
        .dest_y = 2000,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 900,  // Target heading
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle MOVE_TO command");
        return;
    }

    if (nav_state.current_state != STATE_MOVING) {
        FAIL("State not set to MOVING");
        return;
    }

    if (nav_state.target_x != 1000 || nav_state.target_y != 2000) {
        FAIL("Target coordinates not set");
        return;
    }

    if (nav_state.target_heading != 900) {
        FAIL("Target heading not set");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with MOVE action
 */
void test_nav_move_command()
{
    TEST("NAV_HandleCommand - MOVE action");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_MOVE,
        .robot_id = 1,
        .kick_speed = 500,  // Speed for movement
        .pos_x = 1500,
        .pos_y = 2500,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle MOVE command");
        return;
    }

    if (nav_state.current_state != STATE_MOVING) {
        FAIL("State not set to MOVING");
        return;
    }

    if (nav_state.target_x != 1500 || nav_state.target_y != 2500) {
        FAIL("Movement target not set");
        return;
    }

    if (nav_state.movement_speed != 500) {
        FAIL("Movement speed not set");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with ROTATE action
 */
void test_nav_rotate_command()
{
    TEST("NAV_HandleCommand - ROTATE action");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_ROTATE,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 180,  // Rotation speed
        .orientation_w = 1800,  // Target orientation (180 degrees)
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle ROTATE command");
        return;
    }

    if (nav_state.current_state != STATE_ROTATING) {
        FAIL("State not set to ROTATING");
        return;
    }

    if (nav_state.target_heading != 1800) {
        FAIL("Target heading not set");
        return;
    }

    if (nav_state.angular_velocity != 180) {
        FAIL("Angular velocity not set");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with INIT action
 */
void test_nav_init_command()
{
    TEST("NAV_HandleCommand - INIT action");

    // Set some state
    nav_state.current_state = STATE_MOVING;
    nav_state.current_x = 1000;
    nav_state.current_y = 2000;
    nav_state.current_heading = 1000;

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_INIT,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed to handle INIT command");
        return;
    }

    if (nav_state.current_state != STATE_IDLE) {
        FAIL("State not reset to IDLE");
        return;
    }

    if (nav_state.current_x != 0 || nav_state.current_y != 0) {
        FAIL("Position not reset");
        return;
    }

    if (nav_state.current_heading != 0) {
        FAIL("Heading not reset");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with null pointer
 */
void test_nav_null_command()
{
    TEST("NAV_HandleCommand - null pointer");

    nav_state.errors = 0;

    if (nav_handle_command(NULL)) {
        FAIL("Should reject null command");
        return;
    }

    if (nav_state.errors != 1) {
        FAIL("Error not incremented");
        return;
    }

    PASS();
}

/**
 * Test: NAV_HandleCommand with invalid action type
 */
void test_nav_invalid_action()
{
    TEST("NAV_HandleCommand - invalid action type");

    RobotCommand cmd = {
        .action_type = 99,  // Invalid
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    nav_state.errors = 0;

    if (nav_handle_command(&cmd)) {
        FAIL("Should reject invalid action");
        return;
    }

    if (nav_state.errors != 1) {
        FAIL("Error not incremented");
        return;
    }

    PASS();
}

/**
 * Test: Sequential command handling
 */
void test_nav_sequential_commands()
{
    TEST("NAV sequential command handling");

    nav_state.commands_executed = 0;
    nav_state.errors = 0;

    // Command 1: Initialize
    RobotCommand cmd1 = {
        .action_type = ACTION_TYPE_INIT,
        .robot_id = 1,
    };

    if (!nav_handle_command(&cmd1)) {
        FAIL("INIT failed");
        return;
    }

    // Command 2: Move to position
    RobotCommand cmd2 = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 1,
        .dest_x = 1000,
        .dest_y = 1000,
        .orientation_w = 0,
    };

    if (!nav_handle_command(&cmd2)) {
        FAIL("MOVE_TO failed");
        return;
    }

    // Command 3: Kick
    RobotCommand cmd3 = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 1,
        .kick_speed = 2000,
    };

    if (!nav_handle_command(&cmd3)) {
        FAIL("KICK failed");
        return;
    }

    // Command 4: Stop
    RobotCommand cmd4 = {
        .action_type = ACTION_TYPE_STOP,
        .robot_id = 1,
    };

    if (!nav_handle_command(&cmd4)) {
        FAIL("STOP failed");
        return;
    }

    if (nav_state.commands_executed != 4) {
        FAIL("Command count mismatch");
        return;
    }

    PASS();
}

/**
 * Test: int16 coordinate handling
 */
void test_nav_int16_coordinates()
{
    TEST("NAV int16 coordinate handling");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 1,
        .dest_x = 32767,    // Max int16
        .dest_y = -32768,   // Min int16
        .orientation_w = 3599,  // Max valid orientation
    };

    nav_state.errors = 0;

    if (!nav_handle_command(&cmd)) {
        FAIL("Failed with extreme coordinates");
        return;
    }

    if (nav_state.target_x != 32767 || nav_state.target_y != -32768) {
        FAIL("Extreme coordinates not preserved");
        return;
    }

    PASS();
}

/**
 * Test: Command execution tracking
 */
void test_nav_command_tracking()
{
    TEST("NAV command execution tracking");

    nav_state.commands_executed = 0;

    for (int i = 0; i < 10; i++) {
        RobotCommand cmd = {
            .action_type = ACTION_TYPE_STOP,
            .robot_id = 1,
        };

        if (!nav_handle_command(&cmd)) {
            FAIL("Failed to execute command");
            return;
        }
    }

    if (nav_state.commands_executed != 10) {
        FAIL("Command count tracking failed");
        return;
    }

    PASS();
}

int main()
{
    printf("\n" YELLOW "=== NAV Layer Tests ===" RESET "\n\n");

    // Individual command tests
    test_nav_kick_command();
    test_nav_stop_command();
    test_nav_move_to_command();
    test_nav_move_command();
    test_nav_rotate_command();
    test_nav_init_command();

    // Error handling
    printf("\n" YELLOW "--- Error Handling Tests ---\n" RESET);
    test_nav_null_command();
    test_nav_invalid_action();

    // Integration tests
    printf("\n" YELLOW "--- Integration Tests ---\n" RESET);
    test_nav_sequential_commands();
    test_nav_int16_coordinates();
    test_nav_command_tracking();

    printf("\n" YELLOW "=== Test Summary ===" RESET "\n");
    printf(GREEN "Passed: %d\n" RESET, tests_passed);
    printf(RED "Failed: %d\n" RESET, tests_failed);

    if (tests_failed > 0) {
        printf("\n" RED "TESTS FAILED" RESET "\n\n");
        return 1;
    } else {
        printf("\n" GREEN "ALL TESTS PASSED" RESET "\n\n");
        return 0;
    }
}
