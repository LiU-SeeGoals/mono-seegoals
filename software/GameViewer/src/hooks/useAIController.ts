import { useEffect, useState } from 'react';
import { Action } from '../types/Action';

export const useAIController = (
  setRobotActions: React.Dispatch<React.SetStateAction<Action[]>>
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    let aiSocket: WebSocket | null = null;
    let retryTimeout: NodeJS.Timeout | null = null;
    let isMounted = true;

    const connectToAI = () => {
      if (aiSocket) {
        aiSocket.close();
        aiSocket = null;
      }

      const ai_port = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_PORT;
      aiSocket = new WebSocket(`ws://localhost:${ai_port}/ws`);

      aiSocket.onerror = () => {
        if (!isMounted) return;
        setIsConnected(false);
      };

      aiSocket.onopen = () => {
        if (!isMounted) return;
        setIsConnected(true);
        console.log("Connected to AI WebSocket!");
        if (retryTimeout) {
          clearTimeout(retryTimeout);
          retryTimeout = null;
        }
      };

      aiSocket.onclose = () => {
        if (!isMounted) return;
        setIsConnected(false);
        console.log("AI WebSocket closed, retrying in 100 ms...");

        if (!retryTimeout) {
          retryTimeout = setTimeout(() => {
            retryTimeout = null;
            connectToAI();
          }, 100);
        }
      };

      aiSocket.onmessage = (event) => {
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
      if (aiSocket) aiSocket.close();
      if (retryTimeout) clearTimeout(retryTimeout);
    };
  }, [setRobotActions]);

  return { isConnected };
};
