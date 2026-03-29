import {
  buildGitHubCallbackUrl,
  encodeGitHubOAuthState,
  setGitHubOAuthStateCookie,
} from "../../utils/githubOAuth";

const GITHUB_SCOPE = "repo read:user admin:repo_hook";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const type = query.type === "organization" ? "organization" : "user";
  const orgId = typeof query.orgId === "string" ? query.orgId.trim() : "";

  if (type === "organization" && !orgId) {
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        "missing_organization"
      )}`
    );
  }

  const runtimeConfig = useRuntimeConfig(event);
  const githubClientId = runtimeConfig.public.githubClientId;

  if (!githubClientId) {
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        "configuration_error"
      )}`
    );
  }

  const redirectUri = buildGitHubCallbackUrl(event);
  const state = encodeGitHubOAuthState({
    random: crypto.randomUUID().replace(/-/g, ""),
    type,
    orgId: type === "organization" ? orgId : undefined,
  });

  setGitHubOAuthStateCookie(event, state);

  const authUrl = new URL("https://github.com/login/oauth/authorize");
  authUrl.searchParams.set("client_id", githubClientId);
  authUrl.searchParams.set("redirect_uri", redirectUri);
  authUrl.searchParams.set("scope", GITHUB_SCOPE);
  authUrl.searchParams.set("state", state);
  authUrl.searchParams.set("prompt", "select_account");

  return sendRedirect(event, authUrl.toString());
});
