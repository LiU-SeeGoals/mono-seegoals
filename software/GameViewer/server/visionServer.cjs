// Bridges SSL multicast feeds to the browser over one WebSocket:
//   - raw SSL-Vision packets   -> binary frames: all of them with ?source=raw;
//                                 with ?source=tigers only the field geometry,
//                                 on connect and whenever it changes
//   - TIGERs tracker packets   -> {"type":"tracked","update":{...}} (tigers only)
//   - game controller referee  -> {"type":"referee","referee":{...}}
const dgram = require('dgram');
const os = require('os');
const { WebSocketServer, OPEN } = require('ws');
const { decodeTrackedFrame, decodeReferee, geometryPacket } = require('./decode.cjs');

const env = process.env;
const visionAddr = env.SSL_VISION_MULTICAST_ADDR || '224.5.23.2';
// Docker Compose sets SSL_VISION_MULTICAST_PORT to the sim or real port;
// otherwise pick it from ENVIRONMENT.
const visionPort = Number(env.SSL_VISION_MULTICAST_PORT || (env.ENVIRONMENT === 'simulation'
  ? env.SSL_VISION_SIM_MAIN_PORT || 10020
  : env.SSL_VISION_REAL_MAIN_PORT || 10006));
const trackerAddr = env.SSL_TRACKER_ADDR || '224.5.23.2';
const trackerPort = Number(env.SSL_TRACKER_PORT || 10010);
const refereeAddr = env.SSL_GAME_CONTROLLER_PUBLISH_ADDR || '224.5.23.1';
const refereePort = Number(env.SSL_GAME_CONTROLLER_PUBLISH_PORT || 10003);
// VITE_SSL_VISION_WS_ADDR is where the browser connects; listen on all
// interfaces so Docker port mappings work.
const wsPort = Number(env.VITE_SSL_VISION_WS_PORT || 3000);

const MEMBERSHIP_REFRESH_MS = 5000;

const log = (...args) => console.log('[visionServer]', ...args);

function ipv4Interfaces() {
  return Object.values(os.networkInterfaces()).flat()
    .filter((net) => net && net.family === 'IPv4' && !net.internal)
    .map((net) => net.address);
}

function subscribe(name, addr, port, onMessage) {
  const socket = dgram.createSocket({ type: 'udp4', reuseAddr: true });
  socket.on('message', (buffer) => {
    try { onMessage(buffer); }
    catch (error) { console.error(`[visionServer] invalid ${name} packet: ${error.message}`); }
  });
  socket.on('error', (error) => console.error(`[visionServer] ${name} socket error:`, error));
  // Interfaces whose multicast membership succeeded ('' = default interface).
  const joined = new Set();
  const join = (ip) => {
    if (joined.has(ip)) return;
    try {
      if (ip) socket.addMembership(addr, ip);
      else socket.addMembership(addr);
      joined.add(ip);
      log(`${name} joined ${addr} on ${ip || 'default interface'}`);
    } catch (error) {
      console.error(`[visionServer] ${name} membership on ${ip || 'default interface'}: ${error.message}`);
    }
  };
  // Join every IPv4 interface; also retried periodically so a network that
  // comes up after start (or a machine started offline) is picked up.
  const refreshMemberships = () => {
    const interfaces = ipv4Interfaces();
    if (interfaces.length === 0) join('');
    else interfaces.forEach(join);
  };
  socket.bind(port, '0.0.0.0', () => {
    log(`${name} listening on ${addr}:${port}`);
    refreshMemberships();
    setInterval(refreshMemberships, MEMBERSHIP_REFRESH_MS).unref();
  });
  return socket;
}

const wss = new WebSocketServer({ port: wsPort });
let lastGeometry = null; // geometry-only SSL_WrapperPacket for tracker clients

wss.on('listening', () => log(`websocket on port ${wsPort}`));
wss.on('connection', (client, request) => {
  const source = new URL(request.url ?? '/', 'http://localhost').searchParams.get('source');
  client.visionSource = source === 'raw' ? 'raw' : 'tigers';
  log(`browser connected (${client.visionSource})`);
  if (client.visionSource === 'tigers' && lastGeometry) client.send(lastGeometry);
  client.on('close', () => log('browser disconnected'));
});

function broadcast(data, accepts = () => true) {
  for (const client of wss.clients) {
    if (client.readyState === OPEN && accepts(client)) client.send(data);
  }
}

subscribe('vision', visionAddr, visionPort, (buffer) => {
  broadcast(buffer, (client) => client.visionSource === 'raw');
  // Simulators repeat the geometry in every packet; forward it only on change.
  const geometry = geometryPacket(buffer);
  if (geometry && !geometry.equals(lastGeometry ?? Buffer.alloc(0))) {
    lastGeometry = geometry;
    broadcast(geometry, (client) => client.visionSource === 'tigers');
  }
});

subscribe('tracker', trackerAddr, trackerPort, (buffer) => {
  if (wss.clients.size === 0) return;
  const update = decodeTrackedFrame(buffer);
  if (!update) return;
  const message = JSON.stringify({ type: 'tracked', update });
  broadcast(message, (client) => client.visionSource === 'tigers');
});

subscribe('referee', refereeAddr, refereePort, (buffer) => {
  if (wss.clients.size === 0) return;
  broadcast(JSON.stringify({ type: 'referee', referee: decodeReferee(buffer) }));
});
