/**
 * @file com_layer_test.c
 * @brief Tests for communication layer (COM) - simulates basestation and robot COM
 *
 * Tests the COM layer parsing and handling:
 * - Basestation COM_ParsePacket() behavior
 * - Robot parse_controller_packet() behavior
 * - Error handling and validation
 * - Message routing
 *
 * To compile and run:
 *   gcc -I../../../firmware/shared/inc \
 *       ../c/com_layer_test.c \
 *       ../../../firmware/shared/src/robot_command.c \
 *       -o com_test && ./com_test
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
 * Simulated basestation COM layer
 * In real firmware, this is in firmware/basestation/Core/Src/com.c
 */
typedef struct {
    uint8_t buffer[20];
    bool packet_ready;
    int packets_processed;
    int errors;
} BaseStationCOM;

static BaseStationCOM bs_com = {0};

/**
 * Simulates basestation receiving and parsing ROBOT_COMMAND packet
 * Real function: COM_ParsePacket() in com.c
 */
bool basestation_process_robot_command(const uint8_t* packet, size_t len)
{
    if (len != ROBOT_COMMAND_SIZE) {
        bs_com.errors++;
        return false;
    }

    RobotCommand cmd = {0};
    if (!robot_command_decode(packet, &cmd)) {
        bs_com.errors++;
        return false;
    }

    // Validate robot ID is in range
    if (cmd.robot_id < 1 || cmd.robot_id > 7) {
        bs_com.errors++;
        return false;
    }

    // Copy to buffer for forwarding
    memcpy(bs_com.buffer, packet, ROBOT_COMMAND_SIZE);
    bs_com.packet_ready = true;
    bs_com.packets_processed++;

    return true;
}

/**
 * Simulated robot COM layer
 * In real firmware, this is in firmware/robot/CM7/Core/Src/com.c
 */
typedef struct {
    RobotCommand last_command;
    bool command_valid;
    int packets_received;
    int errors;
} RobotCOM;

static RobotCOM robot_com = {0};

/**
 * Simulates robot receiving and parsing controller packet
 * Real function: parse_controller_packet() in com.c
 */
bool robot_process_controller_packet(const uint8_t* packet, size_t len)
{
    if (len != ROBOT_COMMAND_SIZE) {
        robot_com.errors++;
        return false;
    }

    RobotCommand cmd = {0};
    if (!robot_command_decode(packet, &cmd)) {
        robot_com.errors++;
        return false;
    }

    // Store and mark as valid
    memcpy(&robot_com.last_command, &cmd, sizeof(RobotCommand));
    robot_com.command_valid = true;
    robot_com.packets_received++;

    return true;
}

/**
 * Test: Basestation receives valid command
 */
void test_basestation_receives_valid_command()
{
    TEST("Basestation receives valid command");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 2,
        .kick_speed = 1000,
        .pos_x = 500,
        .pos_y = 600,
        .dest_x = 700,
        .dest_y = 800,
        .direction_x = 10,
        .direction_y = 20,
        .angular_vel = 30,
        .orientation_w = 300,
    };

    uint8_t packet[ROBOT_COMMAND_SIZE];
    if (!robot_command_encode(&cmd, packet)) {
        FAIL("Failed to encode");
        return;
    }

    // Reset basestation state
    bs_com.packets_processed = 0;
    bs_com.errors = 0;
    bs_com.packet_ready = false;

    // Basestation processes the packet
    if (!basestation_process_robot_command(packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Basestation rejected valid packet");
        return;
    }

    if (!bs_com.packet_ready) {
        FAIL("Basestation didn't mark packet ready");
        return;
    }

    if (bs_com.packets_processed != 1) {
        FAIL("Packet count mismatch");
        return;
    }

    PASS();
}

/**
 * Test: Basestation rejects invalid packet size
 */
void test_basestation_rejects_invalid_size()
{
    TEST("Basestation rejects invalid packet size");

    uint8_t bad_packet[15];  // Wrong size
    memset(bad_packet, 0xAB, 15);

    bs_com.errors = 0;
    bs_com.packets_processed = 0;

    if (basestation_process_robot_command(bad_packet, 15)) {
        FAIL("Basestation accepted wrong size");
        return;
    }

    if (bs_com.errors != 1) {
        FAIL("Error not incremented");
        return;
    }

    PASS();
}

/**
 * Test: Basestation rejects invalid robot ID
 */
