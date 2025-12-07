import React, { useState, useEffect } from 'react';
import './RobotTable.css';
import LensIcon from '@mui/icons-material/Lens';
import InfoIcon from '@mui/icons-material/Info';
import { Action } from '../../../types/Action';
import { actionToStr } from '../../../helper/defaultValues';

interface RobotTableProps {
  robotActions: Action[];
  visibleRobots: boolean[];
  sslFieldUpdate: SSLFieldUpdate;
}

const RobotTable: React.FC<RobotTableProps> = ({
  robotActions,
  visibleRobots,
  sslFieldUpdate,
}) => {
  const [yellowRobots, setYellowRobots] = useState<Map<number, any>>(new Map());
  const [blueRobots, setBlueRobots] = useState<Map<number, any>>(new Map());

  useEffect(() => {
    setYellowRobots(prev => {
      const updated = new Map(prev);
      sslFieldUpdate.robotsYellow.forEach(robot => {
        updated.set(robot.robotId, robot);
      });
      return updated;
    });

    setBlueRobots(prev => {
      const updated = new Map(prev);
      sslFieldUpdate.robotsBlue.forEach(robot => {
        updated.set(robot.robotId, robot);
      });
      return updated;
    });
  }, [sslFieldUpdate]);

  const tip = 'This only shows if the SSL vision can currenty see the robot';

  return (
    <div>
      <h4>Robots</h4>
      <h5>Yellow</h5>
      <div className="robotTable-wrapper">
        <div className="robotItem header">
          <p>ID</p>
          <p>x</p>
          <p>y</p>
          <p>Angle</p>
        </div>
        {Array.from(yellowRobots.values()).map((robot, index) => (
        <div className="robotItem" key={index}>
          <p>{robot.robotId}</p>
          <p>{robot.x.toFixed(1)}</p>
          <p>{robot.y.toFixed(1)}</p>
          <p>{robot.orientation.toFixed(5)}</p>
        </div>
        ))}
      </div>

      <h5>Blue</h5>
      <div className="robotTable-wrapper">
        <div className="robotItem header">
          <p>ID</p>
          <p>x</p>
          <p>y</p>
          <p>Angle</p>
        </div>
        {Array.from(blueRobots.values()).map((robot, index) => (
        <div className="robotItem" key={index}>
          <p>{robot.robotId}</p>
          <p>{robot.x.toFixed(1)}</p>
          <p>{robot.y.toFixed(1)}</p>
          <p>{robot.orientation.toFixed(5)}</p>
        </div>
        ))}
      </div>
    </div>
  );
};

export default RobotTable;
