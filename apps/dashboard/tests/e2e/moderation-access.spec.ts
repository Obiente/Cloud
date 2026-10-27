import { expect, test } from "@playwright/test";

test.beforeEach(async ({ context }) => {
  await context.addCookies([{ name: "obiente_selected_org_id", value: "org-a", domain: "127.0.0.1", path: "/" }]);
});

for (const scope of ["account", "organization", "permission"]) {
  test(`moderation ${scope} preserves the correct dashboard access`, async ({ page }, testInfo) => {
    await page.goto("/dashboard");
    await expect(page.getByText("Example deployment", { exact: true })).toBeVisible();
    await page.route("**/ListGameServers", (route) => route.fulfill({
      status: 403,
      contentType: "application/json",
      headers: {
        "access-control-allow-origin": "http://127.0.0.1:4310",
        "access-control-allow-credentials": "true",
        "access-control-expose-headers": "X-Obiente-Restriction, X-Obiente-Organization-ID",
        ...(scope === "permission" ? {} : {
          "X-Obiente-Restriction": scope,
          "X-Obiente-Organization-ID": "org-a",
        }),
      },
      body: JSON.stringify({ code: "permission_denied", message: "Access restricted" }),
    }));
    const denial = page.waitForResponse((response) => response.url().endsWith("/ListGameServers") && response.status() === 403);
    await page.getByRole("button", { name: "Refresh", exact: true }).click();
    await denial;
    if (scope === "permission") {
      await expect(page.getByText("Example deployment", { exact: true })).toBeVisible();
      await expect(page.getByRole("heading", { name: /restricted/ })).toHaveCount(0);
      return;
    }
    await expect(page.getByRole("heading", { name: scope === "account" ? "Account restricted" : "Organization restricted" })).toBeVisible();
    await expect(page.getByText("Example deployment", { exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Sign out", exact: true })).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath(`${scope}-restricted.png`), fullPage: true });
    if (scope === "organization") {
      await page.getByRole("button", { name: "Choose another organization" }).click();
      await expect(page).toHaveURL(/\/organizations$/);
    }
  });
}
