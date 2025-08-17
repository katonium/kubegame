// Package websocket provides WebSocket service implementation for real-time game communication.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/katonium/kubegame/backend/domain/entity"
	"github.com/katonium/kubegame/backend/domain/service"
	"github.com/katonium/kubegame/backend/util/logger"
)

// Connection represents an abstract WebSocket connection interface.
type Connection interface {
	io.ReadWriteCloser
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
}

// wsConnection wraps gorilla/websocket.Conn to implement Connection interface.
type wsConnection struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// Read implements io.Reader interface.
func (w *wsConnection) Read(p []byte) (n int, err error) {
	_, data, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n = copy(p, data)
	return n, nil
}

// Write implements io.Writer interface.
func (w *wsConnection) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close implements io.Closer interface.
func (w *wsConnection) Close() error {
	return w.conn.Close()
}

// WriteJSON writes a JSON-encoded object to the connection.
func (w *wsConnection) WriteJSON(v interface{}) error {
	// Log outgoing message for debugging
	if jsonBytes, err := json.Marshal(v); err == nil {
		logger.Debug(context.Background(), "WebSocket SEND: %s", string(jsonBytes))
	}
	
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(v)
}

// ReadJSON reads a JSON-encoded object from the connection.
func (w *wsConnection) ReadJSON(v interface{}) error {
	err := w.conn.ReadJSON(v)
	
	// Log incoming message for debugging
	if err == nil {
		if jsonBytes, jsonErr := json.Marshal(v); jsonErr == nil {
			logger.Debug(context.Background(), "WebSocket RECV: %s", string(jsonBytes))
		}
	}
	
	return err
}

// Client represents an abstract WebSocket client.
type Client interface {
	ID() string
	Send(data []byte) error
	SendJSON(v interface{}) error
	Close() error
}

// client implements the Client interface.
type client struct {
	id         string
	connection Connection
	sendCh     chan []byte
	closeCh    chan struct{}
	once       sync.Once
}

// newClient creates a new client with the given ID and connection.
func newClient(id string, conn Connection) Client {
	c := &client{
		id:         id,
		connection: conn,
		sendCh:     make(chan []byte, 256),
		closeCh:    make(chan struct{}),
	}

	// Start write pump goroutine
	go c.writePump()

	return c
}

// ID returns the client's unique identifier.
func (c *client) ID() string {
	return c.id
}

// Send queues data to be sent to the client.
func (c *client) Send(data []byte) error {
	select {
	case c.sendCh <- data:
		return nil
	case <-c.closeCh:
		return fmt.Errorf("client connection is closed")
	default:
		return fmt.Errorf("client send buffer is full")
	}
}

// SendJSON marshals the given object to JSON and sends it to the client.
func (c *client) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Send(data)
}

// Close closes the client connection and cleanup resources.
func (c *client) Close() error {
	c.once.Do(func() {
		close(c.closeCh)
		c.connection.Close()
	})
	return nil
}

