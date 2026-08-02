import {
  createHash,
  createHmac,
  randomBytes,
  timingSafeEqual,
} from "node:crypto";

export const GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS = 10 * 60;
const GITHUB_APP_INSTALL_STATE_CLOCK_SKEW_SECONDS = 30;

export interface GitHubAppInstallStatePayload {
  random: string;
  orgId: string;
  issuedAt: number;
  installationId?: string;
  setupAction?: string;
}

interface GitHubAppInstallPKCEState {
  stateRandom: string;
  codeVerifier: string;
  issuedAt: number;
}

export function createGitHubAppInstallNonceCore(): string {
  return randomBytes(32).toString("base64url");
}

export function createGitHubAppInstallPKCECore(): {
  codeVerifier: string;
  codeChallenge: string;
} {
  const codeVerifier = randomBytes(32).toString("base64url");
  return {
    codeVerifier,
    codeChallenge: createHash("sha256")
      .update(codeVerifier, "ascii")
      .digest("base64url"),
  };
}

export function buildGitHubAppCallbackUrl(requestHost: string): string {
  let baseUrl: URL;
  try {
    baseUrl = new URL(requestHost.trim());
  } catch {
    throw new Error("DASHBOARD_URL is not a valid absolute URL");
  }

  if (
    !["http:", "https:"].includes(baseUrl.protocol) ||
    baseUrl.username ||
    baseUrl.password
  ) {
    throw new Error("DASHBOARD_URL must use HTTP or HTTPS without credentials");
  }

  return new URL("/api/github/app/callback", baseUrl.origin).toString();
}

export function encodeGitHubAppInstallStateCore(
  secret: string,
  state: Omit<GitHubAppInstallStatePayload, "issuedAt">
): string {
  return encodeSignedPayload(secret, {
    ...state,
    issuedAt: nowSeconds(),
  });
}

export function decodeGitHubAppInstallStateCore(
  state: string
): GitHubAppInstallStatePayload {
  const parsed = decodeSignedPayload(state);
  if (!parsed || typeof parsed !== "object") {
    throw new Error("invalid GitHub App install state payload");
  }

  const { random, orgId, issuedAt, installationId, setupAction } = parsed as {
    random?: unknown;
    orgId?: unknown;
    issuedAt?: unknown;
    installationId?: unknown;
    setupAction?: unknown;
  };

  if (typeof random !== "string" || random.length < 32 || random.length > 128) {
    throw new Error("invalid GitHub App install nonce");
  }
  if (typeof orgId !== "string" || orgId.length === 0 || orgId.length > 255) {
    throw new Error("GitHub App install state is missing orgId");
  }
  if (typeof issuedAt !== "number" || !Number.isInteger(issuedAt)) {
    throw new Error("GitHub App install state is missing issuedAt");
  }
  assertFreshState(issuedAt);

  return {
    random,
    orgId,
    issuedAt,
    installationId:
      typeof installationId === "string" && installationId.length > 0
        ? installationId
        : undefined,
    setupAction:
      typeof setupAction === "string" && setupAction.length > 0
        ? setupAction
        : undefined,
  };
}

export function verifyGitHubAppInstallStateCore(
  secret: string,
  expectedState: string | undefined,
  actualState: string | undefined
): boolean {
  return Boolean(
    expectedState &&
      actualState &&
      safeEqual(expectedState, actualState) &&
      verifySignedPayload(secret, actualState)
  );
}

export function encodeGitHubAppInstallPKCEState(
  secret: string,
  stateRandom: string,
  codeVerifier: string
): string {
  return encodeSignedPayload(secret, {
    stateRandom,
    codeVerifier,
    issuedAt: nowSeconds(),
  });
}

export function decodeGitHubAppInstallPKCEVerifier(
  secret: string,
  value: string | undefined,
  stateRandom: string
): string | undefined {
  if (!value || !verifySignedPayload(secret, value)) {
    return undefined;
  }

  let parsed: unknown;
  try {
    parsed = decodeSignedPayload(value);
  } catch {
    return undefined;
  }
  if (!parsed || typeof parsed !== "object") {
    return undefined;
  }

  const {
    stateRandom: expectedRandom,
    codeVerifier,
    issuedAt,
  } = parsed as Partial<GitHubAppInstallPKCEState>;
  if (
    typeof expectedRandom !== "string" ||
    !safeEqual(expectedRandom, stateRandom) ||
    typeof codeVerifier !== "string" ||
    !/^[A-Za-z0-9._~-]{43,128}$/.test(codeVerifier) ||
    typeof issuedAt !== "number" ||
    !Number.isInteger(issuedAt)
  ) {
    return undefined;
  }
  try {
    assertFreshState(issuedAt);
  } catch {
    return undefined;
  }
  return codeVerifier;
}

function encodeSignedPayload(secret: string, value: object): string {
  const payload = Buffer.from(JSON.stringify(value), "utf-8").toString(
    "base64url"
  );
  return `${payload}.${signPayload(secret, payload)}`;
}

function decodeSignedPayload(value: string): unknown {
  const [payload, signature, extra] = value.split(".");
  if (!payload || !signature || extra !== undefined) {
    throw new Error("invalid GitHub App install state encoding");
  }

  try {
    return JSON.parse(Buffer.from(payload, "base64url").toString("utf-8"));
  } catch {
    throw new Error("invalid GitHub App install state encoding");
  }
}

function signPayload(secret: string, payload: string): string {
  return createHmac("sha256", secret).update(payload).digest("base64url");
}

function verifySignedPayload(secret: string, value: string): boolean {
  const [payload, signature, extra] = value.split(".");
  if (!payload || !signature || extra !== undefined) {
    return false;
  }
  return safeEqual(signPayload(secret, payload), signature);
}

function safeEqual(expectedValue: string, actualValue: string): boolean {
  const expected = Buffer.from(expectedValue, "utf-8");
  const actual = Buffer.from(actualValue, "utf-8");
  return expected.length === actual.length && timingSafeEqual(expected, actual);
}

function assertFreshState(issuedAt: number): void {
  const now = nowSeconds();
  if (
    issuedAt > now + GITHUB_APP_INSTALL_STATE_CLOCK_SKEW_SECONDS ||
    issuedAt < now - GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS
  ) {
    throw new Error("GitHub App install state has expired");
  }
}

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}
