import { useEffect, useState } from 'react';
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { SSLGeometryFieldSize } from '../proto/ssl_vision_geometry';
import { SSLWrapperPacket } from '../proto/ssl_wrapper';

export const useSSLVision = (
  setSSLFieldUpdate: React.Dispatch<React.SetStateAction<SSLFieldUpdate>>,
  setErrorOverlay: React.Dispatch<React.SetStateAction<string | undefined>>,
  setFieldGeometry: React.Dispatch<React.SetStateAction<SSLGeometryFieldSize | null>>,
  source: 'raw' | 'tigers' = 'tigers'
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket;
    let retry: ReturnType<typeof setTimeout>;
    let expiry: ReturnType<typeof setTimeout>;
    const clearDetections = () => {
      setSSLFieldUpdate({ balls: [], robotsBlue: [], robotsYellow: [] });
      setIsConnected(false);
    };
    const acceptFrame = (update: SSLFieldUpdate) => {
      clearTimeout(expiry);
      setSSLFieldUpdate(update);
      setIsConnected(true);
      setErrorOverlay(undefined);
      expiry = setTimeout(clearDetections, 1000);
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
            if (source === 'tigers' && message.type === 'tracked') acceptFrame(message.update);
          } else {
            const packet = SSLWrapperPacket.decode(new Uint8Array(event.data));
            if (packet.geometry?.field) setFieldGeometry(packet.geometry.field);
            if (source === 'raw' && packet.detection) acceptFrame(packet.detection);
          }
        } catch (error) {
          setErrorOverlay('Error parsing vision data');
          console.error(error);
        }
      };
      socket.onerror = () => socket.close();
      socket.onclose = () => {
        if (disposed) return;
        clearTimeout(expiry);
        clearDetections();
        retry = setTimeout(connect, 1000);
      };
    };
    connect();
    return () => {
      disposed = true;
      clearTimeout(retry);
      clearTimeout(expiry);
      socket.close();
    };
  }, [setSSLFieldUpdate, setErrorOverlay, setFieldGeometry, source]);

  return { isConnected };
};
