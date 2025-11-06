import type { ZitadelTokenResponse } from "@obiente/types";
import { getUserData, AUTH_COOKIE_NAME } from "../../utils/auth";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { AuthService } from "@obiente/proto";

export default defineEventHandler(async (event) => {
  try {
    const body = await readBody<{
      email: string;
      password: string;
      rememberMe?: boolean;
    }>(event);

    if (!body.email || !body.password) {
      throw createError({
        statusCode: 400,
        message: "Email and password are required",
      });
    }

    const config = useRuntimeConfig();

    // Create ConnectRPC client for Login (no auth required)
    const transport = createConnectTransport({
      baseUrl: config.public.apiHost,
      httpVersion: "1.1",
      useBinaryFormat: false,
    });

    const client = createClient(AuthService, transport);

    // Call Login RPC method
    const loginResponse = await client.login({
      email: body.email,
      password: body.password,
      rememberMe: body.rememberMe || false,
    });

    if (!loginResponse.success || !loginResponse.accessToken) {
      throw createError({
        statusCode: 401,
        message: loginResponse.message || "Authentication failed",
      });
    }

    // Create a ZitadelTokenResponse-like object for compatibility
    const tokenResponse: ZitadelTokenResponse = {
      access_token: loginResponse.accessToken,
      refresh_token: loginResponse.refreshToken || "",
      expires_in: loginResponse.expiresIn || 3600,
      token_type: "Bearer",
      scope: "openid profile email offline_access",
    };

    // Set the session
    const session = await setUserSession(event, {
      secure: {
        scope: tokenResponse.scope,
        token_type: tokenResponse.token_type,
        expires_in: tokenResponse.expires_in,
        refresh_token: tokenResponse.refresh_token,
        access_token: tokenResponse.access_token,
      },
    });

    await getUserData(event, session);

    // Set the auth cookie
    const expirySeconds = tokenResponse.expires_in || 3600;
    const maxAge = body.rememberMe ? expirySeconds * 7 : expirySeconds - 60;

    setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
      httpOnly: false,
      path: "/",
      maxAge,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      domain: undefined,
    });

    return {
      success: true,
      message: "Login successful",
    };
  } catch (error: any) {
    console.error("Login error:", error);
    throw createError({
      statusCode: error.statusCode || 401,
      message: error.message || "Login failed",
    });
  }
});

