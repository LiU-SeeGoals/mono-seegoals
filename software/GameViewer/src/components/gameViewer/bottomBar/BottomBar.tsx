import React from 'react';
import './BottomBar.css';

interface BottomBarProps {
  style,
  terminalLog: string[];
}

const BottomBar: React.FC<BottomBarProps> = ({ style, terminalLog }) => {
  return (
    <div className="bottomBar-wrapper" style={style}>
      {terminalLog.map((log, index) => (
        <p key={index}>{log}</p>
      ))}
    </div>
  );
};

export default BottomBar;
