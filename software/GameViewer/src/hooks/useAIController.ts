import { useCallback, useEffect, useRef, useState } from 'react';
import { actionsStore, rolesStore, visionMaskStore } from '../state/stores';
import { Action } from '../types/Action';
import { VisionMask } from '../types/VisionMask';

type RobotRoleDTO = {
  Team: number;
  Id: number;
  Role: string;
};

// GameViewerActionPacket in controller/internal/client/webserver_client.go.
type AIControllerPacket = {
  Actions?: Action[] | null;
  RobotRoles?: RobotRoleDTO[] | null;
  VisionMask?: VisionMask | null;
};

const CONNECT_TIMEOUT_MS = 1000;
const MIN_RETRY_MS = 250;
const MAX_RETRY_MS = 5000;
// Each team's controller sends its own packets (both teams in two-team
// simulation). A team that has not sent for this long is dropped.
const TEAM_TIMEOUT_MS = 1000;

export const useAIController = () => {
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ai_address = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_ADDR;
    const ai_port = import.meta.env.VITE_AI_GAME_VIEWER_SOCKET_PORT;
    let retryTimeout: ReturnType<typeof setTimeout> | null = null;
    let retryDelay = MIN_RETRY_MS;
    let isMounted = true;
    // Latest actions per team; a packet replaces its team's actions.
    const teams = new Map<number, { actions: Action[]; receivedAt: number }>();

    const reset = () => {
      teams.clear();
      actionsStore.set([]);
      rolesStore.set({});
      visionMaskStore.set(null);
    };

    const onPacket = (packet: AIControllerPacket) => {
      const now = performance.now();
      const byTeam = new Map<number, Action[]>();
      for (const action of packet.Actions ?? []) {
        const team = action.Team ?? 0;
        byTeam.set(team, [...(byTeam.get(team) ?? []), action]);
      }
      for (const [team, actions] of byTeam) teams.set(team, { actions, receivedAt: now });
      for (const [team, entry] of teams) {
        if (now - entry.receivedAt > TEAM_TIMEOUT_MS) teams.delete(team);
      }
      actionsStore.set([...teams.values()].flatMap((entry) => entry.actions));

      if (packet.RobotRoles) {
        const roles: Record<string, string> = {};
        for (const role of packet.RobotRoles) roles[`${role.Team}:${role.Id}`] = role.Role;
        rolesStore.set(roles);
      }
      const mask = packet.VisionMask ?? null;
      const current = visionMaskStore.get();
      if (mask?.MaskedSign !== current?.MaskedSign || mask?.MarginMM !== current?.MarginMM) {
        visionMaskStore.set(mask);
      }
    };

    const scheduleRetry = () => {
      if (retryTimeout || !isMounted) return;
      retryTimeout = setTimeout(() => {
        retryTimeout = null;
        connect();
      }, retryDelay);
      retryDelay = Math.min(MAX_RETRY_MS, retryDelay * 2);
    };

    const connect = () => {
      wsRef.current?.close();
      const ws = new WebSocket(`ws://${ai_address}:${ai_port}/ws`);
      wsRef.current = ws;
      const connectTimeout = setTimeout(() => {
        if (isMounted && ws.readyState === WebSocket.CONNECTING) ws.close();
      }, CONNECT_TIMEOUT_MS);

      ws.onopen = () => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        retryDelay = MIN_RETRY_MS;
        setIsConnected(true);
        console.log(`[useAIController] connected to ${ws.url}`);
      };
      ws.onerror = () => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        setIsConnected(false);
      };
      ws.onclose = () => {
        if (!isMounted || wsRef.current !== ws) return;
        clearTimeout(connectTimeout);
        setIsConnected(false);
        reset();
        scheduleRetry();
      };
      ws.onmessage = (event) => {
        if (wsRef.current !== ws || !event.data) return;
        try {
          onPacket(JSON.parse(event.data));
        } catch (e) {
          console.error('[useAIController] invalid packet', e);
        }
      };
    };

    connect();
    return () => {
      isMounted = false;
      wsRef.current?.close();
      if (retryTimeout) clearTimeout(retryTimeout);
      reset();
    };
  }, []);

  const controllerSend = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn('[useAIController] not connected; dropped', data);
    }
  }, []);

  return { isConnected, controllerSend };
};
