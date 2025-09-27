import type { RouteLocationNormalizedLoadedGeneric } from 'vue-router';
import type { ZitadelTokenResponse, ZitadelUserProfile } from '@obiente/types';
import type { User } from '@zitadel/proto/zitadel/user_pb';

const AUTH_COOKIE_NAME = 'obiente_auth';
const REFRESH_COOKIE_NAME = 'obiente_refresh';
export const useUser = defineStore('user', () => {
  const config = useRuntimeConfig();
  const OIDC = {
    authority: config.public.oidcBase + '/oauth/v2',
    redirectPath: '/auth/callback',
    postLogoutRedirectUri: '/',
    scope: 'openid profile email',
    responseType: 'code',
    clientId: config.public.oidcClientId,
  };

  // State
  const isAuthenticated = ref(false);
  const user = useState<User | undefined>('user', () => undefined);
  const accessToken = useCookie<string | undefined>(AUTH_COOKIE_NAME);
  const idToken = useState<string | undefined>();
  const refreshToken = useState<string | undefined>();
  const expiresAt = useState<number | undefined>();
  const loading = ref<boolean>();
  const error = ref<string>();

  // Getters
  const isLoggedIn = computed(() => isAuthenticated.value && !!user.value);
  const userName = computed(() => user.value?.userName || user.value?.preferredLoginName);

  // Actions
  async function login() {
    // Redirect to Zitadel OIDC authorize endpoint
    const verifier = await handlePKCE();
    const params = new URLSearchParams({
      client_id: OIDC.clientId,
      redirect_uri: window.location.origin + OIDC.redirectPath,
      response_type: OIDC.responseType,
      scope: OIDC.scope,
      code_challenge: verifier.code_challenge!,
      code_challenge_method: verifier.code_challenge_method!,
      prompt: 'none',
    });
    return `${OIDC.authority}/authorize?${params.toString()}`;
  }

  async function handleCallback(route: RouteLocationNormalizedLoadedGeneric) {
    const { code_verifier } = await handlePKCE(true);
    const code = route.query.code;
    if (typeof code !== 'string') {
      error.value = 'No code found in callback URL.';
      return;
    }
    loading.value = true;
    try {
      // Exchange code for tokens
      const { data: tokenResponse } = await useFetch<ZitadelTokenResponse>(
        `${OIDC.authority}/token`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams({
            grant_type: 'authorization_code',
            code,
            redirect_uri: window.location.origin + OIDC.redirectPath,
            client_id: OIDC.clientId,
            code_verifier,
          }),
        }
      );

      if (!tokenResponse.value) throw new Error('Token exchange failed');
      const tokenData = tokenResponse.value;

      accessToken.value = tokenData.access_token;
      idToken.value = tokenData.id_token;
      refreshToken.value = tokenData.refresh_token;
      expiresAt.value = Date.now() + tokenData.expires_in * 1000;
      // Decode user info from id_token (JWT)
      user.value = idToken.value ? parseJwt(idToken.value) : null;
      isAuthenticated.value = true;
      delete error.value;
      // Clean up URL
    } catch (e: any) {
      error.value = e.message || 'OAuth callback failed.';
      isAuthenticated.value = false;
    } finally {
      loading.value = false;
    }
  }

  function logout() {
    // Clear state
    isAuthenticated.value = false;
    delete user.value;
    delete accessToken.value;
    delete idToken.value;
    delete refreshToken.value;
    delete expiresAt.value;
    // Redirect to Zitadel logout
    const params = new URLSearchParams({
      client_id: OIDC.clientId,
      post_logout_redirect_uri: OIDC.postLogoutRedirectUri,
    });
    window.location.href = `${OIDC.authority}/logout?${params.toString()}`;
  }

  function parseJwt(token: string | null) {
    if (!token) return null;
    const base64Url = token.split('.')[1];
    if (!base64Url) return null;
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
    return JSON.parse(jsonPayload);
  }

  // Token refresh logic (optional, for production)
  async function refreshTokens() {
    if (!refreshToken.value) return;
    try {
      const { data: refreshResponse } = await useFetch<ZitadelTokenResponse>(
        `${OIDC.authority}/token`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams({
            grant_type: 'refresh_token',
            refresh_token: refreshToken.value,
            client_id: OIDC.clientId,
          }),
        }
      );

      if (!refreshResponse.value) throw new Error('Token refresh failed');
      const refreshData = refreshResponse.value;

      accessToken.value = refreshData.access_token;
      idToken.value = refreshData.id_token;
      refreshToken.value = refreshData.refresh_token;
      expiresAt.value = Date.now() + refreshData.expires_in * 1000;
      user.value = idToken.value ? parseJwt(idToken.value) : null;
      isAuthenticated.value = true;
    } catch (e: any) {
      error.value = e.message || 'Token refresh failed.';
      isAuthenticated.value = false;
      logout();
    }
  }

  // Auto-refresh token before expiry (optional, for production)
  let refreshTimeout: any = null;
  function scheduleTokenRefresh() {
    if (refreshTimeout) clearTimeout(refreshTimeout);
    if (!expiresAt.value) return;
    const ms = expiresAt.value - Date.now() - 60000; // 1 min before expiry
    if (ms > 0) {
      refreshTimeout = setTimeout(refreshTokens, ms);
    }
  }

  // Watch for token expiry
  watch(expiresAt, scheduleTokenRefresh);

  // SSR/SPA hydration: try to restore session from localStorage (optional)
  function restoreSession() {
    try {
      const stored = localStorage.getItem('obiente_user');
      if (stored) {
        const parsed = JSON.parse(stored);
        accessToken.value = parsed.accessToken;
        idToken.value = parsed.idToken;
        refreshToken.value = parsed.refreshToken;
        expiresAt.value = parsed.expiresAt;
        user.value = parsed.user;
        isAuthenticated.value = !!parsed.accessToken;
      }
    } catch {}
  }
  // Fetch current user from Zitadel
  async function fetchUser() {
    if (!accessToken.value) return undefined;

    try {
      const { data, error: fetchError } = await useFetch<User>('/api/auth/me', {
        headers: {
          Authorization: `Bearer ${accessToken.value}`,
        },
      });

      if (fetchError) {
        error.value = 'Failed to fetch user profile';
        return undefined;
      }

      return data.value;
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch user profile';
      return undefined;
    }
  }

  // // Check auth status on hydration
  // async function restoreSession() {
  //   const { data: session } = await useFetch('/api/auth/session');
  //   if (session.value?.authenticated) {
  //     isAuthenticated.value = true;
  //     user.value = session.value.user;
  //     expiresAt.value = session.value.expiresAt;
  //   }
  // }
  // Persist session to localStorage
  watch([accessToken, idToken, refreshToken, expiresAt, user], () => {
    localStorage.setItem(
      'obiente_user',
      JSON.stringify({
        accessToken: accessToken.value,
        idToken: idToken.value,
        refreshToken: refreshToken.value,
        expiresAt: expiresAt.value,
        user: user.value,
      })
    );
  });

  // Expose state and actions
  return {
    isAuthenticated,
    isLoggedIn,
    user,
    userName,
    accessToken,
    idToken,
    refreshToken,
    loading,
    error,
    login,
    handleCallback,
    logout,
    refreshTokens,
    restoreSession,
    fetchUser,
  };
});
if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useUser, import.meta.hot));
}
