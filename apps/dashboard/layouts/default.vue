<template>
  <div class="bg-background">
    <div class="flex">
      <!-- Sidebar -->
      <AppSidebar 
        :user="user" 
        @logout="logout" 
      />

      <!-- Main content -->
      <div class="flex-1 flex flex-col">
        <!-- Top bar -->
        <AppHeader
          :current-organization="currentOrganization"
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
  </div>
</template>

<script setup lang="ts">
// Composables
const { user, currentOrganization, logout } = useAuth();

// Notification count (TODO: Replace with actual notification system)
const notificationCount = ref(3);

// Organization switcher data and methods
const organizationOptions = computed(() => {
  // TODO: Replace with actual organizations from API
  return [
    {
      label: currentOrganization.value?.name || 'Personal',
      value: currentOrganization.value?.id || '1',
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
