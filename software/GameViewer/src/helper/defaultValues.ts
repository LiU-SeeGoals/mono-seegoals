import { SSLRobot } from '../types/SSLRobot';
import { Action } from '../types/Action';
import { SSLBall } from '../types/SSLBall';
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { AIRobotUpdate } from '../types/AIRobotUpdate';
import { AIBall } from '../types/AIBall';

export function getDefaultSSLFieldUpdate(): SSLFieldUpdate {
  const fieldUpdate: SSLFieldUpdate = {
    balls: [],
    robotsBlue: [],
    robotsYellow: [],
  };
  return fieldUpdate;
}

export function getDefaultAIRobotUpdate(): AIRobotUpdate {
  const robotUpdate: AIRobotUpdate = {
    Robots: [],
    BallPosition: getDefaultAIBall(),
  };
  return robotUpdate;
}

export function getDefaultRobotPosBlue(): SSLRobot[] {
  const robots: SSLRobot[] = [
    { robotId: 0, orientation: 0, x: 2000, y: 0 },
    { robotId: 1, orientation: 0, x: -4500, y: 3000 },
    { robotId: 2, orientation: 0, x: -1000, y: 1000 },
    { robotId: 3, orientation: 0, x: -1000, y: -1000 },
  ];
  return robots;
}

export function getDefaultRobotPosYellow(): SSLRobot[] {
  const robots: SSLRobot[] = [
    { robotId: 0, orientation: 0, x: -3000, y: 3000 },
    { robotId: 1, orientation: 0, x: 360, y: 150 },
    { robotId: 2, orientation: 0, x: 95, y: 600 },
    { robotId: 3, orientation: 0, x: 300, y: 1000 },
  ];
  return robots;
}

export function getDefaultBallPos(): SSLBall {
  const ball = { x: 500, y: 500 };
  return ball;
}

export function getDefaultAIBall(): AIBall {
  const ball = { VelX: 0, VelY: 0, VelW: 0 };
  return ball;
}

export function getDefaultVisibleRobots(): boolean[] {
  const visible = [false, false, false, false, false, false];
  return visible;
}

export function getDefaultActions(): Action[] {
  const actions: Action[] = [
    // {"Id": 0, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 1, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 2, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 3, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 4, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 5, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
    // {"Id": 6, "Action": -1, "PosX": -3000, "PosY": 3000, "PosW": 0, "DestX": 0, "DestY": 0, "DestW": 0, "Dribble": false, "PreviousAction": -1},
  ];
  return actions;
}

export const actionToStr = (actionCode: number) => {
  switch (actionCode) {
    case 0:
      return 'IDLE';
    case 1:
      return 'STOP';
    case 2:
      return 'KICK';
    case 3:
      return 'MOVE';
    case 4:
      return 'INIT';
    case 5:
      return 'SET_NAVIGATION_DIRECTION';
    case 6:
      return 'ROTATE';
    default:
      return 'UNKNOWN';
  }
};

export function getDefaultVectorSetting(): boolean[] {
  const vectorSetting = [false, false, false, false, false, false];
  return vectorSetting;
}

export function getDefaultTraceSetting(): boolean[] {
  const traceSetting = [true, true, true, true, true, true];
  return traceSetting;
}

export function getDefaultLog(): string[] {
  const defaultLog = [
    '18:51 > Lost connection to controller',
    '17:59 > Using strategy fiskmasen',
    'Logs',
  ];
  return defaultLog;
}
