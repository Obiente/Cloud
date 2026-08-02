import type { H3Event } from "h3";
import { createError, deleteCookie, getCookie, setCookie } from "h3";
import {
  buildGitHubAppCallbackUrl,
  createGitHubAppInstallNonceCore,
  createGitHubAppInstallPKCECore,
  decodeGitHubAppInstallPKCEVerifier,
  decodeGitHubAppInstallStateCore,
  encodeGitHubAppInstallPKCEState,
  encodeGitHubAppInstallStateCore,
  GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS,
  verifyGitHubAppInstallStateCore,
} from "./githubAppInstallStateCore";
import type { GitHubAppInstallStatePayload } from "./githubAppInstallStateCore";

export const createGitHubAppInstallNonce = createGitHubAppInstallNonceCore;
export const createGitHubAppInstallPKCE = createGitHubAppInstallPKCECore;
export const decodeGitHubAppInstallState = decodeGitHubAppInstallStateCore;

export const GITHUB_APP_INSTALL_STATE_COOKIE = "github_app_install_state";
export const GITHUB_APP_INSTALL_PKCE_COOKIE = "github_app_install_pkce";

export function encodeGitHubAppInstallState(
  event: H3Event,
  state: Omit<GitHubAppInstallStatePayload, "issuedAt">
): string {
  return encodeGitHubAppInstallStateCore(
    getGitHubAppInstallStateSecret(event),
    state
  );
}

export function getGitHubAppInstallStateCookie(
  event: H3Event
): string | undefined {
  return getCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE);
}

export function setGitHubAppInstallStateCookie(
  event: H3Event,
  state: string
): void {
  setInstallCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE, state);
}

export function setGitHubAppInstallPKCECookie(
  event: H3Event,
  stateRandom: string,
  codeVerifier: string
): void {
  setInstallCookie(
    event,
    GITHUB_APP_INSTALL_PKCE_COOKIE,
    encodeGitHubAppInstallPKCEState(
      getGitHubAppInstallStateSecret(event),
      stateRandom,
      codeVerifier
    )
  );
}

export function getGitHubAppInstallPKCEVerifier(
  event: H3Event,
  stateRandom: string
): string | undefined {
  return decodeGitHubAppInstallPKCEVerifier(
    getGitHubAppInstallStateSecret(event),
    getCookie(event, GITHUB_APP_INSTALL_PKCE_COOKIE),
    stateRandom
  );
}

export function clearGitHubAppInstallStateCookie(event: H3Event): void {
  clearInstallCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE);
}

export function clearGitHubAppInstallPKCECookie(event: H3Event): void {
  clearInstallCookie(event, GITHUB_APP_INSTALL_PKCE_COOKIE);
}

export function verifyGitHubAppInstallState(
  event: H3Event,
  expectedState: string | undefined,
  actualState: string | undefined
): boolean {
  return verifyGitHubAppInstallStateCore(
    getGitHubAppInstallStateSecret(event),
    expectedState,
    actualState
  );
}

export function getGitHubAppCallbackUrl(event: H3Event): string {
  const config = useRuntimeConfig(event);
  const requestHost = String(config.public.requestHost || "").trim();

  try {
    return buildGitHubAppCallbackUrl(requestHost);
  } catch (error: unknown) {
    throw createError({
      statusCode: 500,
      statusMessage:
        error instanceof Error
          ? error.message
          : "DASHBOARD_URL is not a valid absolute URL",
    });
  }
}

function setInstallCookie(event: H3Event, name: string, value: string): void {
  setCookie(event, name, value, {
    httpOnly: true,
    maxAge: GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS,
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

function clearInstallCookie(event: H3Event, name: string): void {
  deleteCookie(event, name, {
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

function getGitHubAppInstallStateSecret(event: H3Event): string {
  const config = useRuntimeConfig(event);
  const secret =
    config.session?.password ||
    process.env.NUXT_SESSION_PASSWORD ||
    process.env.SESSION_SECRET ||
    process.env.SECRET ||
    "";

  if (
    typeof secret !== "string" ||
    secret.length < 32 ||
    secret === "changeme_dashboard_session_password_please_override" ||
    secret === "changeme_GENERATE_64_CHAR_RANDOM_STRING_HERE"
  ) {
    throw createError({
      statusCode: 500,
      statusMessage:
        "GitHub App install state signing requires NUXT_SESSION_PASSWORD or another strong server secret",
    });
  }

  return secret;
}
