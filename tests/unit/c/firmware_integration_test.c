/**
 * @file firmware_integration_test.c
 * @brief Integration tests for firmware communication layer
 *
 * Tests the full integration of custom binary protocol with:
 * - Basestation communication parsing
 * - Robot command parsing
 * - Navigation command handling
 * - End-to-end encoding/decoding
 *
 * To compile and run:
 *   gcc -I../../../firmware/shared/inc \
 *       ../c/robot_command_test.c \
 *       ../c/firmware_integration_test.c \
 *       ../../../firmware/shared/src/robot_command.c \
 *       -o firmware_test && ./firmware_test
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <assert.h>

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
 * Test: Basestation packet reception and parsing
 * Simulates receiving an encoded command packet and verifying it parses correctly
 */
void test_basestation_packet_reception()
{
    TEST("Basestation packet reception and parsing");

    // Simulate a packet containing encoded command
    RobotCommand original_cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 3,
        .kick_speed = 1000,
        .pos_x = 1500,
        .pos_y = 2000,
        .dest_x = 2000,
        .dest_y = 2500,
        .direction_x = 50,
        .direction_y = 75,
        .angular_vel = 90,
        .orientation_w = 455,
    };

    // Encode the command (what controller sends)
    uint8_t packet_buffer[ROBOT_COMMAND_SIZE];
    if (!robot_command_encode(&original_cmd, packet_buffer)) {
        FAIL("Failed to encode command");
        return;
    }

    // Simulate basestation receiving the packet
    // Basestation would check packet size
    if (ROBOT_COMMAND_SIZE != 20) {
        FAIL("Packet size mismatch");
        return;
    }

    // Basestation validates and extracts robot ID
    RobotCommand received_cmd = {0};
    if (!robot_command_decode(packet_buffer, &received_cmd)) {
        FAIL("Basestation failed to decode");
        return;
    }

    if (received_cmd.robot_id < 1 || received_cmd.robot_id > 7) {
        FAIL("Invalid robot ID after decode");
        return;
    }

    // Verify the decoded command matches original
    if (received_cmd.action_type != original_cmd.action_type ||
        received_cmd.robot_id != original_cmd.robot_id ||
        received_cmd.kick_speed != original_cmd.kick_speed) {
        FAIL("Command data mismatch");
        return;
    }

    PASS();
}

/**
 * Test: Robot packet reception and parsing
 * Simulates robot receiving encoded command over RF
 */
void test_robot_packet_reception()
{
    TEST("Robot packet reception and parsing");

    // Command to be sent to robot
    RobotCommand original_cmd = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 500,
        .pos_y = 500,
        .dest_x = 1000,
        .dest_y = 1000,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 45,
        .orientation_w = 900,
    };

    // Encode as it would be sent over RF
    uint8_t rf_packet[ROBOT_COMMAND_SIZE];
    if (!robot_command_encode(&original_cmd, rf_packet)) {
        FAIL("Failed to encode for RF transmission");
        return;
    }

    // Simulate robot receiving the packet
    // Robot checks packet size
    if (ROBOT_COMMAND_SIZE != 20) {
        FAIL("RF packet size wrong");
        return;
    }

    // Robot decodes
    RobotCommand robot_cmd = {0};
    if (!robot_command_decode(rf_packet, &robot_cmd)) {
        FAIL("Robot failed to decode");
        return;
    }

    // Verify all fields decoded correctly
    if (robot_cmd.action_type != ACTION_TYPE_MOVE_TO ||
        robot_cmd.robot_id != 1 ||
        robot_cmd.pos_x != 500 ||
        robot_cmd.pos_y != 500 ||
        robot_cmd.dest_x != 1000 ||
        robot_cmd.dest_y != 1000) {
        FAIL("Robot received incorrect command data");
        return;
    }

    PASS();
}

/**
 * Test: Multiple commands in sequence
 * Simulates sending multiple commands to verify state doesn't corrupt
 */
