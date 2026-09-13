# Basestation Crash Investigation

## Symptom

Operation works for some time (minutes to hours), then crashes randomly. A simple
reset fixes it instantly. The basestation receives commands from the brain computer
via Ethernet (UDP port 9999), unpacks them, and transmits them to robots via nRF24.

## Architecture

- **MCU:** STM32H563ZI (Cortex-M33)
- **RTOS:** ThreadX + NetX Duo (Azure RTOS)
- **Ethernet RX path:** `ETH_IRQHandler` → HAL → NetX IP helper thread →
  `udp_socket_receive_controller` callback (`app_netxduo.c:380`) →
  `COM_ParsePacket` (`com.c:198`) → `COM_RF_Transmit` (`com.c:127`)
- **nRF24 TX path:** `COM_RF_Transmit` takes semaphore, writes TX_ADDR, enters
  STANDBY1, calls `NRF_Transmit` (blocking HAL SPI), flushes TX FIFO, releases
  semaphore.
- **nRF24 IRQ path:** EXTI3 falling edge → `HAL_GPIO_EXTI_Falling_Callback`
  (`main.c:90`) → `__disable_interrupts()` → `COM_RF_HandleIRQ` (`com.c:106`) →
  multiple blocking SPI transactions → `__enable_interrupts()`
- **No command buffer:** There is no ring buffer or queue. Packets are consumed
  immediately in the NetX IP thread and forwarded synchronously to the radio.
- **No watchdog:** Both IWDG and WWDG are disabled (`stm32h5xx_hal_conf.h:61,86`).

## Root Causes

### Bug 1 — SPI1 corruption from ISR ↔ thread collision

**Files:** `main.c:90-101`, `com.c:106-125`, `nrf24l01.c:89-140`

`COM_RF_HandleIRQ()` runs from EXTI3 at priority 7 and performs multiple blocking
SPI transactions (`NRF_ReadStatus`, `NRF_SetRegisterBit`, `NRF_ReadPayload`).
Meanwhile `COM_RF_Transmit()` runs on the NetX IP helper thread doing SPI
transactions on the same SPI1 peripheral. The semaphore in `com.c:29` only guards
thread-vs-thread — it is never taken by the ISR path.

When the IRQ lands mid-SPI transaction in `COM_RF_Transmit`:
- CSN toggles from the ISR corrupt the radio's command stream.
- `HAL_SPI_TransmitReceive` hits `__HAL_LOCK` and returns `HAL_BUSY`.
- Every SPI error path in `nrf24l01.c` returns **without raising CSN**
  (`nrf24l01.c:96-97`, `:111-116`, `:130-135`), leaving CSN permanently LOW.
- All subsequent nRF commands are misframed until reset.

### Bug 2 — Permanent deadlock from disabled interrupts + blocking SPI timeout

**Files:** `main.c:92`, `nrf24l01.c:199-211`, HAL SPI driver

The EXTI falling callback does `__disable_interrupts()` (masking SysTick and TIM6)
then calls blocking SPI with a 10 ms timeout. HAL SPI timeouts rely on
`HAL_GetTick()`, which is driven by TIM6 interrupt. If the SPI peripheral is stuck
(from Bug 1), the timeout **never fires** because the tick ISR is masked. The
system hangs permanently with interrupts off.

### Bug 3 — Stack overflow from 1000-byte stack buffer on 2–4 KB thread stacks

**Files:** `log.c:71`, `log.h:13`, `app_netxduo.h:75,77`

`LOG_Printf` allocates `char msg_buffer[LOG_MSG_SIZE]` (1000 bytes) on the stack.
The UDP and link threads have only 2 KB stacks (`NX_APP_THREAD_STACK_SIZE =
2 * 1024`). The IP helper thread has 4 KB. When a log call happens from those
threads (`app_netxduo.c:308,320,325,329,375`), the 1000-byte buffer plus
`snprintf`/`vsnprintf` internal overhead overflows into the adjacent thread's
stack in the contiguous `nx_app_byte_pool`. `TX_ENABLE_STACK_CHECKING` is
explicitly forbidden by the ThreadX vendor header (`tx_api.h:2131`), so this goes
undetected.

## Why it's random

The crash requires the EXTI3 IRQ to land during an SPI transaction in
`COM_RF_Transmit` — the timing depends on when the nRF radio asserts its IRQ line
relative to command packets arriving over Ethernet. More traffic = more chance of
collision = quicker crash. A reset works because it re-initializes CSN, SPI, and
the nRF back to a known state.

## Additional bugs found (not crash-related but worth fixing)

### Wrong status bit cleared in COM_RF_Receive

`com.c:195` clears `STATUS_TX_DS` (bit 5) instead of `STATUS_RX_DR` (bit 6) after
reading a payload. RX_DR stays set, the nRF IRQ line stays low, and falling-edge
EXTI gets no further interrupts for received packets.

