# Quick Reference: Code Changes Checklist

## Files to Modify

### 1. Go - Controller
**File:** `software/controller/internal/client/base_station_client.go`

**Location:** Line ~111 in `sendCommands()` function

```diff
- serializedCmd, _ := proto.Marshal(cmd)
+ encoded, err := action.EncodeCommand(cmd)
+ if err != nil {
+     fmt.Printf("Failed to encode: %v\n", err)
+     continue
+ }
+ serializedCmd = encoded
```

**Total changes:** 1 location, 5 lines (was 1 line)

---

### 2. Basestation - Add Include
**File:** `firmware/basestation/Core/Src/com.c`

**Location:** Line ~5 in includes section

```diff
+ #include "robot_command.h"
```

**Total changes:** 1 location, 1 line

---

### 3. Basestation - Update Parser
**File:** `firmware/basestation/Core/Src/com.c`

**Location:** Line ~197 in `COM_ParsePacket()` function

**REPLACE** entire ROBOT_COMMAND case (lines 202-226):

```c
case ROBOT_COMMAND: {
    int length = packet->nx_packet_append_ptr - packet->nx_packet_prepend_ptr;

    if (length != ROBOT_COMMAND_SIZE) {
        LOG_ERROR("Invalid robot command packet size: %d (expected %d)\r\n", 
                  length, ROBOT_COMMAND_SIZE);
        ret = NX_INVALID_PACKET;
        return ret;
    }

    RobotCommand command = {0};
    if (!robot_command_decode(packet->nx_packet_prepend_ptr, &command)) {
        LOG_ERROR("Failed to decode robot command\r\n");
        return NX_INVALID_PACKET;
    }

    if (command.robot_id < 1 || command.robot_id > 7) {
        LOG_ERROR("Invalid robot ID: %d\r\n", command.robot_id);
        return NX_INVALID_PACKET;
    }

    uint8_t data[32];
    data[0] = 1;
    memcpy(data + 1, packet->nx_packet_prepend_ptr, length);
    COM_RF_Transmit(command.robot_id, data, length + 1);

} break;
```

**Total changes:** 1 location, 25 lines (replaces 15 lines)

---

### 4. Robot - Add Include
**File:** `firmware/robot/CM7/Core/Src/com.c`

**Location:** Line ~6 in includes section

```diff
+ #include "robot_command.h"
```

**Total changes:** 1 location, 1 line

---

### 5. Robot - Update Parser
**File:** `firmware/robot/CM7/Core/Src/com.c`

**Location:** Line ~264 in `parse_controller_packet()` function

**REPLACE** entire function:

```c
static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    if (len != ROBOT_COMMAND_SIZE) {
        LOG_DEBUG("Invalid packet size: %d (expected %d)\r\n", len, ROBOT_COMMAND_SIZE);
        return;
    }

    RobotCommand cmd = {0};
    if (!robot_command_decode(payload, &cmd)) {
        LOG_DEBUG("Failed to decode robot command\r\n");
        return;
    }

    NAV_HandleCommand(&cmd);
}
```

**Total changes:** 1 location, 12 lines (replaces 11 lines)

---

### 6. Robot - Update Function Signature
**File:** `firmware/robot/CM7/Core/Inc/nav.h`

```diff
- void NAV_HandleCommand(Command* cmd);
+ void NAV_HandleCommand(RobotCommand* cmd);
```

**Total changes:** 1 location, 1 line

---

### 7. Robot - Update Implementation
**File:** `firmware/robot/CM7/Core/Src/nav.c`

**Location:** Function definition

```diff
- void NAV_HandleCommand(Command* cmd)
+ void NAV_HandleCommand(RobotCommand* cmd)
```

**Note:** No other changes needed - field names are identical!

**Total changes:** 1 location, 1 line

---

### 8. CMakeLists.txt - Basestation

**File:** `firmware/basestation/CMakeLists.txt` (or wherever target_sources is)

**ADD to target_sources:**
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/src/robot_command.c
```

**ADD to target_include_directories:**
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/inc
```

**Total changes:** 2 locations, 2 lines added

---

### 9. CMakeLists.txt - Robot

**File:** `firmware/robot/CMakeLists.txt` (or wherever target_sources is)

