// Common types for KubeGame WebSocket API

export type PodLabel = 'banana' | 'chocolate' | 'strawberry' | 'vanilla';

export type PodStatus = 'pending' | 'scheduling' | 'scheduled' | 'unscheduled' | 'terminating';

export type NodeType = 'player' | 'scheduler';

export type ErrorCode = 
  | 'POD_NOT_FOUND'
  | 'NODE_NOT_FOUND'
  | 'INSUFFICIENT_RESOURCES'
  | 'INVALID_SESSION'
  | 'GAME_NOT_RUNNING'
  | 'CLUSTER_NOT_READY'
  | 'CROSS_SESSION_ACCESS'
  | 'INVALID_REQUEST';

export interface ResourceRequirements {
  cpu: number;
  memory: number;
}

export interface Affinity {
  podAffinity: {
    values: PodLabel[];
  };
  podAntiAffinity: {
    values: PodLabel[];
  };
  nodeAffinity: {
    values: PodLabel[];
  };
  nodeAntiAffinity: {
    values: PodLabel[];
  };
}

export interface Pod {
  id: string;
  name: string;
  label: PodLabel;
  affinity: Affinity;
  requirements: ResourceRequirements;
  status: PodStatus;
  nodeID?: string | null;
}

export interface Node {
  id: string;
  name: string;
  type: NodeType;
  capacity: ResourceRequirements;
  used: ResourceRequirements;
}

export interface ClusterInfo {
  pods: Pod[];
  nodes: Node[];
}

export interface GameInfo {
  timeLeft: number;
  playerScore: number;
  cpuScore: number;
  playerCluster: ClusterInfo;
  schedulerCluster: ClusterInfo;
}
