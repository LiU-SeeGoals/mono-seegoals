# Robot
This is the firmware that all robots (NUCLEO-H755ZI-Q) are running. It receives data from the [basestation](https://github.com/LiU-SeeGoals/basestation) and acts upon this.

**BEWARE** that the wired connections to the motordrivers might behave *funky* when the board is powered through the st-link.

## Contributing
Make sure to follow the [firmware standard](https://github.com/LiU-SeeGoals/wiki/wiki/1.-Processes-&-Standards#seegoal---firmware-standard) and the [feature branch](https://github.com/LiU-SeeGoals/wiki/wiki/1.-Processes-&-Standards#feature-branch-integration) concept.

## Git Submodules 
This project uses several Git Submodules. To get all Submodules at the correct version, use `git submodule update --init --recursive`.
This has to be done the first time you clone, and everytime you change branch to a branch with different submodule versions.

## Building and flashing
This is a cmake project.

To be able to build, make sure you've the `gcc-arm-none-eabi` compiler installed.

Then build with:  
```
# Linux
cmake -B build && make -C build

# Windows (requires MinGW!)
cmake -B build -G "MinGW Makefiles" && make -C build
```

To flash, you can use the `STM32_Programmer_CLI` program downloadable from [here](https://www.st.com/en/development-tools/stm32cubeprog.html).
```
STM32_Programmer_CLI -c port=SWD -w build/robot_CM7.bin 0x08000000 -rst
```

There's also a build rule in make:  
```
make flash_cm7 -C build
```

On Windows, maybe [this GUI](https://www.st.com/en/development-tools/stm32cubeprog.html) is more appropriate.

# Documentation

## Sheets
[NUCLEO-H755ZI-Q](https://www.st.com/resource/en/user_manual/um2408-stm32h7-nucleo144-boards-mb1363-stmicroelectronics.pdf)

## Pins
Use the `robot.ioc` to view the pins, it's opened with [STM32CubeMX](https://www.st.com/en/development-tools/stm32cubemx.html).

## How to use motor driver

Motor driver has 5 pins, set these signals in STM32CubeMX and check the devboard documentation to find the corresponding pin.

```
PWM, - Takes PWM signal to decide speed of motor, generate from devboard with timers.

FGOUT, - Hall sensor tick output, work like a wheel encoder and ticking as the motor spins so that we know how much the motor has spun.

nFault, - Gives fault messages when fault states are entered by going low. Currently does not work, set this pin HIGH 5v or motor driver can enter test mode.

brake - HIGH sets all motor coils high breaking the motor hard.

dir - Decides direction for motor
```

## How to run motor

This will be a overview to give a feeling for the system as things might change with time

What do you need to understand?

* How hardware interuppts work
* How hardware timers work 
* PWM signals.

PWM signals are important to understand to be able to run the motors. And the basics will help to understand how the control loop works. Best is to find some tutorials on youtube

Each motor has a timer which ticks each time FGOUT ticks, this is used in an interrupt which is set to interrupt with a fixed frequency. In the interrupt the delta ticks are counted from the previous interrupt called, this way we know how much/fast the motor has moved.

The control loop is currently a PI loop in MOTOR_SetSpeed (name subject to change) which uses the delta ticks to set a control signal, which is sent as a signal between 0 - 1, this is then scaled and a PWM signal is sent.


## Short about timers

Timers can be set in STM32CubeMX with prescaler and period which decides how fast it will tick compared to the system clock. Sometimes this systemclock is prescaled before coming to the timers, meaning you have to check clock configuration in STM32CubeMX to find the exact frequency, likely this value is 200Mhz (400Mhz prescaled by 2).

The timers are TIM1-12 and LPTIM1-4, LPTIM has less functionality then TIM

EXAMPLE:

TIM1 has prescaler of 10, period of 100

System clock is 400Mhz, however looking at clock config we see that before coming to the timer this is caled by two, meaning the timer gets tick frequncy of 200Mhz.

Now 200Mhz is divided by prescaler 200/10 = 20Mhz and the timer overflows when this has ticked 100 times. 

Now for PWM signals the HIGH part of the pulse width is set by calling

```
  __HAL_TIM_SET_COMPARE(motor->pwm_htim, motor->channel, pwm_speed);
```
  
where pwm_speed is a value between 0 - period.

So if we set pwm_speed = period, the motor will run as fast as possible
and pwm_speed = 0 will turn off the motor.
