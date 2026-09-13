# Implementation Guide: Custom Binary Protocol Integration

This guide provides detailed, copy-paste ready code changes to integrate the custom binary protocol into your system.

## Overview

You'll need to make changes to 3 main components:
1. **Controller** (Go) - Encode commands
2. **Basestation** (C) - Decode & forward commands
3. **Robot** (C) - Decode & execute commands

Total time: ~2-3 hours

## Part 1: Controller (Go)

### File: `software/controller/internal/client/base_station_client.go`

#### Step 1.1: Add Import
Find the imports section at the top of the file and verify this line exists:
```go
import (
	"github.com/LiU-SeeGoals/controller/internal/action"
	// ... other imports
)
```

The `action` package import should already be there since you use `action.Action` interface.

#### Step 1.2: Replace `proto.Marshal()` with `action.EncodeCommand()`

**Location:** In the `sendCommands()` function (around line 111)

**BEFORE:**
```go
func (b *BaseStationClient) sendCommands() {
	for {
		b.queueMutex.Lock()
		if len(b.queue) == 0 {
			b.queueMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		cmd := b.queue[0]
		b.queue = b.queue[1:]
		b.queueMutex.Unlock()

		serializedCmd, _ := proto.Marshal(cmd)  // ← REPLACE THIS
		b.sendMessage(serializedCmd)
	}
}
```

**AFTER:**
```go
func (b *BaseStationClient) sendCommands() {
	for {
		b.queueMutex.Lock()
		if len(b.queue) == 0 {
			b.queueMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		cmd := b.queue[0]
		b.queue = b.queue[1:]
		b.queueMutex.Unlock()

		// Use custom binary encoder instead of protobuf
		encoded, err := action.EncodeCommand(cmd)
		if err != nil {
			fmt.Printf("Failed to encode command for robot %d: %v\n", cmd.RobotId, err)
			continue
		}
		b.sendMessage(encoded)
	}
}
```

#### Step 1.3: Remove Protobuf Import (Optional)

If protobuf is not used elsewhere in this file, remove:
```go
import (
	"google.golang.org/protobuf/proto"  // ← REMOVE THIS
	// ...
)
```

### Verification
```bash
cd software/controller
go build ./cmd/main.go
# Should compile without errors
```

---

## Part 2: Basestation Firmware (C)

### File: `firmware/basestation/Core/Src/com.c`

#### Step 2.1: Add Include

**Location:** At the top with other includes (around line 3)

**ADD:**
```c
#include "robot_command.h"  // ← ADD THIS LINE
```

Full includes should look like:
```c
/* Private includes */
#include "com.h"
#include <log.h>
#include <nrf24l01.h>
#include <nrf_helper_defines.h>
#include <parsed_vision.pb-c.h>
#include "robot_command.h"    // ← NEW
#include <protobuf-c.h>       // ← WILL REMOVE LATER (keep for now)
#include <robot_action.pb-c.h> // ← WILL REMOVE LATER (keep for now)
```

#### Step 2.2: Update `COM_ParsePacket()` Function

**Location:** Around line 197

**BEFORE:**
```c
UINT COM_ParsePacket(NX_PACKET* packet, PACKET_TYPE packet_type)
{
    UINT ret = NX_SUCCESS;

    switch (packet_type) {
    case ROBOT_COMMAND: {
        int length = packet->nx_packet_append_ptr - packet->nx_packet_prepend_ptr;

        if (length > 32) {
            LOG_ERROR("Robot command packet over 32 bytes (%d bytes)\r\n", length);
            ret = NX_INVALID_PACKET;
            return ret;
        }

        Command* command = NULL;
        command = command__unpack(NULL, length, packet->nx_packet_prepend_ptr);
        if (command == NULL) {
            LOG_ERROR("Invalid ethernet packet\r\n");
            return NX_INVALID_PACKET;
        }

        const ProtobufCEnumValue* enum_value = protobuf_c_enum_descriptor_get_value(&action_type__descriptor, command->command_id);

        uint8_t data[32];
        data[0] = 1;
        memcpy(data + 1, packet->nx_packet_prepend_ptr, length);

        COM_RF_Transmit(command->robot_id, data, length + 1);

        protobuf_c_message_free_unpacked(&command->base, NULL);
    } break;
    default:
        LOG_INFO("Unknown packet type: %d\r\n", packet_type);
        break;
    }

    if (ret != NX_SUCCESS) {
        LOG_WARNING("Failed to parse UDP packet\r\n");
    }

    return ret;
}
```

