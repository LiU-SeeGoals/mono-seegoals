import { useEffect, useState, useRef, type Dispatch, type SetStateAction } from 'react';
import { Action } from '../types/Action';

type RobotRoleDTO = {
  Team: number;
  Id: number;
  Role: string;
};

type AIControllerPacket = {
  Actions?: Action[];
  RobotRoles?: RobotRoleDTO[];
};

export const useAIController = (
  setRobotActions: Dispatch<SetStateAction<Action[]>>,
  setRobotRoles?: Dispatch<SetStateAction<Record<string, string>>>
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
          const parsedData: Action[] | AIControllerPacket = JSON.parse(event.data);
          const actions = Array.isArray(parsedData)
            ? parsedData
            : Array.isArray(parsedData.Actions)
              ? parsedData.Actions
              : [];
          if (actions.length === 0 && Array.isArray(parsedData)) return;

          const roleUpdates: Record<string, string> = {};
          if (!Array.isArray(parsedData) && Array.isArray(parsedData.RobotRoles)) {
            for (const role of parsedData.RobotRoles) {
              roleUpdates[`${role.Team}:${role.Id}`] = role.Role;
            }
            if (setRobotRoles) {
              setRobotRoles(roleUpdates);
            }
          } else {
            for (const action of actions) {
              if (action.Team !== undefined && action.Role) {
                roleUpdates[`${action.Team}:${action.Id}`] = action.Role;
              }
            }
            if (setRobotRoles && Object.keys(roleUpdates).length > 0) {
              setRobotRoles((prevRoles) => ({ ...prevRoles, ...roleUpdates }));
            }
          }
          setRobotActions((prevActions) => {
            const updatedActions = [...prevActions, ...actions];
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
  }, [setRobotActions, setRobotRoles]);

  const controllerSend = (data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn("WebSocket not connected");
    }
  };


  return { isConnected, controllerSend };
};
