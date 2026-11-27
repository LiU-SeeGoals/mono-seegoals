import { useEffect, useRef, useState } from 'react';
import './GameViewer.css';
import { SSLGeometryFieldSize } from '../../proto/ssl_vision/ssl_vision_geometry';
import FootballField from './footballField/FootballField';

interface GameViewerProps {
  errorOverlay: string | undefined;
  fieldGeometry: SSLGeometryFieldSize | null;
  controllerSend: (data: unknown) => void;
}

// Fills the space next to the sidebar and tells the field its size. It only
// re-renders when that size changes (window or sidebar resize).
const GameViewer: React.FC<GameViewerProps> = ({ errorOverlay, fieldGeometry, controllerSend }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });

  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const observer = new ResizeObserver(([entry]) => {
      const width = Math.floor(entry.contentRect.width);
      const height = Math.floor(entry.contentRect.height);
      setSize((prev) => (prev.width === width && prev.height === height ? prev : { width, height }));
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return (
    <div className="game-viewer" ref={ref}>
      {size.width > 0 && size.height > 0 && (
        <FootballField
          width={size.width}
          height={size.height}
          fieldGeometry={fieldGeometry}
          controllerSend={controllerSend}
        />
      )}
      {errorOverlay && <div className="game-viewer-error">{errorOverlay}</div>}
    </div>
  );
};

export default GameViewer;
