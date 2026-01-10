import { test, expect } from "../../../fixtures/api";

test.describe("WebSocket Ping/Pong API", () => {
  test("returns pong message when client sends ping", async ({
    wsClient,
    apiAssertions,
    logger,
  }) => {
    let pongMessage: any;

    await test.step("Given a connected WebSocket client", async () => {
      await wsClient.connect();
      // Wait for session_ready to ensure connection is fully established
      await wsClient.waitForMessage("session_ready", 5000);
      logger.info("WebSocket connection established and session ready");
    });

    await test.step("When the client sends a ping message", async () => {
      await wsClient.send({
        type: "ping",
        timestamp: new Date(),
      });
      logger.info("Ping message sent");
    });

    await test.step("Then the server responds with a pong message", async () => {
      await test.step("With message type 'pong'", async () => {
        pongMessage = await wsClient.waitForMessage("pong", 5000);
        apiAssertions.hasMessageType(pongMessage, "pong");
        logger.info("Received pong message", { pongMessage });
      });

      await test.step("And the response includes a timestamp", async () => {
        expect(pongMessage.timestamp).toBeDefined();
        logger.info("Pong message has timestamp", {
          timestamp: pongMessage.timestamp,
        });
      });
    });

    // Cleanup
    await wsClient.disconnect();
  });

  test("handles multiple ping messages correctly", async ({
    wsClient,
    apiAssertions,
    logger,
  }) => {
    await test.step("Given a connected WebSocket client", async () => {
      await wsClient.connect();
      await wsClient.waitForMessage("session_ready", 5000);
      logger.info("WebSocket connection established");
    });

    await test.step("When the client sends multiple ping messages", async () => {
      // Send 3 ping messages
      await wsClient.send({ type: "ping", timestamp: new Date() });
      await wsClient.send({ type: "ping", timestamp: new Date() });
      await wsClient.send({ type: "ping", timestamp: new Date() });
      logger.info("Sent 3 ping messages");
    });

    await test.step("Then the server responds with exactly 3 pong messages", async () => {
      // Wait a bit for all pong messages to arrive
      await new Promise((resolve) => setTimeout(resolve, 1000));

      const pongMessages = wsClient.messages.filter(
        (msg) => msg.type === "pong"
      );

      expect(pongMessages.length).toBe(3);
      logger.info("Received exactly 3 pong messages", { count: pongMessages.length });
    });

    // Cleanup
    await wsClient.disconnect();
  });

  test("responds to pong message without error", async ({
    wsClient,
    logger,
  }) => {
    await test.step("Given a connected WebSocket client", async () => {
      await wsClient.connect();
      await wsClient.waitForMessage("session_ready", 5000);
      logger.info("WebSocket connection established");
    });

    await test.step("When the client sends a pong message", async () => {
      await wsClient.send({
        type: "pong",
        timestamp: new Date(),
      });
      logger.info("Pong message sent");
    });

    await test.step("Then the server accepts it without error", async () => {
      // Wait a bit to see if any error message arrives
      await new Promise((resolve) => setTimeout(resolve, 500));

      const errorMessages = wsClient.messages.filter(
        (msg) => msg.type === "error"
      );

      expect(errorMessages.length).toBe(0);
      logger.info("No error messages received");
    });

    // Cleanup
    await wsClient.disconnect();
  });
});
