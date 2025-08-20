# KubeGame WebSocket API Schema Documentation

This directory contains JSON schemas for all WebSocket communication types in the KubeGame project.

## Schema Organization

All schemas follow the JSON Schema Draft 07 specification and are organized by message type.

## Client-to-Server Messages

These are messages sent from the frontend client to the backend server:

| Schema File | Message Type | Description |
|-------------|--------------|-------------|
| `ping.json` | `ping` | Heartbeat check from client |
| `pong.json` | `pong` | Response to server ping |
| `start_game.json` | `start_game` | Start a new game session |
| `schedule_pod.json` | `schedule_pod` | Player schedules a pod to a node |

## Server-to-Client Events

These are events sent from the backend server to the frontend client:

| Schema File | Event Type | Description |
|-------------|------------|-------------|
| `session_ready.json` | `session_ready` | Session created and ready |
| `game_started.json` | `game_started` | Game has started successfully |
| `game_over.json` | `game_over` | Game ended (timer expired) |
| `game_update.json` | `game_update` | Periodic game state update (every 1s) |
| `pod_created.json` | `pod_created` | New pod created by game engine |
| `pod_scheduled.json` | `pod_scheduled` | Pod assigned to a node |
| `error.json` | `error` | Error occurred during operation |
| `server_ping.json` | `ping` | Server ping for heartbeat |

## Message Structure

All messages follow this general structure:

```json
{
  "type": "message_type",
  "data": { /* payload varies by message type */ },
  "timestamp": "2023-10-01T12:00:00Z"
}
```

### Timestamp Field

All messages include an ISO 8601 timestamp field that indicates when the message/event was created:
- **Format**: `YYYY-MM-DDTHH:mm:ssZ` (UTC timezone)
- **Purpose**: Enable message ordering, latency analysis, and debugging
- **Required**: Yes, for all message types

## Usage

These schemas can be used for:

- **Validation**: Validate incoming/outgoing WebSocket messages
- **Code Generation**: Generate TypeScript interfaces or Go structs
- **Documentation**: Provide clear API documentation
- **Testing**: Ensure message format compliance in tests

## Examples

### Client Message Example
```json
{
  "type": "schedule_pod",
  "data": {
    "pod_id": "pod-123",
    "node_id": "worker-01"
  },
  "timestamp": "2023-10-01T12:00:00Z"
}
```

### Server Event Example
```json
{
  "type": "game_update",
  "data": {
    "timeLeft": 250,
    "playerScore": 5,
    "cpuScore": 3,
    "pods": [...],
    "nodes": [...]
  },
  "timestamp": "2023-10-01T12:00:01Z"
}
```

## Validation Tools

You can validate messages against these schemas using:

- **JavaScript/TypeScript**: [ajv](https://ajv.js.org/)
- **Go**: [gojsonschema](https://github.com/xeipuuv/gojsonschema)
- **Python**: [jsonschema](https://python-jsonschema.readthedocs.io/)
- **Online**: [JSON Schema Validator](https://www.jsonschemavalidator.net/)

## Game-Specific Types

### Pod Labels
Pods are assigned one of four emoji-based labels:
- `banana` 🍌
- `chocolate` 🍫
- `strawberry` 🍓
- `vanilla` 🍦

### Node Types
- `player` - Nodes controllable by the player (worker-01 through worker-04)
- `scheduler` - Nodes controlled by Kubernetes scheduler (scheduler-01)

### Session States
- `ClusterReady` - Session created, cluster ready, but game not started
- `Playing` - Game is actively running

### Error Codes
Common error codes for programmatic handling:
- `POD_NOT_FOUND` - Referenced pod doesn't exist
- `NODE_NOT_FOUND` - Referenced node doesn't exist
- `INSUFFICIENT_RESOURCES` - Node doesn't have enough capacity
- `INVALID_SESSION` - Session ID is invalid or expired
- `GAME_NOT_RUNNING` - Attempted game operation when no game is active
- `CLUSTER_NOT_READY` - Cluster initialization not complete
- `CROSS_SESSION_ACCESS` - Attempting to access resources from different session

## Related Documentation

- [Game Flow Documentation](../game-flow-documentation.md) - Complete game flow and sequence diagrams
- [WebSocket API Update](../../frontend/docs/websocket-api-update.md) - Frontend WebSocket implementation details