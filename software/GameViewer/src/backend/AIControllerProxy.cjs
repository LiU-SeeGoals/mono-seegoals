const dgram = require('dgram');
const ws = require('ws');

const env = process.env.ENVIRONMENT;
const aiAddr = "239.0.0.1";
const aiPort = "9999";
const wsAddr = "127.0.0.1";
const wsPort = "3002";

const udpSocket = dgram.createSocket({type: "udp4", reuseAddr: true });
let wss;

console.log(`[AIControllerProxy.cjs] Subscribing to ${aiAddr}:${aiPort} and passing on to ${wsAddr}:${wsPort}`);

udpSocket.bind(aiPort, () => {
  udpSocket.addMembership(aiAddr);

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

  console.log(`[AIControllerProxy.cjs] Websocket created on ${wss.address().address}:${wss.address().port} (${wss.address().family})`)
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
