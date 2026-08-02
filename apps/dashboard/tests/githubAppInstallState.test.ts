import { createHash } from "node:crypto";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  buildGitHubAppCallbackUrl,
  createGitHubAppInstallNonceCore,
  createGitHubAppInstallPKCECore,
  decodeGitHubAppInstallPKCEVerifier,
  decodeGitHubAppInstallStateCore,
  encodeGitHubAppInstallPKCEState,
  encodeGitHubAppInstallStateCore,
  resolveGitHubAppCallbackHost,
  verifyGitHubAppInstallStateCore,
} from "../server/utils/githubAppInstallStateCore";

const secret = "a".repeat(64);

describe("GitHub App install state", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-02T18:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("signs, verifies, and decodes a short-lived workspace state", () => {
    const state = encodeGitHubAppInstallStateCore(secret, {
      random: createGitHubAppInstallNonceCore(),
      orgId: "workspace-a",
    });

    expect(verifyGitHubAppInstallStateCore(secret, state, state)).toBe(true);
    expect(decodeGitHubAppInstallStateCore(state)).toMatchObject({
      orgId: "workspace-a",
      issuedAt: 1785693600,
    });
  });

  it("rejects a modified state even when it matches the supplied cookie", () => {
    const state = encodeGitHubAppInstallStateCore(secret, {
      random: createGitHubAppInstallNonceCore(),
      orgId: "workspace-a",
    });
    const [payload] = state.split(".");
    const forged = `${payload}.${"x".repeat(43)}`;

    expect(verifyGitHubAppInstallStateCore(secret, forged, forged)).toBe(false);
  });

  it("rejects state from a different browser flow", () => {
    const expected = encodeGitHubAppInstallStateCore(secret, {
      random: createGitHubAppInstallNonceCore(),
      orgId: "workspace-a",
    });
    const actual = encodeGitHubAppInstallStateCore(secret, {
      random: createGitHubAppInstallNonceCore(),
      orgId: "workspace-a",
    });

    expect(verifyGitHubAppInstallStateCore(secret, expected, actual)).toBe(
      false
    );
  });

  it("rejects state after ten minutes", () => {
    const state = encodeGitHubAppInstallStateCore(secret, {
      random: createGitHubAppInstallNonceCore(),
      orgId: "workspace-a",
    });
    vi.advanceTimersByTime(10 * 60 * 1000 + 1000);

    expect(() => decodeGitHubAppInstallStateCore(state)).toThrow(/expired/i);
  });

  it("creates an RFC 7636 S256 verifier and challenge", () => {
    const { codeVerifier, codeChallenge } = createGitHubAppInstallPKCECore();

    expect(codeVerifier).toMatch(/^[A-Za-z0-9._~-]{43,128}$/);
    expect(codeChallenge).toBe(
      createHash("sha256").update(codeVerifier, "ascii").digest("base64url")
    );
  });

  it("binds the PKCE verifier to the authorization state", () => {
    const stateRandom = createGitHubAppInstallNonceCore();
    const { codeVerifier } = createGitHubAppInstallPKCECore();
    const cookie = encodeGitHubAppInstallPKCEState(
      secret,
      stateRandom,
      codeVerifier
    );

    expect(
      decodeGitHubAppInstallPKCEVerifier(secret, cookie, stateRandom)
    ).toBe(codeVerifier);
    expect(
      decodeGitHubAppInstallPKCEVerifier(
        secret,
        cookie,
        createGitHubAppInstallNonceCore()
      )
    ).toBeUndefined();
  });

  it("builds the callback from the configured public origin", () => {
    expect(
      buildGitHubAppCallbackUrl("https://obiente.cloud/settings?ignored=true")
    ).toBe("https://obiente.cloud/api/github/app/callback");
  });

  it("prefers the canonical dashboard URL supplied at runtime", () => {
    expect(
      resolveGitHubAppCallbackHost("http://localhost:3000", {
        DASHBOARD_URL: "https://obiente.cloud",
        NUXT_PUBLIC_REQUEST_HOST: "https://stale.example",
      })
    ).toBe("https://obiente.cloud");
  });
});
