"use client";

import type { DragEvent } from 'react';
import type { Pod } from '@/types';
import { PodCard } from './PodCard';

type PendingPodsPanelProps = {
  pods: Pod[];
  onDragStart: (e: DragEvent<HTMLDivElement>, pod: Pod) => void;
  onPodClick: (pod: Pod) => void;
  selectedPodId: string | null;
};

export function PendingPodsPanel({ pods, onDragStart, onPodClick, selectedPodId }: PendingPodsPanelProps) {
  return (
    <div className="bg-muted/50 p-2 rounded-lg">
        <h3 className="text-sm font-semibold text-center text-muted-foreground mb-2">PENDING PODS</h3>
        <div className="flex items-center justify-center gap-4 flex-wrap min-h-[120px]">
            {pods.length > 0 ? (
            pods.map(pod => (
                <PodCard
                key={pod.id}
                pod={pod}
                onDragStart={(e) => onDragStart(e, pod)}
                onClick={() => onPodClick(pod)}
                isSelected={selectedPodId === pod.id}
                />
            ))
            ) : (
            <div className="text-center text-muted-foreground py-10">
                No pending pods.
            </div>
            )}
        </div>
    </div>
  );
}
