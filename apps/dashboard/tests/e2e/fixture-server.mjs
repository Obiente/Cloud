import http from "node:http";
import { createRequire } from "node:module";
const authenticated = process.env.PLAYWRIGHT_AUTH_TESTS === "true";
const require = createRequire(import.meta.resolve("nuxt"));
const { sealSession } = require("@nuxt/nitro-server/h3");
let authMode = "normal",
  renewals = 0;
const sessionPassword =
  "synthetic-test-session-password-at-least-32-characters";
const authUser = {
  sub: "user-example",
  name: "Example Admin",
  email: "admin@example.test",
  given_name: "Example",
  family_name: "Admin",
  preferred_username: "example",
  email_verified: true,
  locale: "en",
  updated_at: 0,
};

const port = Number(process.env.PLAYWRIGHT_FIXTURE_PORT || 4311);
const timestamp = "2026-01-01T12:00:00Z";
const org = {
  id: "org-a",
  name: "Example Alpha",
  slug: "example-alpha",
  plan: "pro",
  credits: "5000",
};
const quota = {
  organizationId: org.id,
  deploymentsMaxOverride: 7,
  cpuCoresOverride: 2,
  memoryBytesOverride: "4096",
  bandwidthBytesMonthOverride: "8192",
  storageBytesOverride: "16384",
};
const events = Array.from({ length: 128 }, (_, i) => ({
  id: `evt_test_${String(i).padStart(3, "0")}`,
  eventType: "invoice.paid",
  organizationId: org.id,
  organizationName: org.name,
  customerId: "cus_example",
  invoiceId: `inv_${i}`,
  createdAt: timestamp,
  processedAt: timestamp,
}));
http
  .createServer(async (req, res) => {
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Headers", "*");
    res.setHeader("Access-Control-Allow-Methods", "POST,GET,OPTIONS");
    res.setHeader("Content-Type", "application/json");
    if (req.method === "OPTIONS") {
      res.writeHead(204);
      res.end();
      return;
    }
    if (authenticated && req.url?.startsWith("/__fixture/auth/")) {
      const url = new URL(req.url, `http://127.0.0.1:${port}`);
      if (url.pathname.endsWith("/mode")) {
        authMode = url.searchParams.get("value") || "normal";
        res.end("{}");
        return;
      }
      if (url.pathname.endsWith("/stats")) {
        res.end(JSON.stringify({ renewals }));
        return;
      }
      if (url.pathname.endsWith("/cookie")) {
        renewals = 0;
        authMode = "normal";
        const expired = url.searchParams.get("expired") === "true";
        const data = {
          user: authUser,
          secure: {
            scope: "openid",
            token_type: "Bearer",
            access_token: expired ? "expired-access" : "valid-access",
            refresh_token: "synthetic-refresh",
            id_token: "synthetic-id",
            expires_in: 3600,
            expires_at: Date.now() + (expired ? -60000 : 3600000),
          },
        };
        const value = await sealSession(
          {
            context: {
              sessions: {
                h3: { id: "synthetic-session", createdAt: Date.now(), data },
              },
            },
          },
          { password: sessionPassword }
        );
        res.end(JSON.stringify({ value }));
        return;
      }
    }
    if (authenticated && req.url === "/oidc/v1/userinfo") {
      if (authMode === "outage") {
        res.writeHead(503);
        res.end("{}");
        return;
      }
      if (
        !["Bearer valid-access", "Bearer renewed-access"].includes(
          req.headers.authorization
        )
      ) {
        res.writeHead(401);
        res.end("{}");
        return;
      }
      res.end(JSON.stringify(authUser));
      return;
    }
    if (authenticated && req.url === "/oauth/v2/token") {
      let input = "";
      for await (const chunk of req) input += chunk;
      const parameters = new URLSearchParams(input);
      if (authMode === "outage") {
        res.writeHead(503);
        res.end("{}");
        return;
      }
      if (
        parameters.get("refresh_token") !== "synthetic-refresh" ||
        renewals > 0
      ) {
        res.writeHead(400);
        res.end(JSON.stringify({ error: "invalid_grant" }));
        return;
      }
      renewals++;
      res.end(
        JSON.stringify({
          access_token: "renewed-access",
          refresh_token: "rotated-refresh",
          expires_in: 3600,
          token_type: "Bearer",
          scope: "openid",
        })
      );
      return;
    }
    let input = "";
    for await (const chunk of req) input += chunk;
    let body = {};
    try {
      body = JSON.parse(input || "{}");
    } catch {
      /* Non-RPC fixture requests do not have a JSON body. */
    }
    const method = (req.url || "").split("/").at(-1);
    if (
      authenticated &&
      method !== "GetPublicConfig" &&
      !["Bearer valid-access", "Bearer renewed-access"].includes(
        req.headers.authorization
      )
    ) {
      res.writeHead(401);
      res.end(
        JSON.stringify({
          code: "unauthenticated",
          message: "Synthetic access token required",
        })
      );
      return;
    }
    let result = {};
    switch (method) {
      case "GetPublicConfig":
        result = {
          disableAuth: !authenticated,
          billingEnabled: true,
          selfHosted: true,
        };
        break;
      case "GetMySuperadminPermissions":
        result = { isFullSuperadmin: true, permissions: ["superadmin.*"] };
        break;
      case "GetCurrentUser":
        result = {
          user: {
            id: "user-example",
            email: "admin@example.test",
            name: "Example Admin",
          },
        };
        break;
      case "ListOrganizations":
        result = {
          organizations: [
            org,
            { ...org, id: "org-b", name: "Example Beta", slug: "example-beta" },
          ],
          pagination: { total: 2 },
        };
        break;
      case "GetOrganization":
        result = { organization: org };
        break;
      case "GetOverview":
        result = {
          counts: {
            totalOrganizations: "2",
            activeMembers: "3",
            pendingInvites: "1",
            totalDeployments: "2",
          },
          organizations: [
            {
              ...org,
              memberCount: "3",
              inviteCount: "1",
              deploymentCount: "2",
              createdAt: timestamp,
            },
          ],
          usages: [
            {
              organizationId: org.id,
              organizationName: org.name,
              cpuCoreSeconds: "7200",
              memoryByteSeconds: "3865470566400",
              storageBytes: "1048576",
            },
          ],
        };
        break;
      case "GetGameServer":
        result = {
          gameServer: {
            id: body.gameServerId,
            name: "Example game server",
            organizationId: org.id,
            status: "STOPPED",
            gameType: "MINECRAFT",
            createdAt: timestamp,
          },
        };
        break;
      case "ListGameServers":
        result = {
          gameServers: [
            {
              id: "gs-example",
              name: "Example game server",
              organizationId: org.id,
              status: "RUNNING",
              gameType: "MINECRAFT",
            },
            {
              id: "gs-second",
              name: "Second game server",
              organizationId: org.id,
              status: "STOPPED",
              gameType: "VALHEIM",
            },
          ],
        };
        break;
      case "GetDatabase":
        result = {
          database: {
            id: body.databaseId,
            name: "Example database",
            organizationId: org.id,
            type: "POSTGRESQL",
            status: "RUNNING",
            createdAt: timestamp,
          },
        };
        break;
      case "ListDatabases":
        result = { databases: [], pagination: { total: 4 } };
        break;
      case "GetUsage":
        result = {
          organizationId: org.id,
          month: "2026-01",
          current: {
            cpuCoreSeconds: "3600",
            storageMeasurementUnavailable: true,
            costEstimateUnavailable: true,
          },
          estimatedMonthly: {
            storageMeasurementUnavailable: true,
            costEstimateUnavailable: true,
          },
          quota: {},
        };
        break;
      case "GetIncomeOverview":
        result = {
          summary: { totalRevenue: 100, netRevenue: 90 },
          transactions: [
            {
              id: "usage-1",
              organizationId: org.id,
              organizationName: org.name,
              type: "usage",
              amountCents: -1200,
              status: "succeeded",
              note: "Example usage charge",
              createdAt: timestamp,
            },
            {
              id: "refund-1",
              organizationId: org.id,
              organizationName: org.name,
              type: "refund",
              amountCents: -300,
              status: "refunded",
              note: "Example refund",
              createdAt: timestamp,
            },
          ],
        };
        break;
      case "GetOrgQuota":
        result = {
          quota,
          planName: "Example Plan",
          planDefaults: {
            deploymentsMax: 10,
            cpuCores: 4,
            memoryBytes: "8589934592",
            bandwidthBytesMonth: "10737418240",
            storageBytes: "21474836480",
          },
        };
        break;
      case "UpsertOrgQuota":
        result = {};
        break;
      case "ListStripeWebhookEvents": {
        const rows = body.eventType
          ? events.filter((e) => e.eventType === body.eventType)
          : events;
        const offset = body.offset || 0;
        result = {
          events: rows.slice(offset, offset + (body.limit || 50)),
          totalCount: String(rows.length),
        };
        break;
      }
      case "GetAbuseDetection":
        result = {
          suspiciousOrganizations: [],
          suspiciousActivities: [],
          metrics: {},
        };
        break;
      case "ListRoles":
        result = {
          roles: [
            {
              id: "role-reader",
              name: "Database Reader",
              permissionsJson: '["database.read"]',
            },
          ],
        };
        break;
      case "ListPermissions":
        result = {
          permissions: [
            {
              id: "database.read",
              description: "Read databases",
              resourceType: "database",
            },
          ],
        };
        break;
      case "ListMembers":
        result = {
          members: [
            {
              id: "member-example",
              userId: "user-example",
              email: "admin@example.test",
              name: "Example Admin",
              role: "owner",
            },
          ],
        };
        break;
      case "CheckPermission":
        result = { allowed: true };
        break;
      case "ListDeployments":
        result = {
          deployments: [
            {
              id: "dep-example",
              name: "Example deployment",
              organizationId: org.id,
              status: "RUNNING",
              environment: "PRODUCTION",
              createdAt: timestamp,
            },
          ],
          pagination: { total: 1 },
        };
        break;
    }
    res.end(JSON.stringify(result));
  })
  .listen(port, "127.0.0.1", () =>
    console.log(`Fixture API listening on ${port}`)
  );
