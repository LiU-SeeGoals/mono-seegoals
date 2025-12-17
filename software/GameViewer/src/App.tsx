import { useState, useEffect } from 'react';
import './App.css';
import Sidebar from './components/sidebar/Sidebar';
import GameViewer from './components/gameViewer/GameViewer';
import { useSSLVision } from './hooks/useSSLVision';
import { useAIController } from './hooks/useAIController';
import { useGameController } from './hooks/useGameController.ts';
import { SSLGeometryFieldSize } from './proto/ssl_vision_geometry';
// import { parseProto } from './helper/ParseProto';
// import { parseJson } from './helper/ParseJson';

import {
  getDefaultSSLFieldUpdate,
  getDefaultAIRobotUpdate,
  getDefaultTraceSetting,
  getDefaultVectorSetting,
  getDefaultActions,
  getDefaultLog,
  getDefaultVisibleRobots,
} from './helper/defaultValues';

function App() {
  const [sslFieldUpdate, setSSLFieldUpdate] = useState(getDefaultSSLFieldUpdate());
  const [aiRobotUpdate, setAIUpdate] = useState(getDefaultAIRobotUpdate());
  const [robotActions, setRobotActions] = useState(getDefaultActions());
  const [vectorSettingBlue, setVectorSettingBlue] = useState(getDefaultVectorSetting());
  const [vectorSettingYellow, setVectorSettingYellow] = useState(getDefaultVectorSetting());
  const [traceSetting, setTraceSetting] = useState(getDefaultTraceSetting());
  const [visibleRobots, setvisibleRobots] = useState(getDefaultVisibleRobots());
  const [terminalLog, setTerminalLog] = useState(getDefaultLog());
  const [errorOverlay, setErrorOverlay] = useState<string>();
  const [sidebarWidth, setSidebarWidth] = useState(320);
  const [fieldGeometry, setFieldGeometry] = useState<SSLGeometryFieldSize | undefined>(undefined);
  const [aiAddress, setAiAddress] = useState(`${import.meta.env.VITE_AI_CONTROLLER_WS_ADDR}:${import.meta.env.VITE_AI_CONTROLLER_WS_PORT}`);
  const [gameControllerAddress, setGameControllerAddress] = useState(`${import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_ADDR}:${import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_PORT}`);
  // const [sslVisionAddress, setSslVisionAddress] = useState(`${import.meta.env.VITE_SSL_VISION_WS_ADDR}:${import.meta.env.VITE_SSL_VISION_WS_PORT}`);
   const [sslVisionAddress, setSslVisionAddress] = useState(`${import.meta.env.VITE_SSL_VISION_WS_ADDR}:${import.meta.env.VITE_SSL_VISION_WS_PORT}`);

  const { isConnected: isConnectedToVision } = useSSLVision(
    setSSLFieldUpdate,
    setErrorOverlay,
    setFieldGeometry,
    sslVisionAddress
  );

  // Set the config. We need to have the config in backend for it to work when files are static
  useEffect(() => {
    fetch('http://localhost:5174/config') // Or use a proxy if running on same port
      .then(res => res.json())
      .then(data => {
        setAiAddress(`${data.VITE_AI_CONTROLLER_WS_ADDR}:${data.VITE_AI_CONTROLLER_WS_PORT}`);
        setGameControllerAddress(`${data.VITE_SSL_GAME_CONTROLLER_WS_ADDR}:${data.VITE_SSL_GAME_CONTROLLER_WS_PORT}`);
        setSslVisionAddress(`${data.VITE_SSL_VISION_WS_ADDR}:${data.VITE_SSL_VISION_WS_PORT}`)
      });
  }, []);

  const { isConnected: isConnectedToAI } = useAIController(setRobotActions, aiAddress);

  const { isConnected: isConnectedToGameController } = useGameController(gameControllerAddress);
  useEffect(() => {
    document.title = "SeeGoals - GameViewer";
  }, []);

  return (
    <div className="app-container">
      <Sidebar
        vectorSettingBlue={vectorSettingBlue}
        setVectorSettingBlue={setVectorSettingBlue}
        vectorSettingYellow={vectorSettingYellow}
        setVectorSettingYellow={setVectorSettingYellow}
        traceSetting={traceSetting}
        setTraceSetting={setTraceSetting}
        robotActions={robotActions}
        visibleRobots={visibleRobots}
        isConnectedToVision={isConnectedToVision}
        isConnectedToAI={isConnectedToAI}
        isConnectedToGameController={isConnectedToGameController}
        sslFieldUpdate={sslFieldUpdate}
        sidebarWidth={sidebarWidth}
        setSidebarWidth={setSidebarWidth}
      />
      <GameViewer
        sslFieldUpdate={sslFieldUpdate}
        aiRobotUpdate={aiRobotUpdate}
        robotActions={robotActions}
        terminalLog={terminalLog}
        errorOverlay={errorOverlay}
        vectorSettingBlue={vectorSettingBlue}
        vectorSettingYellow={vectorSettingYellow}
        sidebarWidth={sidebarWidth}
        fieldGeometry={fieldGeometry}
      />
    </div>
  );
}

export default App;
