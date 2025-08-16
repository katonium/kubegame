"use client";

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ArrowLeft, Cpu, HelpCircle, MousePointerClick, Star, Timer } from 'lucide-react';

type GameState = 'pre-game' | 'playing' | 'help';

type PreGamePanelProps = {
  gameState: GameState;
  onStart: () => void;
  onShowHelp: () => void;
  onBack: () => void;
};

export function PreGamePanel({ gameState, onStart, onShowHelp, onBack }: PreGamePanelProps) {
  return (
    <div className="fixed inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50">
      <Card className="w-full max-w-md shadow-2xl">
        {gameState === 'pre-game' && (
          <>
            <CardHeader className="text-center">
              <CardTitle className="text-4xl font-bold text-primary">KubeWars</CardTitle>
              <p className="text-muted-foreground">The Kubernetes Scheduling Game</p>
            </CardHeader>
            <CardContent className="flex flex-col gap-4 p-8">
              <Button onClick={onStart} size="lg">
                Start Game
              </Button>
              <Button onClick={onShowHelp} variant="outline" size="lg">
                <HelpCircle className="mr-2 h-4 w-4"/>
                How to Play
              </Button>
            </CardContent>
          </>
        )}
        {gameState === 'help' && (
          <>
            <CardHeader>
                <div className="flex items-center gap-4">
                     <Button onClick={onBack} variant="ghost" size="icon">
                        <ArrowLeft className="h-5 w-5" />
                    </Button>
                    <CardTitle className="text-2xl font-bold">How to Play</CardTitle>
                </div>
            </CardHeader>
            <CardContent className="space-y-4 text-sm text-muted-foreground p-6">
                <div className="flex items-start gap-3">
                    <MousePointerClick className="w-8 h-8 text-primary mt-1 flex-shrink-0" />
                    <div>
                        <h3 className="font-semibold text-foreground">Schedule Pods</h3>
                        <p>Drag & drop pods from the "Pending Pods" panel to a node, or click a pod then click a node to schedule it.</p>
                    </div>
                </div>
                <div className="flex items-start gap-3">
                    <Cpu className="w-8 h-8 text-primary mt-1 flex-shrink-0" />
                    <div>
                        <h3 className="font-semibold text-foreground">Manage Resources</h3>
                        <p>Each pod requires CPU and Memory. A pod will only run if the node has enough capacity. Failed pods return to the pending list.</p>
                    </div>
                </div>
                <div className="flex items-start gap-3">
                    <Star className="w-8 h-8 text-primary mt-1 flex-shrink-0" />
                    <div>
                        <h3 className="font-semibold text-foreground">Earn Points</h3>
                        <p>You earn points for every second a pod is in the 'Running' state. Maximize your running pods to get a high score!</p>
                    </div>
                </div>
                <div className="flex items-start gap-3">
                    <Timer className="w-8 h-8 text-primary mt-1 flex-shrink-0" />
                    <div>
                        <h3 className="font-semibold text-foreground">React to Events</h3>
                        <p>New pods will arrive and existing pods may be terminated unexpectedly. Adapt your strategy to keep your cluster efficient.</p>
                    </div>
                </div>
            </CardContent>
          </>
        )}
      </Card>
    </div>
  );
}
