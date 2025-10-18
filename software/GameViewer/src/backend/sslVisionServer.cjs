const dgram = require('dgram');
const ws = require('ws');

const env = process.env.ENVIRONMENT;
const visionAddr = process.env.SSL_VISION_MULTICAST_ADDR;
const visionPort = env == "simulation" ? process.env.SSL_VISION_SIM_MAIN_PORT :
                                         process.env.SSL_VISION_REAL_MAIN_PORT;
const wsAddr = process.env.VITE_SSL_VISION_WS_ADDR;
const wsPort = process.env.VITE_SSL_VISION_WS_PORT;
const udpSocket = dgram.createSocket('udp4');

const wss = new ws.WebSocketServer({
  address: wsAddr,
  port: wsPort,
  'Access-Control-Allow-Origin': '*',
});

wss.on('connection', (ws) => {
  console.log(`Client connected to WebSocket`);

  ws.on('message', (message) => {
    console.log(`Received message from client: ${message}`);
  });

  ws.on('close', () => {
    console.log(`Client disconnected`);
  });
});

udpSocket.bind(visionPort, () => {
  udpSocket.addMembership(visionAddr);
});

udpSocket.on('message', (msg) => {
  //console.log(`Received multicast message: ${msg}`);

  wss.clients.forEach((client) => {
    if (client.readyState === ws.OPEN) {
      client.send(msg);
    }
  });
});
