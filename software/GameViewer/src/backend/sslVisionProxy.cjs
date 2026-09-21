const dgram = require("dgram");
const ws = require("ws");
const os = require("os");
const { decodeTrackedFrame } = require("./trackedVision.cjs");

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
const visionAddr = process.env.SSL_VISION_MULTICAST_ADDR || "224.5.23.2";
const visionPort = env == "simulation" ? (process.env.SSL_VISION_SIM_MAIN_PORT || 10020) :
                                         (process.env.SSL_VISION_REAL_MAIN_PORT || 10006);
const wsAddr = process.env.VITE_SSL_VISION_WS_ADDR;
const wsPort = Number(process.env.VITE_SSL_VISION_WS_PORT || 3000);
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
  wss.on('connection', (client, request) => {
    client.visionSource = new URL(request.url, 'http://localhost').searchParams.get('source');
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
      if (client.readyState === ws.OPEN) client.send(msg);
    });
  }
});

udpSocket.on("error", (err) => {
  console.log(`[sslVisionProxy.cjs] UDP Socket error: ${err}`); 
  if (wss) {
    wss.close();
  }
});

// Keep raw packets available for field geometry, even in filtered mode.
const trackerSocket = dgram.createSocket({ type: 'udp4', reuseAddr: true });
const trackerAddr = process.env.SSL_TRACKER_ADDR || '224.5.23.2';
const trackerPort = Number(process.env.SSL_TRACKER_PORT || 10010);
trackerSocket.bind(trackerPort, '0.0.0.0', () => {
  for (const ip of getAllIPv4Interfaces()) {
    try { trackerSocket.addMembership(trackerAddr, ip); }
    catch (error) { console.error(`Tracker membership ${ip}: ${error.message}`); }
  }
  console.log(`Tigers tracker listening on ${trackerAddr}:${trackerPort}`);
});
trackerSocket.on('message', (buffer) => {
  try {
    const update = decodeTrackedFrame(buffer);
    if (!update || !wss) return;
    const message = JSON.stringify({ type: 'tracked', update });
    for (const client of wss.clients) {
      if (client.readyState === ws.OPEN && client.visionSource === 'tigers') client.send(message);
    }
  } catch (error) { console.error(`Invalid tracker packet: ${error.message}`); }
});
trackerSocket.on('error', error => console.error('Tracker socket error:', error));
