
# Setup

To setup a new rpi follow the steps bellow

Enable SPI interface

```
sudo raspi-config
```

go to interfaces and enable SPI

```
sudo apt install cmake protobuf-compiler python3-protobuf
```

pip install protobuf dependency
```
python3 -m venv .venv && source .venv/bin/activate && pip install protobuf grpcio-tools
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
