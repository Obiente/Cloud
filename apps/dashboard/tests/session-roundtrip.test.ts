import { createServer, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import {
  createApp,
  eventHandler,
  sealSession,
  toNodeListener,
  unsealSession,
} from "@nuxt/nitro-server/h3";

const password = "synthetic-session-roundtrip-password-at-least-32-characters";
const user = { sub: "example-user", name: "Example User" };
let server: Server;
let base: string;
let cookie: string;
let captured = false;
const config = {
  session: { password },
  public: {
    oidcBase: "https://identity.example.test",
    oidcClientId: "example-client",
  },
};
let provider: ReturnType<typeof vi.fn>;

beforeEach(async () => {
  vi.resetModules();
  captured = false;
  vi.stubEnv("DISABLE_AUTH", "false");
  vi.stubGlobal("useRuntimeConfig", () => ({
    session: { password },
    public: {
      oidcBase: "https://identity.example.test",
      oidcClientId: "example-client",
    },
  }));
  provider = vi.fn();
  vi.stubGlobal("$fetch", provider);
  const { readCurrentSession } = await import(
    "../server/utils/current-session"
  );
  const app = createApp().use(
    eventHandler((event) =>
      readCurrentSession(event, undefined, captured ? config : undefined)
    )
  );
  server = createServer(toNodeListener(app));
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  cookie = await sealSession(
    {
      context: {
        sessions: {
          h3: {
            id: "synthetic-session",
            createdAt: Date.now(),
            data: {
              user,
              secure: {
                access_token: "expired-access",
                refresh_token: "synthetic-refresh",
                expires_in: 3600,
                expires_at: Date.now() - 60000,
              },
            },
          },
        },
      },
    } as never,
    { password }
  );
});

afterEach(async () => {
  if (server) {
    server.closeAllConnections();
    await new Promise<void>((resolve) => server.close(() => resolve()));
  }
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

it("persists renewal in the actual HTTP response cookie without exposing tokens", async () => {
  provider
    .mockRejectedValueOnce({ statusCode: 401 })
    .mockResolvedValueOnce({
      access_token: "renewed-access",
      refresh_token: "rotated-refresh",
      expires_in: 3600,
    })
    .mockResolvedValue(user);
  const response = await fetch(base, { headers: { cookie: `h3=${cookie}` } });
  expect(response.status).toBe(200);
  expect(response.headers.get("cache-control")).toBe("private, no-store");
  expect(await response.json()).toEqual({ id: "synthetic-session", user });
  const sessionCookie = response.headers
    .getSetCookie()
    .find((value) => value.startsWith("h3="))!;
  const sealed = decodeURIComponent(sessionCookie.split(";")[0]!.slice(3));
  const session = await unsealSession({} as never, { password }, sealed);
  expect(session.data.secure.access_token).toBe("renewed-access");
  expect(session.data.secure.refresh_token).toBe("rotated-refresh");
  expect(session.data.secure.expires_at).toBeGreaterThan(Date.now());
  const reloaded = await fetch(base, {
    headers: { cookie: sessionCookie.split(";")[0]! },
  });
  expect(await reloaded.json()).toEqual({ id: "synthetic-session", user });
  expect(
    provider.mock.calls.filter(([url]) => url.endsWith("/token"))
  ).toHaveLength(1);
});

it("keeps a signed existing session when the provider is temporarily unavailable", async () => {
  provider.mockRejectedValue({ statusCode: 503 });
  const response = await fetch(base, { headers: { cookie: `h3=${cookie}` } });
  expect(response.status).toBe(200);
  expect(await response.json()).toEqual({ id: "synthetic-session", user });
  expect(response.headers.getSetCookie()).toEqual([]);
});

it("renews a request using captured configuration after the app context disappears", async () => {
  captured = true;
  vi.stubGlobal("useRuntimeConfig", () => {
    throw new Error("Nuxt context is unavailable after await");
  });
  provider
    .mockRejectedValueOnce({ statusCode: 401 })
    .mockResolvedValueOnce({
      access_token: "renewed-access",
      refresh_token: "rotated-refresh",
      expires_in: 3600,
    })
    .mockResolvedValue(user);
  const response = await fetch(base, { headers: { cookie: `h3=${cookie}` } });
  expect(response.status).toBe(200);
  expect(await response.json()).toEqual({ id: "synthetic-session", user });
  expect(
    response.headers.getSetCookie().some((value) => value.startsWith("h3="))
  ).toBe(true);
});
