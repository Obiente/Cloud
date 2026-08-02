import {
  clearGitHubAppInstallPKCECookie,
  clearGitHubAppInstallStateCookie,
  createGitHubAppInstallNonce,
  createGitHubAppInstallPKCE,
  decodeGitHubAppInstallState,
  encodeGitHubAppInstallState,
  getGitHubAppCallbackUrl,
  getGitHubAppInstallPKCEVerifier,
  getGitHubAppInstallStateCookie,
  setGitHubAppInstallPKCECookie,
  setGitHubAppInstallStateCookie,
  verifyGitHubAppInstallState,
} from "../../../utils/githubAppInstallState";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const state = queryValue(query.state);
  const installationIdValue = queryValue(query.installation_id);
  const setupCode = queryValue(query.code);
  const setupAction = queryValue(query.setup_action);
  const authorizationError = queryValue(query.error);

  const redirectToSettings = (reason: string) =>
    sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        reason
      )}`
    );

  // GitHub can call the setup URL after an installation is updated directly
  // from GitHub. That callback has no Obiente state and must remain
  // non-mutating; the settings page reloads the already-authorized integration.
  if (!state && setupAction === "update") {
    clearInstallCookies(event);
    const installationId = parseInstallationId(installationIdValue);
    if (!installationId) {
      return redirectToSettings("missing_installation");
    }
    const updateQuery = new URLSearchParams({
      tab: "integrations",
      provider: "github",
      success: "true",
      installationUpdated: "true",
      installationId: String(installationId),
    });
    return sendRedirect(event, `/settings?${updateQuery.toString()}`);
  }

  const expectedState = getGitHubAppInstallStateCookie(event);
  if (!verifyGitHubAppInstallState(event, expectedState, state)) {
    clearInstallCookies(event);
    return redirectToSettings("invalid_state");
  }

  let stateData: ReturnType<typeof decodeGitHubAppInstallState>;
  try {
    stateData = decodeGitHubAppInstallState(state);
  } catch {
    clearInstallCookies(event);
    return redirectToSettings("invalid_state");
  }
  clearGitHubAppInstallStateCookie(event);

  if (authorizationError) {
    clearGitHubAppInstallPKCECookie(event);
    return redirectToSettings(
      authorizationError === "access_denied"
        ? "github_authorization_cancelled"
        : "github_authorization_failed"
    );
  }

  const installationIdFromQuery = parseInstallationId(installationIdValue);
  const installationIdFromState = parseInstallationId(
    stateData.installationId || ""
  );
  const installationId = installationIdFromQuery || installationIdFromState;
  if (!installationId) {
    clearGitHubAppInstallPKCECookie(event);
    return redirectToSettings("missing_installation");
  }

  if (!setupCode) {
    clearGitHubAppInstallPKCECookie(event);

    const config = useRuntimeConfig(event);
    const clientId =
      String(config.githubAppClientId || "").trim() ||
      process.env.GITHUB_APP_CLIENT_ID ||
      "";
    if (!clientId) {
      return redirectToSettings("github_app_client_not_configured");
    }

    let callbackUrl: string;
    try {
      callbackUrl = getGitHubAppCallbackUrl(event);
    } catch (error: unknown) {
      logGitHubAppError("Invalid callback URL configuration", error);
      return redirectToSettings("github_app_callback_not_configured");
    }

    const authStateData = {
      random: createGitHubAppInstallNonce(),
      orgId: stateData.orgId,
      installationId: String(installationId),
      setupAction: setupAction || stateData.setupAction || "",
    };
    const authState = encodeGitHubAppInstallState(event, authStateData);
    const { codeVerifier, codeChallenge } = createGitHubAppInstallPKCE();
    setGitHubAppInstallStateCookie(event, authState);
    setGitHubAppInstallPKCECookie(event, authStateData.random, codeVerifier);

    const authUrl = new URL("https://github.com/login/oauth/authorize");
    authUrl.searchParams.set("client_id", clientId);
    authUrl.searchParams.set("state", authState);
    authUrl.searchParams.set("redirect_uri", callbackUrl);
    authUrl.searchParams.set("code_challenge", codeChallenge);
    authUrl.searchParams.set("code_challenge_method", "S256");
    return sendRedirect(event, authUrl.toString());
  }

  const codeVerifier = getGitHubAppInstallPKCEVerifier(event, stateData.random);
  clearInstallCookies(event);
  if (!codeVerifier) {
    return redirectToSettings("invalid_authorization_session");
  }

  const isAuthDisabled = process.env.DISABLE_AUTH === "true";
  const { getServerToken } = await import("../../../utils/serverAuth");
  let userAccessToken = await getServerToken(event);
  if (!userAccessToken && isAuthDisabled) {
    userAccessToken = "dev-dummy-token";
  }
  if (!userAccessToken) {
    return redirectToSettings("login_required");
  }

  try {
    const config = useRuntimeConfig(event);
    const { createConnectTransport } = await import("@connectrpc/connect-node");
    const { createClient } = await import("@connectrpc/connect");
    const { createAuthInterceptor } = await import("~/lib/transport");
    const { AuthService, ConnectOrganizationGitHubAppRequestSchema } =
      await import("@obiente/proto");
    const { create } = await import("@bufbuild/protobuf");

    const getToken = () => Promise.resolve(userAccessToken || undefined);
    const authInterceptor = createAuthInterceptor(getToken);
    const apiHost =
      String(config.apiHostInternal || "").trim() ||
      String(config.public.apiHost || "").trim();
    const transport = createConnectTransport({
      baseUrl: apiHost,
      httpVersion: "1.1",
      useBinaryFormat: false,
      interceptors: [authInterceptor],
      defaultTimeoutMs: 30_000,
    });
    const client = createClient(AuthService, transport);

    const request = create(ConnectOrganizationGitHubAppRequestSchema, {
      organizationId: stateData.orgId,
      installationId: BigInt(installationId),
      setupCode,
      codeVerifier,
      redirectUri: getGitHubAppCallbackUrl(event),
    });

    // The setup code can be exchanged only once. Do not retry this RPC against
    // another endpoint after an ambiguous network failure.
    await client.connectOrganizationGitHubApp(request);

    const successQuery = new URLSearchParams({
      tab: "integrations",
      provider: "github",
      success: "true",
      orgId: stateData.orgId,
      installationId: String(installationId),
    });
    if ((stateData.setupAction || setupAction) === "update") {
      successQuery.set("installationUpdated", "true");
    }
    return sendRedirect(event, `/settings?${successQuery.toString()}`);
  } catch (error: unknown) {
    logGitHubAppError("Failed to save installation", error);
    return redirectToSettings(githubAppConnectionErrorCode(error));
  }
});

function queryValue(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function parseInstallationId(value: string): number | undefined {
  if (!/^\d+$/.test(value)) {
    return undefined;
  }
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
}

function clearInstallCookies(
  event: Parameters<typeof clearGitHubAppInstallStateCookie>[0]
) {
  clearGitHubAppInstallStateCookie(event);
  clearGitHubAppInstallPKCECookie(event);
}

function githubAppConnectionErrorCode(error: unknown): string {
  const message = errorMessage(error).toLowerCase();
  if (
    message.includes("permission_denied") ||
    message.includes("permission denied")
  ) {
    return "github_installer_not_authorized";
  }
  if (
    message.includes("failed_precondition") ||
    message.includes("failed precondition")
  ) {
    return "github_installation_verification_failed";
  }
  if (
    message.includes("already_exists") ||
    message.includes("already exists")
  ) {
    return "github_installation_already_connected";
  }
  return "github_app_connection_failed";
}

function logGitHubAppError(context: string, error: unknown): void {
  const errorRecord =
    error && typeof error === "object"
      ? (error as Record<string, unknown>)
      : undefined;
  console.error(`[GitHub App] ${context}:`, {
    message: errorMessage(error),
    code: errorRecord?.code,
  });
}

function errorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : String(error || "unknown error");
}
