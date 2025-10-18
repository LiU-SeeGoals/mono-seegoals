import React, { useState, useEffect } from 'react';
import './Sidebar.css';
import useResizeSidebar from '../../hooks/useResizeSidebar';
import ExternalLink from './externalLink/ExternalLink';
import Header from './header/Header';
import ToggleSetting from './toggleSetting/ToggleSetting';
import ButtonSetting from './buttonSetting/ButtonSetting';
import RobotTable from './robotTable/RobotTable';
import StatusCircle from './statusCircle/StatusCircle';

import { Action } from '../../types/Action';

interface SidebarProps {
  vectorSettingBlue: boolean[];
  setVectorSettingBlue: React.Dispatch<React.SetStateAction<boolean[]>>;
  vectorSettingYellow: boolean[];
  setVectorSettingYellow: React.Dispatch<React.SetStateAction<boolean[]>>;
  traceSetting: boolean[];
  setTraceSetting: React.Dispatch<React.SetStateAction<boolean[]>>;
  robotActions: Action[];
  visibleRobots: boolean[];
  isConnectedToController: boolean;
  sslFieldUpdate: SSLFieldUpdate;
  sidebarWidth: number,
  setSidebarWidth: React.Dispatch<React.SetStateAction<number>>;
}

const Sidebar: React.FC<SidebarProps> = ({
  vectorSettingBlue,
  setVectorSettingBlue,
  vectorSettingYellow,
  setVectorSettingYellow,
  traceSetting,
  setTraceSetting,
  robotActions,
  visibleRobots,
  isConnectedToController,
  sslFieldUpdate,
  sidebarWidth,
  setSidebarWidth,
}) => {
  const resizerWidth = 5;

  const { value: resizerValue, startResizing, isHidden } = useResizeSidebar(false, sidebarWidth);

  useEffect(() => {
    setSidebarWidth(resizerValue);
  }, [resizerValue, setSidebarWidth]);

  const contentDisplay: string = resizerValue < resizerWidth ? 'none' : 'inline';

  const vectorTip = "Are you stupid? It's kind of self-explanatory";
  const traceTip =
    "Trace shows a line of where the robot has been. It's like a snail trail but for robots.";

  return (
    <div className="sidebar-wrapper">
      <div className="sidebar" style={{
        display: isHidden ? "none" : "inline",
        width: sidebarWidth
      }}>
        <div className="sidebar-content">
          <div className="header-status-container">
            <Header />
            <StatusCircle isConnected={isConnectedToController} />
          </div>
          <div>
              <ExternalLink
                text={'Feature Request'}
                link={'https://github.com/LiU-SeeGoals/GameViewer/issues/new?template=Blank+issue'}
                target="_blank"
              />
              <hr />
              <ExternalLink text={"SSL Game Controller"} link={"http://localhost:8081"} target="_blank" />
              <ExternalLink text={"SSL Vision Client"} link={"http://localhost:8082"} target="_blank" />
              <hr />
              <ToggleSetting
                name={'Show vector'}
                settingsBlue={vectorSettingBlue}
                settingsYellow={vectorSettingYellow}
                setSettingsBlue={setVectorSettingBlue}
                setSettingsYellow={setVectorSettingYellow}
                itemName="Robot"
                tip={vectorTip}
              />
              <ToggleSetting
                name={'Show trace'}
                settingsBlue={traceSetting}
                settingsYellow={traceSetting}
                setSettingsBlue={setTraceSetting}
                setSettingsYellow={setTraceSetting}
                itemName="Robot"
                tip={traceTip}
              />
              <ButtonSetting /> {/* Not implemented yet */}
              <hr />
              <RobotTable
                robotActions={[]}
                visibleRobots={visibleRobots}
                sslFieldUpdate={sslFieldUpdate}
              />
          </div>
        </div>
      </div>
      <div
        className="sidebar-resizer"
        style={{ width: resizerWidth }}
        onMouseDown={startResizing}
      />
    </div>
  );
};

export default Sidebar;
