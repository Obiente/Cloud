<template>
  <div class="bg-surface-base min-h-screen overflow-hidden">
    <!-- Mobile Sidebar Overlay -->
    <Transition name="sidebar-overlay">
      <div
        v-if="isSidebarOpen && user.user && user.isAuthenticated"
        class="fixed inset-0 z-40 flex lg:hidden"
        role="dialog"
        aria-modal="true"
      >
        <div
          class="absolute inset-0 bg-background/80 backdrop-blur-sm"
          @click="closeSidebar"
        />
      </div>
    </Transition>

    <!-- Mobile Sidebar Panel -->
    <Transition name="sidebar-panel" appear>
      <div
        v-if="isSidebarOpen"
        :id="mobileSidebarId"
        class="fixed inset-y-0 left-0 z-50 flex h-full w-72 max-w-[80vw] flex-col lg:hidden"
      >
        <AppSidebar
          class="sidebar-drawer relative h-full overflow-y-auto border-r border-border-muted bg-surface-base shadow-2xl"
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
          aria-label="Close navigation"
        >
          <XMarkIcon class="h-5 w-5" />
        </OuiButton>
      </div>
    </Transition>

    <!-- Authenticated View -->
    <div v-if="user.user && user.isAuthenticated" class="flex min-h-screen">
      <!-- Desktop Sidebar -->
      <div class="hidden lg:block lg:w-64 lg:shrink-0">
        <div class="sticky top-0 h-screen overflow-y-auto bg-surface-base">
          <AppSidebar
            class="w-full relative z-0"
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
      <div class="flex-1 flex flex-col">
        <!-- Header -->
        <div class="sticky top-0 z-30">
          <AppHeader
            class="w-full lg:shadow-sm"
            :notification-count="unreadCount"
            @notifications-click="isNotificationsOpen = !isNotificationsOpen"
          >
            <template #leading>
              <OuiButton
                variant="ghost"
                size="sm"
                class="lg:hidden p-2! text-text-secondary hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
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
          class="flex-1 bg-background overflow-hidden m-2 ml-0 rounded-3xl border border-muted p-6 relative"
        >
          <div class="absolute inset-0 overflow-hidden rounded-xl">
            <div class="h-full w-full overflow-y-auto p-6">
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
      class="main-content flex flex-col justify-center items-center min-h-screen"
    >
      <OuiText v-if="user.isLoading" size="2xl" weight="extrabold"
        >loading</OuiText
      >
      <OuiText v-else-if="user.isAuthenticated" size="2xl" weight="extrabold"
        >logging you in</OuiText
      >
      <OuiButton
        v-else
        size="xl"
        weight="extrabold"
        color="neutral"
        @click="user.popupLogin()"
      >
        Log In
      </OuiButton>
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
  import { Bars3Icon, XMarkIcon } from "@heroicons/vue/24/outline";
  import AppHeader from "~/components/app/AppHeader.vue";

  // Pinia user store
  const user = useAuth();
  const superAdmin = useSuperAdmin();

  // Reset superadmin state when user logs in or changes
  watch(() => user.isAuthenticated, (isAuthenticated) => {
    if (isAuthenticated) {
      // Reset superadmin state when user logs in to force fresh check
      superAdmin.reset();
    }
  }, { immediate: true });

  // Fetch superadmin overview - await on client side too to ensure state is initialized
  if (import.meta.server) {
    await superAdmin.fetchOverview().catch(() => null);
  } else {
    // On client, await the fetch to ensure state is initialized before computing showSuperAdmin
    await superAdmin.fetchOverview().catch(() => null);
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

  const isSidebarOpen = ref(false);
  const mobileSidebarId = "mobile-primary-navigation";

  const closeSidebar = () => {
    isSidebarOpen.value = false;
  };

  const toggleSidebar = () => {
    isSidebarOpen.value = !isSidebarOpen.value;
  };

  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === "Escape") {
      closeSidebar();
    }
  };

  const handleBreakpointChange = () => {
    if (
      import.meta.client &&
      window.matchMedia("(min-width: 1024px)").matches
    ) {
      isSidebarOpen.value = false;
    }
  };

  // Register lifecycle hooks unconditionally (required by Vue)
  onMounted(() => {
    if (import.meta.client) {
      handleBreakpointChange();
      window.addEventListener("keydown", handleKeydown);
      window.addEventListener("resize", handleBreakpointChange);
    }
  });

  onBeforeUnmount(() => {
    if (import.meta.client) {
      window.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("resize", handleBreakpointChange);
    }
  });

  // Organization switcher data and methods (Connect)
  import { useConnectClient } from "~/lib/connect-client";
  import { OrganizationService } from "@obiente/proto";
  const orgClient = useConnectClient(OrganizationService);
  const organizationOptions = computed(() =>
    (user.organizations || []).map((o: any) => ({
      label: o.name || o.slug || o.id,
      value: o.id,
    }))
  );
  const currentOrganization = computed(() => user.currentOrganization || null);
  const selectedOrgId = computed({
    get: () => user.currentOrganizationId || undefined,
    set: (id: string | undefined) => {
      if (id) {
        user.switchOrganization(id);
      }
    },
  });

  const { refresh: refreshOrganizations } = await useAsyncData(
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
