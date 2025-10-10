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
        <div class="absolute inset-0 bg-background/80 backdrop-blur-sm" @click="closeSidebar" />

        <Transition name="sidebar-panel" appear>
          <div
            v-if="isSidebarOpen"
            :id="mobileSidebarId"
            class="relative z-50 flex h-full w-72 max-w-[80vw] flex-col"
          >
            <AppSidebar
              class="sidebar-drawer relative h-full overflow-y-auto border-r border-border-muted bg-surface-base shadow-2xl"
              @navigate="closeSidebar"
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
      </div>
    </Transition>

    <div v-if="user.user && user.isAuthenticated" class="flex min-h-screen">
      <!-- Sidebar -->
      <AppSidebar class="desktop-sidebar" @navigate="closeSidebar" />

      <!-- Main content -->
      <div class="flex-1 flex flex-col">
        <!-- Top bar -->
        <AppHeader
          :organization-options="organizationOptions"
          :notification-count="notificationCount"
          @organization-change="switchOrganization"
          @notifications-click="handleNotificationsClick"
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
      </div>
    </div>
    <div v-else class="main-content flex flex-col justify-center items-center">
      <!-- {{ user }} -->
      <OuiText v-if="user.isLoading" size="2xl" weight="extrabold">loading</OuiText>
      <OuiText v-else-if="user.isAuthenticated" size="2xl" weight="extrabold"
        >logging you in</OuiText
      >
      <OuiButton v-else size="xl" weight="extrabold" @click="user.popupLogin()">Log In</OuiButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue';
import { Bars3Icon, XMarkIcon } from '@heroicons/vue/24/outline';

// Pinia user store
const user = useAuth();
// Notification count (TODO: Replace with actual notification system)
const notificationCount = ref(3);

const isSidebarOpen = ref(false);
const mobileSidebarId = 'mobile-primary-navigation';

const closeSidebar = () => {
  isSidebarOpen.value = false;
};

const toggleSidebar = () => {
  isSidebarOpen.value = !isSidebarOpen.value;
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    closeSidebar();
  }
};

const handleBreakpointChange = () => {
  if (import.meta.client && window.matchMedia('(min-width: 1024px)').matches) {
    isSidebarOpen.value = false;
  }
};

if (import.meta.client) {
  onMounted(() => {
    handleBreakpointChange();
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('resize', handleBreakpointChange);
  });

  onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleKeydown);
    window.removeEventListener('resize', handleBreakpointChange);
  });
}

// Organization switcher data and methods
const organizationOptions = computed(() => {
  // TODO: Replace with actual organizations from API
  return [
    {
      label: 'Personal',
      value: '1',
    },
    { label: 'Acme Corp', value: '2' },
    { label: 'Development Team', value: '3' },
  ];
});

const switchOrganization = async (organizationId: string) => {
  // TODO: Implement organization switching logic
  console.log('Switching to organization:', organizationId);
};

const handleNotificationsClick = () => {
  // TODO: Implement notifications panel/modal
  console.log('Opening notifications');
};
</script>
