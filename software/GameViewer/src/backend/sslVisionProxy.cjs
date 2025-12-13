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

const env = process.env.ENVIRONMENT;
const visionAddr = process.env.SSL_VISION_MULTICAST_ADDR;
const visionPort = env == "simulation" ? process.env.SSL_VISION_SIM_MAIN_PORT :
                                         process.env.SSL_VISION_REAL_MAIN_PORT;
const wsAddr = process.env.VITE_SSL_VISION_WS_ADDR;
const wsPort = process.env.VITE_SSL_VISION_WS_PORT;
const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true});
let wss = null;

console.log(`[sslVisionProxy.cjs] Subscribing to ${visionAddr}:${visionPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(visionPort, "0.0.0.0", () => {
  const interfaces = getAllIPv4Interfaces();
  interfaces.forEach(ip => {
    try {
      udpSocket.addMembership(visionAddr, ip);
      console.log(`[sslVisionProxy.cjs] Joined multicast on ${ip}`);
    } catch (err) {
      console.log(`[sslVisionProxy.cjs] Failed to join on ${ip}: ${err.message}`);
    }
  });
  
  console.log(`[sslVisionProxy.cjs] Listening to ${visionAddr}:${visionPort} on ${udpSocket.address().address}:${udpSocket.address().port} (${udpSocket.address().family})`);
  
  wss = new ws.WebSocketServer({ port: wsPort });
  wss.on('connection', (client) => {
    console.log(`[sslVisionProxy.cjs] Frontend client connected to backend`);
    client.on('close', () => {
      console.log(`[sslVisionProxy.cjs] Frontend client disconnected from backend`);
    });
  });
  console.log(`[sslVisionProxy.cjs] Websocket created on ${wss.address().address}:${wss.address().port} (${wss.address().family})`);
});

udpSocket.on('message', (msg) => {
  if (wss) {
    wss.clients.forEach((client) => {
      client.send(msg);
    });
  }
});

udpSocket.on("error", (err) => {
  console.log(`[sslVisionProxy.cjs] UDP Socket error: ${err}`); 
  if (wss) {
    wss.close();
  }
});
