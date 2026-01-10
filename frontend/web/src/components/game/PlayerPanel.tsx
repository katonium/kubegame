"use client";

import type { DragEvent } from 'react';
import type { Pod, Node } from '@/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { NodeCard } from './NodeCard';
import { ScrollArea } from '@/components/ui/scroll-area';

type PlayerPanelProps = {
  title: string;
  nodes: Node[];
  pods: Pod[];
  onDrop?: (e: DragEvent<HTMLDivElement>, nodeId: string) => void;
  onNodeClick?: (nodeId: string) => void;
  selectedPodId?: string | null;
};

export function PlayerPanel({ title, nodes, pods, onDrop, onNodeClick, selectedPodId }: PlayerPanelProps) {
  const isPlayer = !!onDrop;

  return (
    <Card className="flex flex-col h-full">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden p-2">
        <ScrollArea className="h-full p-2">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 h-full">
            {nodes.map(node => (
              <NodeCard
                key={node.id}
                node={node}
                pods={pods}
                onDrop={onDrop}
                onClick={onNodeClick}
                selectedPodId={selectedPodId}
                isPlayerNode={isPlayer}
              />
            ))}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
