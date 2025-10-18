import './GameViewer.css';
import useResizeSidebar from '../../hooks/useResizeSidebar';
import FootballField from './footballField/FootballField';
import BottomBar from './bottomBar/BottomBar';

interface gameViewerProps {
  sslFieldUpdate: SSLFieldUpdate;
  aiRobotUpdate: AIRobotUpdate;
  robotActions: Action[];
  terminalLog: string[];
  errorOverlay: string;
  vectorSettingBlue: boolean[];
  vectorSettingYellow: boolean[];
  sidebarWidth: number;
}

const GameViewer: React.FC<gameViewerProps> = ({
  sslFieldUpdate,
  aiRobotUpdate,
  robotActions,
  terminalLog,
  errorOverlay,
  vectorSettingBlue,
  vectorSettingYellow,
  sidebarWidth,
}) => {
  const startHeightResizer = 1000;
  const resizerWidth = 5;

  const { value: resizerValue, startResizing, isHidden } = useResizeSidebar(
    true,
    startHeightResizer
  );
  const width: number = window.innerWidth - sidebarWidth;
  const bottomBarHeight: number = isHidden ? 0 : window.innerHeight - resizerValue;
  const footballFieldHeight: number = isHidden
    ? window.innerHeight
    : window.innerHeight - bottomBarHeight;
  return (
    <div className="game-viewer">
      <FootballField
        height={footballFieldHeight}
        width={width}
        sslFieldUpdate={sslFieldUpdate}
        aiRobotUpdate={aiRobotUpdate}
        robotActions={robotActions}
        errorOverlay={errorOverlay}
        vectorSettingBlue={vectorSettingBlue}
        vectorSettingYellow={vectorSettingYellow}
      />

      <div
        className="game-viewer-resizer"
        style={{
            height: resizerWidth,
            zIndex: 11,
        }}
        onMouseDown={startResizing}
      />

      {!isHidden && (
        <BottomBar
          style={{ zIndex: 10, height: bottomBarHeight }}
          terminalLog={terminalLog}
        />
      )}
    </div>
  );
};

export default GameViewer;
