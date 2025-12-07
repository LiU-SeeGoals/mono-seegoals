const dgram = require('dgram');
const ws = require('ws');

const gcAddr = process.env.SSL_GAME_CONTROLLER_PUBLISH_ADDR || '224.5.23.1';
const gcPort = parseInt(process.env.SSL_GAME_CONTROLLER_PUBLISH_PORT || '11003');
const wsAddr = process.env.VITE_SSL_GAME_CONTROLLER_WS_ADDR || '127.0.0.1';
const wsPort = parseInt(process.env.VITE_SSL_GAME_CONTROLLER_WS_PORT || '3001');

const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true });
let wss;

console.log(`[gameControllerProxy.cjs] Subscribing to ${gcAddr}:${gcPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(gcPort, () => {
  udpSocket.addMembership(gcAddr);
  console.log(`[gameControllerProxy.cjs] Listening to ${gcAddr}:${gcPort} on ${udpSocket.address().address}:${udpSocket.address().port} (${udpSocket.address().family})`);

  wss = new ws.WebSocketServer({
    port: wsPort,
  });

  wss.on('connection', (ws) => {
    console.log(`[gameControllerProxy.cjs] Frontend client connected to backend`);
    ws.on('message', (message) => {
      console.log(`Received message from client: ${message}`);
    });
    ws.on('close', () => {
      console.log(`[gameControllerProxy.cjs] Frontend client disconnected from backend`);
    });
  });

  console.log(`[gameControllerProxy.cjs] Websocket created on ${wss.address().address}:${wss.address().port} (${wss.address().family})`)
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
  console.log("[gameControllerProxy.cjs] UDP socket error: ", err);
  if (wss) {
    wss.close();
  }
});
