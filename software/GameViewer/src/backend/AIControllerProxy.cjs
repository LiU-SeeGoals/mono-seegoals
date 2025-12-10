const dgram = require("dgram");
const ws = require("ws");
const os = require("os");

const aiAddr = process.env.AI_ACTIONS_MULTICAST_ADDR;
const aiPort = parseInt(process.env.AI_ACTIONS_MULTICAST_PORT);
const wsAddr = process.env.VITE_AI_CONTROLLER_WS_ADDR;
const wsPort = parseInt(process.env.VITE_AI_CONTROLLER_WS_PORT);

const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true });
let wss;

console.log(`[AIControllerProxy.cjs] Subscribing to ${aiAddr}:${aiPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(aiPort, "0.0.0.0", () => {
  const interfaces = os.networkInterfaces();

  for (const name in interfaces) {
    for (const iface of interfaces[name]) {
      if (iface.family === "IPv4" && !iface.internal) {
        try {
          udpSocket.addMembership(aiAddr, iface.address);
          console.log(`[AIControllerProxy.cjs] Joined multicast group on ${iface.address}`);
        } catch (err) {
          console.log(`[AIControllerProxy.cjs] Failed to join on ${iface.address}: ${err.message}`);
        }
      }
    }
  }

  console.log(`[AIControllerProxy.cjs] Listening to ${aiAddr}:${aiPort} on ${udpSocket.address().address}:${udpSocket.address().port} (${udpSocket.address().family})`);

  wss = new ws.WebSocketServer({
    port: wsPort,
  });

  wss.on('connection', (ws) => {
    console.log(`[AIControllerProxy.cjs] Frontend client connected to backend`);

    ws.on('message', (message) => {
      console.log(`[AIControllerProxy.cjs] Received message from client: ${message}`);
    });

    ws.on('close', () => {
      console.log(`[AIControllerProxy.cjs] Frontend client disconnected from backend`);
    });
  });

  const addr = wss.address();
  if (addr) {
    console.log`[AIControllerProxy.cjs] Websocket created on ${addr.address}:${addr.port} (${addr.family})`;
  } else {
    console.log`[AIControllerProxy.cjs] Websocket created on port ${wsPort}`;
  }
});

udpSocket.on('message', (msg) => {
  if (wss) {
    const text = msg.toString("utf8");
    wss.clients.forEach((client) => {
      if (client.readyState === ws.OPEN) {
        client.send(text);
      }
    });
  }
});

udpSocket.on("error", (err) => {
   console.log(`[AIControllerProxy.cjs] UDP Socket error: ${err}`); 
   if (wss) {
    wss.close();
   }
});
