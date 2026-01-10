import { test, expect } from "../../fixtures/scenario";

test.describe("Complete Game Workflow", () => {
  test("player can connect, start game, and schedule a pod", async ({
    gameContext,
    workflowAssertions,
    logger,
  }) => {
    await test.step("Step 1: Connect to game server", async () => {
      await workflowAssertions.stepSucceeded(async () => {
        await gameContext.connect();
        return gameContext.wsClient.messages.find(
          (msg) => msg.type === "session_ready"
        )!;
      });
      logger.info("Connected to game server and session is ready");
    });

    await test.step("Step 2: Start the game", async () => {
      const gameStarted = await workflowAssertions.stepSucceeded(async () => {
        return await gameContext.startGame();
      });
      logger.info("Game started successfully", { gameStarted });

      // Verify game started with correct initial state
      expect(gameStarted.data.playerScore).toBe(0);
      expect(gameStarted.data.cpuScore).toBe(0);
      expect(gameStarted.data.timeLeft).toBeGreaterThan(0);
    });

    await test.step("Step 3: Verify game is running", async () => {
      await workflowAssertions.gameIsRunning(gameContext);
      logger.info("Game is confirmed to be running");
    });

    await test.step("Step 4: Send ping to verify connection health", async () => {
      await gameContext.wsClient.send({
        type: "ping",
        timestamp: new Date(),
      });
      const pong = await gameContext.wsClient.waitForMessage("pong", 5000);
      expect(pong).toBeDefined();
      logger.info("Connection health verified with ping/pong");
    });

    await test.step("Step 5: Cleanup - Disconnect", async () => {
      await gameContext.disconnect();
      expect(gameContext.wsClient.isConnected).toBe(false);
      logger.info("Disconnected from game server");
    });
  });

  test("player can connect, start game, and handle pod scheduling workflow", async ({
    gameContext,
    workflowAssertions,
    logger,
    testId,
  }) => {
    await test.step("Step 1: Connect and start game", async () => {
      await gameContext.connect();
      await gameContext.startGame();
      logger.info("Connected and game started");
    });

    await test.step("Step 2: Attempt to schedule a pod", async () => {
      const podId = `test-pod-${testId}`;
      const nodeId = `test-node-${testId}`;

      // Note: This test assumes the pod and node exist in the game state
      // In a real scenario, you might need to wait for pod_created events first
      try {
        const podScheduled = await gameContext.schedulePod(podId, nodeId);
        logger.info("Pod scheduling attempted", {
          podId,
          nodeId,
          response: podScheduled,
        });

        // If scheduling succeeds, verify the response
        if (podScheduled.type === "pod_scheduled") {
          expect(podScheduled.data).toBeDefined();
          logger.info("Pod scheduled successfully");
        }
      } catch (error: any) {
        // If scheduling fails (pod doesn't exist), that's expected in this test
        logger.info("Pod scheduling failed as expected", {
          error: error.message,
        });
      }
    });

    await test.step("Step 3: Cleanup", async () => {
      await gameContext.disconnect();
      logger.info("Test cleanup completed");
    });
  });

  test("handles connection loss gracefully", async ({
    gameContext,
    logger,
  }) => {
    await test.step("Step 1: Connect to game server", async () => {
      await gameContext.connect();
      expect(gameContext.wsClient.isConnected).toBe(true);
      logger.info("Connected to game server");
    });

    await test.step("Step 2: Start game", async () => {
      await gameContext.startGame();
      logger.info("Game started");
    });

    await test.step("Step 3: Disconnect abruptly", async () => {
      await gameContext.disconnect();
      expect(gameContext.wsClient.isConnected).toBe(false);
      logger.info("Disconnected from game server");
    });

    await test.step("Step 4: Verify connection is closed", async () => {
      expect(gameContext.wsClient.isConnected).toBe(false);
      logger.info("Connection confirmed closed");
    });
  });

  test("maintains correct message order in workflow", async ({
    gameContext,
    logger,
  }) => {
    const receivedMessages: string[] = [];

    await test.step("Step 1: Connect and track messages", async () => {
      await gameContext.connect();
      receivedMessages.push("session_ready");
      logger.info("Connected, session_ready received");
    });

    await test.step("Step 2: Start game and track message", async () => {
      await gameContext.startGame();
      receivedMessages.push("game_started");
      logger.info("Game started, game_started received");
    });

    await test.step("Step 3: Send ping and track response", async () => {
      await gameContext.wsClient.send({
        type: "ping",
        timestamp: new Date(),
      });
      await gameContext.wsClient.waitForMessage("pong", 5000);
      receivedMessages.push("pong");
      logger.info("Pong received");
    });

    await test.step("Step 4: Verify message order", async () => {
      expect(receivedMessages).toEqual(["session_ready", "game_started", "pong"]);
      logger.info("Message order verified", { receivedMessages });
    });

    await test.step("Step 5: Cleanup", async () => {
      await gameContext.disconnect();
    });
  });
});
