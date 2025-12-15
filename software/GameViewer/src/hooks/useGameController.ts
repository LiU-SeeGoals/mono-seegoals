import { useEffect, useState } from 'react';

export const useGameController = (
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const gc_addr = import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_ADDR;
    const gc_port = import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_PORT;
    console.log(`[useGameController.ts] connecting to ws://${gc_addr}:${gc_port}`);

    const ws = new WebSocket(`ws://${gc_addr}:${gc_port}/`);
    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      setIsConnected(true);
      console.log(`[useGameController.ts] connected on ${ws.url}`);
    };

    ws.onerror = (err) => {
      setIsConnected(false);
      console.error(`[useGameController.ts] error: ${err}`);
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
      } catch (e) {
        console.error('Error parsing game controller message', e);
      }
    };

    return () => {
      ws.close();
    };
  }, []);

  return { isConnected };
};
