import { useState, useEffect } from 'react';
import './App.css';
import Sidebar from './components/sidebar/Sidebar';
import GameViewer from './components/gameViewer/GameViewer';
import { useSSLVision } from './hooks/useSSLVision';
import { useAIController } from './hooks/useAIController';
import { useGameController } from './hooks/useGameController.ts';
import {
  getDefaultSSLFieldUpdate,
  getDefaultAIRobotUpdate,
  getDefaultTraceSetting,
  getDefaultVectorSetting,
  getDefaultActions,
  getDefaultLog,
  getDefaultVisibleRobots,
} from './helper/defaultValues';
import { SSL_GeometryFieldSize } from './proto/ssl_vision_geometry';

function App() {
  const [sslFieldUpdate, setSSLFieldUpdate] = useState(getDefaultSSLFieldUpdate());
  const [aiRobotUpdate, setAIUpdate] = useState(getDefaultAIRobotUpdate());
  const [robotActions, setRobotActions] = useState(getDefaultActions());
  const [vectorSettingBlue, setVectorSettingBlue] = useState(getDefaultVectorSetting());
  const [vectorSettingYellow, setVectorSettingYellow] = useState(getDefaultVectorSetting());
  const [traceSetting, setTraceSetting] = useState(getDefaultTraceSetting());
  const [visibleRobots, setvisibleRobots] = useState(getDefaultVisibleRobots());
  const [terminalLog, setTerminalLog] = useState(getDefaultLog());
  const [errorOverlay, setErrorOverlay] = useState<string | undefined>();
  const [sidebarWidth, setSidebarWidth] = useState(320);
  const [fieldGeometry, setFieldGeometry] = useState<SSL_GeometryFieldSize | null>(null);

  const { isConnected: isConnectedToVision } = useSSLVision(
    setSSLFieldUpdate,
    setErrorOverlay,
    setFieldGeometry
  );

  const { isConnected: isConnectedToAI } = useAIController(setRobotActions);

  const { isConnected: isConnectedToGameController } = useGameController();

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
