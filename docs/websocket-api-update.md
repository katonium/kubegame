# Frontend Updates - WebSocket API Alignment

## Summary of Changes

This document outlines the changes made to align the frontend implementation with the updated backend WebSocket API as documented in `game-flow-documentation.md`.

## Key Changes Made

### 1. WebSocket Service Updates (`/src/lib/websocket.ts`)

**Message Type Alignment:**
- Updated `GameEvent` type to remove timestamp field (matches backend)
- Added `GameSession` type to handle session creation
- Updated `GameState` type to match backend structure (removed state field, added results)
- Updated session management to track connection lifecycle

**API Method Updates:**
- Removed `joinGame()` method (automatic on connection)
- Added `startGame()` method for explicit game start
- Updated `schedulePod()` to use correct parameter names (`pod_id`, `node_id`)
- Added session tracking and automatic session creation handling

### 2. Type System Updates (`/src/types/index.ts`)

**Pod Type Changes:**
- Changed label values from capitalized to lowercase (`'banana'` vs `'Banana'`)
- Renamed `owner` property to `scheduledBy` to match backend
- Updated `nodeType` to be strictly typed as `'player' | 'cpu'`

### 3. WebSocket Hook Updates (`/src/hooks/use-websocket.ts`)

**Event Handling:**
- Added support for `session_created` event (Phase 1 of connection lifecycle)
- Updated to handle `game_started`, `game_update`, `pod_created`, `pod_scheduled` events
- Added proper session state management
- Added `gameStatus` state to track connection/session lifecycle
- Removed automatic game joining (now handled by backend on connection)

**New Features:**
- Added `startGame()` method for explicit game initiation
- Added session information exposure
- Added mock node creation for UI (temporary until backend sends node data)

### 4. Game Component Updates (`/src/components/game/KubeWarsGame.tsx`)

**Game Flow:**
- Updated to use new session-based flow (wait for session_created before allowing game start)
- Fixed UI state synchronization with backend game status
- Updated pod scheduling to use `scheduledBy` instead of `owner`
- Added session information display in connection status

**UX Improvements:**
- Start button now disabled until session is ready
- Better connection status feedback (connecting → session setup → ready)
- Display session ID and cluster ID when connected

### 5. UI Component Updates

**PreGamePanel:**
- Added `canStart` prop to disable start button until session ready
- Updated button text to show "Preparing..." when not ready

**PodCard:**
- Updated label mappings to use lowercase pod labels
- Maintained visual consistency while adapting to backend data format

### 6. Constants Updates (`/src/lib/constants.ts`)

**Configuration:**
- Updated game duration to 300 seconds (5 minutes) to match backend
- Added pod label emoji mappings
- Added node type display labels
- Cleaned up legacy mock data (now handled by backend)

## Connection Lifecycle

The updated frontend now properly follows the backend's connection lifecycle:

1. **Phase 1 - Connection**: Frontend connects to `/ws`, backend creates session
2. **Phase 2 - Session Ready**: Backend sends `session_created`, frontend enables start button  
3. **Phase 3 - Game Start**: User clicks start, frontend sends `start_game`
4. **Phase 4 - Game Playing**: Backend sends game events, frontend responds
5. **Phase 5 - Game End**: Backend sends `game_over`, frontend shows results
6. **Phase 6 - Cleanup**: Connection closes, resources cleaned up

## Event Handling

The frontend now correctly handles all backend events:

- `session_created` → Store session info, enable UI
- `game_started` → Begin game UI, start timer
- `game_update` → Update scores and time
- `pod_created` → Add new pods to pending list
- `pod_scheduled` → Move pods to running state
- `game_over` → Show final results
- `error` → Display error messages
- `ping` → Respond with pong for heartbeat

## API Message Format

**Client → Server:**
```json
{
  "type": "start_game",
  "data": null
}
```

**Server → Client:**
```json
{
  "type": "session_created", 
  "data": {
    "sessionID": "uuid",
    "clusterID": "uuid", 
    "state": "cluster_ready"
  }
}
```

## Testing

To test the updated frontend:

1. Start the backend server
2. Start the frontend development server  
3. Open browser to frontend URL
4. Should see "Connecting..." → "Setting up session..." → "Connected! Session X ready"
5. Start button should be enabled only when session is ready
6. Game should flow through all phases as documented

## Next Steps

- [ ] Test integration with running backend
- [ ] Add error handling for failed connections
- [ ] Add reconnection logic for dropped connections
- [ ] Consider adding loading states for better UX
- [ ] Add proper node data from backend (remove mock nodes)