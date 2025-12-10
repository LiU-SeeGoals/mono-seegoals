const dgram = require("dgram");
const ws = require("ws");
const os = require("os");

function getAllIPv4Interfaces() {
  const nets = os.networkInterfaces();
  const ips = [];
  for (const name of Object.keys(nets)) {
    for (const net of nets[name]) {
      if (net.family === 'IPv4' && !net.internal) {
        ips.push(net.address);
      }
    }
  }
  return ips;
}

const aiAddr = process.env.AI_ACTIONS_MULTICAST_ADDR;
const aiPort = parseInt(process.env.AI_ACTIONS_MULTICAST_PORT);
const wsAddr = process.env.VITE_AI_CONTROLLER_WS_ADDR;
const wsPort = parseInt(process.env.VITE_AI_CONTROLLER_WS_PORT);
const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true });
let wss;

console.log(`[AIControllerProxy.cjs] Subscribing to ${aiAddr}:${aiPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(aiPort, "0.0.0.0", () => {
  const interfaces = getAllIPv4Interfaces();
  interfaces.forEach(ip => {
    try {
      udpSocket.addMembership(aiAddr, ip);
      console.log(`[AIControllerProxy.cjs] Joined multicast on ${ip}`);
    } catch (err) {
      console.log(`[AIControllerProxy.cjs] Failed to join on ${ip}: ${err.message}`);
    }
  });
  
  console.log(`[AIControllerProxy.cjs] Listening to ${aiAddr}:${aiPort} on ${udpSocket.address().address}:${udpSocket.address().port} (${udpSocket.address().family})`);
  
  wss = new ws.WebSocketServer({
    port: wsPort,
  });
  wss.on('connection', (client) => {
    console.log(`[AIControllerProxy.cjs] Frontend client connected to backend`);
    client.on('close', () => {
      console.log(`[AIControllerProxy.cjs] Frontend client disconnected from backend`);
    });
  });
  
  const addr = wss.address();
  if (addr) {
    console.log(`[AIControllerProxy.cjs] Websocket created on ${addr.address}:${addr.port} (${addr.family})`);
  } else {
    console.log(`[AIControllerProxy.cjs] Websocket created on port ${wsPort}`);
  }
});

udpSocket.on('message', (msg) => {
  if (wss) {
    const text = msg.toString("utf8");
    wss.clients.forEach((client) => {
      client.send(text);
    });
  }
});

udpSocket.on("error", (err) => {
  console.log(`[AIControllerProxy.cjs] UDP Socket error: ${err}`); 
  if (wss) {
    wss.close();
  }
});
