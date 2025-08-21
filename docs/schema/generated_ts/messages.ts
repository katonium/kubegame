// WebSocket message types for KubeGame

import { GameInfo, ErrorCode } from './common';

// Base message interface
export interface BaseMessage {
  type: string;
  timestamp: string;
}

// Client-to-Server Messages
export interface PingMessage extends BaseMessage {
  type: 'ping';
}

export interface PongMessage extends BaseMessage {
  type: 'pong';
}

export interface StartGameMessage extends BaseMessage {
  type: 'start_game';
}

export interface SchedulePodMessage extends BaseMessage {
  type: 'schedule_pod';
  data: {
    pod_id: string;
    node_id: string;
  };
}

// Server-to-Client Events
export interface SessionReadyEvent extends BaseMessage {
  type: 'session_ready';
}

export interface GameStartedEvent extends BaseMessage {
  type: 'game_started';
  data: GameInfo;
}

export interface GameUpdateEvent extends BaseMessage {
  type: 'game_update';
  data: GameInfo;
}

export interface GameOverEvent extends BaseMessage {
  type: 'game_over';
  data: GameInfo;
}

export interface PodCreatedEvent extends BaseMessage {
  type: 'pod_created';
  data: GameInfo;
}

export interface PodScheduledEvent extends BaseMessage {
  type: 'pod_scheduled';
  data: GameInfo;
}

export interface ErrorEvent extends BaseMessage {
  type: 'error';
  data: {
    message: string;
    code: ErrorCode;
    details?: Record<string, any>;
  };
}

export interface ServerPingEvent extends BaseMessage {
  type: 'ping';
}

// Union types for message handling
export type ClientMessage = 
  | PingMessage
  | PongMessage
  | StartGameMessage
  | SchedulePodMessage;

export type ServerEvent = 
  | SessionReadyEvent
  | GameStartedEvent
  | GameUpdateEvent
  | GameOverEvent
  | PodCreatedEvent
  | PodScheduledEvent
  | ErrorEvent
  | ServerPingEvent;

export type WebSocketMessage = ClientMessage | ServerEvent;
