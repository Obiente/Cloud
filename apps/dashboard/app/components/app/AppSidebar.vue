<template>
  <nav class="sidebar-nav" :class="$attrs.class">
    <div class="p-6">
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-start space-x-3">
          <div
            class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center mt-1"
          >
            <span class="text-foreground font-bold text-lg">O</span>
          </div>
          <div class="leading-tight">
            <div class="text-xl font-bold text-text-primary">Obiente</div>
            <div
              v-if="props.currentOrganization"
              class="text-sm text-text-secondary"
            >
              {{ props.currentOrganization.name }}
            </div>
          </div>
        </div>

        <div class="ml-2">
          <OrgSwitcher :collection="organization" :multiple="false" />
        </div>
      </div>

      <!-- Navigation -->
      <nav class="space-y-2">
        <AppNavigationLink
          to="/dashboard"
          label="Dashboard"
          :icon="HomeIcon"
          exact-match
          @navigate="handleNavigate"
        />
        <AppNavigationLink
          to="/deployments"
          label="Deployments"
          :icon="RocketLaunchIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/vps"
          label="VPS Instances"
          :icon="ServerIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/databases"
          label="Databases"
          :icon="CircleStackIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/billing"
          label="Billing"
          :icon="CreditCardIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/settings"
          label="Settings"
          :icon="Cog6ToothIcon"
          @navigate="handleNavigate"
        />
      </nav>
    </div>

    <div class="mt-auto border-t border-border-muted bg-surface-subtle">
      <!-- User section -->
      <div class="px-4 py-4">
        <AppUserProfile />
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
  import {
    HomeIcon,
    RocketLaunchIcon,
    ServerIcon,
    CircleStackIcon,
    CreditCardIcon,
    Cog6ToothIcon,
  } from "@heroicons/vue/24/outline";
  import OrgSwitcher from "@/components/oui/OrgSwitcher.vue";
  import { createListCollection, useListCollection } from "@ark-ui/vue";

  interface Organization {
    id: string;
    name: string;
  }

  interface Props {
    currentOrganization?: Organization | null;
    organizationOptions?: Array<{ label: string; value: string | number }>;
  }

  const props = withDefaults(defineProps<Props>(), {
    organizationOptions: () => [],
  });

  const organization = createListCollection({
    items: props.organizationOptions,
  });
  const emit = defineEmits<{
    navigate: [];
    "organization-change": [organizationId: string | string[] | undefined];
  }>();

  const handleNavigate = () => {
    emit("navigate");
  };
</script>