**AFTER:**
```c
UINT COM_ParsePacket(NX_PACKET* packet, PACKET_TYPE packet_type)
{
    UINT ret = NX_SUCCESS;

    switch (packet_type) {
    case ROBOT_COMMAND: {
        int length = packet->nx_packet_append_ptr - packet->nx_packet_prepend_ptr;

        // Custom binary protocol uses fixed 20-byte packets
        if (length != ROBOT_COMMAND_SIZE) {
            LOG_ERROR("Invalid robot command packet size: %d (expected %d)\r\n", 
                      length, ROBOT_COMMAND_SIZE);
            ret = NX_INVALID_PACKET;
            return ret;
        }

        // Decode the custom binary command
        RobotCommand command = {0};
        if (!robot_command_decode(packet->nx_packet_prepend_ptr, &command)) {
            LOG_ERROR("Failed to decode robot command\r\n");
            return NX_INVALID_PACKET;
        }

        // Validate robot ID
        if (command.robot_id < 1 || command.robot_id > 7) {
            LOG_ERROR("Invalid robot ID: %d\r\n", command.robot_id);
            return NX_INVALID_PACKET;
        }

        // Forward to robot via NRF24 (add message type prefix)
        uint8_t data[32];
        data[0] = 1;  // Message type: ACTION
        memcpy(data + 1, packet->nx_packet_prepend_ptr, length);

        COM_RF_Transmit(command.robot_id, data, length + 1);

    } break;
    default:
        LOG_INFO("Unknown packet type: %d\r\n", packet_type);
        break;
    }

    if (ret != NX_SUCCESS) {
        LOG_WARNING("Failed to parse UDP packet\r\n");
    }

    return ret;
}
```

#### Step 2.3: Update CMakeLists.txt

Find the `CMakeLists.txt` for the basestation project and:

**ADD** to `target_sources`:
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/src/robot_command.c
```

**ADD** to `target_include_directories`:
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/inc
```

Example section:
```cmake
add_executable(basestation_app
    Core/Src/com.c
    Core/Src/main.c
    # ... other sources
    ../../firmware/shared/src/robot_command.c  # ← ADD THIS
)

target_include_directories(basestation_app PUBLIC
    Core/Inc
    # ... other includes
    ../../firmware/shared/inc  # ← ADD THIS
)
```

### Verification
```bash
cd firmware/basestation
mkdir -p build
cd build
cmake ..
make
# Should compile without errors
```

---

## Part 3: Robot Firmware (C)

### File: `firmware/robot/CM7/Core/Src/com.c`

#### Step 3.1: Add Include

**Location:** At the top with other includes (around line 5)

**ADD:**
```c
#include "robot_command.h"  // ← ADD THIS LINE
```

Full includes should look like:
```c
#include "com.h"

/* Private includes */
#include "log.h"
#include "main.h"
#include "nav.h"
#include <nrf24l01.h>
#include <nrf_helper_defines.h>
#include "robot_command.h"     // ← NEW
#include <robot_action.pb-c.h> // ← WILL REMOVE LATER (keep for now)
#include <stdint.h>
#include <stdio.h>
```

#### Step 3.2: Update `parse_controller_packet()` Function

**Location:** Around line 264

**BEFORE:**
```c
static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    Command* cmd = NULL;
    cmd = command__unpack(NULL, len, payload);

    if (!cmd) {
        LOG_DEBUG("Decoding PB failed\r\n");
    } else {
        NAV_HandleCommand(cmd);
    }

    protobuf_c_message_free_unpacked((ProtobufCMessage*)cmd, NULL);
}
```

**AFTER:**
```c
static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    // Validate packet size (20 bytes for custom binary protocol)
    if (len != ROBOT_COMMAND_SIZE) {
        LOG_DEBUG("Invalid packet size: %d (expected %d)\r\n", len, ROBOT_COMMAND_SIZE);
        return;
    }

    // Decode the custom binary command
    RobotCommand cmd = {0};
    if (!robot_command_decode(payload, &cmd)) {
        LOG_DEBUG("Failed to decode robot command\r\n");
        return;
    }

    // Process the command
    NAV_HandleCommand(&cmd);
}
```

#### Step 3.3: Update `NAV_HandleCommand()` Signature

**File:** `firmware/robot/CM7/Core/Inc/nav.h`

**BEFORE:**
```c
void NAV_HandleCommand(Command* cmd);
```

**AFTER:**
```c
void NAV_HandleCommand(RobotCommand* cmd);
```

#### Step 3.4: Update `NAV_HandleCommand()` Implementation

**File:** `firmware/robot/CM7/Core/Src/nav.c`

**Find the function and update the signature:**

**BEFORE:**
```c
void NAV_HandleCommand(Command* cmd)
{
    if (!cmd) {
        LOG_ERROR("NAV_HandleCommand: cmd is NULL\r\n");
        return;
    }
    
    // Rest of implementation...
}
```

**AFTER:**
```c
void NAV_HandleCommand(RobotCommand* cmd)
{
    if (!cmd) {
        LOG_ERROR("NAV_HandleCommand: cmd is NULL\r\n");
        return;
    }
    
    // Rest of implementation (field names stay the same!)
    // cmd->action_type, cmd->robot_id, cmd->pos_x, etc.
}
```

**IMPORTANT:** Field names are identical, so no other code changes needed!

#### Step 3.5: Update CMakeLists.txt

Find the `CMakeLists.txt` for the robot project and:

