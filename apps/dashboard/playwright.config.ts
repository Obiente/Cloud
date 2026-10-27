import { tmpdir } from "node:os";
import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.PLAYWRIGHT_DASHBOARD_PORT || 4310);
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  outputDir:
    process.env.PLAYWRIGHT_OUTPUT_DIR ||
    path.join(tmpdir(), "obiente-cloud-playwright", "results"),
  testDir: "./tests/e2e",
  testIgnore:
    process.env.PLAYWRIGHT_AUTH_TESTS === "true"
      ? []
      : ["**/auth-session.spec.ts"],
  grep: process.env.PLAYWRIGHT_GREP
    ? new RegExp(process.env.PLAYWRIGHT_GREP)
    : undefined,
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI
    ? [
        ["list"],
        [
          "html",
          {
            open: "never",
            outputFolder: path.join(
              process.env.PLAYWRIGHT_OUTPUT_DIR ||
                path.join(tmpdir(), "obiente-cloud-playwright"),
              "report"
            ),
          },
        ],
      ]
    : "list",
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium-desktop",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 1000 },
      },
    },
    {
      name: "chromium-mobile",
      use: { ...devices["Pixel 7"] },
    },
  ],
  webServer: [
    {
      command: "node tests/e2e/fixture-server.mjs",
      url: "http://127.0.0.1:4311",
      reuseExistingServer: !process.env.CI,
    },
    {
      command:
        process.env.PLAYWRIGHT_USE_BUILD === "true"
          ? "node .output/server/index.mjs"
          : `corepack pnpm exec nuxt dev --host 127.0.0.1 --port ${port}`,
      env: {
        DISABLE_AUTH:
          process.env.PLAYWRIGHT_AUTH_TESTS === "true" ? "false" : "true",
        NUXT_PUBLIC_OIDC_BASE: "http://127.0.0.1:4311",
        NUXT_PUBLIC_OIDC_CLIENT_ID: "synthetic-client",
        NUXT_PUBLIC_REQUEST_HOST: baseURL,
        NUXT_SESSION_PASSWORD:
          "synthetic-test-session-password-at-least-32-characters",
        NUXT_DEVTOOLS_ENABLED: "false",
        NUXT_PUBLIC_API_HOST:
          process.env.PLAYWRIGHT_API_HOST || "http://127.0.0.1:4311",
        NUXT_API_HOST_INTERNAL:
          process.env.PLAYWRIGHT_API_HOST || "http://127.0.0.1:4311",
        HOST: "127.0.0.1",
        PORT: String(port),
      },
      url: baseURL,
      reuseExistingServer: !process.env.CI,
      timeout: process.env.PLAYWRIGHT_USE_BUILD === "true" ? 120_000 : 300_000,
    },
  ],
});
