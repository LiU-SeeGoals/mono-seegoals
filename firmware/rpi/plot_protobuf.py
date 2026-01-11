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
    "State x": {
        "t": [],
        "signals": {
            "x": [],
        }
    },
    "State y": {
        "t": [],
        "signals": {
            "x": [],
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
            "m1": [],
            "m2": [],
            "m3": [],
            "m4": [],
        }
    },
    "Motor error": {
        "t": [],
        "signals": {
            "m1": [],
            "m2": [],
            "m3": [],
            "m4": [],
        }
    },
    "Motor output": {
        "t": [],
        "signals": {
            "m1": [],
            "m2": [],
            "m3": [],
            "m4": [],
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
    "Vision": {
        "t": [],
        "signals": {
            "x": [],
            "y": [],
            "angle": [],
        }
    }
}

def outlier(val):
    outlier_min = 1
    outlier_max = 1e7
    return val < outlier_min * 1000 or val > outlier_max * 1000


for msg in readFile():
    if not outlier(msg.gyro.timestamp):
        # Angular velocity
        signals["Angular vel"]["t"].append(msg.gyro.timestamp / 1000.0)
        signals["Angular vel"]["signals"]["gyro_z"].append(msg.gyro.z)

    if not outlier(msg.state.timestamp):
        # Robot state
        signals["State x"]["t"].append(msg.state.timestamp / 1000.0)
        signals["State x"]["signals"]["x"].append(msg.state.x)

        signals["State y"]["t"].append(msg.state.timestamp / 1000.0)
        signals["State y"]["signals"]["x"].append(msg.state.y)

        signals["Angle state"]["t"].append(msg.state.timestamp / 1000.0)
        signals["Angle state"]["signals"]["state_z"].append(msg.state.z)

    # Vision updates
    if not outlier(msg.vision.timestamp):
        signals["Vision"]["t"].append(msg.vision.timestamp / 1000.0)
        signals["Vision"]["signals"]["x"].append(msg.vision.x)
        signals["Vision"]["signals"]["y"].append(msg.vision.y)
        signals["Vision"]["signals"]["angle"].append(msg.vision.z)

    if not outlier(msg.m_timestamp):
        # Motor control
        signals["Motor control"]["t"].append(msg.m_timestamp / 1000.0)
        signals["Motor control"]["signals"]["m1"].append(msg.m1.control)
        signals["Motor control"]["signals"]["m2"].append(msg.m2.control)
        signals["Motor control"]["signals"]["m3"].append(msg.m3.control)
        signals["Motor control"]["signals"]["m4"].append(msg.m4.control)

        # Motor reference
        signals["Motor reference"]["t"].append(msg.m_timestamp / 1000.0)
        signals["Motor reference"]["signals"]["m1"].append(msg.m1.ref)
        signals["Motor reference"]["signals"]["m2"].append(msg.m2.ref)
        signals["Motor reference"]["signals"]["m3"].append(msg.m3.ref)
        signals["Motor reference"]["signals"]["m4"].append(msg.m4.ref)

        # Motor reference
        signals["Motor error"]["t"].append(msg.m_timestamp / 1000.0)
        signals["Motor error"]["signals"]["m1"].append(msg.m1.error)
        signals["Motor error"]["signals"]["m2"].append(msg.m2.error)
        signals["Motor error"]["signals"]["m3"].append(msg.m3.error)
        signals["Motor error"]["signals"]["m4"].append(msg.m4.error)

        # Motor reference
        signals["Motor output"]["t"].append(msg.m_timestamp / 1000.0)
        signals["Motor output"]["signals"]["m1"].append(msg.m1.output)
        signals["Motor output"]["signals"]["m2"].append(msg.m2.output)
        signals["Motor output"]["signals"]["m3"].append(msg.m3.output)
        signals["Motor output"]["signals"]["m4"].append(msg.m4.output)

    if not outlier(msg.pos_timestamp):
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

rows = 4
cols = 4

for i, (group_name, group) in enumerate(signals.items(), start=1):
    plt.subplot(rows, cols, i)
    plt.title(group_name)

    t = group["t"]
    tidx = 0
    removes = []

    for name, sig in group["signals"].items():
        plt.plot(t[2:], sig[2:], label=name)

    plt.legend()

plt.tight_layout()
plt.show()
