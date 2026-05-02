import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.PLAYWRIGHT_DASHBOARD_PORT || 4310);
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  fullyParallel: false,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium-desktop",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 1000 } },
    },
    {
      name: "chromium-mobile",
      use: { ...devices["Pixel 7"] },
    },
  ],
  webServer: {
    command: [
      "DISABLE_AUTH=true",
      "NUXT_DEVTOOLS_ENABLED=false",
      "NUXT_PUBLIC_API_HOST=http://api.localhost",
      "NUXT_API_HOST_INTERNAL=http://127.0.0.1:9",
      `pnpm --filter @obiente/dashboard run dev -- --host 127.0.0.1 --port ${port}`,
    ].join(" "),
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
