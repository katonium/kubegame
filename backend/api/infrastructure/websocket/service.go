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
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(v)
}

// ReadJSON reads a JSON-encoded object from the connection.
func (w *wsConnection) ReadJSON(v interface{}) error {
	return w.conn.ReadJSON(v)
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
	clients    map[string]Client  // Connected clients indexed by ID
	broadcast  chan []byte        // Broadcast channel for messages to all clients
	register   chan Client        // Register channel for new clients
	unregister chan Client        // Unregister channel for disconnecting clients
	upgrader   websocket.Upgrader // WebSocket upgrader configuration
	mu         sync.RWMutex       // Mutex for thread-safe access to clients map
}

// NewWebSocketService creates a new WebSocket service instance.
// It initializes the service and starts the background hub goroutine.
func NewWebSocketService() service.WebSocketService {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	svc := &webSocketService{
		clients:    make(map[string]Client),
		broadcast:  make(chan []byte),
		register:   make(chan Client),
		unregister: make(chan Client),
		upgrader:   upgrader,
	}

	// Start the hub goroutine
	go svc.run()

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

	logger.Debug(ctx, "Broadcasting event: %s", event.Type)

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

	logger.Debug(ctx, "Sending event %s to client %s", event.Type, clientID)

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

		case client := <-ws.unregister:
			ws.mu.Lock()
			if _, ok := ws.clients[client.ID()]; ok {
				delete(ws.clients, client.ID())
				client.Close()
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
	logger.Debug(ctx, "Received message from client %s: %s", client.ID(), message.Type)

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
		// TODO: Handle game join logic

	case "schedule_pod":
		logger.Info(ctx, "Client %s requested pod scheduling", client.ID())
		// TODO: Handle pod scheduling request

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
