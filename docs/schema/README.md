# KubeGame WebSocket API Schema Documentation

This directory contains the API specification for all WebSocket communication types in the KubeGame project.

## Schema Organization

The API is now defined using **OpenAPI 3.0.3** specification with automatic code generation for Go and TypeScript.

### OpenAPI Specification

| File | Description |
|------|-------------|
| `websocket-api.yaml` | **Main OpenAPI 3.0.3 specification** containing all WebSocket message types and schemas |

## Automatic Code Generation

Type-safe code generation from OpenAPI specification:

### Backend (Go)
- **Location**: `backend/api/generated/types.go`
- **Tool**: [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
- **Command**: `cd backend/api && task generate`

### Frontend (TypeScript)
- **Location**: `frontend/src/types/generated/api.ts`
- **Tool**: [openapi-typescript](https://github.com/openapi-typescript/openapi-typescript)
- **Command**: `cd frontend && task generate`

### Usage Examples

**Go (Backend):**
```go
import "github.com/katonium/kubegame/backend/generated"

// Use generated types
var gameUpdate generated.GameUpdateEvent
gameUpdate.Type = "game_update"
gameUpdate.Timestamp = time.Now()
gameUpdate.Data = generated.GameInfo{
    TimeLeft: 300,
    PlayerScore: 5,
    // ...
}
```

**TypeScript (Frontend):**
```typescript
import type { components } from '@/types/generated/api';

type GameUpdateEvent = components['schemas']['GameUpdateEvent'];
type PodInfo = components['schemas']['PodInfo'];

// Use generated types
const gameUpdate: GameUpdateEvent = {
  type: 'game_update',
  timestamp: new Date().toISOString(),
  data: {
    timeLeft: 300,
    playerScore: 5,
    // ...
  }
};
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

## Game-Specific Types

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
