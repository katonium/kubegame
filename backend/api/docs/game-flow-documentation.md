# KubeGame - Message Flow and Game Sequence Documentation

## Overview

This document describes the complete message flow and game sequence for the KubeGame backend WebSocket API, including the connection lifecycle, game phases, and real-time communication patterns.

## Architecture Overview

```mermaid
graph LR
    subgraph "Backend"
        WebSocket --> GameSession[Game Session Service]
        WebSocket --> GameUseCase[Game Use Case]
        WebSocket --> GameEngine[Game Engine Use Case]
        
        GameSession --> SessionRepo[Session Repository]
        GameSession --> GameRepo[Game Repository]
        GameSession --> PodRepo[Pod Repository]
        GameSession --> NodeRepo[Node Repository]
        GameSession --> K8sService[Kubernetes Service]
        
        GameEngine --> Scheduler[Scheduler Service]
        GameEngine --> Timers[Game Timers]
        
        subgraph "Per-Connection Resources"
            Session[Game Session]
            Cluster[Fake Cluster]
            Nodes[for Player and for K8s Scheduler]
            Game[Game Instance]
            Pods[Initial 8 Pods]
        end
    end
    subgraph "Frontend"
        FrontendClient[Frontend Client] --> WebSocket[WebSocket Service]
    end
```

## Cluster overview

Each WebSocket connection creates a unique cluster. Each cluster has 2 namespaces for the player and for the k8s scheduler. Everytime game engine request pod schedule, send same request to both namespaces because player and k8s scheduler can schedule pods to their own namespaces, and player try to beat k8s scheduler in the same situation.

```mermaid
graph LR
    subgraph Backend
        subgraph FC1[Fake Cluster]
            subgraph NS1-1["Namespace (Player)"]
                subgraph "Node 2"
                    Pod1-1-3[Pod 3]
                    Pod1-1-4[Pod 4]
                end
                subgraph "Node 1"
                    Pod1-1-1[Pod 1]
                    Pod1-1-2[Pod 2]
                end
            end

            subgraph NS1-2["Namespace (K8s Scheduler)"]
                subgraph "Node 2"
                    Pod1-2-3[Pod 3]
                    Pod1-2-4[Pod 4]
                end
                subgraph "Node 1"
                    Pod1-2-1[Pod 1]
                    Pod1-2-2[Pod 2]
                end
            end
        end

        subgraph FC2[Fake Cluster]
            subgraph NS2-1["Namespace (Player)"]
                subgraph "Node 2"
                    Pod2-1-3[Pod 3]
                    Pod2-1-4[Pod 4]
                end
                subgraph "Node 1"
                    Pod2-1-1[Pod 1]
                    Pod1-1-2[Pod 2]
                end
            end

            subgraph NS2-2["Namespace (K8s Scheduler)"]
                subgraph "Node 2"
                    Pod2-2-3[Pod 3]
                    Pod2-2-4[Pod 4]
                end
                subgraph "Node 1"
                    Pod2-2-1[Pod 1]
                    Pod2-2-2[Pod 2]
                end
            end
        end
    end

    User1[User] --> FC1
    User2[User] --> FC2

```

## Connection-Based Game Lifecycle

### Phase 1: Connection & Session Creation
**Trigger**: Frontend connects to WebSocket `/ws`

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GS as Game Session Service
    participant Repos as Repositories
    
    F->>WS: WebSocket Connect
    WS->>GS: CreateSession(connectionID)
    GS->>Repos: Create session record
    GS->>GS: CreateCluster(connectionID)
    GS->>Repos: Create initial nodes (for player and for k8s scheduler)
    GS-->>WS: Session created
    WS->>F: {"type": "session_created", "data": session}

    Note over F,Repos: Session State: ClusterReady
    Note over F,Repos: Game NOT started - waiting for frontend
