import { useEffect, useState } from 'react';
import { Action } from '../types/Action';

export const useAIController = (
  setRobotActions: React.Dispatch<React.SetStateAction<Action[]>>
) => {
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ai_address = import.meta.env.VITE_AI_CONTROLLER_WS_ADDR;
    const ai_port = import.meta.env.VITE_AI_CONTROLLER_WS_PORT;
    console.log(`[useAIController.ts] connecting to ws://${ai_address}:${ai_port}`);

    const ws = new WebSocket(`ws://${ai_address}:${ai_port}`);

    ws.onopen = () => {
      setIsConnected(true);
      console.log(`[useAIController.ts] connected on ${ws.url}!`);
    };

    ws.onerror = (err) => {
      console.error(`[useAIController.ts] error: ${err}`);
      setIsConnected(false);
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    ws.onmessage = (event) => {
      try {
        if (!event.data) return;
        const parsedData: Action[] = JSON.parse(event.data);
        if (!parsedData) return;
        // Each message is a "tick" snapshot; replace actions so we draw the latest plan.
        setRobotActions(parsedData);
      } catch (e) {
        console.error('Error parsing message JSON', e);
      }
    };

    return () => {
      ws.close();
    };
  }, [setRobotActions]);

  return { isConnected };
};
