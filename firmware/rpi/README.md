
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

run program in this dir?

