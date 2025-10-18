import React from 'react';
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
        {sslFieldUpdate.robotsYellow.map((robot, index) => (
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
        {sslFieldUpdate.robotsBlue.map((robot, index) => (
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
