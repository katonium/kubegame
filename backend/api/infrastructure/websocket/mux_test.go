package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/katonium/kubegame/backend/generated"
)

// TestMessageMux tests the MessageMux implementation.
func TestMessageMux(t *testing.T) {
	ctx := context.Background()
	gameUseCase := &mockGameUseCase{}
	messageMux := NewMessageMux(gameUseCase)

	t.Run("OnConnect", func(t *testing.T) {
		clientID := "test-client"
		responseBuffer := &bytes.Buffer{}

		err := messageMux.OnConnect(ctx, clientID, responseBuffer)
		if err != nil {
			t.Fatalf("OnConnect failed: %v", err)
		}

		// Check that we got a valid JSON response
		var response generated.SessionReadyEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Type != "session_ready" {
			t.Errorf("Expected type 'session_ready', got '%s'", response.Type)
		}
	})

	t.Run("HandlePingMessage", func(t *testing.T) {
		clientID := "test-client"
		responseBuffer := &bytes.Buffer{}

		pingMessage := generated.PingMessage{
			Type:      "ping",
			Timestamp: time.Now(),
		}

		messageBytes, _ := json.Marshal(pingMessage)
		err := messageMux.HandleMessage(ctx, clientID, messageBytes, responseBuffer)
		if err != nil {
			t.Fatalf("HandleMessage failed: %v", err)
		}

		// Check that we got a pong response
		var response generated.PongMessage
		if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Type != "pong" {
			t.Errorf("Expected type 'pong', got '%s'", response.Type)
		}
	})

	t.Run("HandleStartGameMessage", func(t *testing.T) {
		clientID := "test-client"
		responseBuffer := &bytes.Buffer{}

		startGameMessage := generated.StartGameMessage{
			Type:      "start_game",
			Timestamp: time.Now(),
		}

		messageBytes, _ := json.Marshal(startGameMessage)
		err := messageMux.HandleMessage(ctx, clientID, messageBytes, responseBuffer)
		if err != nil {
			t.Fatalf("HandleMessage failed: %v", err)
		}

		// Check that we got a game started response
		var response generated.GameStartedEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Type != "game_started" {
			t.Errorf("Expected type 'game_started', got '%s'", response.Type)
		}

		if response.Data.TimeLeft != 300 {
			t.Errorf("Expected TimeLeft 300, got %d", response.Data.TimeLeft)
		}
	})

	t.Run("HandleSchedulePodMessage", func(t *testing.T) {
		clientID := "test-client"
		responseBuffer := &bytes.Buffer{}

		schedulePodMessage := generated.SchedulePodMessage{
			Type:      "schedule_pod",
			Timestamp: time.Now(),
			Data: struct {
				NodeId string `json:"node_id"`
				PodId  string `json:"pod_id"`
			}{
				PodId:  "test-pod",
				NodeId: "test-node",
			},
		}

		messageBytes, _ := json.Marshal(schedulePodMessage)
		err := messageMux.HandleMessage(ctx, clientID, messageBytes, responseBuffer)
		if err != nil {
			t.Fatalf("HandleMessage failed: %v", err)
		}

		// Check that we got a pod scheduled response
		var response generated.PodScheduledEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Type != "pod_scheduled" {
			t.Errorf("Expected type 'pod_scheduled', got '%s'", response.Type)
		}
	})

	t.Run("HandleInvalidMessage", func(t *testing.T) {
		clientID := "test-client"
		responseBuffer := &bytes.Buffer{}

		// Send invalid JSON
		invalidMessage := json.RawMessage(`{"type": "unknown_type", "timestamp": "2023-10-01T12:00:00Z"}`)
		err := messageMux.HandleMessage(ctx, clientID, invalidMessage, responseBuffer)
		if err != nil {
			t.Fatalf("HandleMessage failed: %v", err)
		}

		// Check that we got an error response
		var response generated.ErrorEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Type != "error" {
			t.Errorf("Expected type 'error', got '%s'", response.Type)
		}
	})

	t.Run("OnDisconnect", func(t *testing.T) {
		clientID := "test-client"

		err := messageMux.OnDisconnect(ctx, clientID)
		if err != nil {
			t.Fatalf("OnDisconnect failed: %v", err)
		}

		// OnDisconnect doesn't return any response, just check it doesn't error
	})
}

