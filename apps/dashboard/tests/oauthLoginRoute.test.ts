import { beforeEach, describe, expect, it, vi } from "vitest";

describe("interactive OAuth login route", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllGlobals();
  });

  it("does not request silent authentication when there is no local session", async () => {
    const sendRedirect = vi.fn();

    vi.stubGlobal("eventHandler", <T>(handler: T) => handler);
    vi.stubGlobal("useRuntimeConfig", () => ({
      public: {
        oidcBase: "https://auth.obiente.cloud",
        oidcClientId: "dashboard-client",
        requestHost: "https://cloud.obiente.org",
      },
    }));
    vi.stubGlobal("handlePKCE", async () => ({
      code_challenge: "test-challenge",
      code_challenge_method: "S256",
    }));
    vi.stubGlobal("sendRedirect", sendRedirect);

    const { default: oauthLogin } = await import(
      "../server/routes/auth/oauth-login"
    );

    await oauthLogin({} as never);

    expect(sendRedirect).toHaveBeenCalledOnce();
    const authorizationUrl = new URL(sendRedirect.mock.calls[0][1] as string);

    expect(authorizationUrl.origin).toBe("https://auth.obiente.cloud");
    expect(authorizationUrl.pathname).toBe("/oauth/v2/authorize");
    expect(authorizationUrl.searchParams.get("redirect_uri")).toBe(
      "https://cloud.obiente.org/auth/callback"
    );
    expect(authorizationUrl.searchParams.get("code_challenge")).toBe(
      "test-challenge"
    );
    expect(authorizationUrl.searchParams.get("code_challenge_method")).toBe(
      "S256"
    );
    expect(authorizationUrl.searchParams.has("prompt")).toBe(false);
  });
});