### FLUSH_TX immediately after NRF_Transmit

`com.c:135` flushes the TX FIFO ~30 us after CE goes high. Whether the in-flight
packet survives depends on nRF timing — if commands are sometimes dropped on the
robot side, this is a suspect.

## Fix plan

### Fix 1 — Move all SPI/IRQ work off the ISR into a ThreadX event-flag worker
thread (fixes Bug 1 + Bug 2)

**Files:** `com.c`, `com.h`, `main.c`

1. Create a `TX_EVENT_FLAGS_GROUP` in `com.c`.
2. In `HAL_GPIO_EXTI_Falling_Callback` (`main.c:90`), replace
   `__disable_interrupts()` / `COM_RF_HandleIRQ()` / `__enable_interrupts()` with:
   ```c
   tx_event_flags_set(&rf_events, EVT_RF_IRQ, TX_OR);
   ```
3. Create a new thread `COM_RF_Thread` (priority above the IP thread, ~2 KB stack)
   that loops on `tx_event_flags_get(&rf_events, EVT_RF_IRQ, TX_OR_CLEAR,
   TX_WAIT_FOREVER)`. When the flag is set, call `COM_RF_HandleIRQ()` from thread
   context.
4. The existing semaphore around `COM_RF_Transmit` already handles
   thread-vs-thread serialization.

### Fix 2 — Fix the CSN leak in nRF24 error paths

**File:** `nrf24l01.c`

Every `NRF_SendCommand`, `NRF_SendWriteCommand`, and `NRF_SendReadCommand` must
call `csn_set()` before returning on error. Currently `csn_set()` is only called
on the happy path (lines 99, 118, 137). This is a latent bug even after Fix 1,
since SPI errors can still happen.

### Fix 3 — Reduce stack usage in LOG_Printf

**File:** `log.c`

Change `char msg_buffer[LOG_MSG_SIZE]` (currently 1000 bytes on stack) to
`static char msg_buffer[LOG_MSG_SIZE]`. This eliminates stack overflow with zero
risk — logging is inherently serialized by the HAL UART `__HAL_LOCK` anyway.
Alternatively reduce `LOG_MSG_SIZE` from 1000 to 256.

### Fix 4 — Enable an IWDG watchdog

**Files:** `main.c` (init), thread context (refresh)

Add an independent watchdog so any remaining hang auto-resets within 1–2 seconds
instead of being permanent. This is the safety net for anything not yet caught.

### Fix 5 — Fix wrong status bit cleared in COM_RF_Receive

**File:** `com.c:195`

Change `STATUS_TX_DS` to `STATUS_RX_DR` so the IRQ line de-asserts properly and
subsequent receive interrupts fire.

## Priority order

1. **Fix 1** (ISR → thread) — eliminates the crash
2. **Fix 2** (CSN leak) — hardens SPI error handling
3. **Fix 3** (stack buffer) — eliminates silent memory corruption
4. **Fix 4** (watchdog) — safety net
5. **Fix 5** (status bit) — correctness

## Fix status (applied 2026-09-13)

- **Fix 1 — DONE.** EXTI callbacks in `main.c` now only call `tx_event_flags_set()`.
  `COM_RF_HandleIRQ()`/`COM_RF_PrintInfo()`/`COM_Test()` moved into
  `COM_RF_Thread_Entry` (`com.c`), a priority-9 ThreadX thread created in
  `App_ThreadX_Init` (`app_threadx.c`) with a 2 KB stack from `tx_app_byte_pool`.
  The existing NRF semaphore now also protects `COM_RF_HandleIRQ` (thread context).
- **Fix 2 — DONE.** `NRF_SendCommand`, `NRF_SendWriteCommand`, `NRF_SendReadCommand`
  now always assert CSN, even on SPI error (`nrf24l01.c`).
- **Fix 3 — DONE.** `msg_buffer` in `LOG_Printf` made `static` (`log.c`).
- **Fix 4 — DONE (register-level, HAL driver not present in project).**
  `IWDG_Init()` in `main.c` configures ~4 s timeout (prescaler 64, reload 2000,
  LSI 32 kHz), window disabled, counter frozen on debugger halt. `IWDG_Feed()`
  is called every loop iteration of the NetX link thread (`app_netxduo.c`).
- **Fix 5 — DONE.** `COM_RF_Receive` now clears `STATUS_RX_DR` instead of
  `STATUS_TX_DS` (`com.c`).
- **Bonus — DONE.** `COM_ParsePacket` payload bound reduced from 32 to 31 bytes:
  previously a 32-byte packet overflowed the 32-byte `data[]` buffer by 1 byte
  and sent a 33-byte payload to the nRF24 (exceeds its 32-byte max).

NOT verified on hardware yet. No ARM toolchain available to compile locally;
build with the normal CMake flow before flashing.
