import { Pod, Node } from '@/types';

export type GameEvent = {
  type: string;
  data: any;
};

export type GameSession = {
  sessionID: string;
  clusterID: string;
  state: 'cluster_ready' | 'playing' | 'game_over';
  playerNamespace: string;
  schedulerNamespace: string;
  gameID?: string;
};

export type GameState = {
  gameID: string;
  timeLeft: number;
  playerScore: number;
  cpuScore: number;
  results?: {
    winner: 'player' | 'cpu' | 'tie';
    finalPlayerScore: number;
    finalCpuScore: number;
  };
};

export class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 1000;
  private eventListeners: Map<string, ((data: any) => void)[]> = new Map();
  private isConnected = false;
  private session: GameSession | null = null;

  constructor(url: string) {
    this.url = url;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.isConnected = true;
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const gameEvent: GameEvent = JSON.parse(event.data);
            this.handleEvent(gameEvent);
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason);
          this.isConnected = false;
          this.handleReconnect();
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  private handleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      
      setTimeout(() => {
        this.connect().catch(console.error);
      }, this.reconnectInterval * this.reconnectAttempts);
    } else {
      console.error('Max reconnection attempts reached');
    }
  }

  private handleEvent(event: GameEvent) {
    console.log('Received WebSocket event:', event.type, event.data);
    
    // Handle session creation automatically
    if (event.type === 'session_created') {
      this.session = event.data as GameSession;
      console.log('Session created:', this.session);
    }
    
    const listeners = this.eventListeners.get(event.type) || [];
    listeners.forEach(listener => {
      try {
        listener(event.data);
      } catch (error) {
        console.error('Error in event listener:', error);
      }
    });
  }

  on(eventType: string, listener: (data: any) => void) {
    if (!this.eventListeners.has(eventType)) {
      this.eventListeners.set(eventType, []);
    }
    this.eventListeners.get(eventType)!.push(listener);
  }

  off(eventType: string, listener: (data: any) => void) {
    const listeners = this.eventListeners.get(eventType);
    if (listeners) {
      const index = listeners.indexOf(listener);
      if (index > -1) {
        listeners.splice(index, 1);
      }
    }
  }

  send(event: { type: string; data?: any }) {
    if (this.ws && this.isConnected && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
    } else {
      console.warn('WebSocket is not connected. Cannot send message:', event);
    }
  }

  // Game-specific methods
  startGame() {
    this.send({
      type: 'start_game',
      data: null
    });
  }

  schedulePod(podId: string, nodeId: string) {
    this.send({
      type: 'schedule_pod',
      data: { pod_id: podId, node_id: nodeId }
    });
  }

  ping() {
    this.send({ type: 'ping', data: null });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
    this.session = null;
    this.eventListeners.clear();
  }

  get connected() {
    return this.isConnected;
  }

  get sessionInfo() {
    return this.session;
  }
}