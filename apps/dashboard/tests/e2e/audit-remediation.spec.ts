import {expect, test} from "@playwright/test";

test.afterEach(async ({page},testInfo)=>{
 await page.screenshot({path:testInfo.outputPath("page.png"),fullPage:true});
});

test.beforeEach(async ({context}) => {
 await context.addCookies([{name:"obiente_selected_org_id",value:"org-a",domain:"127.0.0.1",path:"/"}]);
 await context.addInitScript(()=>{
  if (location.origin !== "http://127.0.0.1:4310") return;
  localStorage.setItem("selectedOrgId","org-a");
  localStorage.setItem("obiente_orgs_cache",JSON.stringify([{id:"org-a",name:"Example Alpha"},{id:"org-b",name:"Example Beta"}]));
 });
});

test("overview displays populated counts and correct units",async({page})=>{
 await page.goto("/superadmin");
 await expect(page.getByRole("heading",{name:"Superadmin Overview",exact:true})).toBeVisible();
 const row=page.getByRole("row").filter({has:page.getByText("Example Alpha",{exact:true})}).first();
 await expect(row).toContainText("3");
 await expect(row).toContainText("2");
 await expect(page.getByText("2 core-hours",{exact:true})).toBeVisible();
 await expect(page).toHaveURL(/\/superadmin$/);
 await expect(page.locator("header.app-topbar")).not.toContainText("Dashboard");
});

test("saved quotas load and zero is sent as an explicit reset",async({page})=>{
 await page.goto("/admin/quotas");
 await expect(page.getByRole("heading",{name:/Organization Quotas/})).toBeVisible();
 const limit=page.getByLabel("Deployments Max",{exact:true});
 await expect(limit).toHaveValue("7");
 await expect(page.getByText("Plan: 10. Effective: 7.", {exact:true})).toBeVisible();
 await limit.fill("0");
 await expect(page.getByText("Plan: 10. Effective: 10.", {exact:true})).toBeVisible();
 const request=page.waitForRequest(r=>r.url().endsWith("/UpsertOrgQuota"));
 await page.getByRole("button",{name:"Save",exact:true}).click();
 expect((await request).postDataJSON().deploymentsMaxOverride ?? 0).toBe(0);
 await expect(page.getByText("Saved",{exact:true})).toBeVisible();
 await page.route("**/GetOrgQuota", route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({quota:{organizationId:"org-b",deploymentsMaxOverride:3},planDefaults:{deploymentsMax:5}})}));
 await page.getByRole("combobox").filter({hasText:"Example Alpha"}).last().click();
 await page.getByRole("option",{name:"Example Beta",exact:true}).click();
 await expect(limit).toHaveValue("3");
 await expect(page.getByText("Plan: 5. Effective: 3.", {exact:true})).toBeVisible();
 expect((await page.context().cookies()).find(cookie=>cookie.name==="obiente_selected_org_id")?.value).toBe("org-a");
});

test("all webhook pages remain reachable and details open",async({page})=>{
 const offsets:number[]=[];
 page.on("request",r=>{if(r.url().endsWith("/ListStripeWebhookEvents"))offsets.push(r.postDataJSON().offset||0)});
 await page.goto("/superadmin/webhook-events");
 await expect(page.getByRole("heading",{name:"Stripe Webhook Events",exact:true})).toBeVisible();
 await expect(page.getByText("evt_test_049",{exact:true})).toBeVisible();
 await page.getByRole("button",{name:/Load More/i}).click();
 await expect(page.getByText("evt_test_099",{exact:true})).toBeVisible();
 await page.getByRole("button",{name:/Load More/i}).click();
 await expect(page.getByText("evt_test_127",{exact:true})).toBeVisible();
 expect(offsets).toEqual([0,50,100]);
 await expect(page.getByRole("button",{name:/Load More/i})).toBeDisabled();
 await page.getByRole("button",{name:/Details/i}).first().click();
 await expect(page.getByRole("dialog")).toBeVisible();
 await expect(page.getByRole("dialog")).toContainText("evt_test_000");
});

