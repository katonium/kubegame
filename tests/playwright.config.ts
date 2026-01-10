import { defineConfig, devices } from "@playwright/test";
import * as path from "path";

/**
 * Read environment variables from .env file.
 */
import * as dotenv from "dotenv";
dotenv.config({ path: path.resolve(__dirname, ".env") });

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: "./",
  testMatch: "**/*.spec.ts",

  // Run tests in files in parallel
  fullyParallel: true,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,

  // Retry on CI only
  retries: process.env.CI ? 2 : 0,

  // Number of workers - use fewer workers for WebSocket tests to avoid overwhelming server
  workers: process.env.CI ? 2 : 4,

  // Reporter to use
  reporter: [
    ["html", { outputFolder: "test-results/html" }],
    ["json", { outputFile: "test-results/results.json" }],
    ["list"],
  ],

  // Shared settings for all the projects below
  use: {
    // Base URL to use in actions like `await page.goto('/')`
    baseURL: process.env.API_BASE_URL || "http://localhost:8080",

    // Collect trace when retrying the failed test
    trace: "on-first-retry",

    // Screenshot on failure
    screenshot: "only-on-failure",

    // Video on failure
    video: "retain-on-failure",
  },

  // Configure projects for different test types
  projects: [
    {
      name: "api-tests",
      testMatch: "api/**/*.spec.ts",
      use: {
        ...devices["Desktop Chrome"],
      },
    },
    {
      name: "scenario-tests",
      testMatch: "scenario/**/*.spec.ts",
      use: {
        ...devices["Desktop Chrome"],
      },
    },
  ],

  // Run local server before starting tests (optional)
  // Uncomment if you want Playwright to start the server
  // webServer: {
  //   command: 'cd ../backend && go run ./api',
  //   url: 'http://localhost:8080',
  //   reuseExistingServer: !process.env.CI,
  //   timeout: 120 * 1000,
  // },
});
