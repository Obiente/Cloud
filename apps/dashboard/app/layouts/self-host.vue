<template>
  <div class="bg-surface-base min-h-screen">
    <!-- Authenticated View -->
    <div v-if="user.user && user.isAuthenticated" class="flex flex-col min-h-screen">
      <!-- Header -->
      <header class="sticky top-0 z-30 border-b border-border-muted bg-surface-base">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="flex h-16 items-center justify-between">
            <!-- Logo and Title -->
            <div class="flex items-center gap-4">
              <ObienteLogo size="md" />
              <OuiStack gap="none" class="leading-tight">
                <OuiText size="lg" weight="bold" color="primary">Self-Hosted DNS</OuiText>
                <OuiText size="xs" color="secondary">Obiente Cloud</OuiText>
              </OuiStack>
            </div>

            <!-- Right side: Org Switcher and Actions -->
            <div class="flex items-center gap-4">
              <!-- Organization Switcher -->
              <div v-if="organizations.length > 0">
                <OrgSwitcher 
                  :collection="organizationCollection" 
                  :multiple="false" 
                  @change="(v: any) => switchOrganization(v)" 
                />
              </div>

              <!-- Actions -->
              <OuiFlex gap="sm">
                <OuiButton variant="ghost" size="sm" @click="navigateTo('/docs/self-hosting')">
                  <BookOpenIcon class="h-4 w-4 mr-2" />
                  Docs
                </OuiButton>
                <OuiButton variant="ghost" size="sm" @click="navigateTo('/dashboard')">
                  <HomeIcon class="h-4 w-4 mr-2" />
                  Dashboard
                </OuiButton>
                <OuiButton variant="ghost" size="sm" @click="logout">
                  <ArrowRightOnRectangleIcon class="h-4 w-4 mr-2" />
                  Sign Out
                </OuiButton>
              </OuiFlex>
            </div>
          </div>
        </div>
      </header>

      <!-- Main Content -->
      <main class="flex-1">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
          <slot />
        </div>
      </main>

      <!-- Footer -->
      <footer class="border-t border-border-muted bg-surface-base py-6">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="flex items-center justify-between">
            <OuiText size="xs" color="muted">
              Self-Hosted DNS Delegation Dashboard
            </OuiText>
            <OuiFlex gap="md">
              <OuiButton variant="ghost" size="xs" @click="navigateTo('/docs/self-hosting')">
                Documentation
              </OuiButton>
              <OuiButton variant="ghost" size="xs" @click="navigateTo('/dashboard')">
                Main Dashboard
              </OuiButton>
            </OuiFlex>
          </div>
        </div>
      </footer>
    </div>

    <!-- Unauthenticated View -->
    <div v-else class="flex min-h-screen items-center justify-center">
      <div class="text-center">
        <OuiStack gap="lg" align="center">
          <LockClosedIcon class="h-16 w-16 text-muted" />
          <OuiStack gap="xs">
            <OuiText size="2xl" weight="bold">Authentication Required</OuiText>
            <OuiText color="muted">Please sign in to access the self-hosted DNS dashboard.</OuiText>
          </OuiStack>
          <OuiFlex v-if="!user.isLoading" gap="md" align="center" justify="center">
            <OuiButton 
              size="lg"
              variant="outline"
              @click="user.popupSignup()"
            >
              Sign Up
            </OuiButton>
            <OuiButton 
              size="lg"
              @click="user.popupLogin()"
            >
              Sign In
            </OuiButton>
          </OuiFlex>
          <OuiStack v-else gap="sm" align="center">
            <ArrowPathIcon class="h-6 w-6 text-muted animate-spin" />
            <OuiText size="sm" color="muted">Loading...</OuiText>
          </OuiStack>
          <OuiButton
            v-if="!user.isLoading"
            variant="ghost"
            size="md"
            @click="navigateTo('/')"
            class="mt-4"
          >
            <HomeIcon class="h-4 w-4 mr-2" />
            Go back home
          </OuiButton>
        </OuiStack>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  HomeIcon,
  BookOpenIcon,
  ArrowRightOnRectangleIcon,
  LockClosedIcon,
  ArrowPathIcon,
} from "@heroicons/vue/24/outline";
import { computed } from "vue";
import { createListCollection } from "@ark-ui/vue";
import { useAuth } from "~/composables/useAuth";
import { useOrganizationsStore } from "~/stores/organizations";
import { useOrganizationLabels } from "~/composables/useOrganizationLabels";
import OrgSwitcher from "~/components/oui/OrgSwitcher.vue";
import ObienteLogo from "~/components/app/ObienteLogo.vue";

interface SelectItem {
  label: string;
  value: string | number;
}

const user = useAuth();
const orgStore = useOrganizationsStore();
orgStore.hydrate();

const organizations = computed(() => orgStore.orgs || []);
const { organizationSelectItems } = useOrganizationLabels(organizations);

// Convert organizations to format expected by OrgSwitcher
const organizationOptions = computed<SelectItem[]>(() => organizationSelectItems.value);

// Create ListCollection for OrgSwitcher
const organizationCollection = computed(() =>
  createListCollection<SelectItem>({ items: organizationOptions.value })
);

const switchOrganization = (orgId: string | null) => {
  if (orgId) {
    orgStore.switchOrganization(orgId);
  }
};

const logout = async () => {
  await user.logout();
  navigateTo("/");
};
</script>

