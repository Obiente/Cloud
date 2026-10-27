import { test, expect, type Page } from "@playwright/test";
async function openAdminPage(page: Page, path: string) {
  await page.goto(path);
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
}
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
test.afterEach(async ({ page }, info) => {
  await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
});

test("database permission can be selected by keyboard and saved in a custom role", async ({
  page,
}) => {
  await page.route("**/CreateRole", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        role: {
          id: "new-role",
          name: "Example Reader",
          permissionsJson: '["database.read"]',
        },
      }),
    })
  );
  await openAdminPage(page, "/admin/roles");
  await expect(
    page.getByRole("combobox").filter({ hasText: "Example Alpha" }).last()
  ).toBeVisible();
  await page.getByLabel("Role Name", { exact: true }).fill("Example Reader");
  const permission = page.getByRole("checkbox", {
    name: "Database",
    exact: true,
  });
  await expect(permission).toBeVisible();
  await permission.focus();
  await page.keyboard.press("Space");
  await expect(permission).toBeChecked();
  const request = page.waitForRequest((r) => r.url().endsWith("/CreateRole"));
  const save = page.getByRole("button", { name: "Create Role", exact: true });
  await save.focus();
  await page.keyboard.press("Enter");
  expect(JSON.parse((await request).postDataJSON().permissionsJson)).toEqual([
    "database.read",
  ]);
});

test("webhook organization action retains canonical organization context", async ({
  page,
}) => {
  await openAdminPage(page, "/superadmin/webhook-events");
  const organization = page
    .getByRole("button", { name: "View Org", exact: true })
    .first();
  await organization.focus();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/superadmin\/organizations\/org-a$/);
});

test("admin navigation is keyboard accessible and fits enlarged layout", async ({
  page,
}) => {
  await openAdminPage(page, "/admin/bindings");
  const quotas = page.getByRole("link", { name: "Quotas", exact: true });
  await quotas.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Organization Quotas", exact: true })
  ).toBeVisible();
  await page.evaluate(() => {
    document.documentElement.style.zoom = "2";
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth + 2
    )
  ).toBe(true);
  const breadcrumb = page.getByRole("navigation", { name: "Breadcrumb" });
  const header = page.locator("header.app-topbar");
  const crumbBox = await breadcrumb.boundingBox();
  const headerBox = await header.boundingBox();
  expect(crumbBox!.width / headerBox!.width).toBeGreaterThan(0.7);
  expect(
    await breadcrumb.evaluate(
      (el) => el.clientHeight / parseFloat(getComputedStyle(el).lineHeight)
    )
  ).toBeLessThan(5);
  await expect(
    page.getByLabel("Deployments Max", { exact: true })
  ).toBeVisible();
});

test("mobile admin tables retain the first column inside the card", async ({
  page,
}) => {
  await page.route("**/GetOrgLeases", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        leases:
          route.request().postDataJSON().organizationId === "org-a"
            ? [
                {
                  vpsId: "example-unknown",
                  organizationId: "org-a",
                  ipAddress: "192.0.2.10",
                  macAddress: "02:00:00:00:00:10",
                },
              ]
            : [],
      }),
    })
  );
  await openAdminPage(page, "/superadmin/vps");
  await page.getByRole("tab", { name: "DHCP Leases" }).click();
  const table = page.getByRole("table");
  await expect(table.getByText("192.0.2.10", { exact: true })).toBeVisible();
  const first = await table.locator("th").first().boundingBox();
  const card = await table
    .locator("xpath=ancestor::*[contains(@class,'oui-card-base')][1]")
    .boundingBox();
  expect(first!.x).toBeGreaterThanOrEqual(card!.x);
});
