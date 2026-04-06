<template>
    <OuiContainer size="full" p="none">
      <OuiStack gap="lg">
        <!-- Header -->
        <OuiStack gap="xs">
          <OuiText as="h1" size="xl" weight="semibold">Settings</OuiText>
          <OuiText size="sm" color="tertiary">
            Manage your account preferences and integrations.
          </OuiText>
        </OuiStack>

        <!-- Tabbed Content -->
        <OuiTabs v-model="activeTab" :tabs="tabs">
          <template #account>
            <SettingsAccount />
          </template>
          <template #integrations>
            <SettingsIntegrations />
          </template>
          <template #preferences>
            <SettingsPreferences />
          </template>
        </OuiTabs>
      </OuiStack>
    </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { TabItem } from "~/components/oui/Tabs.vue";
import {
  Cog6ToothIcon,
  UserIcon,
  LinkIcon,
  AdjustmentsHorizontalIcon,
} from "@heroicons/vue/24/outline";


definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();

const tabs: TabItem[] = [
  { id: "account", label: "Account", icon: UserIcon },
  { id: "integrations", label: "Integrations", icon: LinkIcon },
  { id: "preferences", label: "Preferences", icon: AdjustmentsHorizontalIcon },
];

// Get initial tab from query parameter or default to "account"
const getInitialTab = () => {
  const tabParam = route.query.tab;
  if (typeof tabParam === "string") {
    const tabIds = tabs.map((t) => t.id);
    return tabIds.includes(tabParam) ? tabParam : "account";
  }
  return "account";
};

const activeTab = ref(getInitialTab());

// Watch for tab changes and update query parameter
watch(activeTab, (newTab) => {
  if (route.query.tab !== newTab) {
    router.replace({
      query: {
        ...route.query,
        tab: newTab === "account" ? undefined : newTab,
      },
    });
  }
});

// Watch for query parameter changes (e.g., back/forward navigation)
watch(
  () => route.query.tab,
  (tabParam) => {
    if (typeof tabParam === "string") {
      const tabIds = tabs.map((t) => t.id);
      if (tabIds.includes(tabParam) && activeTab.value !== tabParam) {
        activeTab.value = tabParam;
      }
    } else if (!tabParam && activeTab.value !== "account") {
      activeTab.value = "account";
    }
  }
);
</script>
