import { expect, test } from "@playwright/test";

test("session and cached organizations remain stable during repeated hydration", async ({
  page,
  context,
}, info) => {
  test.setTimeout(90_000);
  await context.addCookies([
    {
      name: "obiente_selected_org_id",
      value: "org-a",
      domain: "127.0.0.1",
      path: "/",
    },
  ]);
  await context.addInitScript(() => {
    if (location.origin !== "http://127.0.0.1:4310") return;
    localStorage.setItem(
      "obiente_orgs_cache",
      JSON.stringify([{ id: "org-a", name: "Cached Alpha" }])
    );
  });
  const errors: string[] = [];
  page.on("console", (message) => {
    if (/hydration.*mismatch|hydration completed/i.test(message.text()))
      errors.push(message.text());
  });
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/auth/session", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 750));
    await route.continue();
  });
  for (let attempt = 0; attempt < 3; attempt++) {
    await page.goto("/deployments");
    await page.waitForFunction(
      () =>
        (document.getElementById("__nuxt") as any)?.__vue_app__?.config
          .globalProperties.$nuxt?.isHydrating === false
    );
    await expect(
      page.getByText("Example deployment", { exact: true })
    ).toBeVisible();
    await page.waitForTimeout(1000);
  }
  expect(errors).toEqual([]);
  await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
});

test("temporary session endpoint outage preserves the rendered account", async ({
  page,
  context,
}, info) => {
  test.setTimeout(60000);
  await context.addCookies([
    {
      name: "obiente_selected_org_id",
      value: "org-a",
      domain: "127.0.0.1",
      path: "/",
    },
  ]);
  await page.route("**/auth/session", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({
        statusCode: 503,
        message: "Temporary authentication outage",
      }),
    })
  );
  const failedRequest = page.waitForResponse(
    (response) =>
      response.url().endsWith("/auth/session") && response.status() === 503
  );
  await page.goto("/deployments");
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
  await failedRequest;
  await expect(
    page.getByText("Example deployment", { exact: true })
  ).toBeVisible();
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          (document.getElementById("__nuxt") as any).__vue_app__.config
            .globalProperties.$nuxt.payload.state["$sobiente-session"]?.user?.id
      )
    )
    .toBe("mem-development");
  await expect(page).toHaveURL(/\/deployments$/);
  await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
});
