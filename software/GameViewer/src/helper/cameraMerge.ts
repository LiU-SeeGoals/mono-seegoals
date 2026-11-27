import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { SSLRobot } from '../types/SSLRobot';

// Raw SSL-Vision sends one packet per camera. Keep each camera's latest
// detections and merge them, so robots seen by camera 0 do not vanish when
// camera 1's packet arrives. A camera that stops sending drops out after
// maxAgeMs.
export class CameraMerger {
  private frames = new Map<number, { update: SSLFieldUpdate; receivedAt: number }>();

  constructor(private readonly maxAgeMs = 500) {}

  add(cameraId: number, update: SSLFieldUpdate, now: number): SSLFieldUpdate {
    this.frames.set(cameraId, { update, receivedAt: now });
    for (const [id, frame] of this.frames) {
      if (now - frame.receivedAt > this.maxAgeMs) this.frames.delete(id);
    }
    const updates = [...this.frames.values()].map((frame) => frame.update);
    return {
      // Highest confidence first, so balls[0] is the best detection.
      balls: updates.flatMap((u) => u.balls)
        .sort((a, b) => (b.confidence ?? 0) - (a.confidence ?? 0)),
      robotsBlue: mergeRobots(updates.map((u) => u.robotsBlue)),
      robotsYellow: mergeRobots(updates.map((u) => u.robotsYellow)),
    };
  }

  clear(): void {
    this.frames.clear();
  }
}

// A robot in the overlap of two cameras is reported by both; keep the more
// confident detection.
function mergeRobots(perCamera: SSLRobot[][]): SSLRobot[] {
  const best = new Map<number, SSLRobot>();
  const unidentified: SSLRobot[] = [];
  for (const robots of perCamera) {
    for (const robot of robots) {
      if (robot.robotId === undefined) {
        unidentified.push(robot);
        continue;
      }
      const current = best.get(robot.robotId);
      if (!current || (robot.confidence ?? 0) > (current.confidence ?? 0)) best.set(robot.robotId, robot);
    }
  }
  return [...best.values(), ...unidentified];
}
