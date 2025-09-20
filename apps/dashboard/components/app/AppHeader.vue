<template>
  <header class="bg-surface-base border-b border-border-muted px-6 py-4">
    <div class="flex items-center justify-between">
      <!-- Page title -->
      <div>
        <h1 class="text-2xl font-bold text-text-primary">
          <slot name="title">
            {{ title }}
          </slot>
        </h1>
        <p v-if="subtitle" class="text-sm text-text-secondary mt-1">
          {{ subtitle }}
        </p>
      </div>

      <!-- Actions -->
      <div class="flex items-center space-x-4">
        <!-- Organization switcher -->
        <OuiSelect
          :items="organizationOptions"
          :modelValue="currentOrganization?.id"
          @update:modelValue="handleOrganizationChange"
          placeholder="Select Organization"
          class="min-w-48"
        />

        <!-- Notifications -->
        <OuiButton 
          variant="ghost" 
          size="sm" 
          title="Notifications" 
          class="!p-2 relative"
          @click="handleNotificationsClick"
        >
          <BellIcon class="w-5 h-5" />
          <!-- Notification badge -->
          <span 
            v-if="notificationCount > 0"
            class="
              absolute -top-1 -right-1 w-5 h-5 bg-danger text-white 
              text-xs font-medium rounded-full 
              flex items-center justify-center
            "
          >
            {{ notificationCount > 99 ? '99+' : notificationCount }}
          </span>
        </OuiButton>

        <!-- Additional actions slot -->
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { BellIcon } from '@heroicons/vue/24/outline'

interface Organization {
  id: string
  name: string
}

interface Props {
  title?: string
  subtitle?: string
  currentOrganization?: Organization | null
  organizationOptions?: Array<{ label: string; value: string }>
  notificationCount?: number
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Dashboard',
  notificationCount: 0,
  organizationOptions: () => []
})

const emit = defineEmits<{
  'organization-change': [organizationId: string]
  'notifications-click': []
}>()

const handleOrganizationChange = (organizationId: string) => {
  emit('organization-change', organizationId)
}

const handleNotificationsClick = () => {
  emit('notifications-click')
}
</script>