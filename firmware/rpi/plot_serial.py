import pickle
import numpy as np
from matplotlib import pyplot as plt
from dataclasses import dataclass, field

@dataclass
class Imu:
    timestamp: int | None = None
    x: float = 0
    y: float = 0
    z: float = 0

@dataclass
class ParsedData:
    imu: Imu = field(default_factory = Imu)

with open("imu.pkl","rb") as file:
    datas = pickle.load(file)

    print(datas)

imu = np.zeros((len(datas),4))

for i in range(len(datas)):
    imu[i, 0] = datas[i].imu.timestamp
    imu[i, 1] = datas[i].imu.x
    imu[i, 2] = datas[i].imu.y
    imu[i, 3] = datas[i].imu.z


plt.plot(imu[:,0],imu[:,3])
print(imu.shape[0],(-datas[0].imu.timestamp + datas[-1].imu.timestamp))
plt.show()

