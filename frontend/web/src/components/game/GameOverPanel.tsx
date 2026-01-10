"use client";

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Award, Bot, User } from 'lucide-react';

type GameOverPanelProps = {
  playerScore: number;
  cpuScore: number;
  onRestart: () => void;
};

export function GameOverPanel({ playerScore, cpuScore, onRestart }: GameOverPanelProps) {
  const winner = playerScore > cpuScore ? 'Player' : playerScore < cpuScore ? 'Kubernetes Scheduler' : 'Draw';
  
  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50">
      <Card className="w-full max-w-md shadow-2xl text-center">
        <CardHeader>
          <CardTitle className="text-4xl font-bold">Game Over</CardTitle>
          <CardDescription>Here are the final scores:</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6 p-8">
            <div className="flex justify-around items-center text-2xl font-semibold">
                <div className="flex flex-col items-center gap-2">
                    <User className="w-10 h-10 text-blue-500"/>
                    <span>You</span>
                    <span className="font-mono text-4xl">{playerScore}</span>
                </div>
                <div className="text-4xl font-bold text-muted-foreground">VS</div>
                 <div className="flex flex-col items-center gap-2">
                    <Bot className="w-10 h-10 text-red-500"/>
                    <span>K8s</span>
                    <span className="font-mono text-4xl">{cpuScore}</span>
                </div>
            </div>
          
            <div className="text-center">
                <div className="flex items-center justify-center gap-2 text-2xl font-bold">
                    <Award className="w-8 h-8 text-yellow-500" />
                    <h2>
                        {winner === 'Draw' ? "It's a Draw!" : `${winner} Wins!`}
                    </h2>
                </div>
            </div>

            <Button onClick={onRestart} size="lg" className="w-full">
                Play Again
            </Button>
        </CardContent>
      </Card>
    </div>
  );
}
