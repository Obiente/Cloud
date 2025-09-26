import type { User, Organization } from '@obiente/types';

export const useAuth = () => {
  // Reactive state
  const user = ref<User | null>(null);
  const currentOrganization = ref<Organization | null>(null);
  const isAuthenticated = computed(() => !!user.value);
  const isLoading = ref(false);

  // Get current user
  const getCurrentUser = async () => {
    try {
      isLoading.value = true;
      // TODO: Implement actual API call to get current user
      // const response = await $fetch('/api/auth/me');
      // user.value = response.user;
      
      // Placeholder for development
      user.value = {
        id: '1',
        externalId: 'zitadel-user-123',
        email: 'admin@example.com',
        name: 'Admin User',
        avatarUrl: 'https://avatar.iran.liara.run/public',
        // preferences: {},
        createdAt: new Date(),
        updatedAt: new Date(),
      };
    } catch (error) {
      console.error('Failed to get current user:', error);
      user.value = null;
    } finally {
      isLoading.value = false;
    }
  };

  // Login function
  const login = async (redirectUrl?: string) => {
    try {
      // TODO: Implement Zitadel OIDC login
      const config = useRuntimeConfig();
      const loginUrl = `${config.public.zitadelUrl}/oauth/v2/authorize?client_id=${config.public.zitadelClientId}&response_type=code&scope=openid email profile&redirect_uri=${encodeURIComponent(redirectUrl || window.location.origin + '/auth/callback')}`;
      
      // Redirect to Zitadel
      window.location.href = loginUrl;
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  };

  // Logout function
  const logout = async () => {
    try {
      // TODO: Implement actual logout
      // await $fetch('/api/auth/logout', { method: 'POST' });
      
      user.value = null;
      currentOrganization.value = null;
      
      // Redirect to login
      await navigateTo('/login');
    } catch (error) {
      console.error('Logout failed:', error);
    }
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

  // Initialize auth state
  onMounted(() => {
    getCurrentUser();
  });

  return {
    user: readonly(user),
    currentOrganization: readonly(currentOrganization),
    isAuthenticated,
    isLoading: readonly(isLoading),
    login,
    logout,
    switchOrganization,
    getCurrentUser,
  };
};