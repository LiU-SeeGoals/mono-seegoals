import { useEffect, useState } from 'react';
import { parseProto } from '../helper/ParseProto';
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { SSLGeometryFieldSize } from '../proto/ssl_vision_geometry';

export const useSSLVision = (
  setSSLFieldUpdate: React.Dispatch<React.SetStateAction<SSLFieldUpdate>>,
  setErrorOverlay: React.Dispatch<React.SetStateAction<string | undefined>>,
  setFieldGeometry: React.Dispatch<React.SetStateAction<SSLGeometryFieldSize | undefined>>,
  vision_ws_addr: string
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    console.log(`[useSSLVision.ts] connecting to ws://${vision_ws_addr}`);

    const ws = new WebSocket(`ws://${vision_ws_addr}/`);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      setIsConnected(true);
      console.log(`[useSSLVision.ts] connected on ${ws.url}`);

    };

    ws.onerror = (err) => {
      setIsConnected(false);
      console.error(`[useSSLVision.ts] error: ${err}`)
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    ws.onmessage = (event) => {
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

    return () => {
      ws.close();
    };
  }, [setSSLFieldUpdate, setErrorOverlay, setFieldGeometry]);

  return { isConnected };
};
