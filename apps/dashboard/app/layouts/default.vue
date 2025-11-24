<template>
  <div class="bg-surface-base min-h-screen overflow-hidden">
    <!-- Mobile Sidebar Overlay -->
    <Transition name="sidebar-overlay">
      <OuiBox
        v-if="isSidebarOpen && user.user && user.isAuthenticated"
        as="div"
        class="fixed inset-0 z-40 flex xl:hidden"
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
      >
        <OuiBox
          as="div"
          class="absolute inset-0 bg-background/80 backdrop-blur-sm"
          @click="closeSidebar"
          aria-hidden="true"
        />
      </OuiBox>
    </Transition>

    <!-- Mobile Sidebar Panel -->
    <Transition name="sidebar-panel" appear>
      <OuiBox
        v-if="isSidebarOpen"
        as="aside"
        :id="mobileSidebarId"
        class="fixed inset-y-0 left-0 z-50 flex h-full w-72 max-w-[80vw] flex-col xl:hidden"
        role="navigation"
        aria-label="Primary navigation"
      >
        <AppSidebar
          class="sidebar-drawer relative h-full border-r border-border-muted bg-surface-base shadow-2xl"
          :organization-options="organizationOptions"
          :current-organization="currentOrganization"
          :show-super-admin="showSuperAdmin"
          @navigate="closeSidebar"
          @organization-change="switchOrganization"
          @new-organization="$router.push('/organizations')"
        />
        <OuiButton
          variant="ghost"
          size="sm"
          class="absolute right-3 top-3 z-50 p-2! text-text-secondary hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
          @click="closeSidebar"
          aria-label="Close navigation menu"
          title="Close menu"
        >
          <XMarkIcon class="h-5 w-5" />
        </OuiButton>
      </OuiBox>
    </Transition>

    <!-- Authenticated View -->
    <div v-if="user.user && user.isAuthenticated" class="flex min-h-screen">
      <!-- Desktop Sidebar -->
      <div class="hidden xl:block xl:w-64 xl:shrink-0">
        <div class="sticky top-0 h-screen bg-surface-base">
          <AppSidebar
            class="w-full h-full relative z-0"
            :organization-options="organizationOptions"
            :current-organization="currentOrganization"
            :show-super-admin="showSuperAdmin"
            @navigate="closeSidebar"
            @organization-change="switchOrganization"
            @new-organization="$router.push('/organizations')"
          />
        </div>
      </div>

      <!-- Main Frame -->
      <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
        <!-- Header -->
        <div class="sticky top-0 z-30 min-w-0">
          <AppHeader
            class="w-full xl:shadow-sm min-w-0"
            :notification-count="unreadCount"
            @notifications-click="isNotificationsOpen = !isNotificationsOpen"
          >
            <template #leading>
              <OuiButton
                variant="ghost"
                size="sm"
                class="xl:hidden p-2! text-text-secondary hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
                @click="toggleSidebar"
                :aria-expanded="isSidebarOpen"
                :aria-controls="mobileSidebarId"
                aria-label="Toggle navigation menu"
              >
                <Bars3Icon class="h-5 w-5" />
              </OuiButton>
            </template>
            <template #title>
              <slot name="title">Dashboard</slot>
            </template>
          </AppHeader>
        </div>

        <!-- Framed Main Content -->
        <main
          class="flex-1 bg-background overflow-hidden m-0 xl:m-2 xl:ml-0 rounded-none xl:rounded-3xl border-0 xl:border border-muted p-2 xl:p-6 relative min-w-0"
        >
          <div class="absolute inset-0 overflow-hidden rounded-none xl:rounded-xl min-w-0">
            <div class="h-full w-full overflow-y-auto overflow-x-hidden p-2 xl:p-6 min-w-0">
              <slot />
            </div>
          </div>
        </main>

        <!-- Notifications -->
        <AppNotifications
          v-model="isNotificationsOpen"
          :items="Array.from(notifications)"
          :anchor-element="notificationButtonElement"
          @update:items="
            (val) => {
              const notify = useNotifications();
              val.forEach(n => {
                if (n.read) notify.markAsRead(n.id);
              });
            }
          "
          @close="isNotificationsOpen = false"
        />

        <!-- Toast Notifications -->
        <OuiToaster :toaster="toaster" />
      </div>
    </div>

    <!-- Unauthenticated View -->
    <div
      v-else
      class="flex min-h-screen items-center justify-center bg-surface-base"
    >
      <div class="text-center">
        <OuiStack gap="lg" align="center">
          <LockClosedIcon class="h-16 w-16 text-muted" />
          <OuiStack gap="xs">
            <OuiText size="2xl" weight="bold">Authentication Required</OuiText>
            <OuiText color="muted">Please sign in to access the dashboard.</OuiText>
          </OuiStack>
          <OuiFlex v-if="!user.isLoading" gap="md" align="center" justify="center">
            <OuiButton 
              size="lg"
              variant="outline"
              @click="user.popupSignup()"
            >
              Sign Up
            </OuiButton>
            <OuiButton 
              size="lg"
              @click="user.popupLogin()"
            >
              Sign In
            </OuiButton>
          </OuiFlex>
          <OuiStack v-else gap="sm" align="center">
            <ArrowPathIcon class="h-6 w-6 text-muted animate-spin" />
            <OuiText size="sm" color="muted">Loading...</OuiText>
          </OuiStack>
          <OuiButton
            v-if="!user.isLoading"
            variant="ghost"
            size="md"
            @click="navigateTo('/')"
            class="mt-4"
          >
            <HomeIcon class="h-4 w-4 mr-2" />
            Go back home
          </OuiButton>
        </OuiStack>
      </div>
    </div>
  </div>
