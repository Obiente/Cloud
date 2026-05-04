import {
  encodeGitHubAppInstallState,
  setGitHubAppInstallStateCookie,
} from "../../../utils/githubAppInstallState";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const orgId = typeof query.orgId === "string" ? query.orgId.trim() : "";

  if (!orgId) {
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        "missing_organization"
      )}`
    );
  }

  const runtimeConfig = useRuntimeConfig(event);
  const appSlug =
    runtimeConfig.public.githubAppSlug ||
    process.env.NUXT_PUBLIC_GITHUB_APP_SLUG ||
    process.env.GITHUB_APP_SLUG ||
    "";

  if (!appSlug) {
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        "github_app_not_configured"
      )}`
    );
  }

  const state = encodeGitHubAppInstallState(event, {
    random: crypto.randomUUID().replace(/-/g, ""),
    orgId,
  });
  setGitHubAppInstallStateCookie(event, state);

  const installUrl = new URL(
    `https://github.com/apps/${encodeURIComponent(appSlug)}/installations/select_target`
  );
  installUrl.searchParams.set("state", state);

  return sendRedirect(event, installUrl.toString());
});
