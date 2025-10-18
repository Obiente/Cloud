<template>
  <!-- {{user}} -->
  <div class="bg-background min-h-screen">
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
          @navigate="closeSidebar"
          @organization-change="switchOrganization"
        />

        <OuiButton
          variant="ghost"
          size="sm"
          class="absolute right-3 top-3 z-50 !p-2 text-text-secondary hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
          @click="closeSidebar"
          aria-label="Close navigation"
        >
          <XMarkIcon class="h-5 w-5" />
        </OuiButton>
      </div>
    </Transition>

    <div v-show="user.user && user.isAuthenticated" class="flex min-h-screen">
      <!-- Sidebar -->
      <AppSidebar
        class="desktop-sidebar"
        :organization-options="organizationOptions"
        :current-organization="currentOrganization"
        @navigate="closeSidebar"
        @organization-change="switchOrganization"
      />

      <!-- Main content -->
      <div class="flex-1 flex flex-col">
        <!-- Top bar -->
        <AppHeader
          :notification-count="unreadCount"
          @notifications-click="isNotificationsOpen = !isNotificationsOpen"
        >
          <template #leading>
            <OuiButton
              variant="ghost"
              size="sm"
              class="hamburger-button !p-2 text-text-secondary hover:text-primary focus-visible:ring-2 focus-visible:ring-primary"
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

        <!-- Page content -->
        <main class="main-content">
          <slot />
        </main>

        <!-- Notifications Modal -->
        <AppNotifications
          v-model="isNotificationsOpen"
          :items="notifications"
          @update:items="(val) => (notifications = val.map(n => ({ ...n, read: !!n.read })))"
          @close="isNotificationsOpen = false"
        />
      </div>
    </div>
    <div
      v-show="!user.user || !user.isAuthenticated"
      class="main-content flex flex-col justify-center items-center"
    >
      <!-- {{ user }} -->
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
        >Log In</OuiButton
      >
    </div>
  </div>
</template>

<script setup lang="ts">
  import { onBeforeUnmount, onMounted, computed, ref } from "vue";
  import { Bars3Icon, XMarkIcon } from "@heroicons/vue/24/outline";

  // Pinia user store
  const user = useAuth();
  // Notifications state
  const isNotificationsOpen = ref(false);
  const notifications = ref<Array<{ id: string; title: string; message: string; timestamp: Date; read: boolean }>>([
    { id: "1", title: "Deployment complete", message: "Your app is live at app.obiente.cloud", timestamp: new Date(), read: false },
    { id: "2", title: "New member joined", message: "Alex added to Acme Corp", timestamp: new Date(Date.now() - 3600_000), read: true },
    { id: "3", title: "Build started", message: "Re-deploy triggered for dashboard", timestamp: new Date(Date.now() - 600_000), read: false },
  ]);
  const unreadCount = computed(() => notifications.value.filter(n => !n.read).length);

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

  if (import.meta.client) {
    onMounted(() => {
      handleBreakpointChange();
      window.addEventListener("keydown", handleKeydown);
      window.addEventListener("resize", handleBreakpointChange);
    });

    onBeforeUnmount(() => {
      window.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("resize", handleBreakpointChange);
    });
  }

  // Organization switcher data and methods
  const organizationOptions = computed(() => {
    // TODO: Replace with actual organizations from API
    return [
      {
        label: "Personal",
        value: "1",
      },
      { label: "Acme Corp", value: "2" },
      { label: "Development Team", value: "3" },
    ];
  });

  const currentOrganization = computed(() => {
    // TODO: Replace with actual current organization from user store/API
    return organizationOptions.value[0]
      ? {
          id: organizationOptions.value[0].value,
          name: organizationOptions.value[0].label,
        }
      : null;
  });

  const switchOrganization = async (
    organizationId: string | string[] | undefined
  ) => {
    // normalise
    const id = Array.isArray(organizationId)
      ? organizationId[0]
      : organizationId;
    if (!id) {
      console.warn("No organization id provided to switchOrganization");
      return;
    }

    // TODO: Implement organization switching logic
    console.log("Switching to organization:", id);
  };

</script>
