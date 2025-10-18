import { SSLBall } from './SSLBall';
import { SSLRobot } from './SSLRobot';

export type SSLFieldUpdate = {
  balls: SSLBall[];
  robotsBlue: SSLRobot[];
  robotsYellow: SSLRobot[];
};
