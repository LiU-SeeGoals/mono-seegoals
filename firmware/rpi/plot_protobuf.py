from read_spi import readFile
from matplotlib import pyplot as plt

zs = []
zs_ts = []
state_zs = []
state_ts = []

m1 = []
m2 = []
m3 = []
m4 = []

m1_r = []
m2_r = []
m3_r = []
m4_r = []

m_ts = []

for msg in readFile():
    zs.append(msg.gyro.z)
    zs_ts.append(msg.gyro.timestamp / 1000.0)
    state_zs.append(msg.state.z)
    state_ts.append(msg.state.timestamp / 1000.0)

    m1.append(msg.m1.control)
    m2.append(msg.m2.control)
    m3.append(msg.m3.control)
    m4.append(msg.m4.control)

    m1_r.append(msg.m1.ref)
    m2_r.append(msg.m2.ref)
    m3_r.append(msg.m3.ref)
    m4_r.append(msg.m4.ref)

    m_ts.append(msg.m_timestamp)

plt.subplot(3,4,1)
plt.title("Angular vel")
plt.plot(zs_ts, zs)

plt.subplot(3,4,2)
plt.title("Angle state")
plt.plot(state_ts, state_zs)

plt.subplot(3,4,3)
plt.title("M2 control signal")
plt.plot(m_ts, m1)

plt.subplot(3,4,4)
plt.title("M2 control signal")
plt.plot(m_ts, m2)

plt.subplot(3,4,5)
plt.title("M3 control signal")
plt.plot(m_ts, m3)

plt.subplot(3,4,6)
plt.title("M4 control signal")
plt.plot(m_ts, m4)

plt.subplot(3,4,7)
plt.title("M2 reference signal")
plt.plot(m_ts, m1_r)

plt.subplot(3,4,8)
plt.title("M2 reference signal")
plt.plot(m_ts, m2_r)

plt.subplot(3,4,9)
plt.title("M3 reference signal")
plt.plot(m_ts, m3_r)

plt.subplot(3,4,10)
plt.title("M4 reference signal")
plt.plot(m_ts, m4_r)

plt.show()

print(sum([zs_ts[i + 1] - zs_ts[i] for i in range(len(zs_ts) - 1)]) / len(zs_ts), "Ts (ms)")
print(sum([1000/(zs_ts[i + 1] - zs_ts[i]) for i in range(len(zs_ts) - 1)]) / len(zs_ts), "Hz")