**ADD** to `target_sources`:
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/src/robot_command.c
```

**ADD** to `target_include_directories`:
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/inc
```

Example section:
```cmake
add_executable(robot_firmware.elf
    Core/Src/com.c
    Core/Src/nav.c
    Core/Src/main.c
    # ... other sources
    ../../firmware/shared/src/robot_command.c  # ← ADD THIS
)

target_include_directories(robot_firmware.elf PUBLIC
    Core/Inc
    # ... other includes
    ../../firmware/shared/inc  # ← ADD THIS
)
```

### Verification
```bash
cd firmware/robot
mkdir -p build
cd build
cmake ..
make
# Should compile without errors
```

---

## Part 4: Testing

### Step 4.1: Run C Unit Tests

```bash
cd firmware/shared/test
gcc -I../inc -Wall -Wextra ../src/robot_command.c robot_command_test.c -o test && ./test
```

Expected output:
```
=== Robot Command Encoder/Decoder Tests ===

[TEST] Verify buffer size is correct
  ✓ PASS
[TEST] Encode/Decode simple kick command
  ✓ PASS
... (8 more tests)
[TEST] Verify byte layout and little-endian encoding
  ✓ PASS

=== Test Summary ===
Passed: 10
Failed: 0

ALL TESTS PASSED
```

### Step 4.2: Compile All Components

```bash
# Go
cd software/controller
go build ./cmd/main.go

# Basestation
cd firmware/basestation/build
cmake ..
make

# Robot
cd firmware/robot/build
cmake ..
make
```

All should compile without errors.

### Step 4.3: End-to-End Test

1. Deploy firmware to basestation and robot
2. Send a command from controller:
   ```bash
   # In controller
   # Send a KICK command to robot 1 with position (1000, 2000)
   ```
3. Monitor basestation logs:
   ```
   Should see: Received UDP packet, decoded robot command, forwarded via NRF24
   ```
4. Monitor robot logs:
   ```
   Should see: Received NRF24 packet, decoded command, executing KICK action
   ```

---

## Part 5: Cleanup (Optional, After Verification)

Once everything works and you've tested in production for a few days:

### Remove Protobuf Includes

**Basestation** (`firmware/basestation/Core/Src/com.c`):
```c
// REMOVE these lines:
#include <protobuf-c.h>
#include <robot_action.pb-c.h>
#include <parsed_vision.pb-c.h>  // If not used elsewhere
```

**Robot** (`firmware/robot/CM7/Core/Src/com.c`):
```c
// REMOVE these lines:
#include <robot_action.pb-c.h>
```

### Remove Protobuf CMake Configuration

In CMakeLists.txt files:
- Remove any `find_package(Protobuf ...)`
- Remove protobuf linking
- Remove .pb-c.h/.pb-c.c generation rules

---

## Troubleshooting

### Compilation Errors

| Error | Solution |
|-------|----------|
| `robot_command.h: No such file` | Add `firmware/shared/inc` to CMakeLists.txt include paths |
| `undefined reference to robot_command_encode` | Add `firmware/shared/src/robot_command.c` to CMakeLists.txt sources |
| `ROBOT_COMMAND_SIZE undeclared` | Verify `#include "robot_command.h"` is present |

### Runtime Errors

| Error | Solution |
|-------|----------|
| "Invalid packet size: X (expected 20)" | Ensure encoder is generating 20 bytes, decoder receives exactly 20 bytes |
| "Failed to decode robot command" | Check input buffer is valid, not corrupted during transmission |
| Segfault on NAV_HandleCommand | Verify signature changed to `RobotCommand* cmd` (not `Command*`) |

### Field Access Issues

Remember: **Field names are identical!** No need to change field access:
- `cmd->action_type` ✓
- `cmd->robot_id` ✓
- `cmd->pos_x` ✓ (now int16 instead of int32)
- `cmd->direction_x` ✓
- etc.

---

## Summary

**Changes Made:**
1. **Go** (3 lines): Replace `proto.Marshal()` with `action.EncodeCommand()`
2. **Basestation C** (15 lines): Add include, replace decode function
3. **Robot C** (20 lines): Add include, replace decode function, update signature
4. **Build System** (4 additions): Add robot_command.c and include path to CMakeLists.txt

**Total:** ~50 lines of code changes + build system updates

**Time:** 2-3 hours including testing and verification

All changes are backward compatible up to compilation - you can keep both implementations side-by-side with a feature flag if needed.

---

## Next Steps

1. ✓ Read this guide
2. ✓ Make changes to Go controller
3. ✓ Make changes to basestation firmware
4. ✓ Make changes to robot firmware
5. ✓ Update CMakeLists.txt files
6. ✓ Run unit tests
7. ✓ Compile all components
8. ✓ Deploy and test end-to-end
9. ✓ Monitor logs for issues
10. ✓ Deploy to production devices

For questions, refer to `firmware/shared/ROBOT_COMMAND.md` or `SIZE_ANALYSIS.md`.
