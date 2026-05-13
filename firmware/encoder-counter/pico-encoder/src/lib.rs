#![no_std]
#![expect(dead_code)]

use cortex_m::interrupt;
use defmt::info;
use heapless::Vec;
use rp235x_hal::{
    self as hal, Clock, Timer,
    clocks::SystemClock,
    gpio::{AnyPin, FunctionNull, Pin, PullUp, ValidFunction},
    pio::{Buffers, PIO, PIOExt, Rx, StateMachine, StateMachineIndex, Tx, UninitStateMachine},
    timer::TimerDevice,
};

pub struct EncoderState {
    speed: i32,
    position: i32,
    step: u32,
}

impl EncoderState {
    fn update_all(&mut self, speed: i32, position: i32, step: u32) {
        (self.speed, self.position, self.step) = (speed, position, step);
    }

    fn update_position(&mut self, position: i32) {
        self.position = position;
    }

    pub fn get_state(&self) -> (i32, i32, u32) {
        (self.speed, self.position, self.step)
    }

    pub fn get_speed(&self) -> i32 {
        self.speed
    }
    pub fn get_position(&self) -> i32 {
        self.position
    }
    pub fn get_step(&self) -> u32 {
        self.step
    }
}

static IDLE_STOP_SAMPLES: u32 = 10;

/// Compute speed in "sub-steps per 2^20 us" from a delta substep position and
/// delta time in microseconds.
fn substep_calc_speed(delta_substep: i32, delta_us: i32) -> i32 {
    // This seems to work the same as the c++ implementation when i tested some
    // examples.
    (((delta_substep as i64) << 20) / delta_us as i64) as i32
}

pub struct Encoder<'pio, P, P1, P2, SMI, D>
where
    P: PIOExt,
    SMI: StateMachineIndex,
    P1: AnyPin,
    P2: AnyPin,
    D: TimerDevice,
{
    pio: &'pio mut PIO<P>,
    sm: StateMachine<(P, SMI), hal::pio::Running>,
    enc_p1: Pin<P1::Id, P::PinFunction, PullUp>,
    enc_p2: Pin<P2::Id, P::PinFunction, PullUp>,
    tx: Tx<(P, SMI)>,
    rx: Rx<(P, SMI)>,
    state: EncoderState,
    phases: [u32; 4],
    timer: Timer<D>,
    clocks_per_us: u32,
    idle_stop_sample_count: u32,
    stopped: bool,
    speed_2_20: i32,
    prev_trans_pos: u32,
    prev_trans_us: u32,
    prev_step_us: u32,
    prev_low: u32,
    prev_high: u32,
    internal_position: u32,
    position_reset: u32,
    steps_per_rotation: u32,
    calib_last_step: u32,
    calib_last_us: u32,
    calib_count: u32,
    calib_sum: [u32; 4],
    calib_data: [u32; 4],
}

