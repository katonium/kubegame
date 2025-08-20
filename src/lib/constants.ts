import type { Pod, Node } from '@/types';

// Game constants that match the backend configuration
export const GAME_DURATION_SECONDS = 300; // 5 minutes - updated to match backend

// WebSocket URL - can be overridden via environment variable
export const DEFAULT_WS_URL = 'ws://localhost:8080/ws';

// Pod label emojis for display
export const POD_LABEL_EMOJIS = {
  banana: '🍌',
  chocolate: '🍫',
  strawberry: '🍓',
  vanilla: '🍦',
} as const;

// Node type display names
export const NODE_TYPE_LABELS = {
  player: 'Player Nodes',
  cpu: 'Kubernetes Scheduler Nodes',
} as const;

// Legacy constants - these are now handled by the backend
export const POINTS_PER_POD_PER_SECOND = 10;
export const GAME_TICK_MS = 1000;
export const EVENT_TICK_MS = 10000;
export const AI_SCHEDULE_INTERVAL_MS = 2500;
