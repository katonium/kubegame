# Generated Types for KubeGame WebSocket API

This directory contains auto-generated type definitions for both TypeScript (frontend) and Go (backend) based on the JSON Schema definitions.

## TypeScript Types (`generated_ts/`)

### Usage

```typescript
import { GameInfo, PodCreatedEvent, MessageHandler } from './generated_ts';

// Type-safe message handling
const handlePodCreated = (event: PodCreatedEvent) => {
  console.log(`New pod created. Time left: ${event.data.timeLeft}`);
  console.log(`Player pods: ${event.data.playerCluster.pods.length}`);
  console.log(`Scheduler pods: ${event.data.schedulerCluster.pods.length}`);
};

// Type-safe message creation
const scheduleMessage: SchedulePodMessage = {
  type: 'schedule_pod',
  timestamp: new Date().toISOString(),
  data: {
    pod_id: 'pod-123',
    node_id: 'node-456'
  }
};
```

### Files

- `common.ts` - Common types (Pod, Node, GameInfo, etc.)
- `messages.ts` - WebSocket message types
- `index.ts` - Re-exports all types

## Go Types (`generated_go/`)

### Usage

```go
import "path/to/kubegame/docs/schema/generated_go"

// Type-safe message handling
func handlePodCreated(event schema.PodCreatedEvent) {
    fmt.Printf("New pod created. Time left: %d\n", event.Data.TimeLeft)
    fmt.Printf("Player pods: %d\n", len(event.Data.PlayerCluster.Pods))
    fmt.Printf("Scheduler pods: %d\n", len(event.Data.SchedulerCluster.Pods))
}

// Type-safe message creation
handler := schema.NewMessageHandler()
scheduleMsg := handler.CreateSchedulePodMessage("pod-123", "node-456")

// JSON marshaling
jsonData, err := json.Marshal(scheduleMsg)
if err != nil {
    log.Fatal(err)
}
```

### Files

- `common.go` - Common types and constants
- `messages.go` - WebSocket message types
- `utils.go` - Message creation and parsing utilities

## Integration

### Frontend Integration

Copy or symlink the `generated_ts` directory to your frontend project:

```bash
# Option 1: Copy
cp -r docs/schema/generated_ts frontend/src/types/api

# Option 2: Symlink
ln -s ../../docs/schema/generated_ts frontend/src/types/api
```

### Backend Integration

Import the package directly:

```go
import "github.com/katonium/kubegame/docs/schema/generated_go"
```

Or copy to your backend module:

```bash
cp -r docs/schema/generated_go backend/pkg/api/types
```

## Type Safety Benefits

### Compile-time Validation
- TypeScript: Catch type mismatches at build time
- Go: Strong typing prevents runtime errors

### IDE Support
- Auto-completion for message fields
- Type hints and documentation
- Refactoring support

### Consistency
- Same data structures across frontend and backend
- Single source of truth from JSON Schema
- Automated generation reduces manual errors

## Regeneration

When JSON Schema files are updated, regenerate these types by running:

```bash
# This will be implemented later with a generation script
./scripts/generate-types.sh
```

## Validation

Both TypeScript and Go types include:
- Required field validation
- Enum value validation  
- Type constraints (string, number, etc.)
- Nested object validation
