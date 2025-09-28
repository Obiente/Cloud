import { appendResponseHeader } from 'h3';
import type { User, Organization, UserSession } from '@obiente/types';

export const useAuth = () => {
  const serverEvent = import.meta.server ? useRequestEvent() : null;

  // Reactive state
  const sessionState = useState<UserSession | null>('obiente-session', () => null);
  const authReadyState = useState('obiente-auth-ready', () => false);
  const user = computed(() => sessionState.value?.user || null);
  const currentOrganization = ref<Organization | null>(null);
  const isAuthenticated = computed(() => sessionState.value && user.value);
  const isLoading = ref(false);

  // Get current user session
  const fetch = async () => {
    try {
      isLoading.value = true;
      sessionState.value = await useRequestFetch()<UserSession>('/auth/session', {
        headers: {
          accept: 'application/json',
        },
        retry: false,
      }).catch(() => null);

      if (!authReadyState.value) {
        authReadyState.value = true;
      }
    } catch (error) {
      console.error('Failed to get current user:', error);
      sessionState.value = null;
    } finally {
      isLoading.value = false;
    }
  };

  // Logout function
  const logout = async () => {
    await useRequestFetch()('/auth/session', {
      method: 'DELETE',
      onResponse({ response: { headers } }) {
        if (import.meta.server && serverEvent) {
          for (const setCookie of headers.getSetCookie()) {
            appendResponseHeader(serverEvent, 'Set-Cookie', setCookie);
          }
        }
      },
    });
    sessionState.value = null;
    currentOrganization.value = null;
    authReadyState.value = false;
  };

  // Switch organization
  const switchOrganization = async (organizationId: string) => {
    try {
      // TODO: Implement organization switching
      // const response = await $fetch(`/api/organizations/${organizationId}/switch`, {
      //   method: 'POST'
      // });
      // currentOrganization.value = response.organization;

      console.log('Switching to organization:', organizationId);
    } catch (error) {
      console.error('Failed to switch organization:', error);
      throw error;
    }
  };

  // Popup authentication support
  const popupListener = (e: StorageEvent) => {
    if (e.key === 'temp-nuxt-auth-utils-popup') {
      fetch();
      window.removeEventListener('storage', popupListener);
    }
  };

  const openInPopup = (route: string, size: { width?: number; height?: number } = {}) => {
    const width = size.width ?? 960;
    const height = size.height ?? 600;
    const top = (window.top?.outerHeight ?? 0) / 2 + (window.top?.screenY ?? 0) - height / 2;
    const left = (window.top?.outerWidth ?? 0) / 2 + (window.top?.screenX ?? 0) - width / 2;

    window.open(
      route,
      '_blank',
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );
  };

  // Initialize auth state
  onMounted(() => {
    fetch();
  });

  return {
    // State
    user: user,
    currentOrganization: readonly(currentOrganization),
    session: readonly(sessionState),
    ready: computed(() => authReadyState.value),
    isAuthenticated,
    isLoading: readonly(isLoading),
    fetch,
    logout,
    switchOrganization,
    getCurrentUser: fetch,
    openInPopup,
  };
};
