# Quick Integration Guide: Custom Binary Protocol

**UPDATED: 20-byte protocol (was 30 bytes)**

## TL;DR - 5 Minute Integration

### 1. Copy Files

Files are already in place:
```
firmware/shared/inc/robot_command.h
firmware/shared/src/robot_command.c
software/controller/internal/action/command_encoder.go
```

### 2. Go/Controller Changes

**File:** `software/controller/internal/client/base_station_client.go`

Replace lines 111-112:
```go
// OLD:
serializedCmd, _ := proto.Marshal(cmd)

// NEW:
encoded, err := action.EncodeCommand(cmd)
if err != nil {
    fmt.Printf("Failed to encode command: %v\n", err)
    continue
}
serializedCmd = encoded
```

### 3. Basestation Firmware Changes

**File:** `firmware/basestation/Core/Src/com.c`

**Add include at top:**
```c
#include "robot_command.h"
```

**Replace lines 211-226:**
```c
// OLD:
Command* command = NULL;
command = command__unpack(NULL, length, packet->nx_packet_prepend_ptr);
if (command == NULL) {
    LOG_ERROR("Invalid ethernet packet\r\n");
    return NX_INVALID_PACKET;
}
// ... use command ...
protobuf_c_message_free_unpacked(&command->base, NULL);

// NEW:
RobotCommand command = {0};
if (!robot_command_decode(packet->nx_packet_prepend_ptr, &command)) {
    LOG_ERROR("Invalid packet\r\n");
    return NX_INVALID_PACKET;
}
// ... use command ...
// (no cleanup needed)
```

### 4. Robot Firmware Changes

**File:** `firmware/robot/CM7/Core/Src/com.c`

**Add include at top:**
```c
#include "robot_command.h"
```

**Replace lines 264-276:**
```c
// OLD:
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

// NEW:
static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    if (len != ROBOT_COMMAND_SIZE) {
        LOG_DEBUG("Invalid packet size: %d\r\n", len);
        return;
    }
    
    RobotCommand cmd = {0};
    if (!robot_command_decode(payload, &cmd)) {
        LOG_DEBUG("Decoding failed\r\n");
        return;
    }
    NAV_HandleCommand(&cmd);
}
```

### 5. Update NAV_HandleCommand Signature

**File:** `firmware/robot/CM7/Core/Inc/nav.h`

**Change:**
```c
// OLD:
void NAV_HandleCommand(Command* cmd);

// NEW:
void NAV_HandleCommand(RobotCommand* cmd);
```

**File:** `firmware/robot/CM7/Core/Src/nav.c`

**Update function implementation:**
All field accesses remain the same since struct field names are identical.

### 6. Update CMakeLists.txt

**Basestation:**
```cmake
# Add to target_sources:
firmware/shared/src/robot_command.c

# Add to target_include_directories:
firmware/shared/inc
```

**Robot:**
```cmake
# Add to target_sources:
firmware/shared/src/robot_command.c

# Add to target_include_directories:
firmware/shared/inc
```

### 7. Remove Protobuf Includes (Optional)

Once everything works, remove these lines:

**Basestation** (`firmware/basestation/Core/Src/com.c`):
```c
#include <protobuf-c.h>
#include <robot_action.pb-c.h>
```

**Robot** (`firmware/robot/CM7/Core/Src/com.c`):
```c
#include <robot_action.pb-c.h>
```

## Verification Checklist

- [ ] Encoder compiles without errors (Go)
- [ ] Decoder compiles without errors (C basestation & robot)
- [ ] Unit tests pass for both C and Go
- [ ] Basestation receives UDP packets and decodes correctly
- [ ] Robot receives NRF24 packets and executes commands
- [ ] Test all 6 action types work
- [ ] Monitor for any decoding errors in logs

## Testing with Actual Commands

```bash
# Run C tests:
cd firmware/shared/test
gcc -I../inc ../src/robot_command.c robot_command_test.c -o test && ./test

# Run Go tests (requires Go toolchain):
cd software/controller
go test -v ./internal/action -run TestEncode
```

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| "Compilation error: robot_command.h not found" | Missing include path | Add `firmware/shared/inc` to CMakeLists.txt |
| "Decoding always fails" | Wrong buffer size passed | Ensure buffer is exactly 20 bytes |
| "Robot ID out of range" | Robot ID not 1-7 | Verify COM_Get_ID() returns 1-7 |
| "Segfault on NAV_HandleCommand" | Signature mismatch | Update `void NAV_HandleCommand(RobotCommand*)` |
| "Direction values wrong" | Didn't update field accesses | Direction_x stays as `direction_x`, same names |

## Rollback Plan

If something breaks:

1. **Keep protobuf headers** during integration
2. **Revert changes** to use old protobuf code
3. **Switch back** without full recompile

Example (use feature flags):
```c
#define USE_CUSTOM_PROTOCOL 0  // Switch back to 1 when ready

#if USE_CUSTOM_PROTOCOL
    // New code
#else
    // Old protobuf code
#endif
```

---

**Protocol Size:** 20 bytes (down from 30!)  
**Estimated time:** 20-30 minutes  
**Risk level:** Low (feature-flagged)  
**Testing required:** ~1 hour
