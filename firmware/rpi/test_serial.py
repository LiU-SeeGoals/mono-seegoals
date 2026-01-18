import serial
import re
import pickle
import time
from dataclasses import dataclass, field

PORT = "/dev/ttyACM1"  # Linux example
BAUD_RATE = 115200

ser = serial.Serial(PORT, BAUD_RATE, timeout=1)
print("Reading from USB port...")

@dataclass
class Imu:
    timestamp: int | None = None
    x: float = 0
    y: float = 0
    z: float = 0

@dataclass
class ParsedData:
    imu: Imu = field(default_factory = Imu)


def parse_uart_msg(data):

    delim = "_"

    def parse_imu(data_imu, output):
        pattern = rf"t(.*)x(.*)y(.*)z(.*){delim}"
        match = re.search(pattern, data_imu)
        if match is None:
            print("Failed parsing imu")
            return None
        
        output.imu.timestamp = int(match.group(1))
        output.imu.x = float(match.group(2))
        output.imu.y = float(match.group(3))
        output.imu.z = float(match.group(4))

        return match.group(0)

    data_order = [parse_imu]

    output = ParsedData()

    for i in range(len(data_order)):
        parsed_output = data_order[i](data, output)

        if parsed_output is None:
            # Failed to parse cannot continue
            break
        data = data[len(parsed_output):]

    return output


datas = []

def avg_diff(l):
    avg = 0
    for i in range(len(l) - 1):
        avg += l[i + 1].imu.timestamp - l[i].imu.timestamp

    return avg/len(l)

def parse_data_uart():
    if ser.in_waiting > 0:
        usb_bytes = ser.readline()
        try:
            line = usb_bytes.decode("utf-8").strip()
        except:
            print(f"Failed decoding {len(usb_bytes)}")
            return

        data_pattern = r"^\[DATA-I\] (.*)$"
        pattern = re.compile(data_pattern)
        match = pattern.search(line)

        if match is not None:
            data_content = match.group(1)
            parsed_data = parse_uart_msg(data_content)
            if parsed_data.imu.timestamp is not None:
                datas.append(parsed_data)

            if len(datas) > 1:
                print(f"Hz: {1000.0/(avg_diff(datas))}")



if __name__ == "__main__":
    ser.reset_input_buffer()
    time.sleep(0.001)
    start = time.time()
    while time.time() - start < 5:
        parse_data_uart()

    with open("imu.pkl", "wb") as file:
        pickle.dump(datas, file)
