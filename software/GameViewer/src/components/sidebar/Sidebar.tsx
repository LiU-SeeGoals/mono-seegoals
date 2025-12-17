import React, { useState, useEffect } from 'react';
import './Sidebar.css';
import useResizeSidebar from '../../hooks/useResizeSidebar';
import ExternalLink from './externalLink/ExternalLink';
import Header from './header/Header';
import RobotTable from './robotTable/RobotTable';
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
  isConnectedToVision: boolean;
  isConnectedToAI: boolean;
  isConnectedToGameController: boolean;
  sslFieldUpdate: SSLFieldUpdate;
  sidebarWidth: number;
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
  isConnectedToVision,
  isConnectedToAI,
  isConnectedToGameController,
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

  return (
    <div className="sidebar-wrapper">
      <div className="sidebar" style={{
        display: isHidden ? "none" : "inline",
        width: sidebarWidth
      }}>
        <div className="sidebar-content">
          <Header />

          <div>
            <hr />
            <ExternalLink text={"SSL Vision Client"} link={"http://localhost:8082"} target="_blank" />
            <ExternalLink text={"SSL Game Controller"} link={"http://localhost:8081"} target="_blank" />
            <hr />

            <div style={{ padding: '10px 15px' }}>
              <div style={{ marginBottom: "10px" }}>Connection status</div>

              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                marginBottom: '8px'
              }}>
                <div style={{
                  width: '10px',
                  height: '10px',
                  borderRadius: '50%',
                  backgroundColor: isConnectedToVision ? '#4CAF50' : '#f44336',
                  boxShadow: isConnectedToVision ? '0 0 8px #4CAF50' : 'none'
                }} />
                <span style={{ fontSize: '14px' }}>SSL Vision</span>
              </div>

              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                marginBottom: '8px'
              }}>
                <div style={{
                  width: '10px',
                  height: '10px',
                  borderRadius: '50%',
                  backgroundColor: isConnectedToGameController ? '#4CAF50' : '#f44336',
                  boxShadow: isConnectedToGameController ? '0 0 8px #4CAF50' : 'none'
                }} />
                  <span style={{ fontSize: '14px' }}>Game Controller</span>
                </div>

              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                marginBottom: '8px'
              }}>
                <div style={{
                  width: '10px',
                  height: '10px',
                  borderRadius: '50%',
                  backgroundColor: isConnectedToAI ? '#4CAF50' : '#f44336',
                  boxShadow: isConnectedToAI ? '0 0 8px #4CAF50' : 'none'
                }} />
                  <span style={{ fontSize: '14px' }}>AI Controller</span>
                </div>
            </div>

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
