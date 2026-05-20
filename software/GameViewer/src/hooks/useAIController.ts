import { useEffect, useState, useRef } from 'react';
import { Action } from '../types/Action';

export const useAIController = (
  setRobotActions: React.Dispatch<React.SetStateAction<Action[]>>
) => {
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ai_address = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_ADDR;
    const ai_port = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_PORT;
    console.log(`[useAIController.ts] connecting to ws://${ai_address}:${ai_port}`);

    const connectTimeoutMs = 1000;
    const retryDelayMs = 100;
    let retryTimeout: ReturnType<typeof setTimeout> | null = null;
    let isMounted = true;

    const connectToAI = () => {
      wsRef.current?.close();

      const ws = new WebSocket(`ws://${ai_address}:${ai_port}/ws`);
      wsRef.current = ws;
      const connectTimeout = setTimeout(() => {
        if (!isMounted || ws.readyState !== WebSocket.CONNECTING) return;

        console.warn(`[useAIController.ts] connection timed out after ${connectTimeoutMs} ms`);
        ws.close();
      }, connectTimeoutMs);

      ws.onerror = (err) => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        console.error(`[useAIController.ts] error: ${err}`);
        setIsConnected(false);
      };

      ws.onopen = () => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        setIsConnected(true);
        console.log(`[useAIController.ts] connected on ${ws.url}!`);
        if (retryTimeout) {
          clearTimeout(retryTimeout);
          retryTimeout = null;
        }
      };

      ws.onclose = () => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        setIsConnected(false);
        console.log(`[useAIController.ts] closed ${ws.url}, retrying in ${retryDelayMs} ms...`);

        if (!retryTimeout) {
          retryTimeout = setTimeout(() => {
            retryTimeout = null;
            connectToAI();
          }, retryDelayMs);
        }
      };

      ws.onmessage = (event) => {
        if (wsRef.current !== ws) return;

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
      wsRef.current?.close();
      if (retryTimeout) clearTimeout(retryTimeout);
    };
  }, [setRobotActions]);

  const controllerSend = (data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn("WebSocket not connected");
    }
  };


  return { isConnected, controllerSend };
};
