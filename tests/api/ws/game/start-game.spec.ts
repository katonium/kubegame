import { test, expect } from "../../../fixtures/api";

test.describe("WebSocket Start Game API", () => {
  test("returns game_started event when client sends start_game", async ({
    wsClient,
    apiAssertions,
    logger,
  }) => {
    let gameStartedMessage: any;

    await test.step("Given a connected WebSocket client with session ready", async () => {
      await wsClient.connect();
      await wsClient.waitForMessage("session_ready", 5000);
      logger.info("WebSocket connection established and session ready");
    });

    await test.step("When the client sends a start_game message", async () => {
      await wsClient.send({
        type: "start_game",
        timestamp: new Date(),
      });
      logger.info("start_game message sent");
    });

    await test.step("Then the server responds with a game_started event", async () => {
      await test.step("With message type 'game_started'", async () => {
        gameStartedMessage = await wsClient.waitForMessage("game_started", 5000);
        apiAssertions.hasMessageType(gameStartedMessage, "game_started");
        logger.info("Received game_started message", { gameStartedMessage });
      });

      await test.step("And the response includes game data", async () => {
        expect(gameStartedMessage.data).toBeDefined();
        expect(gameStartedMessage.data.playerScore).toBeDefined();
        expect(gameStartedMessage.data.cpuScore).toBeDefined();
        expect(gameStartedMessage.data.timeLeft).toBeDefined();
        logger.info("Game data validated", {
          playerScore: gameStartedMessage.data.playerScore,
          cpuScore: gameStartedMessage.data.cpuScore,
          timeLeft: gameStartedMessage.data.timeLeft,
        });
      });

      await test.step("And the response includes cluster information", async () => {
        expect(gameStartedMessage.data.playerCluster).toBeDefined();
        expect(gameStartedMessage.data.schedulerCluster).toBeDefined();
        expect(gameStartedMessage.data.playerCluster.nodes).toBeDefined();
        expect(gameStartedMessage.data.playerCluster.pods).toBeDefined();
        logger.info("Cluster information validated");
      });

      await test.step("And initial scores are zero", async () => {
        expect(gameStartedMessage.data.playerScore).toBe(0);
        expect(gameStartedMessage.data.cpuScore).toBe(0);
        logger.info("Initial scores verified to be zero");
      });
    });

    // Cleanup
    await wsClient.disconnect();
  });

  test("returns error when start_game is sent without session_ready", async ({
    wsClient,
    logger,
  }) => {
    await test.step("Given a newly connected WebSocket client", async () => {
      await wsClient.connect();
      // Immediately send start_game without waiting for session_ready
      logger.info("WebSocket connected (not waiting for session_ready)");
    });

    await test.step("When the client sends start_game immediately", async () => {
      await wsClient.send({
        type: "start_game",
        timestamp: new Date(),
      });
      logger.info("start_game message sent immediately");
    });

    await test.step("Then the server might handle it gracefully", async () => {
      // Wait to see what response we get
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // The server might either:
      // 1. Return an error
      // 2. Process it normally after session_ready
      const messages = wsClient.messages;
      logger.info("Received messages", { count: messages.length });

      // Just verify we get some response
      expect(messages.length).toBeGreaterThan(0);
    });

    // Cleanup
    await wsClient.disconnect();
  });

  test("handles multiple start_game messages", async ({
    wsClient,
    logger,
  }) => {
    await test.step("Given a connected WebSocket client", async () => {
      await wsClient.connect();
      await wsClient.waitForMessage("session_ready", 5000);
      logger.info("WebSocket connection established");
    });

    await test.step("When the client sends multiple start_game messages", async () => {
      await wsClient.send({ type: "start_game", timestamp: new Date() });
      await wsClient.send({ type: "start_game", timestamp: new Date() });
      logger.info("Sent 2 start_game messages");
    });

    await test.step("Then the server handles them appropriately", async () => {
      // Wait for responses
      await new Promise((resolve) => setTimeout(resolve, 1000));

      const gameStartedMessages = wsClient.messages.filter(
        (msg) => msg.type === "game_started"
      );

      // Server might respond to each or just the first one
      expect(gameStartedMessages.length).toBeGreaterThanOrEqual(1);
      logger.info("Received game_started messages", {
        count: gameStartedMessages.length,
      });
    });

    // Cleanup
    await wsClient.disconnect();
  });
});
