import { expect, test } from "@playwright/test";

test("deployment files show volumes, preserve container selection and recover from storage errors", async ({ page, context }, testInfo) => {
  test.setTimeout(Math.max(testInfo.timeout, 90000));
  await context.addCookies([{ name: "obiente_selected_org_id", value: "org-a", domain: "127.0.0.1", path: "/" }]);
  await context.addInitScript(() => {
    localStorage.setItem("selectedOrgId", "org-a");
    localStorage.setItem("obiente_orgs_cache", JSON.stringify([{ id: "org-a", name: "Example Alpha" }]));
  });
  await page.route("**/GetDeployment", route => route.fulfill({ json: { deployment: {
    id: "dep-example", organizationId: "org-a", name: "Example deployment", status: "RUNNING", type: "DOCKER", environment: "PRODUCTION",
  } } }));
  await page.route("**/ListDeploymentContainers", route => route.fulfill({ json: { containers: [
    { containerId: "container-example-a", serviceName: "api", status: "running" },
    { containerId: "container-example-b", serviceName: "worker", status: "running" },
  ] } }));
  let unavailable = true;
  const requests: Record<string, unknown>[] = [];
  await page.route("**/ListContainerFiles", route => {
    const request = route.request().postDataJSON();
    requests.push(request);
    if (request.listVolumes) {
      if (unavailable) return route.fulfill({ status: 503, json: { code: "unavailable", message: "Storage unavailable" } });
      return route.fulfill({ json: { containerRunning: true, volumes: [{ name: "example-data", mountPoint: "/data", isPersistent: true }] } });
    }
    return route.fulfill({ json: { containerRunning: true, files: [{
      name: request.volumeName ? "record.txt" : "app.txt", path: request.volumeName ? "/record.txt" : "/app.txt", size: "4", permissions: "-rw-r--r--",
    }] } });
  });
  await page.route("**/GetContainerFile", route => route.fulfill({ json: {
    content: "saved data", encoding: "text", size: "10", metadata: { name: "record.txt", path: "/record.txt", size: "10" },
  } }));
  await page.goto("/deployments/dep-example?tab=files");
  const tree = page.getByLabel("File tree", { exact: true });
  await expect(tree.getByText("app.txt", { exact: true })).toBeVisible({ timeout: testInfo.timeout });
  await expect(page.getByRole("heading", { name: "Example deployment", exact: true })).toBeVisible();
  await expect(tree).toContainText("Unable to load storage volumes. Try refreshing.");
  await page.screenshot({ path: testInfo.outputPath("storage-error.png"), fullPage: true });
  unavailable = false;
  await page.getByRole("button", { name: "Refresh files", exact: true }).click();
  await expect(tree.getByRole("button", { name: /example-data/ })).toBeVisible();
  await expect(tree).not.toContainText("Unable to load storage volumes");
  const containerSelect = page.getByRole("combobox").filter({ hasText: /Select container|api|worker/ });
  const searchBounds = await page.getByRole("textbox", { name: "Search files...", exact: true }).boundingBox();
  const selectBounds = await containerSelect.boundingBox();
  expect(searchBounds && selectBounds && searchBounds.x < selectBounds.x + selectBounds.width && searchBounds.x + searchBounds.width > selectBounds.x && searchBounds.y < selectBounds.y + selectBounds.height && searchBounds.y + searchBounds.height > selectBounds.y).toBe(false);
  await containerSelect.click();
  await page.getByRole("option", { name: /worker/ }).click();
  await tree.getByRole("button", { name: /example-data/ }).click();
  await expect(tree.getByText("record.txt", { exact: true })).toBeVisible();
  const request = page.waitForRequest(r => r.url().endsWith("/GetContainerFile"));
  await tree.getByText("record.txt", { exact: true }).dblclick();
  expect((await request).postDataJSON()).toMatchObject({ containerId: "container-example-b", serviceName: "worker", volumeName: "example-data" });
  expect(requests.some(r => r.listVolumes && r.containerId === "container-example-b")).toBe(true);
  expect(requests.some(r => r.volumeName === "example-data" && r.containerId === "container-example-b")).toBe(true);
  await expect(page.locator(".monaco-editor .view-lines").first()).toContainText(/saved\s+data/, { timeout: testInfo.timeout });
  await page.screenshot({ path: testInfo.outputPath("volume-files.png"), fullPage: true });
});
