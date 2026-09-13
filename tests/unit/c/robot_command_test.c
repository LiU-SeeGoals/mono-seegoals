/**
 * @file robot_command_test.c
 * @brief Unit tests for robot_command encoding/decoding
 *
 * To compile and run:
 *   gcc -I../inc ../src/robot_command.c robot_command_test.c -o test && ./test
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <assert.h>

#include "robot_command.h"

#define UNUSED(x) (void)(x)

// Color codes for test output
#define RED     "\x1b[31m"
#define GREEN   "\x1b[32m"
#define YELLOW  "\x1b[33m"
#define RESET   "\x1b[0m"

static int tests_passed = 0;
static int tests_failed = 0;

#define TEST(name) printf(YELLOW "[TEST]" RESET " %s\n", name)
#define PASS() printf(GREEN "  ✓ PASS\n" RESET); tests_passed++
#define FAIL(msg) printf(RED "  ✗ FAIL: %s\n" RESET, msg); tests_failed++

void test_encode_decode_simple_kick()
{
    TEST("Encode/Decode simple kick command");

    RobotCommand cmd_orig = {
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
        .orientation_w = 455,  // 45.5 degrees in tenths
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd_decoded = {0};

    // Encode
    if (!robot_command_encode(&cmd_orig, buf)) {
        FAIL("Encode failed");
        return;
    }

    // Decode
    if (!robot_command_decode(buf, &cmd_decoded)) {
        FAIL("Decode failed");
        return;
    }

    // Verify
    if (cmd_decoded.action_type != cmd_orig.action_type) {
        FAIL("action_type mismatch");
        return;
    }
    if (cmd_decoded.robot_id != cmd_orig.robot_id) {
        FAIL("robot_id mismatch");
        return;
    }
    if (cmd_decoded.kick_speed != cmd_orig.kick_speed) {
        FAIL("kick_speed mismatch");
        return;
    }
    if (cmd_decoded.pos_x != cmd_orig.pos_x) {
        FAIL("pos_x mismatch");
        return;
    }
    if (cmd_decoded.pos_y != cmd_orig.pos_y) {
        FAIL("pos_y mismatch");
        return;
    }
    if (cmd_decoded.dest_x != cmd_orig.dest_x) {
        FAIL("dest_x mismatch");
        return;
    }
    if (cmd_decoded.dest_y != cmd_orig.dest_y) {
        FAIL("dest_y mismatch");
        return;
    }
    if (cmd_decoded.direction_x != cmd_orig.direction_x) {
        FAIL("direction_x mismatch");
        return;
    }
    if (cmd_decoded.direction_y != cmd_orig.direction_y) {
        FAIL("direction_y mismatch");
        return;
    }
    if (cmd_decoded.angular_vel != cmd_orig.angular_vel) {
        FAIL("angular_vel mismatch");
        return;
    }
    if (cmd_decoded.orientation_w != cmd_orig.orientation_w) {
        FAIL("orientation_w mismatch");
        return;
    }

    PASS();
}

void test_encode_decode_negative_coords()
{
    TEST("Encode/Decode with negative coordinates");

    RobotCommand cmd_orig = {
        .action_type = ACTION_TYPE_MOVE_TO,
        .robot_id = 3,
        .kick_speed = 0,
        .pos_x = -1000,
        .pos_y = -500,
        .dest_x = 1000,
        .dest_y = 1500,
        .direction_x = -100,
        .direction_y = -50,
        .angular_vel = -45,
        .orientation_w = 1800,  // 180 degrees in tenths
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd_decoded = {0};

    if (!robot_command_encode(&cmd_orig, buf)) {
        FAIL("Encode failed");
        return;
    }

    if (!robot_command_decode(buf, &cmd_decoded)) {
        FAIL("Decode failed");
        return;
    }

    if (cmd_decoded.pos_x != cmd_orig.pos_x ||
        cmd_decoded.pos_y != cmd_orig.pos_y ||
        cmd_decoded.dest_x != cmd_orig.dest_x ||
        cmd_decoded.dest_y != cmd_orig.dest_y ||
        cmd_decoded.angular_vel != cmd_orig.angular_vel) {
        FAIL("Negative coordinate mismatch");
        return;
    }

    PASS();
}

void test_encode_decode_zeros()
{
    TEST("Encode/Decode with all zeros");

    RobotCommand cmd_orig = {0};
    cmd_orig.robot_id = 5;  // At least robot_id needs to be valid for encode

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd_decoded = {0};

    if (!robot_command_encode(&cmd_orig, buf)) {
        FAIL("Encode failed");
        return;
    }

    if (!robot_command_decode(buf, &cmd_decoded)) {
        FAIL("Decode failed");
        return;
    }

    if (cmd_decoded.robot_id != cmd_orig.robot_id) {
        FAIL("robot_id mismatch");
        return;
    }

    PASS();
}

void test_direction_clamping()
{
    TEST("Direction value clamping");

    RobotCommand cmd_orig = {
        .action_type = ACTION_TYPE_MOVE,
        .robot_id = 2,
        .kick_speed = 500,
        .pos_x = 500,
        .pos_y = 500,
        .dest_x = 1000,
        .dest_y = 1000,
        .direction_x = 1000,  // Should clamp to 100
        .direction_y = -1000, // Should clamp to -100
        .angular_vel = 180,
        .orientation_w = 905,  // 90.5 degrees
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd_decoded = {0};

    if (!robot_command_encode(&cmd_orig, buf)) {
        FAIL("Encode failed");
        return;
    }

    if (!robot_command_decode(buf, &cmd_decoded)) {
        FAIL("Decode failed");
        return;
    }

    // After encode/decode, direction values should be clamped
    if (cmd_decoded.direction_x != 100) {
        printf(RED "  ✗ FAIL: direction_x should be 100, got %d\n" RESET, cmd_decoded.direction_x);
        tests_failed++;
        return;
    }

    if (cmd_decoded.direction_y != -100) {
        printf(RED "  ✗ FAIL: direction_y should be -100, got %d\n" RESET, cmd_decoded.direction_y);
        tests_failed++;
        return;
    }

    PASS();
}

void test_orientation_clamping()
{
    TEST("Orientation W clamping [0, 3600)");

    RobotCommand cmd_orig = {
        .action_type = ACTION_TYPE_ROTATE,
        .robot_id = 7,
        .kick_speed = 0,
        .pos_x = 0,
        .pos_y = 0,
        .dest_x = 0,
        .dest_y = 0,
        .direction_x = 0,
        .direction_y = 0,
        .angular_vel = 0,
        .orientation_w = 5000,  // > 3600, should clamp
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];
    RobotCommand cmd_decoded = {0};

    if (!robot_command_encode(&cmd_orig, buf)) {
        FAIL("Encode failed");
        return;
    }

    if (!robot_command_decode(buf, &cmd_decoded)) {
        FAIL("Decode failed");
        return;
    }

    if (cmd_decoded.orientation_w >= 3600) {
        printf(RED "  ✗ FAIL: orientation_w should be < 3600, got %d\n" RESET, cmd_decoded.orientation_w);
        tests_failed++;
        return;
    }

    PASS();
}

void test_encode_invalid_robot_id()
{
    TEST("Reject invalid robot ID");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_STOP,
        .robot_id = 0,  // Invalid, must be 1-7
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];

    if (robot_command_encode(&cmd, buf)) {
        FAIL("Should reject robot_id = 0");
        return;
    }

    cmd.robot_id = 8;  // Invalid, must be 1-7
    if (robot_command_encode(&cmd, buf)) {
        FAIL("Should reject robot_id = 8");
        return;
    }

    PASS();
}

void test_encode_invalid_action_type()
{
    TEST("Reject invalid action type");

    RobotCommand cmd = {
        .action_type = 10,  // Invalid, must be 0-5
        .robot_id = 1,
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];

    if (robot_command_encode(&cmd, buf)) {
        FAIL("Should reject action_type = 10");
        return;
    }

    PASS();
}

void test_buffer_size()
{
    TEST("Verify buffer size is correct");

    if (ROBOT_COMMAND_SIZE != 20) {
        printf(RED "  ✗ FAIL: Expected size 20, got %d\n" RESET, ROBOT_COMMAND_SIZE);
        tests_failed++;
        return;
    }

    PASS();
}

void test_all_action_types()
{
    TEST("Test all action types");

    const uint8_t action_types[] = {
        ACTION_TYPE_KICK,
        ACTION_TYPE_STOP,
        ACTION_TYPE_MOVE_TO,
        ACTION_TYPE_INIT,
        ACTION_TYPE_MOVE,
        ACTION_TYPE_ROTATE,
    };

    for (size_t i = 0; i < sizeof(action_types) / sizeof(action_types[0]); i++) {
        RobotCommand cmd_orig = {
            .action_type = action_types[i],
            .robot_id = 1,
        };

        uint8_t buf[ROBOT_COMMAND_SIZE];
        RobotCommand cmd_decoded = {0};

        if (!robot_command_encode(&cmd_orig, buf)) {
            printf(RED "  ✗ FAIL: Encode failed for action_type %d\n" RESET, action_types[i]);
            tests_failed++;
            return;
        }

        if (!robot_command_decode(buf, &cmd_decoded)) {
            printf(RED "  ✗ FAIL: Decode failed for action_type %d\n" RESET, action_types[i]);
            tests_failed++;
            return;
        }

        if (cmd_decoded.action_type != action_types[i]) {
            printf(RED "  ✗ FAIL: action_type mismatch for type %d\n" RESET, action_types[i]);
            tests_failed++;
            return;
        }
    }

    PASS();
}

void test_byte_layout()
{
    TEST("Verify byte layout and little-endian encoding");

    RobotCommand cmd = {
        .action_type = ACTION_TYPE_KICK,
        .robot_id = 3,  // Valid robot_id (1-7)
        .kick_speed = 0x1234,  // Little-endian: 34 12
        .pos_x = 0x05DC,   // Little-endian: DC 05
    };

    uint8_t buf[ROBOT_COMMAND_SIZE];

    if (!robot_command_encode(&cmd, buf)) {
        FAIL("Encode failed");
        return;
    }

    // Verify byte layout
    if (buf[0] != ACTION_TYPE_KICK) {
        FAIL("Byte 0 (action_type) mismatch");
        return;
    }

    if (buf[1] != 3) {
        FAIL("Byte 1 (robot_id) mismatch");
        return;
    }

    if (buf[2] != 0x34 || buf[3] != 0x12) {
        FAIL("Bytes 2-3 (kick_speed) little-endian mismatch");
        return;
    }

    if (buf[4] != 0xDC || buf[5] != 0x05) {
        FAIL("Bytes 4-5 (pos_x) little-endian mismatch");
        return;
    }

    PASS();
}

int main()
{
    printf("\n" YELLOW "=== Robot Command Encoder/Decoder Tests ===" RESET "\n\n");

    test_buffer_size();
    test_encode_decode_simple_kick();
    test_encode_decode_negative_coords();
    test_encode_decode_zeros();
    test_direction_clamping();
    test_orientation_clamping();
    test_encode_invalid_robot_id();
    test_encode_invalid_action_type();
    test_all_action_types();
    test_byte_layout();

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
