//! Blinks the LED on a Pico board
//!
//! This will blink an LED attached to GP25, which is the pin the Pico uses for the on-board LED.
#![no_std]
#![no_main]

use defmt::*;
use defmt_rtt as _;
use embedded_hal::digital::{InputPin, OutputPin};
use panic_probe as _;
use rp235x_hal::clocks::init_clocks_and_plls;
use rp235x_hal::gpio::{FunctionI2C, Pin, PullNone};
use rp235x_hal::i2c::peripheral::Event;
use rp235x_hal::pio::PIOExt;
use rp235x_hal::{self as hal, entry};
use rp235x_hal::{Clock, pac};

use pico_encoder::Encoder;

use heapless::Vec;

// Provide an alias for our BSP so we can switch targets quickly.
// Uncomment the BSP you included in Cargo.toml, the rest of the code does not need to change.
// use some_bsp;

/// Tell the Boot ROM about our application
#[unsafe(link_section = ".start_block")]
#[used]
pub static IMAGE_DEF: hal::block::ImageDef = hal::block::ImageDef::secure_exe();

#[entry]
fn main() -> ! {
    info!("######### Init #########");
    let mut pac = pac::Peripherals::take().unwrap();
    let core = cortex_m::Peripherals::take().unwrap();
    let mut watchdog = hal::Watchdog::new(pac.WATCHDOG);
    let sio = hal::Sio::new(pac.SIO);

    // External high-speed crystal on the pico board is 12Mhz
    let external_xtal_freq_hz = 12_000_000u32;
    let clocks = init_clocks_and_plls(
        external_xtal_freq_hz,
        pac.XOSC,
        pac.CLOCKS,
        pac.PLL_SYS,
        pac.PLL_USB,
        &mut pac.RESETS,
        &mut watchdog,
    )
    .ok()
    .unwrap();

    let mut delay = cortex_m::delay::Delay::new(core.SYST, clocks.system_clock.freq().to_Hz());
    let timer = hal::Timer::new_timer0(pac.TIMER0, &mut pac.RESETS, &clocks);

    let pins = hal::gpio::Pins::new(
        pac.IO_BANK0,
        pac.PADS_BANK0,
        sio.gpio_bank0,
        &mut pac.RESETS,
    );

    let mut led_pin = pins.gpio25.into_push_pull_output();

    let mut i2c_id_pin = pins.gpio22.into_floating_input();

    // If gpio 22 is low this is pico0 else it is pico1
    let i2c_id: u8 = if i2c_id_pin.is_low().unwrap() {
        0x5
    } else {
        0x6
    };

    // Init SPI
    let sda_pin: Pin<_, FunctionI2C, PullNone> = pins.gpio0.reconfigure();
    let slc_pin: Pin<_, FunctionI2C, PullNone> = pins.gpio1.reconfigure();

    let mut i2c = hal::I2C::new_peripheral_event_iterator(
        pac.I2C0,
        sda_pin, // SDA
        slc_pin, // SCL
        &mut pac.RESETS,
        i2c_id,
    );

    // Init encoder 0
    let (mut pio0, sm0_0, ..) = pac.PIO0.split(&mut pac.RESETS);

    let mut enc0 = Encoder::program_init(
        &mut pio0,
        sm0_0,
        pins.gpio2,
        pins.gpio3,
        timer,
        &clocks.system_clock,
        // The default mode of the AS5047P is 4000 clicks per rotation
        4000,
    );

    // Init encoder 1
    let (mut pio1, sm1_0, ..) = pac.PIO1.split(&mut pac.RESETS);
    let mut enc1 = Encoder::program_init(
        &mut pio1,
        sm1_0,
        pins.gpio14,
        pins.gpio15,
        timer,
        &clocks.system_clock,
        // The default mode of the AS5047P is 4000 clicks per rotation
        4000,
    );

    info!("##### Program loop #####");
    led_pin.set_high().unwrap();

    loop {
        enc0.update();
        enc1.update();

        // React to the I2C events that have happened since the last loop.
        while let Some(event) = i2c.next_event() {
            match event {
                // The controller has requested data so we should send the
                // speeds
                Event::TransferRead => {
                    info!("Transfer Read");
                    let speed0 = enc0.get_raw_encoder_state().get_speed();
                    let speed1 = enc1.get_raw_encoder_state().get_speed();
                    info!("Speed0: {}, Speed1: {}", speed0, speed1);
                    i2c.write(&compose_message(speed0, speed1));
                }
                Event::TransferWrite => {
                    info!("Transfer write");
                    let mut buf = [0; 32];
                    i2c.read(&mut buf);
                    // info!("{:?}", buf);
                }
                // Event::Start => info!("Start"),
                // Event::Restart => info!("Restart"),
                // Event::Stop => info!("Stop"),
                _ => {}
            }
        }

        // info!(
        //     "Speed: {}, Rotation: {}, Steps: {}",
        //     enc.get_speed(),
        //     enc.get_rotation(),
        //     enc.get_steps()
        // );

        delay.delay_ms(20);
    }
}

