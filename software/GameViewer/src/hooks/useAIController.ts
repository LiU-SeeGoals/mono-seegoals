import { useEffect, useState } from 'react';
import { Action } from '../types/Action';

export const useAIController = (
  setRobotActions: React.Dispatch<React.SetStateAction<Action[]>>
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ai_address = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_ADDR;
    const ai_port = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_PORT;
    console.log(`[useAIController.ts] connecting to ws://${ai_address}:${ai_port}`);

    let ws: WebSocket | null = null;
    let retryTimeout: NodeJS.Timeout | null = null;
    let isMounted = true;

    const connectToAI = () => {
      if (ws) {
        ws.close();
        ws = null;
      }

      ws = new WebSocket(`ws://${ai_address}:${ai_port}/ws`);

      ws.onerror = (err) => {
        if (!isMounted) return;
        console.error(`[useAIController.ts] error: ${err}`);
        setIsConnected(false);
      };

      ws.onopen = () => {
        if (!isMounted || !ws) return;
        setIsConnected(true);
        console.log(`[useAIController.ts] connected on ${ws.url}!`);
        if (retryTimeout) {
          clearTimeout(retryTimeout);
          retryTimeout = null;
        }
      };

      ws.onclose = () => {
        if (!isMounted || !ws) return;
        setIsConnected(false);
        console.log(`[useAIController.ts] closed ${ws.url}, retrying in 100 ms...`);

        if (!retryTimeout) {
          retryTimeout = setTimeout(() => {
            retryTimeout = null;
            connectToAI();
          }, 100);
        }
      };

      ws.onmessage = (event) => {
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
      isMounted = false;
      if (ws) ws.close();
      if (retryTimeout) clearTimeout(retryTimeout);
    };
  }, [setRobotActions]);

  return { isConnected };
};
