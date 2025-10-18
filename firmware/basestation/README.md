# Basestation
This is the firmware for the basestation (NUCLEO-H563ZI) which parses messages from the controller and ssl-vision and then sends it to the robots.

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
cmake -B build && make -C build
```

To flash, you can use the `STM32_Programmer_CLI` program downloadable from [here](https://www.st.com/en/development-tools/stm32cubeprog.html).
```
STM32_Programmer_CLI -c port=SWD sn=004C00283232511639353236 -w build/basestation.bin 0x08000000 -rst
```

There's also a build rule in make:  
```
make flash -C build
```

### Soon to be deprecated
This project can also be compiled/debugged through STM32CubeIDE.

There are two compiling options used, `Debug` is used to compile the project, `Compiledb` uses [compiledb](https://github.com/nickdiego/compiledb) to produce a `compile_commands.json` file which can be used with your LSP powered IDE of choice.

By running the project through STM32CubeIDE and having the NUCLEO card connected through USB (st-link marked port) the binary is flashed to the MCU.

# Documentation

## Sheets
[NUCLEO-H563ZI](https://www.st.com/resource/en/user_manual/um3115-stm32h5-nucleo144-board-mb1404-stmicroelectronics.pdf)

## Pins
Use the `basestation.ioc` to view the pins, it's opened with [STM32CubeMX](https://www.st.com/en/development-tools/stm32cubemx.html).
