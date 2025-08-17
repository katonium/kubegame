# KubeGame - Kubernetes Pod Scheduling Game

A real-time multiplayer game where players compete against the Kubernetes scheduler to schedule pods efficiently.

## Architecture

- **Backend**: Go with clean architecture, using real Kubernetes scheduler and WebSocket for real-time communication
- **Frontend**: Next.js with React, TypeScript, and Tailwind CSS
- **Communication**: WebSocket for real-time game updates

## Getting Started

### Prerequisites

- Go 1.24+ 
- Node.js 18+
- npm or yarn

### Backend Setup

1. Navigate to the backend directory:
```bash
cd backend/api
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build and run the backend:
```bash
task build
./build/app
```

The backend will start on port 8080 with WebSocket endpoint at `/ws`.

### Frontend Setup

1. Navigate to the frontend directory:
```bash
cd frontend
```

2. Install dependencies:
```bash
npm install
```

3. Run the development server:
```bash
npm run dev
```

The frontend will start on port 3000.

### Playing the Game

1. Start both backend and frontend servers
2. Open http://localhost:3000 in your browser
3. The game will automatically connect to the backend
4. Click "Start Game" when ready
5. Drag and drop pods to your player nodes to schedule them
6. Compete against the Kubernetes scheduler for points!

## Game Features

- **Real-time Pod Scheduling**: Watch as pods are created, scheduled, and terminated in real-time
- **Player vs. Kubernetes Scheduler**: Compete against the real Kubernetes scheduler
- **Resource Management**: Consider CPU and memory requirements when scheduling
- **Live Updates**: All game events are synchronized in real-time via WebSocket
- **Clean Architecture**: Backend follows clean architecture principles with proper separation of concerns

## Technology Stack

### Backend
- **Go**: Main programming language
- **uber-go/fx**: Dependency injection framework
- **gorilla/websocket**: WebSocket communication
- **client-go**: Kubernetes client library with fake cluster
- **Clean Architecture**: Domain, Use Case, and Infrastructure layers

### Frontend
- **Next.js 15**: React framework
- **TypeScript**: Type safety
- **Tailwind CSS**: Styling
- **WebSocket**: Real-time communication with backend

## Project Structure

```
kubegame/
├── backend/
│   └── api/
│       ├── domain/          # Business entities and interfaces
│       ├── usecase/         # Business logic
│       ├── infrastructure/  # External adapters (WebSocket, Kubernetes)
│       └── util/           # Utilities (logger)
├── frontend/
│   └── src/
│       ├── components/     # React components
│       ├── hooks/         # Custom React hooks
│       ├── lib/          # WebSocket service
│       └── types/        # TypeScript types
└── kubernetes/           # Kubernetes source code for scheduler
```
