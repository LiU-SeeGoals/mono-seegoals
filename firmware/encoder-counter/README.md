This is the firmware for the Raspberry Pi Pico 2:s that live on our encoder counter PCB. It reads the encoders using a PIO program from the [PicoEncoder](https://github.com/pmarques-dev/PicoEncoder) project and transmits the speed over I²C to the nucleo.
# Building & Flashing:
## Installation:
Make sure that you have a Rust toolchain installed I recommend using `rustup` and will assume that you are using it in the following steps.
1. Install the `rustup` package from your system package manager (e.g. `apt install rustup`). If you don't have it available check out the [rust installation instructions.](https://rust-lang.org/tools/install/)
2. Install the rust version for thumbv8m: `rustup target add thumbv8m.main-none-eabihf`
3. Install probe-rs for flashing the chips. (I install with `cargo install probe-rs-tools --locked` this will build from source, if you prefer another method check out the [official instructions](https://probe.rs/docs/getting-started/installation/).)
4. In this directory run: `cargo build`,  if this worked your Rust installation should be ready.
## Flashing:
1. Connect a [Raspberry Pi Debug Probe](https://www.raspberrypi.com/documentation/microcontrollers/debug-probe.html) to the J11 or J12 connector on the encoder counter depending on if you want to flash pico U1 or U2. Connect the JST-SH connector to the port marked **D** on the debug probe and connect the Orange wire to SWCLK, Black wire to GND and Yellow wire to SWDIO.
2. Connect the Raspberry Pi Pico 2 you want to flash to power either over usb or by connecting the encoder counter PCB to power.
3. Run `cargo run --release` to build and flash the pico. After flashing this will open up a RTT connection to the chip and will begin printing status information.
