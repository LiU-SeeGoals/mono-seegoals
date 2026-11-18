import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { VisionMask } from '../types/VisionMask';

export const isMasked = (mask: VisionMask | null, x: number): boolean =>
  mask !== null && mask.MaskedSign * x > mask.MarginMM;

export const applyVisionMask = (update: SSLFieldUpdate, mask: VisionMask | null): SSLFieldUpdate => {
  if (mask === null) return update;
  return {
    balls: update.balls.filter((ball) => !isMasked(mask, ball.x)),
    robotsBlue: update.robotsBlue.filter((robot) => !isMasked(mask, robot.x)),
    robotsYellow: update.robotsYellow.filter((robot) => !isMasked(mask, robot.x)),
  };
};