impl<'pio, P, P1, P2, SMI, D> Encoder<'pio, P, P1, P2, SMI, D>
where
    P: PIOExt,
    SMI: StateMachineIndex,
    P1: AnyPin,
    P2: AnyPin,
    D: TimerDevice,
{
    /// Initialize an encoder, the two pins enc_p1 and enc_p2 have to be sequential
    pub fn program_init(
        pio: &'pio mut PIO<P>,
        uninit_sm: UninitStateMachine<(P, SMI)>,
        enc_p1: P1,
        enc_p2: P2,
        timer: Timer<D>,
        sys_clk: &SystemClock,
        steps_per_rotation: u32,
    ) -> Self
    where
        P1: AnyPin<Function = FunctionNull>,
        P1::Id: ValidFunction<P::PinFunction>,
        P2: AnyPin<Function = FunctionNull>,
        P2::Id: ValidFunction<P::PinFunction>,
    {
        let (enc_p1, enc_p2): (P1::Type, P2::Type) = (enc_p1.into(), enc_p2.into());

        if enc_p1.id().num + 1 != enc_p2.id().num {
            panic!("The pin numbers provided to the encoder are not sequential");
        }

        // Compile the PIO program
        let program = pio::pio_file!("./src/pico_encoder.pio");
        // Install the compiled program to the provided pio unit
        let installed = pio.install(&program.program).unwrap();

        // Conifigure the state machine to the state assumed by the program
        //let (mut sm, mut rx, tx) =
        let builder = hal::pio::PIOBuilder::from_installed_program(installed)
            .in_pin_base(enc_p1.id().num)
            .clock_divisor_fixed_point(1, 0) // Run every cycle
            .in_shift_direction(hal::pio::ShiftDirection::Left)
            .out_shift_direction(hal::pio::ShiftDirection::Right)
            .autopush(true)
            .autopull(false)
            .push_threshold(32)
            .pull_threshold(32)
            .buffers(Buffers::RxTx) // I think this is whats meant by don't join FIFOs
            .set_mov_status_config(hal::pio::MovStatusConfig::Rx(0x12)); // I think this is what the strange bit stuff does
        let (mut sm, mut rx, tx) = builder.build(uninit_sm);

        sm.set_pindirs([
            (enc_p1.id().num, hal::pio::PinDir::Input),
            (enc_p2.id().num, hal::pio::PinDir::Input),
        ]);

        let enc_p1: Pin<_, _, PullUp> = enc_p1.into_pull_type();
        let enc_p2: Pin<_, _, PullUp> = enc_p2.into_pull_type();

        let enc_p1: Pin<P1::Id, P::PinFunction, _> = enc_p1.into_function();

        let enc_p2: Pin<P2::Id, P::PinFunction, _> = enc_p2.into_function();

        // init the state machine according to the current phase. Since we are
        // setting the state running PIO instructions from Rust state, the encoder may
        // step during this initialization. This should not be a problem though,
        // because as long as it is just one step, the state machine will update
        // correctly when it starts. We disable interrupts anyway, to be safe
        let sm_ins = interrupt::free(|_| {
            // to setup the state machine, we need to set the lower 2 bits of OSR to be
            // the negated pin state

            // Read pin state
            sm.exec_instruction(pio::Instruction {
                operands: pio::InstructionOperands::IN {
                    source: pio::InSource::PINS,
                    bit_count: 2,
                },
                delay: 0,
                side_set: None,
            });

            // Set the inverted pin state from ISR to PIO_Y
            sm.exec_instruction(pio::Instruction {
                operands: pio::InstructionOperands::MOV {
                    destination: pio::MovDestination::Y,
                    op: pio::MovOperation::Invert,
                    source: pio::MovSource::ISR,
                },
                delay: 0,
                side_set: None,
            });

            // Set PIO_OSR to the same value as PIO_Y
            sm.exec_instruction(pio::Instruction {
                operands: pio::InstructionOperands::MOV {
                    destination: pio::MovDestination::OSR,
                    op: pio::MovOperation::None,
                    source: pio::MovSource::Y,
                },
                delay: 0,
                side_set: None,
            });

            // Push ISR to the rx FIFO, this also clears ISR,
            // blocking should not matter here as the FIFO should be empty.
            sm.exec_instruction(pio::Instruction {
                operands: pio::InstructionOperands::PUSH {
                    if_full: false,
                    block: true,
                },
                delay: 0,
                side_set: None,
            });

            // Read the pin values from the FIFO, and with three to just keep
            // the lowest two bits.
            let pin_state = (rx.read().expect("Could not read pin value from rx FIFO") & 3) as u8;

            let position = match pin_state {
                0 => 0,
                1 => 3,
                2 => 1,
                3 => 2,
                // We & 3 pin_state above so it has to be between 0 and 3.
                _ => unreachable!(),
            };

            // Set PIO_Y to the value calculated above.
            sm.exec_instruction(pio::Instruction {
                operands: pio::InstructionOperands::SET {
                    destination: pio::SetDestination::Y,
                    data: position,
                },
                delay: 0,
                side_set: None,
            });
            // enable
            sm.start()
        });

        let clocks_per_us = (sys_clk.freq().to_Hz() + 500000) / 1000000;

        Self {
            pio,
            sm: sm_ins,
            enc_p1,
            enc_p2,
            rx,
            tx,
            state: EncoderState {
                speed: 0,
                position: 0,
                step: 0,
            },
            phases: [0, 64, 128, 192],
            timer,
            clocks_per_us,
            idle_stop_sample_count: 0,
            stopped: true,
            speed_2_20: 0,
            prev_trans_pos: 0,
            prev_trans_us: 0,
            prev_step_us: 0,
            prev_low: 0,
            prev_high: 0,
            internal_position: 0,
            position_reset: 0,
            steps_per_rotation,
            calib_last_step: 0,
            calib_last_us: 0,
            calib_count: 0,
            calib_sum: [0, 0, 0, 0],
            calib_data: [0, 0, 0, 0],
        }
    }

    /// Gets the raw data from the encoder
    /// Output: Option<(cycles, step, time{us})>
    /// This method is very defenceivly coded so it might be a bit slow
    fn get_counts(&mut self) -> Option<(i32, u32, u32)> {
        // The singular rx fifo is a 4x32-bit bus so this array should be big
        // enough. REF: 11.2.4.4. in the datasheet. We throw this value away
        // every time we call this method so we have the possibility of mabye
        // going out of sync (hopefully not).
        let mut pairs = Vec::<u32, 4>::new();
        let mut cycles: i32 = 0;
        let mut step: u32 = 0;

        let mut t: u32 = 0;
        // Read the data without interupts to not have a big time gap between
        // reading the data and the current us.
        // The maximum number of values in the rx fifo is 4 so we only want to
        // get that many, if we don't we will jsut keep reading.
        interrupt::free(|_| {
            for _ in 0..4 {
                if let Some(i) = self.rx.read() {
                    // info!("Recived: {}", i);
                    pairs.push(i).expect("Can not push to pairs vector")
                } else {
                    break;
                }
            }
            t = self.timer.get_counter_low(); // I hope that this is the same as time_us_32()
        });

        // info!("{:?}", &pairs[0..pairs.len()]);

        // If we don't have a pair we should not output anything.
        // If we got an odd number of messages wait until we get a new one.
        // Just waiting should be fine as the only reason there should be an
        // odd number of messages is if the read is perfectly timed between the
        // two IN instructions.
        if pairs.len() == 0 {
            return None;
        } else if pairs.len() % 2 == 1 {
            info!("Odd number of messages recived");
            // wait until we get a message
            loop {
                let message = self.rx.read();
                match message {
                    Some(i) => {
                        pairs.push(i).expect("Can not push to vector");
                        break;
                    }
                    None => continue,
                }
            }
        }

        for i in (0..(pairs.len() >> 1)).step_by(2) {
            // Read the pairs cycles come first then step.
            cycles = pairs[i] as i32;
            step = pairs[i + 1];
        }
        // info!("{:?}", (cycles, step, t));
        Some((cycles, step, t))
    }

    fn read_pio_data(&mut self) -> Option<(u32, u32, u32, bool)> {
        let (mut cycles, step, step_us) = match self.get_counts() {
            Some(i) => i,
            None => return None,
        };

        // Copied directly from the cpp code:
        // when the PIO program detects a transition, it sets cycles to either zero
        // (when step is incrementing) or 2^31 (when step is decrementing) and keeps
        // decrementing it on each 13 clock loop. We can use this information to get
        // the time and direction of the last transition
        let forward = if cycles < 0 {
            cycles = -cycles;
            true
        } else {
            cycles = i32::MIN.wrapping_sub(cycles);
            false
        };
        let transition_us =
            step_us.wrapping_sub((cycles.wrapping_mul(13)) as u32 / self.clocks_per_us);
        // info!(
        //     "step_us: {}, cycles: {}, transition_us: {}",
        //     step_us, cycles, transition_us
        // );

        return Some((step, step_us, transition_us, forward));
    }

    /// Read data from the pio and update the speed / position estimate.
    /// Should be called once in the control loop just before one of the values
    /// are needed.
    pub fn update(&mut self) {
        let transition_pos;

        // Read the current encoder state
        let (new_step, step_us, transition_us, forward) = match self.read_pio_data() {
            Some(i) => i,
            None => return,
        };

        let (step, mut speed) = (self.state.step, self.state.speed);

        // From the current step we can get the low and high boundaries in
        // substeps of the current position.
        let low = self.get_step_start_transition_pos(new_step);
        let high = self.get_step_start_transition_pos(new_step.wrapping_add(1));
        // info!("new_step: {}, low: {}, high: {}", new_step, low, high);

        // if we were not stopped, but the last transition was more than
        // "idle_stop_samples" ago, we are stopped now
        if new_step == step {
            self.idle_stop_sample_count += 1;
        } else {
            self.idle_stop_sample_count = 0;
        }

        if !self.stopped && self.idle_stop_sample_count >= IDLE_STOP_SAMPLES {
            speed = 0;
            self.speed_2_20 = 0;
            self.stopped = true;
        }

        // when we are at a different step now, there is certainly a transition
        if step != new_step {
            // the transition position depends on the direction of the move
            transition_pos = if forward { low } else { high };

            // if we are not stopped, that means there is valid previous transition
            // we can use to estimate the current speed
            if !self.stopped {
                self.speed_2_20 = substep_calc_speed(
                    (transition_pos as i32).wrapping_sub(self.prev_trans_pos as i32),
                    (transition_us as i32).wrapping_sub(self.prev_trans_us as i32),
                );
            }

            // if we have a transition, we are not stopped now
            self.stopped = false;
            // save the timestamp and position of this transition to use later to
            // estimate speed
            self.prev_trans_pos = transition_pos;
            self.prev_trans_us = transition_us;
        }

        // if we are stopped, speed is zero and the position estimate remains
        // constant. If we are not stopped, we have to update the position and speed
        if !self.stopped {
            // although the current step doesn't give us a precise position, it does
            // give boundaries to the position, which together with the last
            // transition gives us boundaries for the speed value. This can be very
            // useful especially in two situations:
            // - we have been stopped for a while and start moving quickly: although
            //   we only have one transition initially, the number of steps we moved
            //   can already give a non-zero speed estimate
            // - we were moving but then stop: without any extra logic we would just
            //   keep the last speed for a while, but we know from the step
            //   boundaries that the speed must be decreasing

            // if there is a transition between the last sample and now, and that
            // transition is closer to now than the previous sample time, we should
            // use the slopes from the last sample to the transition as these will
            // have less numerical issues
            let (speed_high, speed_low) = if self.prev_trans_us > self.prev_step_us
                && (self.prev_trans_us as i32 - self.prev_step_us as i32)
                    > (step_us as i32 - self.prev_trans_us as i32)
            {
                (
                    substep_calc_speed(
                        self.prev_trans_pos as i32 - self.prev_low as i32,
                        self.prev_trans_us as i32 - self.prev_step_us as i32,
                    ),
                    substep_calc_speed(
                        self.prev_trans_pos as i32 - self.prev_high as i32,
                        self.prev_trans_us as i32 - self.prev_step_us as i32,
                    ),
                )
            } else {
                // otherwise use the slopes from the last transition to now
                (
                    substep_calc_speed(
                        high as i32 - self.prev_trans_pos as i32,
                        step_us as i32 - self.prev_trans_us as i32,
                    ),
                    substep_calc_speed(
                        low as i32 - self.prev_trans_pos as i32,
                        step_us as i32 - self.prev_trans_us as i32,
                    ),
                )
            };

            // Make shure the current speed estimate is between the maximum and
            // minimum values obtained from the step slopes.
            if self.speed_2_20 > speed_high {
                self.speed_2_20 = speed_high;
            }
            if self.speed_2_20 < speed_low {
                self.speed_2_20 = speed_low;
            }

            // Convert the speed units from "substeps per 2^20" us to "substeps
            // per second".
            speed = ((self.speed_2_20 as i64 * 62500i64) >> 16) as i32;

            // estimate the current position by applying the speed estimate to the
            // most recent transition
            // I'm very unshure if the casting here is correct
            self.internal_position = self.prev_trans_pos.wrapping_add(
                ((self.speed_2_20 as i64 * (step_us as i64 - transition_us as i64)) >> 20) as u32,
            );

            // make sure the position estimate is between "low" and "high", as we
            // can be sure the actual current position must be in this range
            if self.internal_position as i32 - high as i32 > 0 {
                self.internal_position = high;
            } else if (self.internal_position as i32 - low as i32) < 0 {
                self.internal_position = low;
            }
        }

        // compute the user position, as the difference to the reset position
        let position = self.internal_position as i32 - self.position_reset as i32;

        self.prev_low = low;
        self.prev_high = high;
        self.prev_step_us = step_us;
        self.state.update_all(speed, position, new_step);
    }

    /// Returns the `EncoderState` data structure used internally to keep track
    /// of the current state. The `speed` and `position` fields are in substeps
    /// so are 64x bigger than the normal `step` value.
    pub fn get_raw_encoder_state(&self) -> &EncoderState {
        &self.state
    }

    /// Returns the encoder speed in rpm
    pub fn get_speed(&self) -> f32 {
        (self.state.speed as f32 * 60.0) / (64 * self.steps_per_rotation) as f32
    }

    /// Returns the rotation position, with decimals representing a fraction of
    /// a rotation.
    pub fn get_rotation(&self) -> f32 {
        self.state.position as f32 / (64 * self.steps_per_rotation) as f32
    }

    /// Returns the number of steps, sign indicates direction.
    pub fn get_steps(&self) -> i32 {
        self.state.step as i32
    }

    /// Incrementally update the phase measurements, so that the substep
    /// estimation takes phase sizes into account. This method should be called
    /// at high frequency to be able to measure step sizes.
    pub fn auto_calibrate_phases(&mut self) {
        // read raw encoder data. Reading encoder data is an idempotent operation,
        // so we can still continue calling update and everything should just work
        let (step, _, cur_us, forward) = self.read_pio_data().unwrap();
        // if we are still on the same step as before, there is nothing to see
        if step == self.calib_last_step {
            // info!("New step is same as last step");
            return;
        }

        // if calib_last_us is zero, that means we haven't started yet, so don't try
        // to use a delta to nothing
        let delta = if self.calib_last_us == 0 {
            0
        } else {
            cur_us.wrapping_sub(self.calib_last_us)
        };

        let steps = self.calib_last_step.wrapping_sub(step) as i32;

        self.calib_last_step = step;
        self.calib_last_us = cur_us;

        // if we've skipped a step, we can not use this information (and we need to
        // reset the data). Also, do the same if we didn't skip a step but the last
        // step was just too slow to be usable (> 20ms)
        if steps.abs() > 1 || delta > 20000 || delta == 0 {
            self.calib_data = [0, 0, 0, 0];
            // info!("We've skipped a step or gone too long between steps");
            return;
        }

        // save the step period in the correct data slot
        if forward {
            self.calib_data[((step.wrapping_sub(1)) & 3) as usize] = delta;
        } else {
            self.calib_data[((step.wrapping_add(1)) & 3) as usize] = delta;
        }

        // if we don't have a measure of all the steps yet, just continue
        if self.calib_data.contains(&0) {
            // info!("Calib_data has 0");
            return;
        }

        // otherwise, use the measurement. Sum the just acquired 4 step sizes to the
        // step size total accumulator. Check if the values in the accumulator are
        // getting too big and halve them in that case, to keep them manageable
        let mut need_rescale = false;
        for i in 0..4 {
            self.calib_sum[i] += self.calib_data[i];
            self.calib_data[i] = 0;
            if self.calib_sum[i] > 2500000 {
                need_rescale = true;
            }
        }

        let mut total = 0;
        for i in 0..4 {
            if need_rescale {
                self.calib_sum[i] >>= 1;
            }
            total += self.calib_sum[i];
        }
        self.calib_count += 1;

        // if we don't have at least 32 full measurements, don't use them yet, as
        // we may still have a big bias (this is just an heuristic)
        if self.calib_count < 1024 {
            // info!("Calib count lower than 32: {}", self.calib_count);
            return;
        }

        // scale the sizes to a total of 256 to be used as sub-steps
        self.phases[0] = 0;
        self.phases[1] = (self.calib_sum[0] * 256 + total / 2) / total;
        self.phases[2] = ((self.calib_sum[0] + self.calib_sum[1]) * 256 + total / 2) / total;
        self.phases[3] =
            ((self.calib_sum[0] + self.calib_sum[1] + self.calib_sum[2]) * 256 + total / 2) / total;
    }

    /// return true if the phase auto calibration is ready
    pub fn auto_calibration_done(&self) -> bool {
        if self.calib_count < 1024 { false } else { true }
    }

    /// Will return the curent phase calibration
    /// If the phase has not been auto calibrated or set manualy this will be
    /// 0x404040
    pub fn get_phases(&self) -> u32 {
        self.phases[1]
            | ((self.phases[2] - self.phases[1]) << 8)
            | ((self.phases[3] - self.phases[2]) << 16)
    }

    /// Set the phase sizes using the result from a previous calibration.
    pub fn set_phases(&mut self, phases: u32) {
        self.phases[0] = 0;
        self.phases[1] = phases & 0xFF;
        self.phases[2] = self.phases[1] + ((phases >> 8) & 0xFF);
        self.phases[3] = self.phases[2] + ((phases >> 16) & 0xFF);
    }

    /// Returns the raw phase size array used by the program
    pub fn get_raw_phases(&self) -> &[u32; 4] {
        &self.phases
    }

    /// Resets the phase calibration values to the default.
    /// It is unlikely that this would need to be called unless there was a change
    /// in encoder geometry.
    pub fn reset_auto_calibration(&mut self) {
        self.calib_sum = [0, 0, 0, 0];
        self.calib_count = 0;
        self.calib_last_us = 0;
    }

    /// Reset the position value to 0
    pub fn reset_position(&mut self) {
        self.position_reset = self.internal_position;
        self.state.update_position(0);
    }

    /// Get the sub-step position of the start of a step
    fn get_step_start_transition_pos(&self, step: u32) -> u32 {
        // I'm not shure what this does or how it works, but it is copied from
        // the c++ code.
        ((step << 6) & 0xFFFFFF00) | &self.phases[(step & 3) as usize]
    }
}