// TestMessageMuxIntegration tests the MessageMux with integration patterns.
func TestMessageMuxIntegration(t *testing.T) {
	ctx := context.Background()
	gameUseCase := &mockGameUseCase{}
	wsService := NewWebSocketServiceWithMessageMux(gameUseCase)

	t.Run("CompleteFlow", func(t *testing.T) {
		clientID := "integration-test-client"
		responseBuffer := &bytes.Buffer{}

		// 1. Client connects
		err := wsService.HandleClientConnection(ctx, clientID, responseBuffer)
		if err != nil {
			t.Fatalf("HandleClientConnection failed: %v", err)
		}

		// Verify connection response
		var sessionReady generated.SessionReadyEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &sessionReady); err != nil {
			t.Fatalf("Failed to unmarshal connection response: %v", err)
		}
		if sessionReady.Type != "session_ready" {
			t.Errorf("Expected session_ready event on connection")
		}

		// 2. Client sends ping
		responseBuffer.Reset()
		pingMsg := `{"type": "ping", "timestamp": "2023-10-01T12:00:00Z"}`
		err = wsService.HandleClientMessage(ctx, clientID, json.RawMessage(pingMsg), responseBuffer)
		if err != nil {
			t.Fatalf("HandleClientMessage (ping) failed: %v", err)
		}

		// Verify pong response
		var pongResponse generated.PongMessage
		if err := json.Unmarshal(responseBuffer.Bytes(), &pongResponse); err != nil {
			t.Fatalf("Failed to unmarshal pong response: %v", err)
		}
		if pongResponse.Type != "pong" {
			t.Errorf("Expected pong response to ping")
		}

		// 3. Client starts game
		responseBuffer.Reset()
		startGameMsg := `{"type": "start_game", "timestamp": "2023-10-01T12:00:01Z"}`
		err = wsService.HandleClientMessage(ctx, clientID, json.RawMessage(startGameMsg), responseBuffer)
		if err != nil {
			t.Fatalf("HandleClientMessage (start_game) failed: %v", err)
		}

		// Verify game started response
		var gameStarted generated.GameStartedEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &gameStarted); err != nil {
			t.Fatalf("Failed to unmarshal game started response: %v", err)
		}
		if gameStarted.Type != "game_started" {
			t.Errorf("Expected game_started response")
		}

		// 4. Client schedules pod
		responseBuffer.Reset()
		schedulePodMsg := `{"type": "schedule_pod", "data": {"pod_id": "pod-1", "node_id": "node-1"}, "timestamp": "2023-10-01T12:00:02Z"}`
		err = wsService.HandleClientMessage(ctx, clientID, json.RawMessage(schedulePodMsg), responseBuffer)
		if err != nil {
			t.Fatalf("HandleClientMessage (schedule_pod) failed: %v", err)
		}

		// Verify pod scheduled response
		var podScheduled generated.PodScheduledEvent
		if err := json.Unmarshal(responseBuffer.Bytes(), &podScheduled); err != nil {
			t.Fatalf("Failed to unmarshal pod scheduled response: %v", err)
		}
		if podScheduled.Type != "pod_scheduled" {
			t.Errorf("Expected pod_scheduled response")
		}

		// 5. Client disconnects
		err = wsService.HandleClientDisconnection(ctx, clientID)
		if err != nil {
			t.Fatalf("HandleClientDisconnection failed: %v", err)
		}
	})
}

// BenchmarkMessageMux benchmarks the MessageMux performance.
func BenchmarkMessageMux(b *testing.B) {
	ctx := context.Background()
	gameUseCase := &mockGameUseCase{}
	messageMux := NewMessageMux(gameUseCase)
	clientID := "benchmark-client"

	pingMessage := generated.PingMessage{
		Type:      "ping",
		Timestamp: time.Now(),
	}
	messageBytes, _ := json.Marshal(pingMessage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		responseBuffer := &bytes.Buffer{}
		messageMux.HandleMessage(ctx, clientID, messageBytes, responseBuffer)
	}
}
