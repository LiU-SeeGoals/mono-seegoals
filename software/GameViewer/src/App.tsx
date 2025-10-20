import { useState, useEffect } from 'react';
import './App.css';
import Sidebar from './components/sidebar/Sidebar';
import GameViewer from './components/gameViewer/GameViewer';
import { SSL_GeometryFieldSize } from './proto/ssl_vision_geometry';
import { parseProto } from './helper/ParseProto';
import { parseJson } from './helper/ParseJson';
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
  // The useStates are defined here
  const [sslFieldUpdate, setSSLFieldUpdate] = useState(
    getDefaultSSLFieldUpdate()
  );
  const [aiRobotUpdate, setAIUpdate] = useState(getDefaultAIRobotUpdate());
  const [robotActions, setRobotActions] = useState(getDefaultActions());
  const [vectorSettingBlue, setVectorSettingBlue] = useState(
    getDefaultVectorSetting()
  );
  const [vectorSettingYellow, setVectorSettingYellow] = useState(
    getDefaultVectorSetting()
  );
  const [traceSetting, setTraceSetting] = useState(getDefaultTraceSetting());
  const [visibleRobots, setvisibleRobots] = useState(getDefaultVisibleRobots());
  const [terminalLog, setTerminalLog] = useState(getDefaultLog());
  const [errorOverlay, setErrorOverlay] = useState();
  const [isConnectedToController, setIsConnectedToController] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(320);
  const [fieldGeometry, setFieldGeometry] = useState<SSL_GeometryFieldSize | null>(null);

  useEffect(() => {
    const vision_ws_addr = import.meta.env.VITE_SSL_VISION_WS_ADDR;
    const vision_ws_port = import.meta.env.VITE_SSL_VISION_WS_PORT;
    const ssl_vision_socket = new WebSocket(`ws://${vision_ws_addr}:${vision_ws_port}/`);
    ssl_vision_socket.binaryType = 'arraybuffer';

    ssl_vision_socket.onmessage = (event) => {
      try {
        if (!event.data) return;
        const buffer = new Uint8Array(event.data);
        if (!buffer) {
          console.error('Expected ArrayBuffer, got', typeof event.data);
          return;
        }
        parseProto(buffer, setSSLFieldUpdate, setErrorOverlay, setFieldGeometry);
      } catch (e) {
        console.error('Error parsing message JSON', e);
      }
    };

    let aiSocket: WebSocket;
    let retryInterval: number;
    const connectToAI = () => {
      const ai_address = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_ADDR;
      const ai_port = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_PORT;
      aiSocket = new WebSocket(`ws://localhost:${ai_port}/ws`);

      aiSocket.onerror = () => {
        setIsConnectedToController(false);
        if (!retryInterval) {
          retryInterval = setInterval(() => {
            console.log("Retrying AI WebSocket connection...");
            connectToAI();
          }, 1000);
        }
      };

      aiSocket.onopen = () => {
        setIsConnectedToController(true);
        console.log("Connected to AI WebSocket!");

        if (retryInterval) {
          clearInterval(retryInterval);
          retryInterval = null;
        }
      };

      aiSocket.onmessage = (event) => {
        try {
          if (!event.data) return;
          const parsedData: Action[] = JSON.parse(event.data);
          if (!parsedData) return;
          setRobotActions((prevActions) => {
            const updatedActions = [...prevActions, ...parsedData];
            return updatedActions.slice(-10);
          });
        } catch (e) {
          console.error('Error parsing message JSON', e);
        }
      };
    };

    connectToAI();

    return () => {
      ssl_vision_socket.close();
      if (aiSocket) aiSocket.close();
      if (retryInterval) clearInterval(retryInterval);
    };
  }, []);

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
        isConnectedToController={isConnectedToController}
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
