import { useEffect, useRef, useState, useCallback } from 'react';
import { WebSocketService, GameEvent, GameState, GameSession } from '@/lib/websocket';
import type { Pod, Node } from '@/types';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

export function useWebSocket() {
  const wsRef = useRef<WebSocketService | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [session, setSession] = useState<GameSession | null>(null);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [pods, setPods] = useState<Pod[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [gameStatus, setGameStatus] = useState<'waiting' | 'ready' | 'playing' | 'game_over'>('waiting');

  const connect = useCallback(async () => {
    if (wsRef.current?.connected) return;

    try {
      wsRef.current = new WebSocketService(WS_URL);
      
      // Set up event listeners
      wsRef.current.on('pong', () => {
        console.log('Received pong from server');
      });

      wsRef.current.on('session_created', (data: GameSession) => {
        console.log('Session created:', data);
        setSession(data);
        setGameStatus('ready');
      });

      wsRef.current.on('game_started', (data: GameState) => {
        console.log('Game started:', data);
        setGameState(data);
        setGameStatus('playing');
      });

      wsRef.current.on('game_update', (data: GameState) => {
        console.log('Game state update:', data);
        setGameState(data);
      });

      wsRef.current.on('pod_created', (data: Pod) => {
        console.log('Pod created:', data);
        setPods(prevPods => {
          const existingIndex = prevPods.findIndex(p => p.id === data.id);
          if (existingIndex >= 0) {
            // Update existing pod
            const newPods = [...prevPods];
            newPods[existingIndex] = data;
            return newPods;
          } else {
            // Add new pod
            return [...prevPods, data];
          }
        });
      });

      wsRef.current.on('pod_scheduled', (data: { podID: string; nodeID: string; scheduledBy: 'player' | 'cpu' }) => {
        console.log('Pod scheduled:', data);
        setPods(prevPods => 
          prevPods.map(pod => 
            pod.id === data.podID 
              ? { ...pod, nodeId: data.nodeID, scheduledBy: data.scheduledBy, status: 'Running' as const }
              : pod
          )
        );
      });

      wsRef.current.on('game_over', (data: GameState) => {
        console.log('Game over:', data);
        setGameState(data);
        setGameStatus('game_over');
      });

      wsRef.current.on('error', (data: { message: string }) => {
        console.error('WebSocket error:', data.message);
      });

      // Connect to WebSocket
      await wsRef.current.connect();
      setIsConnected(true);
      
    } catch (error) {
      console.error('Failed to connect to WebSocket:', error);
      setIsConnected(false);
    }
  }, []);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.disconnect();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const startGame = useCallback(() => {
    if (wsRef.current?.connected && gameStatus === 'ready') {
      wsRef.current.startGame();
    } else {
      console.warn('Cannot start game: WebSocket not connected or session not ready');
    }
  }, [gameStatus]);

  const schedulePod = useCallback((podId: string, nodeId: string) => {
    if (wsRef.current?.connected) {
      wsRef.current.schedulePod(podId, nodeId);
    } else {
      console.warn('Cannot schedule pod: WebSocket not connected');
    }
  }, []);

  const ping = useCallback(() => {
    if (wsRef.current?.connected) {
      wsRef.current.ping();
    }
  }, []);

  // Auto-connect on mount
  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  // Ping server periodically to keep connection alive
  useEffect(() => {
    if (!isConnected) return;

    const pingInterval = setInterval(() => {
      ping();
    }, 30000); // Ping every 30 seconds

    return () => clearInterval(pingInterval);
  }, [isConnected, ping]);

  // Create mock nodes for UI (since backend creates them but may not send node events)
  useEffect(() => {
    if (gameStatus === 'ready' && nodes.length === 0) {
      // Create mock nodes for display - in reality these would come from backend
      const mockNodes: Node[] = [
        { id: 'worker-01', name: 'worker-01', nodeType: 'player', capacity: { cpu: 2, memory: 4 } },
        { id: 'worker-02', name: 'worker-02', nodeType: 'player', capacity: { cpu: 1, memory: 2 } },
        { id: 'worker-03', name: 'worker-03', nodeType: 'player', capacity: { cpu: 3, memory: 6 } },
        { id: 'worker-04', name: 'worker-04', nodeType: 'player', capacity: { cpu: 2, memory: 4 } },
        { id: 'scheduler-01', name: 'scheduler-01', nodeType: 'cpu', capacity: { cpu: 8, memory: 16 } },
      ];
      setNodes(mockNodes);
    }
  }, [gameStatus, nodes.length]);

  return {
    isConnected,
    session,
    gameStatus,
    connect,
    disconnect,
    startGame,
    schedulePod,
    ping,
    gameState,
    pods,
    nodes,
  };
}