test("abuse errors do not present a healthy result and refresh recovers",async({page})=>{
 let failed=true;
 await page.route("**/GetAbuseDetection",route=>route.fulfill({status:failed?503:200,contentType:"application/json",body:JSON.stringify(failed?{code:"unavailable",message:"Temporary fixture outage"}:{suspiciousOrganizations:[],suspiciousActivities:[],metrics:{}})}));
 await page.goto("/superadmin/abuse");
 await page.getByRole("button",{name:"Refresh",exact:true}).click();
 await expect(page.getByText("Abuse detection is unavailable",{exact:true})).toBeVisible();
 failed=false;
 await page.getByRole("button",{name:"Refresh",exact:true}).click();
 await expect(page.getByText("Abuse detection is unavailable",{exact:true})).toHaveCount(0);
 await expect(page.getByRole("heading",{name:"Abuse Detection",exact:true})).toBeVisible();
});

test("role binding requires an explicit member and role",async({page})=>{
 await page.goto("/admin/bindings");
 await expect(page.getByRole("button",{name:"Bind",exact:true})).toBeDisabled();
 await expect(page.getByPlaceholder("Search for a member...")).toHaveValue("");
 await expect(page.getByPlaceholder("Search for a role...")).toHaveValue("");
});

test("deployment refresh recovers from an outage without reloading",async({page})=>{
 await page.goto("/deployments");
 await expect(page.getByText("Example deployment",{exact:true})).toBeVisible();
 let failed=true;
 await page.route("**/ListDeployments",route=>route.fulfill({status:failed?503:200,contentType:"application/json",body:JSON.stringify(failed?{code:"unavailable",message:"Temporary fixture outage"}:{deployments:[{id:"dep-recovered",name:"Recovered deployment",organizationId:"org-a",status:"RUNNING"}],pagination:{total:1}})}));
 await page.evaluate(()=>window.dispatchEvent(new Event("online")));
 await expect(page.getByText("Failed to load deployments",{exact:true})).toBeVisible();
 failed=false;
 await page.evaluate(()=>window.dispatchEvent(new Event("online")));
 await expect(page.getByText("Recovered deployment",{exact:true})).toBeVisible();
 await expect(page.getByText("Failed to load deployments",{exact:true})).toHaveCount(0);
 expect(await page.evaluate(()=>performance.getEntriesByType("navigation").length)).toBe(1);
});

test("a late deployment response cannot replace the selected organization",async({page})=>{
 await page.goto("/deployments");
 await page.waitForFunction(() => (document.getElementById("__nuxt") as any)?.__vue_app__?.config.globalProperties.$nuxt?.isHydrating === false);
 await expect(page.getByText("Example deployment",{exact:true})).toBeVisible();
 let release!:()=>void;
 const held=new Promise<void>(resolve=>{release=resolve});
 let received!:()=>void;
 const started=new Promise<void>(resolve=>{received=resolve});
 await page.route("**/ListDeployments",async route=>{
  const id=route.request().postDataJSON().organizationId;
  if(id==="org-a"){received();await held;}
  await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({deployments:[{id:`dep-${id}`,name:id==="org-a"?"Late Alpha deployment":"Beta deployment",organizationId:id,status:"RUNNING"}],pagination:{total:1}})});
 });
 await page.evaluate(()=>window.dispatchEvent(new Event("online")));
 await started;
 // Storage events are the supported cross-tab organization-switch path.
 await page.evaluate(()=>{localStorage.setItem("selectedOrgId","org-b");window.dispatchEvent(new StorageEvent("storage",{key:"selectedOrgId",newValue:"org-b"}));});
 await expect(page.getByText("Beta deployment",{exact:true})).toBeVisible();
 release();
 await expect(page.getByText("Late Alpha deployment",{exact:true})).toHaveCount(0);
 await expect(page.getByText("Beta deployment",{exact:true})).toBeVisible();
});

test("new DNS secrets stay available until the dialog closes",async({page})=>{
 await page.route("**/CreateDNSDelegationAPIKey",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({apiKey:"test-only-dns-secret"})}));
 await page.goto("/superadmin/dns");
 await page.getByRole("button",{name:"Create API Key",exact:true}).click();
 const dialog=page.getByRole("dialog");
 await dialog.getByPlaceholder("e.g., Self-hosted instance at example.com").fill("Example integration");
 await dialog.getByRole("button",{name:"Create API Key",exact:true}).click();
 await expect(dialog.getByText("test-only-dns-secret",{exact:true})).toBeVisible();
 await page.clock.install();
 await page.clock.fastForward(6000);
 await expect(dialog.getByText("test-only-dns-secret",{exact:true})).toBeVisible();
 await dialog.getByRole("button",{name:"Close",exact:true}).click();
 await expect(page.getByText("test-only-dns-secret",{exact:true})).toHaveCount(0);
});


