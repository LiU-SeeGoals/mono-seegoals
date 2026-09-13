# Test Suite Quick Start

## Run All Tests
```bash
cd /home/anton/projects/mono-seegoals/tests
bash run_all_tests.sh
```

Expected output: `✓ ALL TESTS PASSED - READY TO DEPLOY`

## Test Files Location
- **Protocol tests**: `tests/unit/c/robot_command_test.c`
- **Firmware integration tests**: `tests/unit/c/firmware_integration_test.c`
- **COM layer tests**: `tests/unit/c/com_layer_test.c`
- **NAV layer tests**: `tests/unit/c/nav_layer_test.c`
- **Test runner**: `tests/run_all_tests.sh`

## Quick Test Commands

### Run protocol tests only
```bash
cd tests/unit/c
gcc -I../../../firmware/shared/inc -Wall -Wextra \
    robot_command_test.c \
    ../../../firmware/shared/src/robot_command.c \
    -o test && ./test
```

### Run firmware integration tests only
```bash
cd tests/unit/c
gcc -I../../../firmware/shared/inc -Wall -Wextra \
    firmware_integration_test.c \
    ../../../firmware/shared/src/robot_command.c \
    -o test && ./test
```

### Run COM layer tests only
```bash
cd tests/unit/c
gcc -I../../../firmware/shared/inc -Wall -Wextra \
    com_layer_test.c \
    ../../../firmware/shared/src/robot_command.c \
    -o test && ./test
```

### Run NAV layer tests only
```bash
cd tests/unit/c
gcc -I../../../firmware/shared/inc -Wall -Wextra \
    nav_layer_test.c \
    ../../../firmware/shared/src/robot_command.c \
    -o test && ./test
```

## Test Summary
- **Protocol Encoder/Decoder**: 10 tests ✓
- **Firmware Integration**: 10 tests ✓
- **COM Layer**: 10 tests ✓
- **NAV Layer**: 11 tests ✓
- **Total**: 41 tests ✓

## What's Tested

### Protocol Layer
✓ Encoding/decoding all data types  
✓ int16 boundary values (min/max)  
✓ Value clamping (direction, orientation)  
✓ All 6 action types  
✓ Robot ID validation (1-7)  

### Firmware Integration
✓ Basestation packet handling  
✓ Robot packet handling  
✓ NRF24 size compliance  
✓ Error handling and recovery  

### Communication Layer
✓ Basestation COM (receiving, validating, forwarding)  
✓ Robot COM (receiving, parsing, storing)  
✓ Multiple sequential packets  
✓ Error conditions  

### Navigation Layer
✓ All 6 command types (KICK, STOP, MOVE_TO, MOVE, ROTATE, INIT)  
✓ State transitions  
✓ Error handling  
✓ Sequential command chains  

## Continuous Integration
The test runner has proper exit codes:
- Exit 0: All tests passed ✓
- Exit 1: Some tests failed ✗

Perfect for CI/CD pipelines!

## More Information
See `tests/README.md` for detailed documentation.
