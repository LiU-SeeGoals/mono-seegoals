import React from 'react';
import { SSLFieldUpdate } from '../../../types/SSLFieldUpdate';
import './RobotTable.css';
import { Action } from '../../../types/Action';

interface RobotTableProps {
  robotActions: Action[];
  visibleRobots: boolean[];
  sslFieldUpdate: SSLFieldUpdate;
}

const RobotTable: React.FC<RobotTableProps> = ({
  sslFieldUpdate,
}) => {
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
        {[...sslFieldUpdate.robotsYellow]
          .sort((a, b) => (a.robotId ?? 0) - (b.robotId ?? 0))
          .map((robot, index) => (
        <div className="robotItem" key={index}>
          <p>{robot.robotId}</p>
          <p>{robot.x.toFixed(1)}</p>
          <p>{robot.y.toFixed(1)}</p>
          <p>{robot.orientation?.toFixed(5) ?? '—'}</p>
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
        {[...sslFieldUpdate.robotsBlue]
          .sort((a, b) => (a.robotId ?? 0) - (b.robotId ?? 0))
          .map((robot, index) => (
        <div className="robotItem" key={index}>
          <p>{robot.robotId}</p>
          <p>{robot.x.toFixed(1)}</p>
          <p>{robot.y.toFixed(1)}</p>
          <p>{robot.orientation?.toFixed(5) ?? '—'}</p>
        </div>
        ))}
      </div>
    </div>
  );
};

export default RobotTable;
