import { useEffect, useState } from 'react';
import './App.css';
import Sidebar from './components/sidebar/Sidebar';
import GameViewer from './components/gameViewer/GameViewer';
import { useVisionServer, type VisionSource } from './hooks/useVisionServer';
import { useAIController } from './hooks/useAIController';

function App() {
  const [visionSource, setVisionSource] = useState<VisionSource>(() =>
    localStorage.getItem('visionSource') === 'raw' ? 'raw' : 'tigers');
  useEffect(() => { localStorage.setItem('visionSource', visionSource); }, [visionSource]);
  const [errorOverlay, setErrorOverlay] = useState<string | undefined>();

  const { isConnected: isConnectedToVision, fieldGeometry, referee } =
    useVisionServer(setErrorOverlay, visionSource);
  const { isConnected: isConnectedToAI, controllerSend } = useAIController();

  useEffect(() => {
    document.title = 'SeeGoals - GameViewer';
  }, []);

  return (
    <div className="app-container">
      <Sidebar
        visionSource={visionSource}
        setVisionSource={setVisionSource}
        isConnectedToVision={isConnectedToVision}
        isConnectedToAI={isConnectedToAI}
        referee={referee}
        controllerSend={controllerSend}
      />
      <GameViewer
        errorOverlay={errorOverlay}
        fieldGeometry={fieldGeometry}
        controllerSend={controllerSend}
      />
    </div>
  );
}

export default App;
