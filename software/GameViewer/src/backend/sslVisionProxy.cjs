const dgram = require('dgram');
const ws = require('ws');

const env = process.env.ENVIRONMENT;
const visionAddr = process.env.SSL_VISION_MULTICAST_ADDR;
const visionPort = env == "simulation" ? process.env.SSL_VISION_SIM_MAIN_PORT :
                                         process.env.SSL_VISION_REAL_MAIN_PORT;
const wsAddr = process.env.VITE_SSL_VISION_WS_ADDR;
const wsPort = process.env.VITE_SSL_VISION_WS_PORT;

const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true });
let wss;

console.log(`[sslVisionProxy.cjs] Subscribing to ${visionAddr}:${visionPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(visionPort, () => {
  udpSocket.addMembership(visionAddr);

  console.log(`[sslVisionProxy.cjs] Listening to ${visionAddr}:${visionPort} on ${udpSocket.address().address}:${udpSocket.address().port} (${udpSocket.address().family})`);

  wss = new ws.WebSocketServer({
    port: wsPort,
  });
  
  wss.on('connection', (ws) => {
    console.log(`[sslVisionProxy.cjs] Client connected to SSL Vision WebSocket`);
  
    ws.on('message', (message) => {
      console.log(`[sslVisionProxy.cjs] Received message from client: ${message}`);
    });
  
    ws.on('close', () => {
      console.log(`[sslVisionProxy.cjs] Client disconnected`);
    });
  });

  console.log(`[sslVisionProxy.cjs] Websocket created on ${wss.address().address}:${wss.address().port} (${wss.address().family})`)
});

udpSocket.on('message', (msg) => {
  if (wss) {
    wss.clients.forEach((client) => {
      if (client.readyState === ws.OPEN) {
        client.send(msg);
      }
    });
  }
});

udpSocket.on("error", (err) => {
   console.log(`[sslVisionProxy.cjs] UDP Socket error: ${err}`); 
   if (wss) {
    wss.close();
   }
});
