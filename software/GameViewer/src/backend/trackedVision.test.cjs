const { test } = require('node:test');
const assert = require('node:assert/strict');
const { decodeTrackedFrame, TrackerWrapperPacket } = require('./trackedVision.cjs');
const encode = frame => TrackerWrapperPacket.encode(TrackerWrapperPacket.create({
  uuid: 'test', trackedFrame: { frameNumber: 1, timestamp: 1, ...frame },
})).finish();
test('converts both teams and balls from meters to millimeters', () => {
  const update = decodeTrackedFrame(encode({ robots: [
    { robotId: { id: 0, teamColor: 1 }, pos: { x: 1, y: -2 }, orientation: 0 },
    { robotId: { id: 3, teamColor: 2 }, pos: { x: -1, y: 2 }, orientation: 1 },
  ], balls: [{ pos: { x: 1, y: 2, z: 0 } }] }));
  assert.deepEqual(update.robotsYellow, [{ robotId: 0, x: 1000, y: -2000, orientation: 0 }]);
  assert.equal(update.robotsBlue[0].robotId, 3);
  assert.deepEqual(update.balls, [{ x: 1000, y: 2000 }]);
});
test('drops explicitly invisible objects and replaces the whole frame', () => {
  assert.deepEqual(decodeTrackedFrame(encode({ robots: [
    { robotId: { id: 1, teamColor: 1 }, pos: { x: 1, y: 1 }, orientation: 0, visibility: 0 },
  ], balls: [{ pos: { x: 1, y: 1, z: 0 }, visibility: 0 }] })),
  { balls: [], robotsYellow: [], robotsBlue: [] });
  assert.deepEqual(decodeTrackedFrame(encode({})), { balls: [], robotsYellow: [], robotsBlue: [] });
});
test('metadata packets do not replace current detections', () => {
  assert.equal(decodeTrackedFrame(TrackerWrapperPacket.encode({ uuid: 'test' }).finish()), null);
});
