"use client";

import type { DragEvent } from 'react';
import type { Pod } from '@/types';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Cpu, MemoryStick } from 'lucide-react';
import { cn } from '@/lib/utils';
import { BananaIcon } from '../icons/BananaIcon';
import { ChocolateIcon } from '../icons/ChocolateIcon';
import { StrawberryIcon } from '../icons/StrawberryIcon';
import { VanillaIcon } from '../icons/VanillaIcon';

type PodCardProps = {
  pod: Pod;
  onDragStart?: (e: DragEvent<HTMLDivElement>) => void;
  onClick?: () => void;
  isSelected?: boolean;
  isCompact?: boolean;
};

const labelIcons: Record<Pod['label'], React.ReactNode> = {
  Banana: <BananaIcon className="w-5 h-5" />,
  Chocolate: <ChocolateIcon className="w-5 h-5" />,
  Strawberry: <StrawberryIcon className="w-5 h-5" />,
  Vanilla: <VanillaIcon className="w-5 h-5" />,
};

const statusColors: Record<Pod['status'], string> = {
  Pending: 'bg-gray-500',
  Scheduling: 'bg-blue-500 animate-pulse',
  Running: 'bg-green-500',
  Terminated: 'bg-neutral-700',
  Failed: 'bg-red-500',
};

export function PodCard({ pod, onDragStart, onClick, isSelected, isCompact = false }: PodCardProps) {
  const isActionable = pod.status === 'Pending' && (!!onDragStart || !!onClick);

  if (isCompact) {
    return (
        <div className={cn(
            "flex items-center gap-3 p-2 bg-secondary rounded-lg",
            pod.status === 'Failed' && 'bg-red-200 dark:bg-red-900'
        )}>
            <div className={cn("w-2 h-2 rounded-full", statusColors[pod.status])}></div>
            <div className="flex-shrink-0">{labelIcons[pod.label]}</div>
            <div className="flex-1 truncate text-sm font-medium">{pod.name}</div>
        </div>
    );
  }

  return (
    <Card
      draggable={isActionable}
      onDragStart={onDragStart}
      onClick={onClick}
      className={cn(
        'transition-all duration-200 shadow-md',
        isActionable ? 'cursor-grab' : 'cursor-default',
        isSelected ? 'border-primary ring-2 ring-primary shadow-lg' : '',
        !isActionable && pod.status !== 'Running' ? 'opacity-60' : ''
      )}
    >
      <CardHeader className="p-4">
        <div className="flex justify-between items-start">
            <div className="flex items-center gap-3">
                {labelIcons[pod.label]}
                <CardTitle className="text-lg">{pod.name}</CardTitle>
            </div>
            <Badge variant="secondary" className={cn("text-white", statusColors[pod.status])}>{pod.status}</Badge>
        </div>
      </CardHeader>
      <CardContent className="p-4 pt-0 text-sm">
        <div className="flex items-center gap-4 text-muted-foreground">
            <div className="flex items-center gap-1">
                <Cpu className="w-4 h-4" />
                <span>{pod.requirements.cpu} Core(s)</span>
            </div>
            <div className="flex items-center gap-1">
                <MemoryStick className="w-4 h-4" />
                <span>{pod.requirements.memory} GB</span>
            </div>
        </div>
      </CardContent>
    </Card>
  );
}
