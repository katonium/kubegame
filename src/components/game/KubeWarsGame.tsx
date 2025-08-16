"use client";

import { useState, useEffect, useCallback, useMemo, DragEvent } from 'react';
import type { Pod, Node } from '@/types';
import { PlayerPanel } from './PlayerPanel';
import { GameStats } from './GameStats';
import { PreGamePanel } from './PreGamePanel';
import { GameOverPanel } from './GameOverPanel';
import { PendingPodsPanel } from './PendingPodsPanel';
import { useToast } from '@/hooks/use-toast';
import {
  INITIAL_PODS,
  INITIAL_PLAYER_NODES,
  INITIAL_CPU_NODES,
  POINTS_PER_POD_PER_SECOND,
  GAME_TICK_MS,
  EVENT_TICK_MS,
  GAME_DURATION_SECONDS,
  AI_SCHEDULE_INTERVAL_MS,
} from '@/lib/constants';

type GameState = 'pre-game' | 'playing' | 'help' | 'game-over';

export function KubeWarsGame() {
  const [gameState, setGameState] = useState<GameState>('pre-game');
  const [pods, setPods] = useState<Pod[]>([]);
  const [playerNodes, setPlayerNodes] = useState<Node[]>([]);
  const [cpuNodes, setCpuNodes] = useState<Node[]>([]);
  const [playerScore, setPlayerScore] = useState(0);
  const [cpuScore, setCpuScore] = useState(0);
  const [time, setTime] = useState(GAME_DURATION_SECONDS);
  const [draggedPod, setDraggedPod] = useState<Pod | null>(null);
  const [selectedPodId, setSelectedPodId] = useState<string | null>(null);
  const { toast } = useToast();

  const resetGame = useCallback(() => {
    setPods(JSON.parse(JSON.stringify(INITIAL_PODS)));
    setPlayerNodes(JSON.parse(JSON.stringify(INITIAL_PLAYER_NODES)));
    setCpuNodes(JSON.parse(JSON.stringify(INITIAL_CPU_NODES)));
    setPlayerScore(0);
    setCpuScore(0);
    setTime(GAME_DURATION_SECONDS);
    setSelectedPodId(null);
    setDraggedPod(null);
  }, []);

  useEffect(() => {
    resetGame();
  }, [resetGame]);

  const handleStartGame = () => {
    resetGame();
    setGameState('playing');
  };

  const handleShowHelp = () => setGameState('help');
  const handleBackToMenu = () => setGameState('pre-game');

  const schedulePod = useCallback((podId: string, nodeId: string, isPlayer: boolean) => {
      const allNodes = isPlayer ? playerNodes : cpuNodes;
      const node = allNodes.find(n => n.id === nodeId);
      const podToSchedule = pods.find(p => p.id === podId);
      if (!node || !podToSchedule) return;

      setPods(prevPods => prevPods.map(p =>
          p.id === podId ? { ...p, status: 'Scheduling', nodeId, owner: isPlayer ? 'player' : 'cpu' } : p
      ));
      
      setTimeout(() => {
        setPods(currentPods => {
            const currentPodToSchedule = currentPods.find(p => p.id === podId);
            if (!currentPodToSchedule) return currentPods;
            
            const podsOnNode = currentPods.filter(
                p => p.nodeId === nodeId && p.status !== 'Failed'
            );
            
            const usedCpu = podsOnNode.reduce((acc, p) => acc + p.requirements.cpu, 0);
            const usedMemory = podsOnNode.reduce((acc, p) => acc + p.requirements.memory, 0);

            const canSchedule =
                node.capacity.cpu >= usedCpu + currentPodToSchedule.requirements.cpu &&
                node.capacity.memory >= usedMemory + currentPodToSchedule.requirements.memory;

            if (canSchedule) {
                if(isPlayer) toast({ title: "Pod Scheduled!", description: `${currentPodToSchedule.name} is now running.` });
                return currentPods.map(p => p.id === podId ? { ...p, status: 'Running' } : p);
            } else {
                if(isPlayer) toast({ title: "Scheduling Failed!", description: `${node.name} has insufficient resources.`, variant: 'destructive' });
                return currentPods.map(p => p.id === podId ? { ...p, status: 'Failed' } : p);
            }
        });
      }, 1000);
    }, [playerNodes, cpuNodes, pods, toast]
  );
  
  useEffect(() => {
    const failedPods = pods.filter(p => p.status === 'Failed');
    if (failedPods.length > 0) {
      const timer = setTimeout(() => {
        setPods(prevPods =>
          prevPods.map(p => (p.status === 'Failed' ? { ...p, status: 'Pending', nodeId: null, owner: null } : p))
        );
      }, 2000);
      return () => clearTimeout(timer);
    }
  }, [pods]);

  // Game tick for score and time
  useEffect(() => {
    if (gameState !== 'playing') return;
    const timer = setInterval(() => {
      setTime(t => {
        if (t <= 1) {
            setGameState('game-over');
            return 0;
        }
        return t - 1;
      });

      const runningPlayerPods = pods.filter(p => p.status === 'Running' && p.owner === 'player');
      if (runningPlayerPods.length > 0) {
        setPlayerScore(s => s + runningPlayerPods.length * POINTS_PER_POD_PER_SECOND);
      }
      
      const runningCpuPods = pods.filter(p => p.status === 'Running' && p.owner === 'cpu');
      if (runningCpuPods.length > 0) {
        setCpuScore(s => s + runningCpuPods.length * POINTS_PER_POD_PER_SECOND);
      }

    }, GAME_TICK_MS);
    return () => clearInterval(timer);
  }, [gameState, pods]);

  // Game engine for random events
  useEffect(() => {
    if (gameState !== 'playing') return;
    const eventTimer = setInterval(() => {
      const eventType = Math.random();
      if (eventType < 0.5) { // Terminate a pod
        const runningPods = pods.filter(p => p.status === 'Running');
        if (runningPods.length > 0) {
            const podToTerminate = runningPods[Math.floor(Math.random() * runningPods.length)];
            setPods(currentPods => currentPods.map(p => p.id === podToTerminate.id ? { ...p, status: 'Pending', nodeId: null, owner: null } : p));
        }
      } else { // Add a new pod
        const newPod: Pod = {
          id: `pod-${Date.now()}`,
          name: `pod-gen-${Math.random().toString(36).substring(7)}`,
          label: ['Banana', 'Chocolate', 'Strawberry', 'Vanilla'][Math.floor(Math.random() * 4)] as Pod['label'],
          requirements: { cpu: Math.ceil(Math.random() * 2), memory: Math.ceil(Math.random() * 4) },
          status: 'Pending',
          nodeId: null,
          owner: null
        };
        setPods(currentPods => [...currentPods, newPod]);
      }
    }, EVENT_TICK_MS);
    return () => clearInterval(eventTimer);
  }, [gameState, pods]);

  // AI Scheduler
   useEffect(() => {
    if (gameState !== 'playing') return;
    const aiScheduler = setInterval(() => {
        const pending = pods.filter(p => p.status === 'Pending');
        const availableNodes = cpuNodes.map(node => {
            const podsOnNode = pods.filter(p => p.nodeId === node.id && p.status !== 'Failed');
            const usedCpu = podsOnNode.reduce((acc, p) => acc + p.requirements.cpu, 0);
            const usedMemory = podsOnNode.reduce((acc, p) => acc + p.requirements.memory, 0);
            return {
                ...node,
                availableCpu: node.capacity.cpu - usedCpu,
                availableMemory: node.capacity.memory - usedMemory
            };
        }).sort((a, b) => b.availableCpu - a.availableCpu); // Prioritize nodes with more CPU

        if (pending.length > 0 && availableNodes.length > 0) {
            const podToSchedule = pending[0];
            const bestNode = availableNodes.find(n => n.availableCpu >= podToSchedule.requirements.cpu && n.availableMemory >= podToschedule.requirements.memory);
            
            if (bestNode) {
                schedulePod(podToSchedule.id, bestNode.id, false);
            }
        }
    }, AI_SCHEDULE_INTERVAL_MS);
    return () => clearInterval(aiScheduler);
  }, [gameState, pods, cpuNodes, schedulePod]);


  const handleDragStart = (e: DragEvent<HTMLDivElement>, pod: Pod) => {
    setDraggedPod(pod);
    setSelectedPodId(null);
  };
  
  const handleDrop = (e: DragEvent<HTMLDivElement>, nodeId: string) => {
    e.preventDefault();
    if (draggedPod) {
      schedulePod(draggedPod.id, nodeId, true);
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
      schedulePod(selectedPodId, nodeId, true);
      setSelectedPodId(null);
    }
  };

  const pendingPods = useMemo(() => pods.filter(p => p.status === 'Pending'), [pods]);

  if (gameState === 'pre-game' || gameState === 'help') {
    return <PreGamePanel gameState={gameState} onStart={handleStartGame} onShowHelp={handleShowHelp} onBack={handleBackToMenu} />;
  }

  if (gameState === 'game-over') {
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
        <PlayerPanel title="Kubernetes AI Nodes" nodes={cpuNodes} pods={pods} />
      </div>
    </div>
  );
}
