const { test } = require('node:test');
const assert = require('node:assert/strict');
const {
  decodeTrackedFrame, decodeReferee, geometryPacket,
  TrackerWrapperPacket, SSLWrapperPacket, Referee,
} = require('./decode.cjs');
const encode = frame => TrackerWrapperPacket.encode(TrackerWrapperPacket.create({
  uuid: 'test', trackedFrame: { frameNumber: 1, timestamp: 1, ...frame },
})).finish();
test('converts both teams and balls from meters to millimeters', () => {
  const update = decodeTrackedFrame(encode({ robots: [
    { robotId: { id: 0, team: 1 }, pos: { x: 1, y: -2 }, orientation: 0 },
    { robotId: { id: 3, team: 2 }, pos: { x: -1, y: 2 }, orientation: 1 },
  ], balls: [{ pos: { x: 1, y: 2, z: 0 } }] }));
  assert.deepEqual(update.robotsYellow, [{ robotId: 0, x: 1000, y: -2000, orientation: 0 }]);
  assert.equal(update.robotsBlue[0].robotId, 3);
  assert.deepEqual(update.balls, [{ x: 1000, y: 2000 }]);
});
test('drops explicitly invisible objects and replaces the whole frame', () => {
  assert.deepEqual(decodeTrackedFrame(encode({ robots: [
    { robotId: { id: 1, team: 1 }, pos: { x: 1, y: 1 }, orientation: 0, visibility: 0 },
  ], balls: [{ pos: { x: 1, y: 1, z: 0 }, visibility: 0 }] })),
  { balls: [], robotsYellow: [], robotsBlue: [] });
  assert.deepEqual(decodeTrackedFrame(encode({})), { balls: [], robotsYellow: [], robotsBlue: [] });
});
test('metadata packets do not replace current detections', () => {
  assert.equal(decodeTrackedFrame(TrackerWrapperPacket.encode({ uuid: 'test' }).finish()), null);
});
test('geometryPacket keeps only the geometry of a raw packet', () => {
  const detection = SSLWrapperPacket.encode({ detection: {
    frameNumber: 1, tCapture: 0, tSent: 0, cameraId: 0, balls: [], robotsYellow: [], robotsBlue: [],
  } }).finish();
  const geometry = SSLWrapperPacket.encode({ geometry: { field: {
    fieldLength: 9000, fieldWidth: 6000, goalWidth: 1000, goalDepth: 180, boundaryWidth: 300,
  }, calib: [] } }).finish();
  assert.equal(geometryPacket(detection), null);
  const both = SSLWrapperPacket.encode({
    ...SSLWrapperPacket.decode(detection), ...SSLWrapperPacket.decode(geometry),
  }).finish();
  const extracted = SSLWrapperPacket.decode(geometryPacket(both));
  assert.equal(extracted.detection, null);
  assert.equal(extracted.geometry.field.fieldLength, 9000);
  assert.ok(geometryPacket(both).equals(geometryPacket(geometry)));
});
test('referee packets decode to enum names and scores', () => {
  const team = (name, score) => ({ name, score, redCards: 0, yellowCardTimes: [], yellowCards: 0,
    timeouts: 0, timeoutTime: 0, goalkeeper: 0 });
  const referee = decodeReferee(Referee.encode({
    packetTimestamp: 1, stage: 1, command: 2, commandCounter: 1, commandTimestamp: 1,
    yellow: team('SeeGoals', 2), blue: team('TIGERs', 1),
  }).finish());
  assert.deepEqual(referee, { stage: 'NORMAL_FIRST_HALF', command: 'NORMAL_START',
    yellow: { name: 'SeeGoals', score: 2 }, blue: { name: 'TIGERs', score: 1 } });
});
