<template>
  <OuiBox as="nav" class="flex flex-col h-full min-h-0 bg-surface-base" :class="$attrs.class">
    <!-- Header - Fixed at top -->
    <OuiBox as="header" class="shrink-0 p-6 border-b border-border-muted">
      <OuiFlex align="center" justify="between">
        <OuiFlex align="start" gap="md">
          <ObienteLogo size="md" class="mt-1" />
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

        <OuiBox class="ml-2 shrink-0">
          <OrgSwitcher :collection="organization" :multiple="false" @change="(v:any)=>emit('organization-change', v)" @create="emit('new-organization')" />
        </OuiBox>
      </OuiFlex>
    </OuiBox>

    <!-- Navigation - Scrollable middle section -->
    <OuiBox class="flex-1 min-h-0 overflow-y-auto sidebar-scrollable">
      <nav class="px-6 pt-6 pb-20 space-y-2">
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
          v-if="billingEnabled"
          to="/billing"
          label="Billing"
          :icon="CreditCardIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/support"
          label="Support"
          :icon="ChatBubbleLeftRightIcon"
          @navigate="handleNavigate"
        />

        <AppNavigationLink
          to="/docs"
          label="Documentation"
          :icon="BookOpenIcon"
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
          to="/audit-logs"
          label="Audit Logs"
          :icon="ClipboardDocumentListIcon"
          @navigate="handleNavigate"
        />
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
            v-if="hasPagePermission('/superadmin')"
            to="/superadmin"
            label="Overview"
            :icon="ShieldCheckIcon"
            @navigate="handleNavigate"
            exact-match
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/audit-logs')"
            to="/superadmin/audit-logs"
            label="Global Audit Logs"
            :icon="ClipboardDocumentListIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/organizations')"
            to="/superadmin/organizations"
            label="Organizations"
            :icon="BuildingOfficeIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/plans')"
            to="/superadmin/plans"
            label="Plans"
            :icon="CubeIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/deployments')"
            to="/superadmin/deployments"
            label="Deployments"
            :icon="RocketLaunchIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/vps')"
            to="/superadmin/vps"
            label="VPS Instances"
            :icon="ServerIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/nodes')"
            to="/superadmin/nodes"
            label="Nodes"
            :icon="ServerIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/users')"
            to="/superadmin/users"
            label="Users"
            :icon="UsersIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/usage')"
            to="/superadmin/usage"
            label="Usage"
            :icon="ChartBarIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/dns')"
            to="/superadmin/dns"
            label="DNS"
            :icon="ServerIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/abuse')"
            to="/superadmin/abuse"
            label="Abuse Detection"
            :icon="ShieldExclamationIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/income')"
            to="/superadmin/income"
            label="Income Overview"
            :icon="CurrencyDollarIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/invoices')"
            to="/superadmin/invoices"
            label="Invoices"
            :icon="DocumentTextIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/webhook-events')"
            to="/superadmin/webhook-events"
            label="Webhook Events"
            :icon="BoltIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/roles')"
            to="/superadmin/roles"
            label="Roles"
            :icon="ShieldCheckIcon"
            @navigate="handleNavigate"
          />
          <AppNavigationLink
            v-if="hasPagePermission('/superadmin/role-bindings')"
            to="/superadmin/role-bindings"
            label="Role Bindings"
            :icon="UserGroupIcon"
            @navigate="handleNavigate"
          />
        </template>
      </nav>
    </OuiBox>

    <!-- Footer - Fixed at bottom -->
    <OuiBox as="footer" class="shrink-0 border-t border-border-muted bg-surface-subtle">
      <!-- User section -->
      <OuiBox class="px-4 py-4">
        <AppUserProfile />
      </OuiBox>
    </OuiBox>
  </OuiBox>
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
  UserGroupIcon,
  ShieldCheckIcon,
  ShieldExclamationIcon,
  BuildingOfficeIcon,
  ChartBarIcon,
  CubeIcon,
  ChatBubbleLeftRightIcon,
  BookOpenIcon,
  CurrencyDollarIcon,
  DocumentTextIcon,
  ClipboardDocumentListIcon,
  BoltIcon,
} from "@heroicons/vue/24/outline";
import OrgSwitcher from "@/components/oui/OrgSwitcher.vue";
import { createListCollection } from "@ark-ui/vue";
import { computed } from 'vue';
import ObienteLogo from "./ObienteLogo.vue";

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

const config = useConfig();
const billingEnabled = computed(() => config.billingEnabled.value === true);

const superAdmin = useSuperAdmin();

// Map pages to required permissions
const pagePermissions: Record<string, string> = {
  "/superadmin": "superadmin.overview.read",
  "/superadmin/audit-logs": "superadmin.overview.read", // Audit logs are part of overview
  "/superadmin/organizations": "superadmin.overview.read", // Organizations are part of overview
  "/superadmin/plans": "superadmin.plans.read",
  "/superadmin/deployments": "superadmin.deployments.read",
  "/superadmin/vps": "superadmin.vps.read",
  "/superadmin/nodes": "superadmin.nodes.read",
  "/superadmin/users": "superadmin.users.read",
  "/superadmin/usage": "superadmin.overview.read", // Usage is part of overview
  "/superadmin/dns": "superadmin.dns.read",
  "/superadmin/abuse": "superadmin.abuse.read",
  "/superadmin/income": "superadmin.income.read",
  "/superadmin/invoices": "superadmin.invoices.read",
  "/superadmin/webhook-events": "superadmin.webhooks.read",
  "/superadmin/roles": "admin.roles.read",
  "/superadmin/role-bindings": "admin.bindings.read",
};

// Check if user has permission for a page
const hasPagePermission = (path: string): boolean => {
  if (!props.showSuperAdmin) return false;
  if (superAdmin.isFullSuperadmin.value) return true;
  
  const requiredPerm = pagePermissions[path];
  if (!requiredPerm) return false;
  
  return superAdmin.hasPermission(requiredPerm);
};

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
