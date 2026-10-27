import { test, expect } from "@playwright/test";
const fixture = "http://127.0.0.1:4311/__fixture/auth";
test.setTimeout(process.env.PLAYWRIGHT_USE_BUILD === "true" ? 60000 : 120000);
test.beforeEach(async ({ context, request }) => {
  const response = await request.get(`${fixture}/cookie?expired=true`);
  const { value } = await response.json();
  await context.addCookies([
    { name: "h3", value, domain: "127.0.0.1", path: "/", httpOnly: true },
    {
      name: "obiente_selected_org_id",
      value: "org-a",
      domain: "127.0.0.1",
      path: "/",
    },
  ]);
});
test("server renewal reaches the browser and survives hydration and reload", async ({
  page,
  request,
  context,
}, info) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const response = await page.goto("/deployments");
  expect(response?.status()).toBe(200);
  expect(response?.headers()["cache-control"]).toContain("no-store");
  expect(
    (await response!.headersArray())
      .filter((h) => h.name.toLowerCase() === "set-cookie")
      .some((h) => h.value.startsWith("h3="))
  ).toBe(true);
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
  await expect(
    page.getByText("Example deployment", { exact: true })
  ).toBeVisible();
  const token = await page.evaluate(async () =>
    (await fetch("/auth/token")).json()
  );
  expect(token.accessToken).toBe("renewed-access");
  expect(
    (await context.cookies()).some(
      (cookie) =>
        cookie.name === "obiente_auth" && cookie.value === "renewed-access"
    )
  ).toBe(true);
  await page.reload();
  await expect(
    page.getByText("Example deployment", { exact: true })
  ).toBeVisible();
  expect(await (await request.get(`${fixture}/stats`)).json()).toEqual({
    renewals: 1,
  });
  expect(errors).toEqual([]);
  await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
});
test("provider outage preserves a known account on first render and recovers without reload", async ({
  page,
  request,
  context,
}, info) => {
  const { value } = await (
    await request.get(`${fixture}/cookie?expired=false`)
  ).json();
  await context.addCookies([
    { name: "h3", value, domain: "127.0.0.1", path: "/", httpOnly: true },
  ]);
  await request.get(`${fixture}/mode?value=outage`);
  await page.goto("/deployments");
  await page.waitForFunction(
    () =>
      (document.getElementById("__nuxt") as any)?.__vue_app__?.config
        .globalProperties.$nuxt?.isHydrating === false
  );
  await expect(
    page.getByText("Example deployment", { exact: true })
  ).toBeVisible();
  await expect(page).toHaveURL(/\/deployments$/);
  const session = await page.evaluate(async () =>
    (await fetch("/auth/session")).json()
  );
  expect(session.user.sub).toBe("user-example");
  expect(session.secure).toBeUndefined();
  await request.get(`${fixture}/mode?value=normal`);
  const recovered = await page.evaluate(async () =>
    (await fetch("/auth/session")).json()
  );
  expect(recovered.user.sub).toBe("user-example");
  await expect(
    page.getByText("Example deployment", { exact: true })
  ).toBeVisible();
  await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
});
