const dgram = require('dgram');
const ws = require('ws');

const gcAddr = process.env.SSL_GAME_CONTROLLER_PUBLISH_ADDR || '224.5.23.1';
const gcPort = parseInt(process.env.SSL_GAME_CONTROLLER_PUBLISH_PORT || '11003');
const wsAddr = process.env.VITE_SSL_GAME_CONTROLLER_WS_ADDR || '127.0.0.1';
const wsPort = parseInt(process.env.VITE_SSL_GAME_CONTROLLER_WS_PORT || '3001');

const udpSocket = dgram.createSocket('udp4');
const wss = new ws.WebSocketServer({
  port: wsPort,
});

wss.on('connection', (ws) => {
  console.log(`Client connected to Game Controller WebSocket`);
  ws.on('message', (message) => {
    console.log(`Received message from client: ${message}`);
  });
  ws.on('close', () => {
    console.log(`Client disconnected from Game Controller`);
  });
});

udpSocket.bind(gcPort, () => {
  console.log(`Listening for Game Controller multicast on ${gcAddr}:${gcPort}`);
  udpSocket.addMembership(gcAddr);
});

udpSocket.on('message', (msg) => {
  wss.clients.forEach((client) => {
    if (client.readyState === ws.OPEN) {
      client.send(msg);
    }
  });
});

console.log(`Game Controller WebSocket server running on ws://${wsAddr}:${wsPort}`);
