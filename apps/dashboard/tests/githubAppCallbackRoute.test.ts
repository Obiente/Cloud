import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  clearPKCECookie: vi.fn(),
  clearStateCookie: vi.fn(),
  setPKCECookie: vi.fn(),
  setStateCookie: vi.fn(),
}));

vi.mock("../server/utils/githubAppInstallState", () => ({
  clearGitHubAppInstallPKCECookie: mocks.clearPKCECookie,
  clearGitHubAppInstallStateCookie: mocks.clearStateCookie,
  createGitHubAppInstallNonce: () => "authorization-nonce",
  createGitHubAppInstallPKCE: () => ({
    codeVerifier: "v".repeat(43),
    codeChallenge: "pkce-challenge",
  }),
  decodeGitHubAppInstallState: () => ({
    random: "installation-nonce",
    orgId: "workspace-a",
    issuedAt: 1,
  }),
  encodeGitHubAppInstallState: () => "signed-authorization-state",
  getGitHubAppCallbackUrl: () =>
    "https://obiente.cloud/api/github/app/callback",
  getGitHubAppInstallPKCEVerifier: () => undefined,
  getGitHubAppInstallStateCookie: () => "signed-installation-state",
  setGitHubAppInstallPKCECookie: mocks.setPKCECookie,
  setGitHubAppInstallStateCookie: mocks.setStateCookie,
  verifyGitHubAppInstallState: (
    _event: unknown,
    expected: string,
    actual: string
  ) => expected === actual,
}));

describe("GitHub App callback route", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vi.stubGlobal("defineEventHandler", <T>(handler: T) => handler);
    vi.stubGlobal("useRuntimeConfig", () => ({
      githubAppClientId: "github-client-id",
      public: {},
    }));
    vi.stubGlobal(
      "sendRedirect",
      vi.fn((_event, location: string) => location)
    );
  });

  it("turns the verified installation callback into a PKCE authorization", async () => {
    vi.stubGlobal("getQuery", () => ({
      state: "signed-installation-state",
      installation_id: "42",
      setup_action: "install",
    }));
    const { default: callback } = await import(
      "../server/api/github/app/callback.get"
    );

    const location = await callback({} as never);
    const url = new URL(String(location));

    expect(url.origin).toBe("https://github.com");
    expect(url.pathname).toBe("/login/oauth/authorize");
    expect(url.searchParams.get("client_id")).toBe("github-client-id");
    expect(url.searchParams.get("state")).toBe("signed-authorization-state");
    expect(url.searchParams.get("redirect_uri")).toBe(
      "https://obiente.cloud/api/github/app/callback"
    );
    expect(url.searchParams.get("code_challenge")).toBe("pkce-challenge");
    expect(url.searchParams.get("code_challenge_method")).toBe("S256");
    expect(mocks.setStateCookie).toHaveBeenCalledWith(
      expect.anything(),
      "signed-authorization-state"
    );
    expect(mocks.setPKCECookie).toHaveBeenCalledWith(
      expect.anything(),
      "authorization-nonce",
      "v".repeat(43)
    );
  });

  it("rejects a callback that does not return the install state", async () => {
    vi.stubGlobal("getQuery", () => ({ installation_id: "42" }));
    const { default: callback } = await import(
      "../server/api/github/app/callback.get"
    );

    const location = await callback({} as never);

    expect(String(location)).toContain("error=invalid_state");
    expect(mocks.setPKCECookie).not.toHaveBeenCalled();
  });

  it("accepts a non-mutating direct installation update callback", async () => {
    vi.stubGlobal("getQuery", () => ({
      installation_id: "42",
      setup_action: "update",
    }));
    const { default: callback } = await import(
      "../server/api/github/app/callback.get"
    );

    const location = await callback({} as never);
    const url = new URL(String(location), "https://obiente.cloud");

    expect(url.pathname).toBe("/settings");
    expect(url.searchParams.get("success")).toBe("true");
    expect(url.searchParams.get("installationUpdated")).toBe("true");
    expect(url.searchParams.get("installationId")).toBe("42");
    expect(mocks.setPKCECookie).not.toHaveBeenCalled();
  });
});
