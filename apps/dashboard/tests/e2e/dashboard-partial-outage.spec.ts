import { expect, test } from "@playwright/test";

test.beforeEach(async ({ context }) => {
  await context.addCookies([
    {
      name: "obiente_selected_org_id",
      value: "org-a",
      domain: "127.0.0.1",
      path: "/",
    },
  ]);
});

for (const outage of ["ListGameServers", "ListDatabases", "ListVPS"]) {
  test(`dashboard retains healthy data during ${outage} outage and recovers`, async ({
    page,
    isMobile,
  }, testInfo) => {
    let failed = true;
    await page.route(`**/${outage}`, (route) =>
      failed
        ? route.fulfill({
            status: 503,
            contentType: "application/json",
            body: JSON.stringify({
              code: "unavailable",
              message: "Synthetic service outage",
            }),
          })
        : route.continue()
    );
    await page.goto("/deployments");
    await page.waitForFunction(
      () =>
        (document.getElementById("__nuxt") as any)?.__vue_app__?.config
          .globalProperties.$nuxt?.isHydrating === false
    );
    if (isMobile)
      await page
        .getByRole("button", { name: "Toggle navigation menu", exact: true })
        .click();
    await page.locator('a[href="/dashboard"]:visible').first().click();
    await expect(
      page.getByText("Example deployment", { exact: true })
    ).toBeVisible();
    const cards = page.locator(".app-stat-card");
    const label =
      outage === "ListGameServers"
        ? "Game Servers"
        : outage === "ListDatabases"
        ? "Databases"
        : "VPS Instances";
    await expect(
      cards.filter({ has: page.getByText(label, { exact: true }) })
    ).toContainText("Unavailable");
    await expect(
      cards.filter({
        has: page.getByText("Active Deployments", { exact: true }),
      })
    ).toContainText("1");
    await expect(cards.getByText("Unavailable", { exact: true })).toHaveCount(
      1
    );
    await page.screenshot({
      path: testInfo.outputPath("partial-outage.png"),
      fullPage: true,
    });
    failed = false;
    await page.getByRole("button", { name: "Refresh", exact: true }).click();
    await expect(cards.getByText("Unavailable", { exact: true })).toHaveCount(
      0
    );
    await expect(
      page.getByText("Example deployment", { exact: true })
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => performance.getEntriesByType("navigation").length
      )
    ).toBe(1);
  });
}
