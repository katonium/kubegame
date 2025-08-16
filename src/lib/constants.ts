import type { Pod, Node } from '@/types';

export const INITIAL_PODS: Pod[] = [
  {
    id: 'pod-1',
    name: 'pod-banana-alpha',
    label: 'Banana',
    requirements: { cpu: 1, memory: 2 },
    status: 'Pending',
    nodeId: null,
    owner: null,
  },
  {
    id: 'pod-2',
    name: 'pod-chocolate-bravo',
    label: 'Chocolate',
    requirements: { cpu: 2, memory: 1 },
    status: 'Pending',
    nodeId: null,
    owner: null,
  },
  {
    id: 'pod-3',
    name: 'pod-strawberry-charlie',
    label: 'Strawberry',
    requirements: { cpu: 1, memory: 3 },
    status: 'Pending',
    nodeId: null,
    owner: null,
  },
  {
    id: 'pod-4',
    name: 'pod-vanilla-delta',
    label: 'Vanilla',
    requirements: { cpu: 2, memory: 4 },
    status: 'Pending',
    nodeId: null,
    owner: null,
  },
];

export const INITIAL_PLAYER_NODES: Node[] = [
  {
    id: 'player-node-1',
    name: 'player-node-a',
    capacity: { cpu: 4, memory: 8 },
  },
  {
    id: 'player-node-2',
    name: 'player-node-b',
    capacity: { cpu: 8, memory: 16 },
  },
];

export const INITIAL_CPU_NODES: Node[] = [
  {
    id: 'cpu-node-1',
    name: 'kube-ai-node-x',
    capacity: { cpu: 4, memory: 8 },
  },
  {
    id: 'cpu-node-2',
    name: 'kube-ai-node-y',
    capacity: { cpu: 8, memory: 16 },
  },
];


export const POINTS_PER_POD_PER_SECOND = 10;
export const GAME_TICK_MS = 1000;
export const EVENT_TICK_MS = 10000;
export const GAME_DURATION_SECONDS = 60;
export const AI_SCHEDULE_INTERVAL_MS = 2500;