```

### Phase 2: Frontend Game Start
**Trigger**: Frontend sends `start_game` message

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GS as Game Session Service
    participant GE as Game Engine
    participant Repos as Repositories
    
    F->>WS: {"type": "start_game"}
    WS->>GS: StartGame(connectionID)
    GS->>Repos: Create game entity
    GS->>GS: createInitialPods()
    GS->>Repos: Create initial pods
    GS-->>WS: Game started
    WS->>GE: StartGame(gameID)
    GE->>GE: Start timers & background processes
    WS->>F: {"type": "game_started", "data": session}
    
    Note over F,Repos: Session State: Playing
    Note over F,Repos: Game engine running with timers
```

### Phase 3: Game Interaction
**Trigger**: Various frontend actions

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GU as Game Use Case
    participant GE as Game Engine
    
    alt Pod Scheduling
        F->>WS: {"type": "schedule_pod", "data": {"pod_id": "...", "node_id": "..."}}
        WS->>GU: SchedulePodToNode(podID, nodeID, connectionID)
        GU->>GU: Validate ownership & capacity
        GU->>Repos: Update pod status
        GU-->>WS: Success/Error
        WS->>F: {"type": "pod_scheduled"} or {"type": "error"}
    
    else Ping/Pong
        F->>WS: {"type": "ping"}
        WS->>F: {"type": "pong"}

    else Ping/Pong (Server)
        WS->>F: {"type": "ping"}
        F->>WS: {"type": "pong"}

    else Game Control
        F->>WS: {"type": "start_game"}
        Note over WS: Game already started, return game state (assuming user clicked start again)
    end
```

### Phase 4: Background Game Events
**Trigger**: Automatic timers and game engine

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GU as Game Use Case
    participant GE as Game Engine
    participant Scheduler as Kubernetes Scheduler
    
    loop Every 1-5 seconds randomly
        GE->>GE: Generate random number of pods for user and k8s scheduler
        GE->>Repos: Create new pods
        GE->>WS: BroadcastEvent("pod_created")
        WS->>F: {"type": "pod_created", "data": pod}
        GE->>Scheduler: Schedule pod placement
        Scheduler->>GE: Pod placement result
        GE->>WS: BroadcastEvent("pod_scheduled")
        WS->>F: {"type": "pod_scheduled", "data": pod, node}
    end
    
    loop Every 1 second
        GE->>GE: Update game timer & scores
        GE->>Repos: Update game state
        GE->>WS: BroadcastEvent("game_update")
        WS->>F: {"type": "game_update", "data": {timeLeft, scores, pods, nodes}}
    end
```

### Phase 5: Game End & Cleanup
**Trigger**: Game timer expires or manual stop

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GS as Game Session Service
    participant GU as Game Use Case
    participant GE as Game Engine
    participant Scheduler as Kubernetes Scheduler
    participant Repos as Repositories

    alt Timer Expiration
        GE->>GE: Timer reaches 0
        GE->>GS: Game time expired
        GS->>GS: StopGame(connectionID)
        GE->>WS: BroadcastEvent("game_over")
        WS->>F: {"type": "game_over", "data": {finalScores, results}}
    end

    Note over GS: Session State: ClusterReady (can restart)
```

### Phase 6: Connection Cleanup
**Trigger**: Frontend disconnects

```mermaid
sequenceDiagram
    participant F as Frontend
    participant WS as WebSocket Service
    participant GS as Game Session Service
    participant Repos as Repositories
    participant FC as Fake Cluster
    
    F->>WS: WebSocket Disconnect
    WS->>GS: CloseSession(connectionID)
    GS->>GS: StopGame(connectionID) if running
    GS->>GS: DeleteCluster(connectionID)
    GS->>FC: Cleanup 
    GS->>Repos: Delete all nodes for cluster
    GS->>Repos: Delete all pods for cluster  
    GS->>Repos: Delete session record
    WS->>WS: Remove client from registry
    
    Note over F,Repos: All resources cleaned up
