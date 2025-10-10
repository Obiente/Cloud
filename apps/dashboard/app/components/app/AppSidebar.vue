<template>
  <nav class="sidebar-nav" :class="$attrs.class">
    <div class="p-6">
      <!-- Logo -->
      <div class="flex items-center space-x-2 mb-8">
        <div
          class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center"
        >
          <span class="text-white font-bold text-lg">O</span>
        </div>
        <span class="text-xl font-bold text-text-primary">Obiente</span>
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

    <!-- User section -->
    <AppUserProfile />
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

const emit = defineEmits<{
  navigate: [];
}>();

const handleNavigate = () => {
  emit("navigate");
};
</script>
