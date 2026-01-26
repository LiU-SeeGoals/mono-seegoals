
# Setup

To setup a new rpi follow the steps bellow

Enable SPI interface

```
sudo raspi-config
```

go to interfaces and enable SPI

```
sudo apt install cmake protobuf-compiler python3-protobuf python3-grpcio
```

Build and install wiringPi

```
cd WiringPi && ./build
```

# How to run

ssh into the robot and go to rpi folder

Build the rpi

```
cmake . -B build
cd build
make
```

Now run the binary to start reading spi from robot

```
./spiSpeed
```

This creates a file output.txt in the build folder which contains protobuf serialised messages
Decode each line using the data_sample proto message in your favorite programming language


## Using output from rpi

After running the rpi for some time and logging data you can scp the files over and plot the data.

```
scp rpi@rpi.local:/<file location>
cd build
make protobuf
python3 <plotfile>
```

# Live Telemetry

## Setup
Follow the setup instructions above for logging.

To use the live telemetry you need to have AdvantageScope installed on your
machine installation instructions can be found at:
https://docs.advantagescope.org/overview/installation

### Configuring AdvantageScope
AdvantageScope needs to be configured to use fetdatorns ip address,
this setting is found in `App > Show Preferences`.
Then set `Robot Address` to `192.168.1.1`.

## Running live telemetry
1. Start Glass on fetdatorn
2. Run the script on the raspberry pi. `./live_telem.py` (this requires [uv](https://docs.astral.sh/uv/))
3. Start AdvantageScope on your machine and press `Ctrl+k` or navigate to `File > Connect to Robot` to connect to the server.