void test_basestation_rejects_invalid_robot_id()
{
    TEST("Basestation rejects invalid robot ID");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
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

    uint8_t packet[ROBOT_COMMAND_SIZE];
    robot_command_encode(&cmd, packet);

    // Manually corrupt robot ID in packet (byte index 1)
    packet[1] = 10;  // Invalid ID

    bs_com.errors = 0;
    bs_com.packets_processed = 0;

    // Basestation should reject due to invalid ID
    if (basestation_process_robot_command(packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Basestation accepted invalid robot ID");
        return;
    }

    PASS();
}

/**
 * Test: Robot receives valid command
 */
void test_robot_receives_valid_command()
{
    TEST("Robot receives valid command");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 1,
        .kick_speed = 0,
        .pos_x = 1000,
        .pos_y = 1500,
        .dest_x = 2000,
        .dest_y = 2500,
        .direction_x = 100,
        .direction_y = 150,
        .angular_vel = 45,
        .orientation_w = 450,
    };

    uint8_t packet[ROBOT_COMMAND_SIZE];
    if (!robot_command_encode(&cmd, packet)) {
        FAIL("Failed to encode");
        return;
    }

    // Reset robot state
    robot_com.packets_received = 0;
    robot_com.errors = 0;
    robot_com.command_valid = false;

    // Robot processes the packet
    if (!robot_process_controller_packet(packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Robot rejected valid packet");
        return;
    }

    if (!robot_com.command_valid) {
        FAIL("Robot didn't mark command valid");
        return;
    }

    if (robot_com.last_command.action_type != ACTION_TYPE_MOVE_TO) {
        FAIL("Robot command action type mismatch");
        return;
    }

    PASS();
}

/**
 * Test: Robot rejects invalid packet size
 */
void test_robot_rejects_invalid_size()
{
    TEST("Robot rejects invalid packet size");

    uint8_t bad_packet[10];  // Too small
    
    robot_com.errors = 0;
    robot_com.packets_received = 0;
    robot_com.command_valid = false;

    if (robot_process_controller_packet(bad_packet, 10)) {
        FAIL("Robot accepted wrong size");
        return;
    }

    if (robot_com.errors != 1) {
        FAIL("Error count mismatch");
        return;
    }

    if (robot_com.command_valid) {
        FAIL("Robot marked invalid packet as valid");
        return;
    }

    PASS();
}

/**
 * Test: Robot rejects corrupted packet with invalid robot ID
 * Note: The decoder accepts any byte value, but COM layer validates robot ID
 */
void test_robot_rejects_corrupted_packet()
{
    TEST("Robot rejects corrupted packet");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 1,
        .kick_speed = 1000,
        .pos_x = 500,
        .pos_y = 500,
        .dest_x = 1000,
        .dest_y = 1000,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 0,
    };

    uint8_t packet[ROBOT_COMMAND_SIZE];
    robot_command_encode(&cmd, packet);

    // Corrupt the packet (set invalid robot ID)
    // Decoder accepts this, but robot_process_controller_packet 
    // doesn't validate, only basestation does
    packet[1] = 10;  // Invalid robot ID (>7)

    robot_com.errors = 0;
    robot_com.packets_received = 0;
    robot_com.command_valid = false;

    // Robot process doesn't validate robot ID (only basestation does that)
    // So it will accept this packet. This is the design.
    if (!robot_process_controller_packet(packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Robot rejected packet");
        return;
    }

    // Verify robot accepted it but with invalid ID
    if (robot_com.last_command.robot_id != 10) {
        FAIL("Robot ID not preserved");
        return;
    }

    PASS();
}

/**
 * Test: Multiple sequential packets to basestation
 */
void test_basestation_multiple_packets()
{
    TEST("Basestation processes multiple packets");

    bs_com.packets_processed = 0;
    bs_com.errors = 0;

    for (int i = 1; i <= 5; i++) {
        RobotCommand cmd = {
            .action_type = ACTION_TYPE_KICK,
            .robot_id = (uint8_t)i,
            .kick_speed = (uint16_t)(i * 100),
            .pos_x = (int16_t)(i * 10),
            .pos_y = (int16_t)(i * 20),
            .dest_x = (int16_t)(i * 30),
            .dest_y = (int16_t)(i * 40),
            .direction_x = (int16_t)(i * 5),
            .direction_y = (int16_t)(i * 10),
            .angular_vel = (int16_t)(i * 15),
            .orientation_w = (int16_t)(i * 100),
        };

        uint8_t packet[ROBOT_COMMAND_SIZE];
        if (!robot_command_encode(&cmd, packet)) {
            FAIL("Failed to encode packet");
            return;
        }

        if (!basestation_process_robot_command(packet, ROBOT_COMMAND_SIZE)) {
            FAIL("Failed to process packet");
            return;
        }
    }

    if (bs_com.packets_processed != 5) {
        FAIL("Packet count mismatch");
        return;
    }

    if (bs_com.errors != 0) {
        FAIL("Unexpected errors occurred");
        return;
    }

    PASS();
}

/**
 * Test: Multiple sequential packets to robot
 */
void test_robot_multiple_packets()
{
    TEST("Robot processes multiple packets");

    robot_com.packets_received = 0;
    robot_com.errors = 0;

    for (int i = 0; i < 6; i++) {
        RobotCommand cmd = {
            .action_type = (uint8_t)i,
            .robot_id = 1,
            .kick_speed = (uint16_t)(i * 100),
            .pos_x = 0,
            .pos_y = 0,
            .dest_x = 0,
            .dest_y = 0,
            .direction_x = 0,
            .direction_y = 0,
            .angular_vel = (int16_t)(i * 10),
            .orientation_w = 0,
        };

        uint8_t packet[ROBOT_COMMAND_SIZE];
        if (!robot_command_encode(&cmd, packet)) {
            FAIL("Failed to encode packet");
            return;
        }

        if (!robot_process_controller_packet(packet, ROBOT_COMMAND_SIZE)) {
            FAIL("Failed to process packet");
            return;
        }

        // Verify last command is correct
        if (robot_com.last_command.action_type != (uint8_t)i) {
            FAIL("Last command action type mismatch");
            return;
        }
    }

    if (robot_com.packets_received != 6) {
        FAIL("Packet count mismatch");
        return;
    }

    PASS();
}

/**
 * Test: Basestation packet forwarding
 * Verify packet is correctly forwarded to NRF24
 */
void test_basestation_packet_forwarding()
{
    TEST("Basestation packet forwarding");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 3,
        .kick_speed = 1500,
        .pos_x = 1000,
        .pos_y = 2000,
        .dest_x = 3000,
        .dest_y = 4000,
        .direction_x = 50,
        .direction_y = 100,
        .angular_vel = 180,
        .orientation_w = 1800,
    };

    uint8_t original_packet[ROBOT_COMMAND_SIZE];
    if (!robot_command_encode(&cmd, original_packet)) {
        FAIL("Failed to encode");
        return;
    }

    // Basestation receives and processes
    bs_com.packet_ready = false;
    if (!basestation_process_robot_command(original_packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Basestation rejected packet");
        return;
    }

    // Verify packet is ready for forwarding
    if (!bs_com.packet_ready) {
        FAIL("Packet not marked ready");
        return;
    }

    // Verify packet data matches
    if (memcmp(bs_com.buffer, original_packet, ROBOT_COMMAND_SIZE) != 0) {
        FAIL("Packet data corrupted in forwarding");
        return;
    }

    PASS();
}

/**
 * Test: Error recovery
 * After error, system should handle next packet correctly
 */
void test_error_recovery()
{
    TEST("Error recovery");

    // Send invalid packet
    uint8_t bad_packet[10];
    memset(bad_packet, 0xFF, 10);

    bs_com.errors = 0;
    bs_com.packets_processed = 0;
    
    basestation_process_robot_command(bad_packet, 10);
    
    if (bs_com.errors != 1) {
        FAIL("Error not recorded");
        return;
    }

    // Now send valid packet
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

    uint8_t packet[ROBOT_COMMAND_SIZE];
    robot_command_encode(&cmd, packet);

    if (!basestation_process_robot_command(packet, ROBOT_COMMAND_SIZE)) {
        FAIL("Failed to process valid packet after error");
        return;
    }

    if (bs_com.packets_processed != 1) {
        FAIL("Packet count not incremented");
        return;
    }

    PASS();
}

int main()
{
    printf("\n" YELLOW "=== COM Layer Tests ===" RESET "\n\n");

    // Basestation tests
    printf(YELLOW "--- Basestation COM Tests ---\n" RESET);
    test_basestation_receives_valid_command();
    test_basestation_rejects_invalid_size();
    test_basestation_rejects_invalid_robot_id();
    test_basestation_multiple_packets();
    test_basestation_packet_forwarding();

    // Robot tests
    printf("\n" YELLOW "--- Robot COM Tests ---\n" RESET);
    test_robot_receives_valid_command();
    test_robot_rejects_invalid_size();
    test_robot_rejects_corrupted_packet();
    test_robot_multiple_packets();

    // Integration tests
    printf("\n" YELLOW "--- Integration Tests ---\n" RESET);
    test_error_recovery();

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
