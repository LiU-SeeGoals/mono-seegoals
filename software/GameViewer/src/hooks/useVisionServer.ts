import { useEffect, useState, type Dispatch, type SetStateAction } from 'react';
import { SSLGeometryFieldSize } from '../proto/ssl_vision/ssl_vision_geometry';
import { SSLWrapperPacket } from '../proto/ssl_vision/ssl_wrapper';
import { CameraMerger } from '../helper/cameraMerge';
import { EMPTY_FIELD, fieldStore } from '../state/stores';
import { Referee } from '../types/Referee';
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';

export type VisionSource = 'raw' | 'tigers';

const FRAME_TIMEOUT_MS = 1000;
const REFEREE_TIMEOUT_MS = 2000;
const RETRY_MS = 1000;

// Connects to server/visionServer.cjs. Detections go to fieldStore (read by
// the canvas at frame rate); connection state, geometry and referee state are
// React state because they change rarely.
export const useVisionServer = (
  setErrorOverlay: Dispatch<SetStateAction<string | undefined>>,
  source: VisionSource = 'tigers'
) => {
  const [isConnected, setIsConnected] = useState(false);
  const [fieldGeometry, setFieldGeometry] = useState<SSLGeometryFieldSize | null>(null);
  const [referee, setReferee] = useState<Referee | null>(null);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket;
    let retry: ReturnType<typeof setTimeout>;
    let frameExpiry: ReturnType<typeof setTimeout>;
    let refereeExpiry: ReturnType<typeof setTimeout>;
    let lastReferee = '';
    let lastGeometry = '';
    const cameras = new CameraMerger();

    const clearDetections = () => {
      cameras.clear();
      fieldStore.set(EMPTY_FIELD);
      setIsConnected(false);
    };
    const clearReferee = () => {
      lastReferee = '';
      setReferee(null);
    };
    const acceptFrame = (update: SSLFieldUpdate) => {
      clearTimeout(frameExpiry);
      fieldStore.set(update);
      setIsConnected(true);
      setErrorOverlay(undefined);
      frameExpiry = setTimeout(clearDetections, FRAME_TIMEOUT_MS);
    };
    const acceptReferee = (next: Referee) => {
      clearTimeout(refereeExpiry);
      // The game controller repeats its state ~10 times a second; only
      // re-render when something changed.
      const key = JSON.stringify(next);
      if (key !== lastReferee) {
        lastReferee = key;
        setReferee(next);
      }
      refereeExpiry = setTimeout(clearReferee, REFEREE_TIMEOUT_MS);
    };
    const acceptGeometry = (geometry: SSLGeometryFieldSize) => {
      const key = JSON.stringify(geometry);
      if (key === lastGeometry) return;
      lastGeometry = key;
      setFieldGeometry(geometry);
    };

    clearDetections();
    const connect = () => {
      const addr = import.meta.env.VITE_SSL_VISION_WS_ADDR || window.location.hostname;
      const port = import.meta.env.VITE_SSL_VISION_WS_PORT || 3000;
      socket = new WebSocket(`ws://${addr}:${port}/?source=${source}`);
      socket.binaryType = 'arraybuffer';
      socket.onmessage = (event) => {
        if (disposed) return;
        try {
          if (typeof event.data === 'string') {
            const message = JSON.parse(event.data);
            if (message.type === 'tracked' && source === 'tigers') acceptFrame(message.update);
            else if (message.type === 'referee') acceptReferee(message.referee);
          } else {
            const packet = SSLWrapperPacket.decode(new Uint8Array(event.data));
            if (packet.geometry?.field) acceptGeometry(packet.geometry.field);
            if (source === 'raw' && packet.detection) {
              acceptFrame(cameras.add(packet.detection.cameraId, packet.detection, performance.now()));
            }
          }
        } catch (error) {
          setErrorOverlay('Error parsing vision data');
          console.error(error);
        }
      };
      socket.onerror = () => socket.close();
      socket.onclose = () => {
        if (disposed) return;
        clearTimeout(frameExpiry);
        clearTimeout(refereeExpiry);
        clearDetections();
        clearReferee();
        retry = setTimeout(connect, RETRY_MS);
      };
    };
    connect();
    return () => {
      disposed = true;
      clearTimeout(retry);
      clearTimeout(frameExpiry);
      clearTimeout(refereeExpiry);
      socket.close();
    };
  }, [setErrorOverlay, source]);

  return { isConnected, fieldGeometry, referee };
};
