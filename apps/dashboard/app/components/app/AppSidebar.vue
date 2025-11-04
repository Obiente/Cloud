<template>
  <nav class="flex flex-col min-h-screen bg-surface-base" :class="$attrs.class">
    <div class="p-6">
      <OuiFlex align="center" justify="between" class="mb-6">
        <OuiFlex align="start" gap="md">
          <OuiBox
            class="w-8 h-8 bg-primary rounded-xl mt-1"
          >
            <OuiFlex align="center" justify="center" class="h-full">
              <OuiText size="lg" weight="bold" color="primary">O</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiStack gap="none" class="leading-tight">
            <OuiText size="xl" weight="bold" color="primary">Obiente</OuiText>
            <OuiText
              v-if="props.currentOrganization"
              size="sm"
              color="secondary"
            >
              {{ props.currentOrganization.name }}
            </OuiText>
          </OuiStack>
        </OuiFlex>

        <div class="ml-2">
          <OrgSwitcher :collection="organization" :multiple="false" @change="(v:any)=>emit('organization-change', v)" @create="emit('new-organization')" />
        </div>
      </OuiFlex>

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
          to="/gameservers"
          label="Game Servers"
          :icon="CubeIcon"
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

        <AppNavigationLink
          to="/organizations"
          label="Organizations"
          :icon="UsersIcon"
          @navigate="handleNavigate"
        />

        <!-- Admin -->
        <div class="mt-4">
          <OuiText size="xs" transform="uppercase" class="tracking-wide px-2" color="secondary">Admin</OuiText>
        </div>
        <AppNavigationLink
          to="/admin/quotas"
          label="Quotas"
          :icon="Cog6ToothIcon"
          @navigate="handleNavigate"
        />
        <AppNavigationLink
          to="/admin/roles"
          label="Roles"
          :icon="Cog6ToothIcon"
          @navigate="handleNavigate"
        />
        <AppNavigationLink
          to="/admin/bindings"
          label="Bindings"
          :icon="Cog6ToothIcon"
          @navigate="handleNavigate"
        />
        <template v-if="props.showSuperAdmin">
          <div class="mt-4">
            <OuiText size="xs" transform="uppercase" class="tracking-wide px-2" color="secondary">
              Superadmin
            </OuiText>
          </div>
          <AppNavigationLink
            to="/superadmin"
            label="Overview"
            :icon="ShieldCheckIcon"
            @navigate="handleNavigate"
            exact-match
          />
          <AppNavigationLink
            to="/superadmin/organizations"
            label="Organizations"
            :icon="BuildingOfficeIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            to="/superadmin/deployments"
            label="Deployments"
            :icon="RocketLaunchIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            to="/superadmin/invites"
            label="Invites"
            :icon="UsersIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            to="/superadmin/usage"
            label="Usage"
            :icon="ChartBarIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            to="/superadmin/dns"
            label="DNS"
            :icon="ServerIcon"
            @navigate="handleNavigate"
          />
        </template>
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
  UsersIcon,
  ShieldCheckIcon,
  BuildingOfficeIcon,
  ChartBarIcon,
  CubeIcon,
} from "@heroicons/vue/24/outline";
import OrgSwitcher from "@/components/oui/OrgSwitcher.vue";
import { createListCollection } from "@ark-ui/vue";
import { computed } from 'vue';

interface Organization {
  id: string;
  name?: string | null;
}

interface Props {
  currentOrganization?: Organization | null;
  organizationOptions?: Array<{ label: string; value: string | number }>;
  showSuperAdmin?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  organizationOptions: () => [],
  showSuperAdmin: false,
});

// Recompute the Ark collection whenever options change to ensure OrgSwitcher updates
const organization = computed(() =>
  createListCollection({ items: props.organizationOptions })
);
const emit = defineEmits<{
  navigate: [];
  "organization-change": [organizationId: string | string[] | undefined];
  "new-organization": [];
}>();

const handleNavigate = () => {
  emit("navigate");
};
</script>