// writePump handles writing messages to the WebSocket connection.
func (c *client) writePump() {
	defer c.connection.Close()

	for {
		select {
		case data, ok := <-c.sendCh:
			if !ok {
				return
			}
			if _, err := c.connection.Write(data); err != nil {
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

// webSocketService implements the WebSocket service for game communication.
type webSocketService struct {
	clients           map[string]Client  // Connected clients indexed by ID
	broadcast         chan []byte        // Broadcast channel for messages to all clients
	register          chan Client        // Register channel for new clients
	unregister        chan Client        // Unregister channel for disconnecting clients
	upgrader          websocket.Upgrader // WebSocket upgrader configuration
	mu                sync.RWMutex       // Mutex for thread-safe access to clients map
	gameSessionService service.GameSessionService // Session management
	gameEngine        GameEngine         // Game engine interface
	gameUseCase       GameUseCase       // Game use case interface
}

// Interfaces to avoid circular dependencies
type GameEngine interface {
	StartGame(ctx context.Context, gameID string) error
	StopGame(ctx context.Context, gameID string) error
}

type GameUseCase interface {
	StartGame(ctx context.Context, gameID string) error
	StopGame(ctx context.Context, gameID string) error
	SchedulePodToNode(ctx context.Context, podID, nodeID, connectionID string) error
}

// NewWebSocketService creates a new WebSocket service instance.
// It initializes the service and starts the background hub goroutine.
func NewWebSocketService(gameSessionService service.GameSessionService, gameEngine GameEngine, gameUseCase GameUseCase) service.WebSocketService {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	svc := &webSocketService{
		clients:            make(map[string]Client),
		broadcast:          make(chan []byte),
		register:           make(chan Client),
		unregister:         make(chan Client),
		upgrader:           upgrader,
		gameSessionService: gameSessionService,
		gameEngine:         gameEngine,
		gameUseCase:        gameUseCase,
	}

	// Start the hub goroutine
	go svc.run()
	
	// Start the timer update goroutine
	go svc.runTimerUpdates()

	return svc
}

// RegisterClient registers a new WebSocket client with the service.
func (ws *webSocketService) RegisterClient(ctx context.Context, clientID string, conn interface{}) error {
	wsConn, ok := conn.(*websocket.Conn)
	if !ok {
		return fmt.Errorf("invalid connection type, expected *websocket.Conn")
	}

	// Wrap the connection
	connection := &wsConnection{conn: wsConn}

	// Create client
	client := newClient(clientID, connection)

	// Register the client
	ws.register <- client

	logger.Info(ctx, "WebSocket client %s registered", clientID)

	// Start read pump for this client
	go ws.readPump(ctx, client)

	return nil
}

// UnregisterClient removes a WebSocket client from the service.
func (ws *webSocketService) UnregisterClient(ctx context.Context, clientID string) error {
	ws.mu.RLock()
	client, exists := ws.clients[clientID]
	ws.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	ws.unregister <- client
	logger.Info(ctx, "WebSocket client %s unregistered", clientID)

	return nil
}

// GetConnectedClients returns a list of all currently connected client IDs.
func (ws *webSocketService) GetConnectedClients(ctx context.Context) ([]string, error) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	clientIDs := make([]string, 0, len(ws.clients))
	for _, client := range ws.clients {
		clientIDs = append(clientIDs, client.ID())
	}

	return clientIDs, nil
}

// BroadcastEvent broadcasts a game event to all connected clients.
func (ws *webSocketService) BroadcastEvent(ctx context.Context, event *entity.GameEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logger.Debug(ctx, "Broadcasting event: %s - content: %s", event.Type, string(data))

	select {
	case ws.broadcast <- data:
		return nil
	default:
		return fmt.Errorf("broadcast channel is full")
	}
}

// SendEventToClient sends a game event to a specific client.
func (ws *webSocketService) SendEventToClient(ctx context.Context, clientID string, event *entity.GameEvent) error {
	ws.mu.RLock()
	client, exists := ws.clients[clientID]
	ws.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	// Log the complete outgoing event for debugging
	if eventBytes, err := json.Marshal(event); err == nil {
		logger.Debug(ctx, "Sending event to client %s: %s", clientID, string(eventBytes))
	} else {
		logger.Debug(ctx, "Sending event %s to client %s (marshal error: %v)", event.Type, clientID, err)
	}

	return client.SendJSON(event)
}

// BroadcastGameState broadcasts the current game state to all connected clients.
func (ws *webSocketService) BroadcastGameState(ctx context.Context, gameState *entity.Game) error {
	event := &entity.GameEvent{
		Type:      "game_state_update",
		Data:      gameState,
		Timestamp: gameState.UpdatedAt,
	}

	return ws.BroadcastEvent(ctx, event)
}

// BroadcastPodUpdate broadcasts a pod update event to all connected clients.
func (ws *webSocketService) BroadcastPodUpdate(ctx context.Context, pod *entity.Pod) error {
	event := &entity.GameEvent{
		Type:      "pod_update",
		Data:      pod,
		Timestamp: pod.CreatedAt,
	}

	return ws.BroadcastEvent(ctx, event)
}

// BroadcastNodeUpdate broadcasts a node update event to all connected clients.
func (ws *webSocketService) BroadcastNodeUpdate(ctx context.Context, node *entity.Node) error {
	event := &entity.GameEvent{
		Type:      "node_update",
		Data:      node,
		Timestamp: time.Now(),
	}

	return ws.BroadcastEvent(ctx, event)
}

// run is the main hub goroutine that handles client registration, unregistration, and broadcasting.
func (ws *webSocketService) run() {
	for {
		select {
		case client := <-ws.register:
			ws.mu.Lock()
			ws.clients[client.ID()] = client
			ws.mu.Unlock()

			// Create session and cluster for new client
			ctx := context.Background()
			_, err := ws.gameSessionService.CreateSession(ctx, client.ID())
			if err != nil {
				logger.Error(ctx, "Failed to create session for client %s: %v", client.ID(), err)
			} else {
				// Create cluster for the session
				if err := ws.gameSessionService.CreateCluster(ctx, client.ID()); err != nil {
					logger.Error(ctx, "Failed to create cluster for client %s: %v", client.ID(), err)
				} else {
					logger.Info(ctx, "Created session and cluster for client %s", client.ID())
					// Note: Game will be started when frontend sends "start_game" message
					
					// Send welcome message with session info
					session, _ := ws.gameSessionService.GetSession(ctx, client.ID())
					welcomeEvent := &entity.GameEvent{
						Type:      "session_created",
						Data:      session,
						Timestamp: time.Now(),
					}
					client.SendJSON(welcomeEvent)
				}
			}

		case client := <-ws.unregister:
			ws.mu.Lock()
			if _, ok := ws.clients[client.ID()]; ok {
				delete(ws.clients, client.ID())
				client.Close()
				
				// Close session and cleanup cluster
				ctx := context.Background()
				if err := ws.gameSessionService.CloseSession(ctx, client.ID()); err != nil {
					logger.Error(ctx, "Failed to close session for client %s: %v", client.ID(), err)
				} else {
					logger.Info(ctx, "Closed session and cleaned up cluster for client %s", client.ID())
				}
			}
			ws.mu.Unlock()

		case message := <-ws.broadcast:
			ws.mu.RLock()
			for clientID, client := range ws.clients {
				if err := client.Send(message); err != nil {
					// Remove client if send fails
					delete(ws.clients, clientID)
					client.Close()
				}
			}
			ws.mu.RUnlock()
		}
	}
}

// readPump handles reading messages from the WebSocket connection.
func (ws *webSocketService) readPump(ctx context.Context, cli Client) {
	defer func() {
		ws.unregister <- cli
	}()

	for {
		var incomingEvent struct {
			Type string      `json:"type"`
			Data interface{} `json:"data"`
		}

		// Use the client's connection to read JSON directly
		if clientImpl, ok := cli.(*client); ok {
			if err := clientImpl.connection.ReadJSON(&incomingEvent); err != nil {
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error(ctx, "WebSocket read error for client %s: %v", cli.ID(), err)
				}
				break
			}

			ws.handleClientMessage(ctx, cli, &incomingEvent)
		} else {
			break
		}
	}
}

// handleClientMessage processes incoming messages from a WebSocket client.
func (ws *webSocketService) handleClientMessage(ctx context.Context, client Client, message *struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}) {
	// Log the complete incoming message for debugging
	if msgBytes, err := json.Marshal(message); err == nil {
		logger.Debug(ctx, "Processing message from client %s: %s", client.ID(), string(msgBytes))
	} else {
		logger.Debug(ctx, "Received message from client %s: type=%s (marshal error: %v)", client.ID(), message.Type, err)
	}

	// Update session activity
	if err := ws.gameSessionService.UpdateSessionActivity(ctx, client.ID()); err != nil {
		logger.Error(ctx, "Failed to update session activity for client %s: %v", client.ID(), err)
	}

	// Handle different message types
	switch message.Type {
	case "ping":
		// Respond with pong
		response := &entity.GameEvent{
			Type:      "pong",
			Data:      nil,
			Timestamp: time.Now(),
		}
		client.SendJSON(response)

	case "join_game":
		logger.Info(ctx, "Client %s joined the game", client.ID())
		// Send session info and initial game state
		if session, err := ws.gameSessionService.GetSession(ctx, client.ID()); err != nil {
			logger.Error(ctx, "Failed to get session for client %s: %v", client.ID(), err)
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Failed to get session"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
		} else {
			// Send session ready event
			sessionReadyEvent := &entity.GameEvent{
				Type:      "session_ready",
				Data:      session,
				Timestamp: time.Now(),
			}
			client.SendJSON(sessionReadyEvent)
		}

	case "start_game":
		logger.Info(ctx, "Client %s requested to start game", client.ID())
		if err := ws.gameSessionService.StartGame(ctx, client.ID()); err != nil {
			logger.Error(ctx, "Failed to start game for client %s: %v", client.ID(), err)
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Failed to start game"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
		} else {
			// Start the game engine
			session, _ := ws.gameSessionService.GetSession(ctx, client.ID())
			if session != nil && session.GameID != nil {
				if err := ws.gameEngine.StartGame(ctx, *session.GameID); err != nil {
					logger.Error(ctx, "Failed to start game engine for client %s: %v", client.ID(), err)
				} else {
					logger.Info(ctx, "Started game engine for client %s", client.ID())
				}
			}
			
			// Get the updated session state with all details
			session, _ = ws.gameSessionService.GetSession(ctx, client.ID())
			
			// Send game started confirmation with full game state
			gameStartedEvent := &entity.GameEvent{
				Type:      "game_started",
				Data:      session,
				Timestamp: time.Now(),
			}
			client.SendJSON(gameStartedEvent)
			
			// Also send the current pods and nodes state
			if session != nil {
				ws.sendCompleteGameState(ctx, client.ID(), session.ClusterID)
			}
		}

	case "stop_game":
		logger.Info(ctx, "Client %s requested to stop game", client.ID())
		if err := ws.gameSessionService.StopGame(ctx, client.ID()); err != nil {
			logger.Error(ctx, "Failed to stop game for client %s: %v", client.ID(), err)
		} else {
			// Send game stopped confirmation
			session, _ := ws.gameSessionService.GetSession(ctx, client.ID())
			gameStoppedEvent := &entity.GameEvent{
				Type:      "game_stopped",
				Data:      session,
				Timestamp: time.Now(),
			}
			client.SendJSON(gameStoppedEvent)
		}

	case "restart_game":
		logger.Info(ctx, "Client %s requested to restart game", client.ID())
		if err := ws.gameSessionService.RestartGame(ctx, client.ID()); err != nil {
			logger.Error(ctx, "Failed to restart game for client %s: %v", client.ID(), err)
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Failed to restart game"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
		} else {
			// Send game restarted confirmation
			session, _ := ws.gameSessionService.GetSession(ctx, client.ID())
			gameRestartedEvent := &entity.GameEvent{
				Type:      "game_restarted",
				Data:      session,
				Timestamp: time.Now(),
			}
			client.SendJSON(gameRestartedEvent)
		}

	case "get_session_state":
		logger.Info(ctx, "Client %s requested session state", client.ID())
		session, err := ws.gameSessionService.GetSessionState(ctx, client.ID())
		if err != nil {
			logger.Error(ctx, "Failed to get session state for client %s: %v", client.ID(), err)
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Failed to get session state"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
		} else {
			sessionStateEvent := &entity.GameEvent{
				Type:      "session_state",
				Data:      session,
				Timestamp: time.Now(),
			}
			client.SendJSON(sessionStateEvent)
		}

	case "schedule_pod":
		logger.Info(ctx, "Client %s requested pod scheduling", client.ID())
		
		// Parse scheduling request
		data, ok := message.Data.(map[string]interface{})
		if !ok {
			logger.Error(ctx, "Invalid schedule_pod data format from client %s", client.ID())
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Invalid schedule_pod data format"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
			return
		}

		podID, ok := data["pod_id"].(string)
		if !ok {
			logger.Error(ctx, "Missing pod_id in schedule_pod request from client %s", client.ID())
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Missing pod_id"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
			return
		}

		nodeID, ok := data["node_id"].(string)
		if !ok {
			logger.Error(ctx, "Missing node_id in schedule_pod request from client %s", client.ID())
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Missing node_id"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
			return
		}

		// Call game use case to schedule pod
		if ws.gameUseCase != nil {
			if err := ws.gameUseCase.SchedulePodToNode(ctx, podID, nodeID, client.ID()); err != nil {
				logger.Error(ctx, "Failed to schedule pod %s to node %s for client %s: %v", podID, nodeID, client.ID(), err)
				errorEvent := &entity.GameEvent{
					Type:      "error",
					Data:      map[string]string{"message": fmt.Sprintf("Failed to schedule pod: %v", err)},
					Timestamp: time.Now(),
				}
				client.SendJSON(errorEvent)
			} else {
				logger.Info(ctx, "Successfully scheduled pod %s to node %s for client %s", podID, nodeID, client.ID())
				successEvent := &entity.GameEvent{
					Type:      "pod_scheduled",
					Data:      map[string]string{"pod_id": podID, "node_id": nodeID},
					Timestamp: time.Now(),
				}
				client.SendJSON(successEvent)
			}
		} else {
			logger.Error(ctx, "Game use case not available for client %s", client.ID())
			errorEvent := &entity.GameEvent{
				Type:      "error",
				Data:      map[string]string{"message": "Game use case not available"},
				Timestamp: time.Now(),
			}
			client.SendJSON(errorEvent)
		}

	default:
		logger.Warn(ctx, "Unknown message type %s from client %s", message.Type, client.ID())
	}
}

// ServeHTTP upgrades HTTP connections to WebSocket and registers the client.
// This method implements http.Handler interface for easy integration with HTTP servers.
func (ws *webSocketService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(context.Background(), "Failed to upgrade connection: %v", err)
		return
	}

	// Generate client ID (in real implementation, you might get this from session/auth)
	clientID := fmt.Sprintf("client_%d", len(ws.clients)+1)

	ctx := context.Background()
	ws.RegisterClient(ctx, clientID, conn)
}

