from read_spi import readFile
from matplotlib import pyplot as plt

signals = {
    "Angular vel": {
        "t": [],
        "signals": {
            "gyro_z": []
        }
    },
    "Angle state": {
        "t": [],
        "signals": {
            "state_z": []
        }
    },
    "Motor control": {
        "t": [],
        "signals": {
            "m1": [],
            "m2": [],
            "m3": [],
            "m4": [],
        }
    },
    "Motor reference": {
        "t": [],
        "signals": {
            "m1_r": [],
            "m2_r": [],
            "m3_r": [],
            "m4_r": [],
        }
    },
    "Position control": {
        "t": [],
        "signals": {
            "x": [],
            "y": [],
            "angle": [],
        }
    },
    "Position reference": {
        "t": [],
        "signals": {
            "x_r": [],
            "y_r": [],
            "angle_r": [],
        }
    },
}

for msg in readFile():

    # Angular velocity
    signals["Angular vel"]["t"].append(msg.gyro.timestamp / 1000.0)
    signals["Angular vel"]["signals"]["gyro_z"].append(msg.gyro.z)

    # Angle state
    signals["Angle state"]["t"].append(msg.state.timestamp / 1000.0)
    signals["Angle state"]["signals"]["state_z"].append(msg.state.z)

    # Motor control
    signals["Motor control"]["t"].append(msg.m_timestamp / 1000.0)
    signals["Motor control"]["signals"]["m1"].append(msg.m1.control)
    signals["Motor control"]["signals"]["m2"].append(msg.m2.control)
    signals["Motor control"]["signals"]["m3"].append(msg.m3.control)
    signals["Motor control"]["signals"]["m4"].append(msg.m4.control)

    # Motor reference
    signals["Motor reference"]["t"].append(msg.m_timestamp / 1000.0)
    signals["Motor reference"]["signals"]["m1_r"].append(msg.m1.ref)
    signals["Motor reference"]["signals"]["m2_r"].append(msg.m2.ref)
    signals["Motor reference"]["signals"]["m3_r"].append(msg.m3.ref)
    signals["Motor reference"]["signals"]["m4_r"].append(msg.m4.ref)

    # Position control
    signals["Position control"]["t"].append(msg.pos_timestamp / 1000.0)
    signals["Position control"]["signals"]["x"].append(msg.pos_x.control)
    signals["Position control"]["signals"]["y"].append(msg.pos_y.control)
    signals["Position control"]["signals"]["angle"].append(msg.pos_angle.control)

    # Position reference
    signals["Position reference"]["t"].append(msg.pos_timestamp / 1000.0)
    signals["Position reference"]["signals"]["x_r"].append(msg.pos_x.ref)
    signals["Position reference"]["signals"]["y_r"].append(msg.pos_y.ref)
    signals["Position reference"]["signals"]["angle_r"].append(msg.pos_angle.ref)

plt.figure(figsize=(12, 8))

rows = 3
cols = 3

for i, (group_name, group) in enumerate(signals.items(), start=1):
    plt.subplot(rows, cols, i)
    plt.title(group_name)

    t = group["t"]
    for name, sig in group["signals"].items():
        plt.plot(t, sig, label=name)

    plt.legend()

plt.tight_layout()
plt.show()