test("income distinguishes usage charges and refresh updates the displayed data", async ({page}) => {
 await page.goto("/superadmin/income");
 const usage=page.getByRole("row").filter({hasText:"Example usage charge"});
 await expect(usage).toContainText("succeeded");
 await expect(usage).not.toContainText("refunded");
 await expect(page.getByRole("row").filter({hasText:"Example refund"})).toContainText("refunded");
 await page.route("**/GetIncomeOverview",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({transactions:[{id:"new-payment",organizationName:"Example Alpha",type:"payment",status:"succeeded",note:"New payment after refresh"}]})}));
 await page.getByRole("button",{name:"Refresh",exact:true}).click();
 await expect(page.getByText("New payment after refresh",{exact:true})).toBeVisible();
});

test("dashboard shows existing game servers and databases", async ({page}) => {
 await page.goto("/dashboard");
 const game=page.locator(".app-stat-card").filter({hasText:"Game Servers"});
 const databases=page.locator(".app-stat-card").filter({hasText:"Databases"});
 await expect(game.getByRole("heading")).toHaveText("2");
 await expect(databases.getByRole("heading")).toHaveText("4");
});


test("database backups explain unavailable storage and offer supported SQL export", async ({page}) => {
 await page.goto("/databases/db-example");
 await page.getByRole("tab", {name:"Backups",exact:true}).click();
 await expect(page.getByText("Managed backups are unavailable",{exact:true})).toBeVisible();
 await expect(page.getByRole("button",{name:"Create backup",exact:true})).toHaveCount(0);
 await expect(page.getByRole("button",{name:"Export SQL",exact:true})).toBeVisible();
});

test("users refresh replaces the displayed result", async ({page}) => {
 await page.goto("/superadmin/users");
 await page.route("**/ListUsers",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({users:[{id:"new-user",name:"New Example User",email:"new@example.test"}],pagination:{page:1,perPage:50,total:1,totalPages:1}})}));
 await page.getByRole("button",{name:"Refresh",exact:true}).click();
 await expect(page.getByText("New Example User",{exact:true})).toBeVisible();
});


test("game server enum filters select populated results and can clear", async ({page}) => {
 await page.goto("/gameservers");
 await expect(page.getByText("Example game server",{exact:true})).toBeVisible();
 await page.getByRole("combobox").filter({hasText:"All Status"}).click();
 await page.getByRole("option",{name:"Stopped",exact:true}).click();
 await expect(page.getByText("Second game server",{exact:true})).toBeVisible();
 await expect(page.getByText("Example game server",{exact:true})).toHaveCount(0);
 await page.getByRole("combobox").filter({hasText:"Stopped"}).click();
 await page.getByRole("option",{name:"All Status",exact:true}).click();
 await page.getByRole("combobox").filter({hasText:"All Game Types"}).click();
 await page.getByRole("option",{name:"Minecraft",exact:true}).click();
 await expect(page.getByText("Example game server",{exact:true})).toBeVisible();
 await expect(page.getByText("Second game server",{exact:true})).toHaveCount(0);
});

test("database detail follows backend changes without navigation", async ({page}) => {
 await page.goto("/databases/db-example");
 await expect(page.getByRole("heading",{name:"Example database",exact:true})).toBeVisible();
 await page.route("**/GetDatabase", route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({database:{id:"db-example",name:"Updated database",organizationId:"org-a",type:"POSTGRESQL",status:"STOPPED"}})}));
 await page.evaluate(()=>window.dispatchEvent(new Event("online")));
 await expect(page.getByRole("heading",{name:"Updated database",exact:true})).toBeVisible();
 expect(await page.evaluate(()=>performance.getEntriesByType("navigation").length)).toBe(1);
});