```

## Message Types Reference

### Client-to-Server Messages

| Type | Description | Payload | Response |
|------|-------------|---------|----------|
| `ping` | Heartbeat check | `null` | `pong` event |
| `start_game` | Start new game (if stopped) | `null` | `game_started` or `error` |
| `schedule_pod` | Player schedules pod to node | `{"pod_id": "...", "node_id": "..."}` | `pod_scheduled` or `error` |

### Server-to-Client Events

| Type | Description | Payload | Response |
|------|-------------|----------------|----------|
| `session_ready` | Session info after join | `{"sessionID": "...", "clusterID": "...", "state": "..."}` | No response |
| `game_started` | Game started confirmation | `{"gameID": "...", "timeLeft": 300}` | No response |
| `game_over` | Game ended (timer) | `{"results": {...}, "playerScore": 0, "cpuScore": 0}` | No response |
| `game_update` | Periodic game state | `{"timeLeft": 250, "playerScore": 5, "cpuScore": 3}` | No response |
| `pod_created` | New pod available | `{"id": "...", "name": "...", "requirements": {...}}` | No response |
| `pod_scheduled` | Pod assigned to node | `{"podID": "...", "nodeID": "...", "scheduledBy": "player\|cpu"}` | No response |
| `error` | Error occurred | `{"message": "Error description"}` | No response |
| `ping` | Ping request from server | `null` | `pong` |

## Game Rules & Scoring

### Resource Allocation
- **Player Nodes**: 4 nodes with varying CPU/Memory capacity (players can schedule here)
- **K8s Scheduler Node**: 1 node with high capacity for automatic scheduling
- **Initial Pods**: 8 pods created at game start with random resource requirements

### Scoring System
- **Player Points**: Earned when player successfully schedules pods to player nodes
- **K8s Scheduler Points**: Earned when Kubernetes scheduler assigns pods to the scheduler node
- **Game Duration**: 5 minutes (300 seconds) by default

### Pod Labels & Types
Pods are randomly assigned one of four labels:
- `banana` 🍌
- `chocolate` 🍫  
- `strawberry` 🍓
- `vanilla` 🍦

### Node Types & Constraints
- **Player Nodes**: `worker-01`, `worker-02`, `worker-03`, `worker-04` - user controllable
- **K8s Scheduler Node**: `scheduler-01` - automatic scheduling only

## Session Isolation

Each WebSocket connection gets completely isolated resources:

1. **Unique Session ID**: UUID-based session identifier
2. **Dedicated Cluster**: Cluster name format: `cluster-{connectionID}`
3. **Isolated Nodes**: Node IDs include cluster suffix: `player-node-1-cluster-{connectionID}`
4. **Separate Game State**: Each session has independent game timer, score, and pod set
5. **Independent Lifecycle**: Sessions don't affect each other

## Error Handling

### Common Error Scenarios

| Scenario | Error Message | Recovery |
|----------|---------------|----------|
| Invalid pod scheduling | `"Pod does not exist"` | Check pod ID |
| Node capacity exceeded | `"Insufficient resources on node"` | Try different node |
| Cross-session access | `"Node does not belong to session"` | Use session's own nodes |
| Game not running | `"No active game for session"` | Start game first |
| Cluster not ready | `"Cluster not ready for session"` | Wait for cluster creation |

### WebSocket Error Handling
- Connection errors trigger automatic session cleanup
- Invalid message types log warnings but don't disconnect
- Repository errors are caught and returned as error events

## Performance Characteristics

### Scalability
- **Memory**: ~1MB per active session (nodes + pods + game state)
- **CPU**: Background timers run per game (3 timers × number of active games)
- **Cleanup**: Automatic on disconnect, manual cleanup available

### Real-time Updates
- **Game State**: 1-second intervals
- **Pod Generation**: 2-second intervals  
- **Scheduler Actions**: 3-second intervals
- **Heartbeat**: 30-second ping intervals (frontend-initiated)

## Development Notes

### Testing Strategy
- Integration tests create isolated test environments
- Each test gets fresh dependency injection container
- WebSocket connections tested with `gorilla/websocket` test client
- Session isolation verified through multi-client tests

### Future Extensions
- Game difficulty levels
    - Easy mode: No node creation/destruction
    - Normal mode: User controls only Pod scheduling, automatic node management
    - Hard mode: User can create/destroy nodes
- Team/multiplayer modes
- Ranking system for competitive play
