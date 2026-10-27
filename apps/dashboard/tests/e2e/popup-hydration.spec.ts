import { test, expect } from "@playwright/test";

test("fresh pages hydrate popup controls without mismatches or an error page", async ({
  browser,
}, info) => {
  test.setTimeout(process.env.PLAYWRIGHT_USE_BUILD === "true" ? 120_000 : 240_000);
  for (let attempt = 0; attempt < 6; attempt++) {
    const context = await browser.newContext({
      baseURL: info.project.use.baseURL,
      viewport: info.project.use.viewport,
      userAgent: info.project.use.userAgent,
      deviceScaleFactor: info.project.use.deviceScaleFactor,
      isMobile: info.project.use.isMobile,
      hasTouch: info.project.use.hasTouch,
    });
    try {
      await context.addCookies([
        { name: "obiente_selected_org_id", value: "org-a", domain: "127.0.0.1", path: "/" },
      ]);
      await context.addInitScript(() => {
        localStorage.setItem("selectedOrgId", "org-a");
        localStorage.setItem("obiente_orgs_cache", JSON.stringify([
          { id: "org-a", name: "Example Alpha" },
          { id: "org-b", name: "Example Beta" },
        ]));
      });
      const page = await context.newPage();
      const errors: string[] = [];
      page.on("console", (message) => {
        if (/hydration.*mismatch|hydration completed/i.test(message.text())) errors.push(message.text());
      });
      page.on("pageerror", (error) => errors.push(error.message));
      const response = await page.goto("/deployments");
      expect(response?.status()).toBe(200);
      const popupCount = await page.evaluate((html) => {
        const document = new DOMParser().parseFromString(html, "text/html");
        return document.querySelectorAll('[data-scope="select"][data-part="positioner"], [data-scope="combobox"][data-part="positioner"]').length;
      }, await response!.text());
      // Closed popups must mount after hydration. SSR body teleports can attach
      // to a different control when asynchronous layout and page work settles.
      expect(popupCount).toBe(0);
      await page.waitForFunction(() => (document.getElementById("__nuxt") as any)?.__vue_app__?.config.globalProperties.$nuxt?.isHydrating === false);
      await expect(page.getByText("Example deployment", { exact: true })).toBeVisible();
      const status = page.getByRole("combobox").filter({ hasText: "All Status" });
      await status.click();
      await page.getByRole("option", { name: "Running", exact: true }).click();
      await expect(page.getByText("Example deployment", { exact: true })).toBeVisible();
      expect(await page.evaluate(() => (document.getElementById("__nuxt") as any)?.__vue_app__?.config.globalProperties.$nuxt?.payload.error ?? null)).toBeNull();
      expect(errors).toEqual([]);
      if (attempt === 5) await page.screenshot({ path: info.outputPath("page.png"), fullPage: true });
    } finally {
      await context.close();
    }
  }
});
