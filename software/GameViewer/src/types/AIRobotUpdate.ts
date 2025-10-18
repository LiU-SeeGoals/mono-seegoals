import { AIBall } from './AIBall';
import { AIRobot } from './AIRobot';

export type AIRobotUpdate = {
  Robots: AIRobot[];
  BallPosition: AIBall;
};
