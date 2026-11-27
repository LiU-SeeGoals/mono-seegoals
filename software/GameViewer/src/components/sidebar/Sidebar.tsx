import React from 'react';
import './Sidebar.css';
import useResizeSidebar from '../../hooks/useResizeSidebar';
import ExternalLink from './externalLink/ExternalLink';
import Header from './header/Header';
import RobotTable from './robotTable/RobotTable';
import { VisionSource } from '../../hooks/useVisionServer';
import { Referee } from '../../types/Referee';

interface SidebarProps {
  visionSource: VisionSource;
  setVisionSource: (source: VisionSource) => void;
  isConnectedToVision: boolean;
  isConnectedToAI: boolean;
  referee: Referee | null;
  controllerSend: (data: unknown) => void;
}

const RESIZER_WIDTH = 5;
const START_WIDTH = 320;

const StatusRow: React.FC<{ ok: boolean; label: string }> = ({ ok, label }) => (
  <div className="sidebar-status-row">
    <div className="sidebar-status-dot" style={{
      backgroundColor: ok ? '#4CAF50' : '#f44336',
      boxShadow: ok ? '0 0 8px #4CAF50' : 'none',
    }} />
    <span>{label}</span>
  </div>
);

const Sidebar: React.FC<SidebarProps> = ({
  visionSource,
  setVisionSource,
  isConnectedToVision,
  isConnectedToAI,
  referee,
  controllerSend,
}) => {
  const { value: sidebarWidth, startResizing, isHidden } = useResizeSidebar(false, START_WIDTH);

  const changeScenario = (type: 'Game' | 'Manual') =>
    controllerSend({ Command: 'CHANGE_SCENARIO', Type: type });

  return (
    <div className="sidebar-wrapper">
      <div className="sidebar" style={{ display: isHidden ? 'none' : 'inline', width: sidebarWidth }}>
        <div className="sidebar-content">
          <Header />
          <hr />
          <ExternalLink text="SSL Vision Client" link="http://localhost:8082" />
          <ExternalLink text="SSL Game Controller" link="http://localhost:8081" />
          <hr />

          <div style={{ padding: '10px 15px' }}>
            <label style={{ display: 'block', marginBottom: 12 }}>
              Vision source
              <select aria-label="Vision source" value={visionSource}
                onChange={(event) => setVisionSource(event.target.value as VisionSource)}
                style={{ display: 'block', width: '100%', marginTop: 6 }}>
                <option value="tigers">Tigers filter</option>
                <option value="raw">Raw SSL Vision</option>
              </select>
            </label>
            <div style={{ marginBottom: 10 }}>Connection status</div>
            <StatusRow ok={isConnectedToVision} label={visionSource === 'tigers' ? 'Tigers filter' : 'SSL Vision'} />
            <StatusRow ok={referee !== null} label="Game Controller" />
            <StatusRow ok={isConnectedToAI} label="AI Controller" />
            {referee && (
              <div className="sidebar-referee">
                <div><b>{referee.command}</b> · {referee.stage}</div>
                <div>
                  <span className="sidebar-team-yellow">{referee.yellow.name || 'Yellow'}</span>
                  {' '}{referee.yellow.score} : {referee.blue.score}{' '}
                  <span className="sidebar-team-blue">{referee.blue.name || 'Blue'}</span>
                </div>
              </div>
            )}
          </div>

          <hr />
          <RobotTable />
          <button className="sidebar-button" onClick={() => changeScenario('Game')}>Game</button>
          <button className="sidebar-button" onClick={() => changeScenario('Manual')}>Manual</button>
        </div>
      </div>
      <div className="sidebar-resizer" style={{ width: RESIZER_WIDTH }} onMouseDown={startResizing} />
    </div>
  );
};

export default Sidebar;