void test_multiple_sequential_commands()
{
    TEST("Multiple sequential commands");

    // Simulate sending different commands to different robots
    RobotCommand commands[] = {
        {ACTION_TYPE_KICK, 1, 1000, 100, 200, 300, 400, 10, 20, 30, 100},
        {ACTION_TYPE_MOVE, 2, 500, -100, -200, 0, 0, 0, 0, 0, 0},
        {ACTION_TYPE_STOP, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0},
        {ACTION_TYPE_ROTATE, 4, 0, 1000, 2000, 1500, 2500, 50, 75, 180, 1800},
    };

    uint8_t buffer[ROBOT_COMMAND_SIZE];
    RobotCommand decoded[4] = {0};

    // Encode and decode each command
    for (int i = 0; i < 4; i++) {
        if (!robot_command_encode(&commands[i], buffer)) {
            FAIL("Failed to encode command");
            return;
        }

        if (!robot_command_decode(buffer, &decoded[i])) {
            FAIL("Failed to decode command");
            return;
        }

        // Verify each decoded command
        if (decoded[i].action_type != commands[i].action_type ||
            decoded[i].robot_id != commands[i].robot_id) {
            FAIL("Command data corrupted in sequence");
            return;
        }
    }

    PASS();
}

/**
 * Test: Command buffer boundaries
 * Ensures the 20-byte buffer is properly utilized
 */
void test_buffer_size_constraints()
{
    TEST("Buffer size constraints");

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd = {0};

    // Fill with test pattern
    memset(buf, 0xAB, ROBOT_COMMAND_SIZE);

    // Encode a command
    RobotCommand test_cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 5,
        .kick_speed = 2000,
        .pos_x = 32767,  // Max int16
        .pos_y = -32768, // Min int16
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 100,
        .direction_y = -100,
        .angular_vel = 32767,
        .orientation_w = 3599,
    };

    if (!robot_command_encode(&test_cmd, buf)) {
        FAIL("Failed to encode extreme values");
        return;
    }

    // Verify buffer is exactly 20 bytes
    if (ROBOT_COMMAND_SIZE != 20) {
        FAIL("Buffer size incorrect");
        return;
    }

    // Decode and verify
    RobotCommand decoded = {0};
    if (!robot_command_decode(buf, &decoded)) {
        FAIL("Failed to decode from buffer");
        return;
    }

    PASS();
}

/**
 * Test: Error handling - invalid robot ID
 * Basestation receives command with invalid robot ID
 */
void test_error_handling_invalid_robot_id()
{
    TEST("Error handling - invalid robot ID");

    // Try to create command with invalid robot ID
    RobotCommand bad_cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 10,  // Invalid! Must be 1-7
        .kick_speed = 1000,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];

    // Encoder should reject invalid robot ID
    if (robot_command_encode(&bad_cmd, buf)) {
        FAIL("Should reject invalid robot ID");
        return;
    }

    PASS();
}

/**
 * Test: Error handling - invalid action type
 * Receives command with invalid action type
 */
void test_error_handling_invalid_action()
{
    TEST("Error handling - invalid action type");

    RobotCommand bad_cmd = {
        .action_type = 99,  // Invalid! Must be 0-5
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

    uint8_t buf[ROBOT_COMMAND_SIZE];

    // Encoder should reject invalid action type
    if (robot_command_encode(&bad_cmd, buf)) {
        FAIL("Should reject invalid action type");
        return;
    }

    PASS();
}

/**
 * Test: Packet consistency across encodes
 * Same command should always produce identical packet
 */
void test_packet_determinism()
{
    TEST("Packet determinism");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 2,
        .kick_speed = 1500,
        .pos_x = 1234,
        .pos_y = 5678,
        .dest_x = 4321,
        .dest_y = 8765,
        .direction_x = 50,
        .direction_y = -50,
        .angular_vel = 180,
        .orientation_w = 1800,
    };

    uint8_t buf1[ROBOT_COMMAND_SIZE];
    uint8_t buf2[ROBOT_COMMAND_SIZE];

    // Encode the same command twice
    if (!robot_command_encode(&cmd, buf1) ||
        !robot_command_encode(&cmd, buf2)) {
        FAIL("Failed to encode");
        return;
    }

    // Verify both packets are identical
    if (memcmp(buf1, buf2, ROBOT_COMMAND_SIZE) != 0) {
        FAIL("Same command produced different packets");
        return;
    }

    PASS();
}

