import type { ZitadelTokenResponse } from "@obiente/types";
import { getUserData, AUTH_COOKIE_NAME } from "../../utils/auth";

export default defineEventHandler(async (event) => {
  try {
    const body = await readBody<{
      name: string;
      email: string;
      password: string;
    }>(event);

    if (!body.name || !body.email || !body.password) {
      throw createError({
        statusCode: 400,
        message: "Name, email, and password are required",
      });
    }

    // Validate password strength
    if (body.password.length < 8) {
      throw createError({
        statusCode: 400,
        message: "Password must be at least 8 characters",
      });
    }

    const config = useRuntimeConfig();

    // Create user via Zitadel Management API or User API
    // This requires proper API credentials and setup
    // For now, we'll use a simplified approach that creates the user
    // and then authenticates them
    
    // Note: In production, you'd need to:
    // 1. Get a service account token from Zitadel
    // 2. Use Zitadel's Management API to create the user
    // 3. Then authenticate the user
    
    // For this implementation, we'll attempt to create via OAuth2 registration endpoint
    // or use Zitadel's User API if available
    
    let userId: string;
    
    try {
      // Try to create user via Zitadel's User API
      // This requires a service account with proper permissions
      const createUserResponse = await $fetch<{ userId: string }>(
        `${config.public.oidcBase}/management/v1/users/human/_import`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            // In production, you'd need proper authentication here
            // Authorization: `Bearer ${serviceAccountToken}`,
          },
          body: {
            userName: body.email,
            email: {
              email: body.email,
              isEmailVerified: false, // User will need to verify email
            },
            profile: {
              firstName: body.name.split(" ")[0] || body.name,
              lastName: body.name.split(" ").slice(1).join(" ") || "",
            },
            password: {
              password: body.password,
              changeRequired: false,
            },
          },
        }
      ).catch(async () => {
        // If Management API fails, try OAuth2 registration flow
        // This requires the registration endpoint to be enabled
        throw new Error("User creation via Management API not available");
      });
      
      userId = createUserResponse.userId;
    } catch (err: any) {
      // Fallback: Use OAuth2 registration endpoint if available
      // This is a simplified approach - in production, use Management API
      console.warn("User creation via Management API failed, using OAuth2 registration:", err);
      
      // For now, return an error suggesting to use the web registration
      throw createError({
        statusCode: 501,
        message: "Direct signup is not available. Please use the web registration flow.",
      });
    }

    // After user creation, authenticate them
    let tokenResponse: ZitadelTokenResponse;

    try {
      tokenResponse = await $fetch<ZitadelTokenResponse>(
        `${config.public.oidcBase}/oauth/v2/token`,
        {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: new URLSearchParams({
            grant_type: "password",
            username: body.email,
            password: body.password,
            client_id: config.public.oidcClientId,
            scope: "openid profile email offline_access",
          }),
        }
      );
    } catch (err: any) {
      throw createError({
        statusCode: 500,
        message: "Account created but authentication failed. Please try logging in.",
      });
    }

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
    const maxAge = expirySeconds - 60;

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
      message: "Account created successfully",
    };
  } catch (error: any) {
    console.error("Signup error:", error);
    throw createError({
      statusCode: error.statusCode || 500,
      message: error.message || "Signup failed",
    });
  }
});