test("game server detail refreshes and reconnects metrics after transport failure", async ({page}) => {
 let streams=0;
 const frame=(flag:number,value:object)=>{const body=Buffer.from(JSON.stringify(value));const header=Buffer.alloc(5);header[0]=flag;header.writeUInt32BE(body.length,1);return Buffer.concat([header,body]);};
 await page.route("**/StreamGameServerMetrics", route=>{
  streams++;
  if(streams===1) return route.fulfill({status:503,contentType:"application/json",body:JSON.stringify({code:"unavailable",message:"Temporary fixture outage"})});
  return route.fulfill({status:200,contentType:"application/connect+json",body:Buffer.concat([frame(0,{gameServerId:"gs-example",cpuUsagePercent:42,memoryUsageBytes:"1048576"}),frame(2,{})])});
 });
 await page.goto("/gameservers/gs-example");
 await expect(page.getByRole("heading",{name:"Example game server",exact:true})).toBeVisible();
 await page.route("**/GetGameServer",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({gameServer:{id:"gs-example",name:"Updated game server",organizationId:"org-a",status:"RUNNING",gameType:"MINECRAFT"}})}));
 await page.evaluate(()=>window.dispatchEvent(new Event("online")));
 await expect(page.getByRole("heading",{name:"Updated game server",exact:true})).toBeVisible({timeout:15_000});
 await expect.poll(()=>streams).toBeGreaterThanOrEqual(2);
 expect(await page.evaluate(()=>performance.getEntriesByType("navigation").length)).toBe(1);
 const menu=page.getByRole("button",{name:"Toggle navigation menu",exact:true});
 if (await menu.isVisible()) await menu.click();
 await page.getByRole("link",{name:"Dashboard",exact:true}).first().click();
 await expect(page).toHaveURL(/\/dashboard$/);
 const stoppedAt=streams;
 await page.waitForTimeout(1600);
 expect(streams).toBe(stoppedAt);
});

test("DHCP filters distinguish missing expiry and return to all leases",async({page})=>{
 const now=Date.now();
 await page.route("**/GetOrgLeases",route=>route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({leases:route.request().postDataJSON().organizationId==="org-a"?[
  {vpsId:"example-expiring",organizationId:"org-a",ipAddress:"192.0.2.10",macAddress:"02:00:00:00:00:10",expiresAt:new Date(now+3600000).toISOString()},
  {vpsId:"example-active",organizationId:"org-a",ipAddress:"192.0.2.11",macAddress:"02:00:00:00:00:11",expiresAt:new Date(now+172800000).toISOString()},
  {vpsId:"example-unknown",organizationId:"org-a",ipAddress:"192.0.2.12",macAddress:"02:00:00:00:00:12"},
 ]:[]})}));
 await page.goto("/superadmin/vps");
 await page.getByRole("tab",{name:"DHCP Leases",exact:true}).click();
 await expect(page.getByText("example-unknown",{exact:true})).toBeVisible();
 await expect(page.getByRole("row").filter({hasText:"example-unknown"})).toContainText("Unknown");
 await page.getByRole("combobox").filter({hasText:"Expiry Status"}).click();
 await page.getByRole("option",{name:"Active (24h or more)",exact:true}).click();
 await expect(page.getByText("example-active",{exact:true})).toBeVisible();
 await expect(page.getByText("example-unknown",{exact:true})).toHaveCount(0);
 await page.getByRole("combobox").filter({hasText:"Active (24h or more)"}).click();
 await page.getByRole("option",{name:"All expiry states",exact:true}).click();
 await expect(page.getByText("example-unknown",{exact:true})).toBeVisible();
 await expect(page.getByRole("button",{name:"Copy IP address 192.0.2.12",exact:true})).toBeVisible();
});

test("unavailable storage does not appear as a zero cost estimate",async({page})=>{
 await page.goto("/dashboard");
 const costs=page.locator(".app-panel").filter({has:page.getByRole("heading",{name:"Cost Breakdown",exact:true})});
 await expect(costs).toContainText("Unavailable");
 await expect(costs).not.toContainText("$0.00");
 await page.goto("/billing");
 const estimate=page.locator(".oui-card-base").filter({has:page.getByText("Monthly Estimate",{exact:true})});
 await expect(estimate).toContainText("Unavailable");
});
