import React from 'react';
import './RobotTable.css';
import { applyVisionMask } from '../../../helper/visionMask';
import { useThrottledStore } from '../../../state/liveStore';
import { fieldStore, visionMaskStore } from '../../../state/stores';
import { SSLRobot } from '../../../types/SSLRobot';

// Human-readable numbers do not need 60 Hz; refresh the table 5 times a second.
const REFRESH_MS = 200;

const TeamTable: React.FC<{ title: string; robots: SSLRobot[] }> = ({ title, robots }) => (
  <>
    <h5>{title}</h5>
    <div className="robotTable-wrapper">
      <div className="robotItem header">
        <p>ID</p>
        <p>x</p>
        <p>y</p>
        <p>Angle</p>
      </div>
      {[...robots]
        .sort((a, b) => (a.robotId ?? 0) - (b.robotId ?? 0))
        .map((robot, index) => (
          <div className="robotItem" key={robot.robotId ?? `unknown-${index}`}>
            <p>{robot.robotId}</p>
            <p>{robot.x.toFixed(1)}</p>
            <p>{robot.y.toFixed(1)}</p>
            <p>{robot.orientation?.toFixed(5) ?? '—'}</p>
          </div>
        ))}
    </div>
  </>
);

const RobotTable: React.FC = () => {
  const field = useThrottledStore(fieldStore, REFRESH_MS);
  const mask = useThrottledStore(visionMaskStore, REFRESH_MS);
  const visible = applyVisionMask(field, mask);
  return (
    <div>
      <h4>Robots</h4>
      <TeamTable title="Yellow" robots={visible.robotsYellow} />
      <TeamTable title="Blue" robots={visible.robotsBlue} />
    </div>
  );
};

export default RobotTable;