</template>
<style>
  /* Apply to html and body for global effect */
  html,
  body {
    scrollbar-width: thin;
    scrollbar-color: var(--scroll-thumb) var(--scroll-track);
  }

  /* WebKit (Chrome, Edge, Safari) */
  ::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }

  ::-webkit-scrollbar-track {
    background: var(--scroll-track);
    border-radius: 8px;
  }

  ::-webkit-scrollbar-thumb {
    background-color: var(--scroll-thumb);
    border-radius: 8px;
    border: 2px solid var(--scroll-track);
  }
</style>
<script setup lang="ts">
  import { onBeforeUnmount, onMounted, computed, ref, watch, type ComponentPublicInstance } from "vue";
  import { Bars3Icon, XMarkIcon, LockClosedIcon, ArrowPathIcon, HomeIcon } from "@heroicons/vue/24/outline";
  import AppHeader from "~/components/app/AppHeader.vue";

  // Pinia user store
  const user = useAuth();
  const superAdmin = useSuperAdmin();
  const config = useConfig();
  
  // Fetch config on mount (non-blocking)
  // On server, await briefly then continue
  // On client, fetch in background
  if (import.meta.server) {
    const configPromise = config.fetchConfig();
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 300));
    await Promise.race([configPromise, timeoutPromise]);
  } else {
    // Client-side: fetch in background, don't block
    config.fetchConfig().catch(() => null);
  }

  // Reset superadmin state when user logs in or changes
  watch(() => user.isAuthenticated, (isAuthenticated) => {
    if (isAuthenticated) {
      // Reset superadmin state when user logs in to force fresh check
      superAdmin.reset();
    }
  }, { immediate: true });

  // Fetch superadmin overview (non-blocking)
  // On server, await briefly then continue
  // On client, fetch in background
  if (import.meta.server) {
    const overviewPromise = superAdmin.fetchOverview();
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 300));
    await Promise.race([overviewPromise, timeoutPromise]);
  } else {
    // Client-side: fetch in background, don't block
    superAdmin.fetchOverview().catch(() => null);
  }
  
  // Show superadmin sidebar if allowed is explicitly true (not null or false)
  const showSuperAdmin = computed(() => superAdmin.allowed.value === true);
  // Notifications state
  const { notifications, unreadCount } = useNotifications();
  const isNotificationsOpen = ref(false);
  const headerRef = ref<ComponentPublicInstance<typeof AppHeader> | null>(null);
  const notificationButtonElement = computed(() => {
    if (import.meta.client && headerRef.value?.notificationButtonRef) {
      return headerRef.value.notificationButtonRef;
    }
    return null;
  });
  // Toast notifications
  const { toaster } = useToast();

  // Use robust sidebar composable with Tailwind breakpoint
  const sidebar = useSidebar({
    desktopBreakpoint: "xl", // Matches xl:hidden, xl:block classes
    lockBodyScroll: true,
    trapFocus: true,
  });
  
  const isSidebarOpen = sidebar.isOpen;
  const mobileSidebarId = sidebar.sidebarId;
  const closeSidebar = sidebar.close;
  const toggleSidebar = sidebar.toggle;

  // Organization switcher data and methods (Connect)
  import { useConnectClient } from "~/lib/connect-client";
  import { OrganizationService } from "@obiente/proto";
  import { useOrganizationLabels } from "~/composables/useOrganizationLabels";
  const orgClient = useConnectClient(OrganizationService);
  const organizations = computed(() => user.organizations || []);
  const { organizationSelectItems } = useOrganizationLabels(organizations);
  const organizationOptions = computed(() => organizationSelectItems.value);
  const currentOrganization = computed(() => user.currentOrganization || null);
  const selectedOrgId = computed({
    get: () => user.currentOrganizationId || undefined,
    set: (id: string | undefined) => {
      if (id) {
        user.switchOrganization(id);
      }
    },
  });

  const { refresh: refreshOrganizations } = await useClientFetch(
    "organizations",
    async () => {
      if (!user.isAuthenticated) return [];
      // Only show user's own organizations in the select, even for superadmins
      const res = await orgClient.listOrganizations({ onlyMine: true });
      user.setOrganizations(res.organizations || []);
      return res.organizations || [];
    },
    {
      watch: [() => user.isAuthenticated],
    }
  );

  // Explicitly refresh organizations when user becomes authenticated
  // This ensures organizations are loaded immediately after login
  watch(() => user.isAuthenticated, (isAuthenticated, wasAuthenticated) => {
    if (isAuthenticated && !wasAuthenticated) {
      // User just logged in - refresh organizations immediately
      refreshOrganizations().catch((err) => {
        console.error("Failed to refresh organizations after login:", err);
      });
    }
  });

  const switchOrganization = async (
    organizationId: string | string[] | undefined
  ) => {
    const id = Array.isArray(organizationId)
      ? organizationId[0]
      : organizationId;
    if (!id) return;
    await user.switchOrganization(id);
  };
</script>
