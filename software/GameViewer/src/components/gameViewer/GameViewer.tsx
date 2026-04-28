import './GameViewer.css';
import { SSL_GeometryFieldSize } from '../../proto/ssl_vision_geometry';
import useResizeSidebar from '../../hooks/useResizeSidebar';
import FootballField from './footballField/FootballField';

interface gameViewerProps {
  sslFieldUpdate: SSLFieldUpdate;
  aiRobotUpdate: AIRobotUpdate;
  robotActions: Action[];
  errorOverlay: string;
  vectorSettingBlue: boolean[];
  vectorSettingYellow: boolean[];
  sidebarWidth: number;
  fieldGeometry: SSL_GeometryFieldSize | null;
}

const GameViewer: React.FC<gameViewerProps> = ({
  sslFieldUpdate,
  aiRobotUpdate,
  robotActions,
  errorOverlay,
  vectorSettingBlue,
  vectorSettingYellow,
  sidebarWidth,
  fieldGeometry,
  controllerSend,
}) => {
  const startHeightResizer = 1000;
  const resizerWidth = 5;

  const { value: resizerValue, startResizing, isHidden } = useResizeSidebar(
    true,
    startHeightResizer
  );
  const width: number = window.innerWidth - sidebarWidth;
  const footballFieldHeight: number = window.innerHeight
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
        fieldGeometry={fieldGeometry}
        controllerSend={controllerSend}
      />

      <div
        className="game-viewer-resizer"
        style={{
            height: resizerWidth,
            zIndex: 11,
        }}
        onMouseDown={startResizing}
      />
    </div>
  );
};

export default GameViewer;
