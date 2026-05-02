import { execSync } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";
import { expect, test, type Page } from "@playwright/test";

const dynamicSegmentSamples: Record<string, string> = {
  id: "e2e-resource",
  buildId: "e2e-build",
  orgId: "e2e-org",
  userId: "e2e-user",
  vpsId: "e2e-vps",
};

const ignoredConsoleFragments = [
  "Failed to fetch public config",
  "Config fetch failed",
  "No access token available",
  "Failed to get current user",
  "Silent auth failed",
  "favicon.ico",
  "An iframe which has both allow-scripts and allow-same-origin",
];

function pageFileToRoute(filePath: string) {
  const pageRoot = "apps/dashboard/app/pages/";
  let routePath = filePath.slice(pageRoot.length).replace(/\.vue$/, "");

  routePath = routePath
    .split("/")
    .filter((segment) => segment !== "index")
    .map((segment) => {
      const dynamic = segment.match(/^\[(.+)]$/);
      if (dynamic) return dynamicSegmentSamples[dynamic[1] || ""] || `e2e-${dynamic[1]}`;
      return segment;
    })
    .join("/");

  return `/${routePath}`.replace(/\/+$/, "") || "/";
}

function trackedDashboardRoutes() {
  const repoRoot = path.resolve(__dirname, "../../../..");
  const output = execSync("git ls-files 'apps/dashboard/app/pages/**/*.vue' 'apps/dashboard/app/pages/*.vue'", {
    cwd: repoRoot,
    encoding: "utf8",
  });

  return [...new Set(output.trim().split("\n"))]
    .filter(Boolean)
    .filter((filePath) => existsSync(path.join(repoRoot, filePath)))
    .filter((filePath) =>
      filePath
        .slice("apps/dashboard/app/pages/".length)
        .split("/")
        .every((segment) => !segment.startsWith("_"))
    )
    .map((filePath) => ({ filePath, route: pageFileToRoute(filePath) }))
    .sort((a, b) => a.route.localeCompare(b.route));
}

async function installApiMocks(page: Page) {
  await page.context().addCookies([
    {
      name: "obiente_selected_org_id",
      value: "org-a",
      domain: "127.0.0.1",
      path: "/",
      sameSite: "Lax",
    },
  ]);

  await page.route("**/auth/session", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        user: {
          sub: "dev-user",
          email: "dev@example.com",
          name: "Development User",
          preferred_username: "dev",
        },
      }),
    });
  });

  await page.route("**/auth/token", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ accessToken: "e2e-token", expiresIn: 3600 }),
    });
  });

  await page.route("**/auth/refresh", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ accessToken: "e2e-token", expiresIn: 3600 }),
    });
  });

  await page.route("http://api.localhost/**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: "{}",
    });
  });
}

for (const { filePath, route } of trackedDashboardRoutes()) {
  test(`route smoke: ${route} (${filePath})`, async ({ page }) => {
    const severeConsole: string[] = [];

    page.on("console", (message) => {
      if (!["error", "warning"].includes(message.type())) return;
      const text = message.text();
      if (ignoredConsoleFragments.some((fragment) => text.includes(fragment))) return;
      severeConsole.push(`${message.type()}: ${text}`);
    });

    page.on("pageerror", (error) => {
      severeConsole.push(`pageerror: ${error.message}`);
    });

    await installApiMocks(page);

    const navigationStartedAt = Date.now();
    const response = await page.goto(route, { waitUntil: "domcontentloaded" });
    const navigationMs = Date.now() - navigationStartedAt;
    expect(response?.status(), `${route} should not return an HTTP error`).toBeLessThan(500);
    expect(navigationMs, `${route} should render before backend request timeouts`).toBeLessThan(15_000);

    const main = page.locator("main, body").first();
    await expect(main, `${route} should render visible content`).toBeVisible();

    const visibleText = (await main.innerText()).trim();
    expect(visibleText.length, `${route} should not render a blank page`).toBeGreaterThan(20);
    expect(visibleText, `${route} should not render Nuxt fatal errors`).not.toMatch(
      /(?:^|\n)\s*(?:500|internal server error|page not found)(?:\n|$)/i
    );

    const hasHorizontalOverflow = await page.evaluate(() => {
      const root = document.documentElement;
      const body = document.body;
      return Math.max(root.scrollWidth, body.scrollWidth) > root.clientWidth + 2;
    });
    expect(hasHorizontalOverflow, `${route} should not create viewport-level horizontal overflow`).toBe(false);

    expect(severeConsole, `${route} should not emit severe console errors`).toEqual([]);
  });
}
