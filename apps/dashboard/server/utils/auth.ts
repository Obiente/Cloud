import type { User, UserSession, ZitadelTokenResponse } from "@obiente/types";
import { useRequestEvent } from "nuxt/app";
import type { H3Event } from "h3";

export const AUTH_COOKIE_NAME = "obiente_auth";
export const REFRESH_COOKIE_NAME = "obiente_refresh";

const config = useRuntimeConfig();
export async function exchangeCodeForTokens(
  code: string,
  code_verifier: string,
  redirect_uri: string
): Promise<ZitadelTokenResponse> {
  const response = await $fetch<ZitadelTokenResponse>(
    `${config.public.oidcBase}/oauth/v2/token`,
    {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        code,
        code_verifier,
        redirect_uri,
        client_id: config.public.oidcClientId,
      }),
    }
  ).catch((error) => {
    throw new Error(error);
  });
  return response;
}

export async function getUserData(
  event: H3Event,
  session: UserSession
): Promise<void> {
  const config = useRuntimeConfig();

  const response = await $fetch<User>(
    `${config.public.oidcBase}/oidc/v1/userinfo`,
    {
      headers: {
        Authorization: `Bearer ${session.secure?.access_token}`,
      },
    }
  ).catch(() => null);
  if (response) await setUserSession(event, { user: response });
}
