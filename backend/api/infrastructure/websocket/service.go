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
	"github.com/katonium/kubegame/backend/generated"
	"github.com/katonium/kubegame/backend/usecase"
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
	Send(ctx context.Context, data []byte) error
	SendJSON(ctx context.Context, v interface{}) error
	Close() error
}

// TODO: move notifier to its own file
type notifier struct {
	cli Client
}

func NewNotifier(cli Client) usecase.Notifier {
	return &notifier{cli: cli}
}

func (n *notifier) NotifyGameUpdate(ctx context.Context, event generated.GameUpdateEvent) error {
	return n.cli.SendJSON(ctx, event)
}
func (n *notifier) NotifyPodCreated(ctx context.Context, event generated.PodCreatedEvent) error {
	return n.cli.SendJSON(ctx, event)
}
func (n *notifier) NotifyPodScheduled(ctx context.Context, event generated.PodScheduledEvent) error {
	return n.cli.SendJSON(ctx, event)
}
func (n *notifier) NotifyGameOver(ctx context.Context, event generated.GameOverEvent) error {
	return n.cli.SendJSON(ctx, event)
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
func (c *client) Send(ctx context.Context, data []byte) error {
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
func (c *client) SendJSON(ctx context.Context, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Send(ctx, data)
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
	handler    MessageHandler     // Message handler for processing incoming messages
}

// NewWebSocketService creates a new WebSocket service instance.
// It initializes the service and starts the background hub goroutine.
func NewWebSocketService(handler MessageHandler) service.WebSocketService {
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
		handler:    handler,
		mu:         sync.RWMutex{},
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

	return client.SendJSON(ctx, event)
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
			// Register new client
			ws.registerClient(context.Background(), client)

			// Create session and cluster for new client
			ctx := context.Background()
			err := ws.handler.OnConnect(ctx, client)
			if err != nil {
				logger.Error(ctx, "Failed to create session for client %s: %v", client.ID(), err)
			}
		case client := <-ws.unregister:
			// Unregister client
			ws.unregisterClient(context.Background(), client)
			ws.handler.OnDisconnect(context.Background(), client.ID())
		case message := <-ws.broadcast:
			// Broadcast message to all clients
			ws.broadcastMessage(context.Background(), message)
		}
	}
}

func (ws *webSocketService) registerClient(ctx context.Context, client Client) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.clients[client.ID()] = client
	logger.Info(ctx, "Client %s registered", client.ID())
}

func (ws *webSocketService) unregisterClient(ctx context.Context, client Client) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if _, ok := ws.clients[client.ID()]; ok {
		delete(ws.clients, client.ID())
		client.Close()
		logger.Info(ctx, "Client %s unregistered", client.ID())
	}
}

func (ws *webSocketService) broadcastMessage(ctx context.Context, message []byte) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for clientID, client := range ws.clients {
		if err := client.Send(ctx, message); err != nil {
			// Remove client if send fails
			delete(ws.clients, clientID)
			client.Close()
			logger.Error(ctx, "Failed to send message to client %s: %v", clientID, err)
		}
	}
}

// readPump handles reading messages from the WebSocket connection.
func (ws *webSocketService) readPump(ctx context.Context, cli Client) {
	defer func() {
		ws.unregister <- cli
	}()

	for {
		// parse JSON raw message
		var buf = json.RawMessage{}
		// Use the client's connection to read JSON directly
		if clientImpl, ok := cli.(*client); ok {
			if err := clientImpl.connection.ReadJSON(&buf); err != nil {
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Error(ctx, "WebSocket read error for client %s: %v", cli.ID(), err)
				}
				break
			}

			// Process the incoming message
			ws.handler.HandleMessage(ctx, buf, cli)
		} else {
			break
		}
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