static CRC_TABLE: [u8; 256] = [
    0, 94, 188, 226, 97, 63, 221, 131, 194, 156, 126, 32, 163, 253, 31, 65, 157, 195, 33, 127, 252,
    162, 64, 30, 95, 1, 227, 189, 62, 96, 130, 220, 35, 125, 159, 193, 66, 28, 254, 160, 225, 191,
    93, 3, 128, 222, 60, 98, 190, 224, 2, 92, 223, 129, 99, 61, 124, 34, 192, 158, 29, 67, 161,
    255, 70, 24, 250, 164, 39, 121, 155, 197, 132, 218, 56, 102, 229, 187, 89, 7, 219, 133, 103,
    57, 186, 228, 6, 88, 25, 71, 165, 251, 120, 38, 196, 154, 101, 59, 217, 135, 4, 90, 184, 230,
    167, 249, 27, 69, 198, 152, 122, 36, 248, 166, 68, 26, 153, 199, 37, 123, 58, 100, 134, 216,
    91, 5, 231, 185, 140, 210, 48, 110, 237, 179, 81, 15, 78, 16, 242, 172, 47, 113, 147, 205, 17,
    79, 173, 243, 112, 46, 204, 146, 211, 141, 111, 49, 178, 236, 14, 80, 175, 241, 19, 77, 206,
    144, 114, 44, 109, 51, 209, 143, 12, 82, 176, 238, 50, 108, 142, 208, 83, 13, 239, 177, 240,
    174, 76, 18, 145, 207, 45, 115, 202, 148, 118, 40, 171, 245, 23, 73, 8, 86, 180, 234, 105, 55,
    213, 139, 87, 9, 235, 181, 54, 104, 138, 212, 149, 203, 41, 119, 244, 170, 72, 22, 233, 183,
    85, 11, 136, 214, 52, 106, 43, 117, 151, 201, 74, 20, 246, 168, 116, 42, 200, 150, 21, 75, 169,
    247, 182, 232, 10, 84, 215, 137, 107, 53,
];

fn calc_crc(array: &[u8], start: usize) -> u8 {
    let mut crc = 0;
    for i in &array[start..] {
        crc = CRC_TABLE[(crc ^ i) as usize]
    }
    crc
}

fn compose_message(speed0: i32, speed1: i32) -> [u8; 11] {
    let mut message: Vec<u8, 11> = Vec::new();
    // STX, Unwrap because this should not be able to fail
    message.push(0x2).unwrap();
    // Speed_0
    message.extend(speed0.to_be_bytes());
    // Speed_1
    message.extend(speed1.to_be_bytes());
    // CRC
    let crc = calc_crc(&message, 1);
    // This will always have availible space so unwrap.
    message.push(crc).unwrap();
    // EXT
    message.push(0x3).unwrap();

    message.into_array().expect("Message size less than 11")
}

/// Program metadata for `picotool info`
#[unsafe(link_section = ".bi_entries")]
#[used]
pub static PICOTOOL_ENTRIES: [rp235x_hal::binary_info::EntryAddr; 5] = [
    rp235x_hal::binary_info::rp_cargo_bin_name!(),
    rp235x_hal::binary_info::rp_cargo_version!(),
    rp235x_hal::binary_info::rp_program_description!(c"RP2350 Template"),
    rp235x_hal::binary_info::rp_cargo_homepage_url!(),
    rp235x_hal::binary_info::rp_program_build_attribute!(),
];

// End of file