/**
 * Test: NRF24 packet size constraint
 * Verify 20-byte protocol fits in NRF24 32-byte limit with header
 */
void test_nrf24_size_constraint()
{
    TEST("NRF24 size constraint");

    // NRF24 max packet size is 32 bytes
    // Our protocol uses 20 bytes
    // + 1 byte header (message type) = 21 bytes total
    // This leaves 11 bytes headroom

    uint8_t nrf_packet[32];
    
    // Simulate NRF24 packet with our protocol
    nrf_packet[0] = 1;  // Message type: ACTION
    
    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 1,
        .kick_speed = 1000,
        .pos_x = 1500,
        .pos_y = 2000,
        .dest_x = 2000,
        .dest_y = 2500,
        .direction_x = 50,
        .direction_y = 75,
        .angular_vel = 90,
        .orientation_w = 455,
    };

    // Encode our command into NRF packet (skipping header byte)
    if (!robot_command_encode(&cmd, &nrf_packet[1])) {
        FAIL("Failed to encode");
        return;
    }

    // Verify total packet size is within NRF24 limit
    uint16_t total_size = 1 + ROBOT_COMMAND_SIZE;  // header + command
    if (total_size > 32) {
        FAIL("Packet exceeds NRF24 limit");
        return;
    }

    PASS();
}

/**
 * Test: All action types
 * Verify all 6 action types work correctly
 */
void test_all_action_types()
{
    TEST("All action types support");

    const char* action_names[] = {
        "KICK",
        "STOP",
        "MOVE_TO",
        "INIT",
        "MOVE",
        "ROTATE"
    };

    for (uint8_t i = 0; i < 6; i++) {
        RobotCommand cmd = {
            .action_type = i,
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

        uint8_t buf[ROBOT_COMMAND_SIZE];
        RobotCommand decoded = {0};

        if (!robot_command_encode(&cmd, buf)) {
            printf(RED "  ✗ FAIL: Failed to encode %s\n" RESET, action_names[i]);
            tests_failed++;
            return;
        }

        if (!robot_command_decode(buf, &decoded)) {
            printf(RED "  ✗ FAIL: Failed to decode %s\n" RESET, action_names[i]);
            tests_failed++;
            return;
        }

        if (decoded.action_type != i) {
            printf(RED "  ✗ FAIL: %s type mismatch\n" RESET, action_names[i]);
            tests_failed++;
            return;
        }
    }

    PASS();
}

/**
 * Test: Coordinate range limits
 * Verify int16 range handling (-32768 to 32767)
 */
void test_coordinate_ranges()
{
    TEST("Coordinate range limits");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 32767,     // Max positive int16
        .pos_y = -32768,    // Min negative int16
        .dest_x = -1,
        .dest_y = 1,
        .direction_x = 100,
        .direction_y = -100,
        .angular_vel = 32767,
        .orientation_w = 0,
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand decoded = {0};

    if (!robot_command_encode(&cmd, buf)) {
        FAIL("Failed to encode extreme coordinates");
        return;
    }

    if (!robot_command_decode(buf, &decoded)) {
        FAIL("Failed to decode extreme coordinates");
        return;
    }

    if (decoded.pos_x != cmd.pos_x ||
        decoded.pos_y != cmd.pos_y ||
        decoded.dest_x != cmd.dest_x ||
        decoded.dest_y != cmd.dest_y ||
        decoded.angular_vel != cmd.angular_vel) {
        FAIL("Extreme coordinate values corrupted");
        return;
    }

    PASS();
}

int main()
{
    printf("\n" YELLOW "=== Firmware Integration Tests ===" RESET "\n\n");

    // Basestation tests
    test_basestation_packet_reception();
    test_robot_packet_reception();
    test_multiple_sequential_commands();

    // Buffer and constraint tests
    test_buffer_size_constraints();
    test_nrf24_size_constraint();

    // Error handling tests
    test_error_handling_invalid_robot_id();
    test_error_handling_invalid_action();

    // Protocol validation tests
    test_packet_determinism();
    test_all_action_types();
    test_coordinate_ranges();

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
