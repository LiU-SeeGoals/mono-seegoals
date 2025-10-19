import { useEffect, useState } from 'react';

export const useGameController = (
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const gc_addr = import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_ADDR;
    const gc_port = import.meta.env.VITE_SSL_GAME_CONTROLLER_WS_PORT;
    const game_controller_socket = new WebSocket(`ws://${gc_addr}:${gc_port}/`);
    game_controller_socket.binaryType = 'arraybuffer';

        const wsUrl = `ws://${gc_addr}:${gc_port}/`;
    console.log('Attempting to connect to Game Controller:', wsUrl);

    game_controller_socket.onopen = () => {
      setIsConnected(true);
      console.log("Connected to Game Controller!");
    };

    game_controller_socket.onerror = (error) => {
      console.error('Game Controller WebSocket error:', error);
      setIsConnected(false);
    };

    game_controller_socket.onclose = () => {
      console.log('Game Controller WebSocket closed:', event.code, event.reason);
      setIsConnected(false);
    };

    game_controller_socket.onmessage = (event) => {
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
      game_controller_socket.close();
    };
  }, []);

  return { isConnected };
};
