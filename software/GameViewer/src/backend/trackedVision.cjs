const path = require('path');
const protobuf = require('protobufjs');
const root = new protobuf.Root();
root.resolvePath = (_origin, target) => path.join(__dirname, 'proto', target);
root.loadSync('ssl_vision/ssl_wrapper_tracked.proto');
const TrackerWrapperPacket = root.lookupType('TrackerWrapperPacket');

function decodeTrackedFrame(buffer) {
  const frame = TrackerWrapperPacket.decode(buffer).trackedFrame;
  if (!frame) return null;
  // Visibility is optional in the protocol. Explicitly invisible predictions
  // must not remain on screen as if they were current observations.
  const visible = (object) => !Object.prototype.hasOwnProperty.call(object, 'visibility') || object.visibility > 0;
  const update = { balls: [], robotsBlue: [], robotsYellow: [] };
  for (const robot of frame.robots) {
    if (!visible(robot) || !robot.pos || !robot.robotId) continue;
    const team = robot.robotId.teamColor === 1 ? update.robotsYellow
      : robot.robotId.teamColor === 2 ? update.robotsBlue : null;
    team?.push({ robotId: robot.robotId.id, x: robot.pos.x * 1000,
      y: robot.pos.y * 1000, orientation: robot.orientation });
  }
  update.balls = frame.balls.filter(visible).filter(ball => ball.pos)
    .map(ball => ({ x: ball.pos.x * 1000, y: ball.pos.y * 1000 }));
  return update;
}
module.exports = { decodeTrackedFrame, TrackerWrapperPacket };