// sendCompleteGameState sends the complete game state (pods, nodes, game info) to a specific client
func (ws *webSocketService) sendCompleteGameState(ctx context.Context, clientID, clusterID string) {
	// Send initial game state notification
	stateEvent := &entity.GameEvent{
		Type:      "game_state_init",
		Data:      map[string]string{
			"cluster_id": clusterID,
			"message": "Game initialized - pods and nodes created",
		},
		Timestamp: time.Now(),
	}
	ws.SendEventToClient(ctx, clientID, stateEvent)
}

// runTimerUpdates runs a background goroutine that sends timer updates to active game clients
func (ws *webSocketService) runTimerUpdates() {
	ticker := time.NewTicker(1 * time.Second) // Update every second
	defer ticker.Stop()
	
	ctx := context.Background()
	
	for range ticker.C {
		ws.mu.RLock()
		clients := make(map[string]Client)
		for k, v := range ws.clients {
			clients[k] = v
		}
		ws.mu.RUnlock()
		
		// For each connected client, check if they have an active game
		for clientID := range clients {
			session, err := ws.gameSessionService.GetSession(ctx, clientID)
			if err != nil || session == nil || session.GameID == nil {
				continue
			}
			
			// Get the current game state and send timer update
			// Note: We can't access game repository directly from here
			// For now, send a simple timer tick event
			timerEvent := &entity.GameEvent{
				Type: "timer_tick",
				Data: map[string]interface{}{
					"session_id": session.SessionID,
					"game_id": *session.GameID,
					"timestamp": time.Now().Unix(),
				},
				Timestamp: time.Now(),
			}
			
			ws.SendEventToClient(ctx, clientID, timerEvent)
		}
	}
}
