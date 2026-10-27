<template>
  <div class="app-shell bg-background min-h-screen overflow-hidden">
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
      >
        <AppSidebar
          class="sidebar-drawer relative h-full shadow-2xl"
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
    <OuiFlex v-if="user.user && user.isAuthenticated" class="min-h-screen">
      <!-- Desktop Sidebar -->
      <OuiBox class="hidden xl:block xl:w-60 xl:shrink-0 xl:pl-3 xl:py-3">
        <OuiBox position="sticky" class="top-0 h-[calc(100vh-1.5rem)]">
          <AppSidebar
            class="w-full h-full relative z-0"
            :organization-options="organizationOptions"
            :current-organization="currentOrganization"
            :show-super-admin="showSuperAdmin"
            @navigate="closeSidebar"
            @organization-change="switchOrganization"
            @new-organization="$router.push('/organizations')"
          />
        </OuiBox>
      </OuiBox>

      <!-- Main Frame -->
      <OuiStack class="grow min-w-0 overflow-hidden xl:px-3 xl:py-3" gap="none">
        <!-- Header -->
        <OuiBox position="sticky" class="top-0 z-30 min-w-0 px-2 pt-2 pb-2 xl:px-0 xl:pt-0 xl:pb-2">
          <AppHeader
            class="w-full min-w-0"
            :title="pageContext.title"
            :subtitle="pageContext.section"
            :subtitle-to="currentPageRoute.path.startsWith('/superadmin') ? '/superadmin' : '/admin/quotas'"
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
              <slot name="title">{{ pageContext.title }}</slot>
            </template>
          </AppHeader>
        </OuiBox>

        <!-- Main Content -->
        <main
          class="app-main-frame flex-1 overflow-hidden mt-2 mx-2 mb-2 xl:m-0 rounded-xl p-0 relative min-w-0"
        >
          <OuiBox position="absolute" overflow="hidden" rounded="xl" class="inset-0 min-w-0">
            <OuiBox class="app-main-scroll h-full w-full overflow-y-auto overflow-x-hidden p-4 xl:p-5 min-w-0">
              <slot />
            </OuiBox>
          </OuiBox>
        </main>

        <!-- Notifications -->
        <AppNotifications
          v-model="isNotificationsOpen"
          :items="Array.from(notifications)"
          :is-loading="isLoading"
          @update:items="(val) => val.forEach((n) => n.read && markNotificationAsRead(n.id))"
          @clear="clearAllNotifications"
          @close="isNotificationsOpen = false"
        />

        <!-- Toast Notifications -->
        <OuiToaster :toaster="toaster" />
      </OuiStack>
    </OuiFlex>

    <!-- Unauthenticated View -->
    <OuiFlex
      v-else
      align="center"
      justify="center"
      class="min-h-screen bg-background px-6"
    >
      <OuiStack gap="lg" align="center" class="max-w-sm w-full text-center">
        <OuiStack gap="xs" align="center">
          <LockClosedIcon class="h-10 w-10 text-tertiary mx-auto" />
          <OuiText size="xl" weight="semibold">Sign in to continue</OuiText>
          <OuiText size="sm" color="tertiary">Access your cloud infrastructure dashboard.</OuiText>
        </OuiStack>
        <OuiFlex v-if="!user.isLoading" gap="sm" align="center" justify="center">
          <OuiButton 
            size="md"
            variant="outline"
            @click="user.popupSignup()"
          >
            Sign Up
          </OuiButton>
          <OuiButton 
            size="md"
            color="primary"
            @click="user.popupLogin()"
          >
            Sign In
          </OuiButton>
        </OuiFlex>
        <OuiFlex v-else justify="center">
          <ArrowPathIcon class="h-5 w-5 text-secondary animate-spin" />
        </OuiFlex>
        <OuiButton
          v-if="!user.isLoading"
          variant="ghost"
          size="sm"
          @click="navigateTo('/')"
        >
          <HomeIcon class="h-4 w-4 mr-1.5" />
          Back to home
        </OuiButton>
      </OuiStack>
    </OuiFlex>
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
import { adminPageContext } from "~/utils/adminPresentation";
  import { onBeforeUnmount, onMounted, computed, ref, watch, type ComponentPublicInstance } from "vue";
  import { Bars3Icon, XMarkIcon, LockClosedIcon, ArrowPathIcon, HomeIcon } from "@heroicons/vue/24/outline";
  import AppHeader from "~/components/app/AppHeader.vue";

  // Pinia user store
  const user = useAuth();
  // Resolve one stable session before rendering. Starting unawaited session
  // requests from every useAuth caller could change the SSR tree mid-render.
  await (user.ready ? Promise.resolve() : user.fetch());
  const superAdmin = useSuperAdmin();
  const config = useConfig();
  
  // Keep the same async setup boundary on server and client so Vue generates
  // matching control IDs during hydration. Existing payload state stays intact.
  await Promise.race([
    config.fetchConfig(),
    new Promise(resolve => setTimeout(resolve, 300)),
  ]);

  // A layout mount is not a login. Reset only when the signed-in identity changes.
  watch(() => user.isAuthenticated ? user.user?.sub : null, () => {
    superAdmin.reset();
    if (user.isAuthenticated) void superAdmin.fetchOverview();
  });

  await superAdmin.fetchOverview();

  // Show superadmin sidebar if allowed is explicitly true (not null or false)
  const showSuperAdmin = computed(() => superAdmin.allowed.value === true);
  // Notifications state
  const {
    notifications,
    unreadCount,
    isLoading,
    clearAll: clearAllNotifications,
    markAsRead: markNotificationAsRead,
  } = useNotifications();
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

  const { refresh: refreshOrganizations } = useClientFetch(
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
      server: false,
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
const currentPageRoute = useRoute();
const pageContext = computed(() => adminPageContext(currentPageRoute.path));
</script>
