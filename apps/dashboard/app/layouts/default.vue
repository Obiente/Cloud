<template>
  <!-- {{user}} -->
  <div class="bg-background">
    <div v-if="user.user && user.isAuthenticated" class="flex">
      <!-- Sidebar -->
      <AppSidebar />

      <!-- Main content -->
      <div class="flex-1 flex flex-col">
        <!-- Top bar -->
        <AppHeader
          :organization-options="organizationOptions"
          :notification-count="notificationCount"
          @organization-change="switchOrganization"
          @notifications-click="handleNotificationsClick"
        >
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
// Pinia user store
const user = useAuth();
// Notification count (TODO: Replace with actual notification system)
const notificationCount = ref(3);

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
