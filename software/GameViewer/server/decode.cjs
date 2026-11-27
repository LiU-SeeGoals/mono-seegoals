const protobuf = require('protobufjs');
// Generated from software/proto-messages by its build_ts.sh.
const root = protobuf.Root.fromJSON(require('./proto/ssl.json').root);
const TrackerWrapperPacket = root.lookupType('TrackerWrapperPacket');
const SSLWrapperPacket = root.lookupType('SSL_WrapperPacket');
const Referee = root.lookupType('Referee');
const STAGE_NAMES = Referee.lookupEnum('Stage').valuesById;
const COMMAND_NAMES = Referee.lookupEnum('Command').valuesById;

function decodeTrackedFrame(buffer) {
  const frame = TrackerWrapperPacket.decode(buffer).trackedFrame;
  if (!frame) return null;
  // Visibility is optional in the protocol. Explicitly invisible predictions
  // must not remain on screen as if they were current observations.
  const visible = (object) => !Object.prototype.hasOwnProperty.call(object, 'visibility') || object.visibility > 0;
  const update = { balls: [], robotsBlue: [], robotsYellow: [] };
  for (const robot of frame.robots) {
    if (!visible(robot) || !robot.pos || !robot.robotId) continue;
    const team = robot.robotId.team === 1 ? update.robotsYellow
      : robot.robotId.team === 2 ? update.robotsBlue : null;
    team?.push({ robotId: robot.robotId.id, x: robot.pos.x * 1000,
      y: robot.pos.y * 1000, orientation: robot.orientation });
  }
  update.balls = frame.balls.filter(visible).filter(ball => ball.pos)
    .map(ball => ({ x: ball.pos.x * 1000, y: ball.pos.y * 1000 }));
  return update;
}

// The geometry of a raw SSL_WrapperPacket, re-encoded as a geometry-only
// packet, or null if it has none. In tracker mode the browser needs only this,
// not every camera's detections.
function geometryPacket(buffer) {
  const { geometry } = SSLWrapperPacket.decode(buffer);
  return geometry ? Buffer.from(SSLWrapperPacket.encode({ geometry }).finish()) : null;
}

// The small part of a referee packet the viewer shows.
function decodeReferee(buffer) {
  const msg = Referee.decode(buffer);
  return {
    stage: STAGE_NAMES[msg.stage] ?? String(msg.stage),
    command: COMMAND_NAMES[msg.command] ?? String(msg.command),
    yellow: { name: msg.yellow?.name ?? '', score: msg.yellow?.score ?? 0 },
    blue: { name: msg.blue?.name ?? '', score: msg.blue?.score ?? 0 },
  };
}

module.exports = {
  decodeTrackedFrame, decodeReferee, geometryPacket,
  TrackerWrapperPacket, SSLWrapperPacket, Referee,
};
