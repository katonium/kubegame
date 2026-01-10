"use client";

import { useEffect, useState } from 'react';
import { Timer, Star, Bot, User } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { cn } from '@/lib/utils';

type GameStatsProps = {
  playerScore: number;
  cpuScore: number;
  time: number;
};

export function GameStats({ playerScore, cpuScore, time }: GameStatsProps) {
  const [pulsePlayer, setPulsePlayer] = useState(false);
  const [pulseCpu, setPulseCpu] = useState(false);

  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60)
      .toString()
      .padStart(2, '0');
    const secs = (seconds % 60).toString().padStart(2, '0');
    return `${minutes}:${secs}`;
  };

  useEffect(() => {
    if (playerScore > 0) {
      setPulsePlayer(true);
      const timer = setTimeout(() => setPulsePlayer(false), 300);
      return () => clearTimeout(timer);
    }
  }, [playerScore]);
  
  useEffect(() => {
    if (cpuScore > 0) {
      setPulseCpu(true);
      const timer = setTimeout(() => setPulseCpu(false), 300);
      return () => clearTimeout(timer);
    }
  }, [cpuScore]);

  return (
    <Card className="p-2 px-4 shadow-lg flex items-center gap-6">
      <div className="flex items-center gap-2 text-foreground">
        <User className="w-5 h-5 text-blue-500" />
        <span
          className={cn(
            'font-mono text-lg font-semibold tabular-nums transition-transform duration-300',
            pulsePlayer && 'scale-125 text-primary'
          )}
        >
          {playerScore}
        </span>
      </div>
      <div className="flex items-center gap-2 text-foreground">
        <Timer className="w-5 h-5 text-primary" />
        <span className="font-mono text-lg font-semibold tabular-nums">
          {formatTime(time)}
        </span>
      </div>
      <div className="flex items-center gap-2 text-foreground">
        <Bot className="w-5 h-5 text-red-500" />
        <span
          className={cn(
            'font-mono text-lg font-semibold tabular-nums transition-transform duration-300',
            pulseCpu && 'scale-125 text-destructive'
          )}
        >
          {cpuScore}
        </span>
      </div>
    </Card>
  );
}
