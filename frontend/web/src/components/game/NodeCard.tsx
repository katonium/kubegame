"use client";

import type { DragEvent } from 'react';
import type { Pod, Node } from '@/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { PodCard } from './PodCard';
import { Cpu, MemoryStick } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Progress } from '@/components/ui/progress';

type NodeCardProps = {
  node: Node;
  pods: Pod[];
  onDrop?: (e: DragEvent<HTMLDivElement>, nodeId: string) => void;
  onClick?: (nodeId: string) => void;
  selectedPodId?: string | null;
  isPlayerNode?: boolean;
};

export function NodeCard({ node, pods, onDrop, onClick, selectedPodId, isPlayerNode = false }: NodeCardProps) {
  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };
  
  const podsOnNode = pods.filter(p => p.nodeId === node.id);
  
  const runningPods = podsOnNode.filter(p => p.status === 'Running' || p.status === 'Scheduling');
  const usedCpu = runningPods.reduce((acc, pod) => acc + pod.requirements.cpu, 0);
  const usedMemory = runningPods.reduce((acc, pod) => acc + pod.requirements.memory, 0);
  const cpuUsagePercent = (usedCpu / node.capacity.cpu) * 100;
  const memoryUsagePercent = (usedMemory / node.capacity.memory) * 100;

  const isActionable = isPlayerNode && (!!onDrop || !!onClick);

  return (
    <Card
      onDrop={(e) => onDrop?.(e, node.id)}
      onDragOver={handleDragOver}
      onClick={() => onClick?.(node.id)}
      className={cn(
        "flex flex-col h-full transition-all duration-200",
        isActionable && selectedPodId ? "cursor-pointer hover:border-primary hover:shadow-lg" : ""
      )}
    >
      <CardHeader>
        <CardTitle>{node.name}</CardTitle>
        <CardDescription>Capacity & Usage</CardDescription>
        <div className="space-y-2 pt-2 text-sm">
            <div className="flex items-center gap-2">
                <Cpu className="w-4 h-4 text-muted-foreground" />
                <span>CPU: {usedCpu.toFixed(1)} / {node.capacity.cpu.toFixed(1)} Cores</span>
            </div>
            <Progress value={cpuUsagePercent} />
            <div className="flex items-center gap-2">
                <MemoryStick className="w-4 h-4 text-muted-foreground" />
                <span>Memory: {usedMemory.toFixed(1)} / {node.capacity.memory.toFixed(1)} GB</span>
            </div>
            <Progress value={memoryUsagePercent} />
        </div>
      </CardHeader>
      <Separator />
      <CardContent className="p-4 flex-1 overflow-y-auto">
        <div className="space-y-2">
          {podsOnNode.length > 0 ? (
            podsOnNode.map(pod => <PodCard key={pod.id} pod={pod} isCompact={true}/>)
          ) : (
            <div className="text-center text-muted-foreground py-4">
              {isActionable ? "Drop Pods Here" : "No Pods"}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
