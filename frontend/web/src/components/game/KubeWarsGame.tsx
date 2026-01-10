"use client";

import { useState, useEffect, useCallback, useMemo, DragEvent } from 'react';
import type { Pod, Node } from '@/types';
import { PlayerPanel } from './PlayerPanel';
import { GameStats } from './GameStats';
import { PreGamePanel } from './PreGamePanel';
import { GameOverPanel } from './GameOverPanel';
import { PendingPodsPanel } from './PendingPodsPanel';
import { useToast } from '@/hooks/use-toast';
import { useWebSocket } from '@/hooks/use-websocket';
import {
  GAME_DURATION_SECONDS,
} from '@/lib/constants';

type GameState = 'pre-game' | 'playing' | 'help' | 'game-over';

export function KubeWarsGame() {
  const [uiState, setUiState] = useState<GameState>('pre-game');
  const [draggedPod, setDraggedPod] = useState<Pod | null>(null);
  const [selectedPodId, setSelectedPodId] = useState<string | null>(null);
  const { toast } = useToast();
  
  // Use WebSocket hook for real-time game data
  const { 
    isConnected,
    session,
    gameStatus,
    gameState: backendGameState, 
    pods, 
    nodes, 
    startGame,
    schedulePod,
    connect,
    disconnect
  } = useWebSocket();

  // Separate player and CPU nodes
  const playerNodes = useMemo(() => 
    nodes.filter(node => node.nodeType === 'player'), 
    [nodes]
  );
  
  const cpuNodes = useMemo(() => 
    nodes.filter(node => node.nodeType === 'cpu'), 
    [nodes]
  );

  // Extract scores and time from backend game state
  const playerScore = backendGameState?.playerScore || 0;
  const cpuScore = backendGameState?.cpuScore || 0;
  const time = backendGameState?.timeLeft || GAME_DURATION_SECONDS;

  // Reset UI state when game resets
  const resetGame = useCallback(() => {
    setSelectedPodId(null);
    setDraggedPod(null);
    setUiState('pre-game');
  }, []);

  // Sync UI state with backend game state
  useEffect(() => {
    if (gameStatus === 'game_over') {
      setUiState('game-over');
    } else if (gameStatus === 'playing') {
      setUiState('playing');
    } else if (gameStatus === 'ready') {
      // Stay in pre-game until user clicks start
    }
  }, [gameStatus]);

  const handleStartGame = () => {
    if (gameStatus === 'ready') {
      startGame();
      setUiState('playing');
    } else {
      console.warn('Cannot start game: session not ready');
    }
  };

  const handleShowHelp = () => setUiState('help');
  const handleBackToMenu = () => setUiState('pre-game');

  const handleSchedulePod = useCallback((podId: string, nodeId: string, isPlayer: boolean) => {
    if (!isPlayer) {
      // Only allow player scheduling through UI
      return;
    }

    const node = playerNodes.find(n => n.id === nodeId);
    const podToSchedule = pods.find(p => p.id === podId);
    
    if (!node || !podToSchedule) {
      console.warn('Invalid pod or node for scheduling');
      return;
    }

    if (!isConnected) {
      toast({ 
        title: "Connection Error", 
        description: "Not connected to game server", 
        variant: 'destructive' 
      });
      return;
    }

    // Send scheduling request to backend
    schedulePod(podId, nodeId);
    
    // Show immediate feedback
    toast({ 
      title: "Scheduling Pod...", 
      description: `Requesting to schedule ${podToSchedule.name} to ${node.name}` 
    });
  }, [playerNodes, pods, isConnected, schedulePod, toast]);

  // Show success/failure notifications for pod scheduling
  useEffect(() => {
    pods.forEach(pod => {
      if (pod.scheduledBy === 'player') {
        if (pod.status === 'Running' && pod.nodeId) {
          const node = playerNodes.find(n => n.id === pod.nodeId);
          if (node) {
            toast({ 
              title: "Pod Scheduled!", 
              description: `${pod.name} is now running on ${node.name}` 
            });
          }
        } else if (pod.status === 'Failed') {
          toast({ 
            title: "Scheduling Failed!", 
            description: `${pod.name} failed to schedule - insufficient resources`, 
            variant: 'destructive' 
          });
        }
      }
    });
  }, [pods, playerNodes, toast]);


  const handleDragStart = (e: DragEvent<HTMLDivElement>, pod: Pod) => {
    setDraggedPod(pod);
    setSelectedPodId(null);
  };
  
  const handleDrop = (e: DragEvent<HTMLDivElement>, nodeId: string) => {
    e.preventDefault();
    if (draggedPod) {
      handleSchedulePod(draggedPod.id, nodeId, true);
    }
    setDraggedPod(null);
  };

  const handlePodClick = (pod: Pod) => {
    if (pod.status === 'Pending') {
      setSelectedPodId(currentId => currentId === pod.id ? null : pod.id);
    }
  };

  const handleNodeClick = (nodeId: string) => {
    if (selectedPodId) {
      handleSchedulePod(selectedPodId, nodeId, true);
      setSelectedPodId(null);
    }
  };

  const pendingPods = useMemo(() => pods.filter(p => p.status === 'Pending'), [pods]);

  if (uiState === 'pre-game' || uiState === 'help') {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen p-4">
        <PreGamePanel 
          gameState={uiState} 
          onStart={handleStartGame} 
          onShowHelp={handleShowHelp} 
          onBack={handleBackToMenu}
          canStart={gameStatus === 'ready'}
        />
        {!isConnected && (
          <div className="mt-4 p-4 bg-yellow-100 border border-yellow-400 rounded-lg">
            <p className="text-yellow-800">Connecting to game server...</p>
          </div>
        )}
        {isConnected && gameStatus === 'waiting' && (
          <div className="mt-4 p-4 bg-blue-100 border border-blue-400 rounded-lg">
            <p className="text-blue-800">Setting up game session...</p>
          </div>
        )}
        {isConnected && gameStatus === 'ready' && session && (
          <div className="mt-4 p-4 bg-green-100 border border-green-400 rounded-lg">
            <p className="text-green-800">Connected! Session {session.sessionID} ready</p>
            <p className="text-green-700 text-sm">Cluster: {session.clusterID}</p>
          </div>
        )}
      </div>
    );
  }

  if (uiState === 'game-over') {
    return <GameOverPanel playerScore={playerScore} cpuScore={cpuScore} onRestart={handleStartGame} />
  }

  return (
    <div className="flex flex-col h-screen p-4 gap-4 relative font-sans">
      <div className="absolute top-4 left-1/2 -translate-x-1/2 z-20">
        <GameStats playerScore={playerScore} cpuScore={cpuScore} time={time} />
      </div>

      <h1 className="text-3xl font-bold text-primary text-center pt-2">KubeWars</h1>
      
      <PendingPodsPanel pods={pendingPods} onPodClick={handlePodClick} onDragStart={handleDragStart} selectedPodId={selectedPodId} />
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 flex-1 min-h-0 pt-4">
        <PlayerPanel title="Your Nodes" nodes={playerNodes} pods={pods} onDrop={handleDrop} onNodeClick={handleNodeClick} selectedPodId={selectedPodId} />
        <PlayerPanel title="Kubernetes Scheduler Nodes" nodes={cpuNodes} pods={pods} />
      </div>
    </div>
  );
}
