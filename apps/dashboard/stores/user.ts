import { defineStore } from 'pinia';

// TODO: Replace with your Zitadel OIDC config
const OIDC_CONFIG = {
  authority: 'https://auth.obiente.cloud/oauth/v2',
  clientId: 'your-client-id',
  redirectUri: '/auth/callback',
  postLogoutRedirectUri: '/',
  scope: 'openid profile email',
  responseType: 'code',
};
const config = useRuntimeConfig();
export const useUserStore = defineStore('user', () => {
  // State
  const isAuthenticated = ref(false);
  const user = ref<any>(null);
  const accessToken = ref<string | null>(null);
  const idToken = ref<string | null>(null);
  const refreshToken = ref<string | null>(null);
  const expiresAt = ref<number | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  const router = useRouter();

  // Getters
  const isLoggedIn = computed(() => isAuthenticated.value && !!user.value);
  const userName = computed(
    () => user.value?.name || user.value?.preferred_username || user.value?.email || ''
  );

  // Actions
  function login() {
    // Redirect to Zitadel OIDC authorize endpoint
    const params = new URLSearchParams({
      client_id: config.public.oidcClientId,
      redirect_uri: config.public.oidcBase + OIDC_CONFIG.redirectUri,
      response_type: OIDC_CONFIG.responseType,
      scope: OIDC_CONFIG.scope,
    });
    window.location.href = `${OIDC_CONFIG.authority}/authorize?${params.toString()}`;
  }

  async function handleCallback() {
    // Parse code from URL
    const url = new URL(window.location.href);
    const code = url.searchParams.get('code');
    if (!code) {
      error.value = 'No code found in callback URL.';
      return;
    }
    loading.value = true;
    try {
      // Exchange code for tokens
      const tokenRes = await fetch(`${OIDC_CONFIG.authority}/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'authorization_code',
          code,
          redirect_uri: OIDC_CONFIG.redirectUri,
          client_id: OIDC_CONFIG.clientId,
        }),
      });
      const tokenData = await tokenRes.json();
      if (!tokenRes.ok) throw new Error(tokenData.error_description || 'Token exchange failed');
      accessToken.value = tokenData.access_token;
      idToken.value = tokenData.id_token;
      refreshToken.value = tokenData.refresh_token;
      expiresAt.value = Date.now() + tokenData.expires_in * 1000;
      // Decode user info from id_token (JWT)
      user.value = idToken.value ? parseJwt(idToken.value) : null;
      isAuthenticated.value = true;
      error.value = null;
      // Clean up URL
      router.replace({ path: '/' });
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
    user.value = null;
    accessToken.value = null;
    idToken.value = null;
    refreshToken.value = null;
    expiresAt.value = null;
    // Redirect to Zitadel logout
    const params = new URLSearchParams({
      client_id: OIDC_CONFIG.clientId,
      post_logout_redirect_uri: OIDC_CONFIG.postLogoutRedirectUri,
    });
    window.location.href = `${OIDC_CONFIG.authority}/logout?${params.toString()}`;
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
      const res = await fetch(`${OIDC_CONFIG.authority}/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'refresh_token',
          refresh_token: refreshToken.value,
          client_id: OIDC_CONFIG.clientId,
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error_description || 'Token refresh failed');
      accessToken.value = data.access_token;
      idToken.value = data.id_token;
      refreshToken.value = data.refresh_token;
      expiresAt.value = Date.now() + data.expires_in * 1000;
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
    if (typeof window === 'undefined') return;
    try {
      const stored = window.localStorage.getItem('obiente_user');
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

  // Persist session to localStorage
  watch([accessToken, idToken, refreshToken, expiresAt, user], () => {
    if (typeof window === 'undefined') return;
    window.localStorage.setItem(
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
  };
});
