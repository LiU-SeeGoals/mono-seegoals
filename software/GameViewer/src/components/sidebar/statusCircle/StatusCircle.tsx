import React from 'react';
import './StatusCircle.css'; // Import the CSS file

interface StatusCircleProps {
  isConnected: boolean;
}

const StatusCircle: React.FC<StatusCircleProps> = ({ isConnected }) => {
  return (
    <div className="circle-container">
      <div
        className="status-circle"
        style={{
          backgroundColor: isConnected ? 'green' : 'red', // Change circle color based on connection status
        }}
      />
      <span className="hover-text">
        Controller Status: {isConnected ? 'Connected' : 'Disconnected'}
      </span>
    </div>
  );
};

export default StatusCircle;
