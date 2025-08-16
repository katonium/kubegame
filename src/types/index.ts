export type PodStatus = 'Pending' | 'Scheduling' | 'Running' | 'Terminated' | 'Failed';
export type PodOwner = 'player' | 'cpu' | null;

export type Pod = {
  id: string;
  name: string;
  label: 'Banana' | 'Chocolate' | 'Strawberry' | 'Vanilla';
  requirements: {
    cpu: number; // in cores
    memory: number; // in GB
  };
  status: PodStatus;
  nodeId: string | null;
  owner: PodOwner;
};

export type Node = {
  id: string;
  name: string;
  capacity: {
    cpu: number;
    memory: number;
  };
};
