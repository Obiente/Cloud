import { test, expect } from "@playwright/test";

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

test("managed backup creation, restore and deletion track durable operations", async ({
  page,
}, info) => {
  test.setTimeout(process.env.PLAYWRIGHT_USE_BUILD === "true" ? 60000 : 120000);
  let state = "empty",
    polls = 0;
  const backup: any = {
    id: "00000000-0000-4000-8000-000000000001",
    databaseId: "db-example",
    name: "Example backup",
    databaseName: "primary_db",
    status: "BACKUP_COMPLETED",
    sizeBytes: "4096",
    createdAt: "2026-01-01T12:00:00Z",
    completedAt: "2026-01-01T12:01:00Z",
  };
  await page.route("**/ListBackups", (route) => {
    polls++;
    if (state === "creating" && polls >= 2) state = "ready";
    if (state === "restoring" && polls >= 2) {
      state = "ready";
      backup.restoreStatus = "completed";
    }
    if (state === "deleting" && polls >= 2) state = "empty";
    const rows =
      state === "empty"
        ? []
        : [
            {
              ...backup,
              status:
                state === "creating"
                  ? "BACKUP_CREATING"
                  : state === "deleting"
                  ? "BACKUP_DELETING"
                  : "BACKUP_COMPLETED",
            },
          ];
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        backups: rows,
        managedAvailable: true,
        retainedCount: 10,
        pagination: { page: 1, perPage: 50, total: rows.length },
      }),
    });
  });
  await page.route("**/CreateBackup", (route) => {
    state = "creating";
    polls = 0;
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        backup: { ...backup, status: "BACKUP_CREATING" },
      }),
    });
  });
  await page.route("**/RestoreBackup", (route) => {
    const input = route.request().postDataJSON();
    expect(input.targetDatabaseName).toBe("restored_example");
    expect(input.organizationId).toBe("org-a");
    state = "restoring";
    polls = 0;
    backup.restoreStatus = "queued";
    backup.restoreTarget = input.targetDatabaseName;
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ backup, message: "Restore queued" }),
    });
  });
  await page.route("**/DeleteBackup", (route) => {
    state = "deleting";
    polls = 0;
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      body: '{"success":true}',
    });
  });
  await page.goto("/databases/db-example");
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
  await page.getByRole("tab", { name: "Backups", exact: true }).click();
  await expect(
    page.getByText("No backups yet.", { exact: true })
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Create backup", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Restore backup Example backup",
      exact: true,
    })
  ).toBeEnabled({ timeout: 15000 });
  await page
    .getByRole("button", { name: "Restore backup Example backup", exact: true })
    .click();
  await page.screenshot({
    path: info.outputPath("restore-dialog.png"),
    fullPage: true,
  });
  const target = page.getByLabel("New database name");
  await target.fill("primary_db");
  await expect(
    page.getByRole("button", { name: "Restore to new database", exact: true })
  ).toBeDisabled();
  await target.fill("restored_example");
  await page
    .getByRole("button", { name: "Restore to new database", exact: true })
    .click();
  await expect(
    page.getByText("Restored to restored_example.", { exact: false })
  ).toBeVisible({ timeout: 15000 });
  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain('Delete backup "Example backup"');
    await dialog.accept();
  });
  await page
    .getByRole("button", { name: "Delete backup Example backup", exact: true })
    .click();
  await expect(page.getByText("No backups yet.", { exact: true })).toBeVisible({
    timeout: 15000,
  });
});

test("backup availability recovers after a request failure without reloading", async ({
  page,
}) => {
  let fail = true;
  await page.route("**/ListBackups", (route) =>
    route.fulfill({
      status: fail ? 503 : 200,
      contentType: "application/json",
      body: fail
        ? JSON.stringify({
            code: "unavailable",
            message: "Temporary storage outage",
          })
        : JSON.stringify({
            backups: [],
            managedAvailable: true,
            retainedCount: 10,
            pagination: { total: 0 },
          }),
    })
  );
  await page.goto("/databases/db-example");
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
  await page.getByRole("tab", { name: "Backups", exact: true }).click();
  await expect(
    page.getByText("Could not refresh backups", { exact: true })
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Create backup", exact: true })
  ).toHaveCount(0);
  fail = false;
  await expect(
    page.getByRole("button", { name: "Create backup", exact: true })
  ).toBeEnabled({ timeout: 15000 });
  await expect(
    page.getByText("Could not refresh backups", { exact: true })
  ).toHaveCount(0);
});
