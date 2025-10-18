import serial
import pdb
import time
import matplotlib.pyplot as plt
import numpy as np

class kalmanFilterCV1d:

    def __init__(self, T):
        self.P = np.diag([1,1])
        self.G = np.array([[T**2/2,0],
                           [0,T]])

        self.q = np.diag([1,1])

        self.Q = self.G @ self.q @ self.G.T
        self.x = np.zeros(2)

        self.F = np.array([[1,T],
                           [0,1]])

        self.B = np.array([[T**2/2,0],
                           [0,T]])

    def predict(self, u):
        self.x = self.F @ self.x + self.B @ np.array([u,u])
        self.P = self.F @ self.P @ self.F.T + self.Q
        

ser = serial.Serial('/dev/ttyACM0', baudrate=115200)

print(f"connected to: {ser.portstr}")


COLLECT_DATA = False


data_name = "imu-move-desk"
NUM_DATA = 1000

start_t = time.time()
if COLLECT_DATA:
    collected_data = 0
    data = np.zeros([3, NUM_DATA])

    while collected_data < NUM_DATA:
        line = str(ser.readline())
        if "DATA" in line and len(line.split("===")) == 3:
            x,y,z = map(float, line.split("===")[1].split(";")[1:])

            data[:, collected_data] = np.array([x,y,z])
            collected_data += 1
            print(f"{collected_data / NUM_DATA}%")

    collection_time = np.array([time.time() - start_t])

    ser.close()

    np.save(f"{data_name}.npy", data)
    np.save(f"{data_name}-time.npy", collection_time)

calibration = np.load(f"imu.npy")

data = np.load(f"{data_name}.npy")
collection_time = np.load(f"{data_name}-time.npy")

print(f"Collection of {len(data[0,:])} imu samples took {abs(collection_time)} seconds i.e. {abs(collection_time)/len(data[0,:])} Hz")

f, axs = plt.subplots(3,1)
times = np.linspace(0, abs(collection_time), len(data[0,:]))

bias = np.mean(calibration, axis=1)

# data = data - bias[:,None]

axs[0].plot(times, data[0,:] - bias[0])
axs[1].plot(times, data[1,:] - bias[1])
axs[2].plot(times, data[2,:] - bias[2])

plt.show()
print("kalmanning this bith")

dt = (abs(collection_time)/len(data[0,:]))[0]

filterx = kalmanFilterCV1d(dt)
filtery = kalmanFilterCV1d(dt)
filterz = kalmanFilterCV1d(dt)

pos_predictions = np.zeros([3, data.shape[1]])
vel_predictions = np.zeros([3, data.shape[1]])

filters = [filterx, filtery, filterz]

# This is where the data enters the kalman

for i in range(data.shape[1]):
    for j in range(3):
        if abs(data[j,i] - bias[j]) > 0.25:
            filters[j].predict(data[j,i] - bias[j])

        pos_predictions[j,i] = filters[j].x[0]
        vel_predictions[j,i] = filters[j].x[1]


f, axs = plt.subplots(3,3)

axs[0,0].set_title("Position (x,y,z)")
axs[0,0].plot(times, pos_predictions[0,:])
axs[1,0].plot(times, pos_predictions[1,:])
axs[2,0].plot(times, pos_predictions[2,:])

axs[0,1].set_title("Velocity (x,y,z)")
axs[0,1].plot(times, vel_predictions[0,:])
axs[1,1].plot(times, vel_predictions[1,:])
axs[2,1].plot(times, vel_predictions[2,:])

axs[0,2].set_title("Raw data minus bias (x,y,z)")
axs[0,2].plot(times, data[0,:] - bias[0])
axs[1,2].plot(times, data[1,:] - bias[1])
axs[2,2].plot(times, data[2,:] - bias[2])

plt.show()
