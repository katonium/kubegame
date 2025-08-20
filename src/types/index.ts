export type PodStatus = 'Pending' | 'Scheduling' | 'Running' | 'Terminated' | 'Failed';
export type PodOwner = 'player' | 'cpu' | null;

export type Pod = {
  id: string;
  name: string;
  label: 'banana' | 'chocolate' | 'strawberry' | 'vanilla'; // lowercase to match backend
  requirements: {
    cpu: number; // in cores
    memory: number; // in GB
  };
  status: PodStatus;
  nodeId: string | null;
  scheduledBy: 'player' | 'cpu' | null; // renamed from owner to match backend
};

export type Node = {
  id: string;
  name: string;
  capacity: {
    cpu: number;
    memory: number;
  };
  nodeType: 'player' | 'cpu'; // player nodes vs cpu/scheduler nodes
};
