import { createLiveStore } from './liveStore';
import { Action } from '../types/Action';
import { SSLFieldUpdate } from '../types/SSLFieldUpdate';
import { VisionMask } from '../types/VisionMask';

export const EMPTY_FIELD: SSLFieldUpdate = Object.freeze({ balls: [], robotsBlue: [], robotsYellow: [] });

// Latest vision frame (tracked, or merged raw cameras), before masking.
export const fieldStore = createLiveStore<SSLFieldUpdate>(EMPTY_FIELD);
// Latest AI controller actions of every team that is currently sending.
export const actionsStore = createLiveStore<Action[]>([]);
// Robot roles keyed by `${team}:${id}`.
export const rolesStore = createLiveStore<Record<string, string>>({});
export const visionMaskStore = createLiveStore<VisionMask | null>(null);