**ADD to target_sources:**
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/src/robot_command.c
```

**ADD to target_include_directories:**
```cmake
${PROJECT_SOURCE_DIR}/../../firmware/shared/inc
```

**Total changes:** 2 locations, 2 lines added

---

## Summary of Changes

| Component | File | Lines Changed | Type |
|-----------|------|---------------|------|
| Go | base_station_client.go | 5 | Replace |
| Basestation | com.c (include) | 1 | Add |
| Basestation | com.c (parser) | 25 | Replace |
| Basestation | CMakeLists.txt | 2 | Add |
| Robot | com.c (include) | 1 | Add |
| Robot | com.c (parser) | 12 | Replace |
| Robot | nav.h | 1 | Replace |
| Robot | nav.c | 1 | Replace |
| Robot | CMakeLists.txt | 2 | Add |
| **TOTAL** | **9 files** | **~50 lines** | **Mixed** |

---

## Testing Checklist

```bash
# 1. Run unit tests
cd firmware/shared/test
gcc -I../inc ../src/robot_command.c robot_command_test.c -o test && ./test
# Expected: 10/10 PASSED ✓

# 2. Compile Go
cd software/controller
go build ./cmd/main.go
# Expected: No errors ✓

# 3. Compile Basestation
cd firmware/basestation/build
cmake .. && make
# Expected: No errors ✓

# 4. Compile Robot
cd firmware/robot/build
cmake .. && make
# Expected: No errors ✓
```

---

## Quick Copy-Paste Sections

### Go Change (Full Function)
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

### Basestation Parser Case (Full Case)
```c
case ROBOT_COMMAND: {
    int length = packet->nx_packet_append_ptr - packet->nx_packet_prepend_ptr;

    if (length != ROBOT_COMMAND_SIZE) {
        LOG_ERROR("Invalid robot command packet size: %d (expected %d)\r\n", 
                  length, ROBOT_COMMAND_SIZE);
        ret = NX_INVALID_PACKET;
        return ret;
    }

    RobotCommand command = {0};
    if (!robot_command_decode(packet->nx_packet_prepend_ptr, &command)) {
        LOG_ERROR("Failed to decode robot command\r\n");
        return NX_INVALID_PACKET;
    }

    if (command.robot_id < 1 || command.robot_id > 7) {
        LOG_ERROR("Invalid robot ID: %d\r\n", command.robot_id);
        return NX_INVALID_PACKET;
    }

    uint8_t data[32];
    data[0] = 1;
    memcpy(data + 1, packet->nx_packet_prepend_ptr, length);
    COM_RF_Transmit(command.robot_id, data, length + 1);

} break;
```

### Robot Parser Function (Full Function)
```c
static void parse_controller_packet(uint8_t* payload, uint8_t len)
{
    if (len != ROBOT_COMMAND_SIZE) {
        LOG_DEBUG("Invalid packet size: %d (expected %d)\r\n", len, ROBOT_COMMAND_SIZE);
        return;
    }

    RobotCommand cmd = {0};
    if (!robot_command_decode(payload, &cmd)) {
        LOG_DEBUG("Failed to decode robot command\r\n");
        return;
    }

    NAV_HandleCommand(&cmd);
}
```

---

## Common Mistakes to Avoid

❌ **Don't forget to:**
- Update NAV_HandleCommand signature in BOTH header (.h) and implementation (.c)
- Add includes for robot_command.h in both basestation and robot
- Update CMakeLists.txt in BOTH basestation and robot
- Keep field names the same - cmd->pos_x, not cmd->posX or cmd->position_x

❌ **Don't:**
- Change field access patterns (they're identical)
- Forget the size check: `if (len != ROBOT_COMMAND_SIZE)`
- Mix old protobuf code with new binary code in the same build
- Forget error handling in encode/decode

✓ **Do:**
- Use `action.EncodeCommand()` for encoding
- Use `robot_command_decode()` for decoding
- Validate packet size before decoding
- Check return values
- Add logging for errors

---

## If Something Goes Wrong

### Revert Strategy
1. Keep old protobuf includes (comment out, don't delete)
2. Add feature flag:
   ```c
   #define USE_CUSTOM_PROTOCOL 0  // Change to 1 to use new protocol
   ```
3. Use `#if USE_CUSTOM_PROTOCOL` to switch between implementations
4. Revert to 0 if issues found
5. No rebuild needed, just relink

---

For detailed explanations and troubleshooting, see `IMPLEMENTATION_GUIDE.md`.
