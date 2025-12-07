<template>
  <nav class="flex flex-col h-full min-h-0 bg-surface-base" :class="$attrs.class">
    <!-- Navigation - Scrollable middle section -->
    <div class="flex-1 min-h-0 overflow-y-auto sidebar-scrollable">
      <nav class="px-6 pt-6 pb-20 space-y-2">
        <AppNavigationLink
          to="/docs"
          label="Overview"
          :icon="BookOpenIcon"
          exact-match
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/getting-started"
          label="Getting Started"
          :icon="RocketLaunchIcon"
          @navigate="handleNavigate"
        />

        <!-- Section Divider -->
        <div class="mt-4">
          <OuiText size="xs" transform="uppercase" class="tracking-wide px-2" color="secondary">
            Features
          </OuiText>
        </div>

        <AppNavigationLink
          to="/docs/dashboard"
          label="Dashboard"
          :icon="HomeIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/deployments"
          label="Deployments"
          :icon="RocketLaunchIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/gameservers"
          label="Game Servers"
          :icon="CubeIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/vps"
          label="VPS Instances"
          :icon="ServerIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/databases"
          label="Databases"
          :icon="CircleStackIcon"
          @navigate="handleNavigate"
        />

        <!-- Section Divider -->
        <div class="mt-4">
          <OuiText size="xs" transform="uppercase" class="tracking-wide px-2" color="secondary">
            Management
          </OuiText>
        </div>

        <AppNavigationLink
          v-if="billingEnabled"
          to="/docs/billing"
          label="Billing"
          :icon="CreditCardIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/organizations"
          label="Organizations"
          :icon="BuildingOfficeIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/permissions"
          label="Permissions"
          :icon="ShieldCheckIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs/self-hosting"
          label="Self-Hosting"
          :icon="ServerIcon"
          @navigate="handleNavigate"
        />

        <!-- Section Divider -->
        <div class="mt-4">
          <OuiText size="xs" transform="uppercase" class="tracking-wide px-2" color="secondary">
            Help
          </OuiText>
        </div>

        <AppNavigationLink
          to="/docs/troubleshooting"
          label="Troubleshooting"
          :icon="QuestionMarkCircleIcon"
          @navigate="handleNavigate"
        />
      </nav>
    </div>
  </nav>
</template>

<script setup lang="ts">
import {
  BookOpenIcon,
  HomeIcon,
  RocketLaunchIcon,
  CubeIcon,
  ServerIcon,
  CircleStackIcon,
  CreditCardIcon,
  BuildingOfficeIcon,
  ShieldCheckIcon,
  QuestionMarkCircleIcon,
} from "@heroicons/vue/24/outline";

const emit = defineEmits<{
  navigate: [];
}>();

const config = useConfig();
const billingEnabled = computed(() => config.billingEnabled.value === true);

const handleNavigate = () => {
  emit("navigate");
};
</script>

<style scoped>
/* Hide scrollbar for sidebar scrollable section */
.sidebar-scrollable {
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
}

.sidebar-scrollable::-webkit-scrollbar {
  display: none; /* Chrome, Safari, Opera */
}
</style>
