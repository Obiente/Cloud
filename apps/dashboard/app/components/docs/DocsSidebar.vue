<template>
  <nav
    class="flex flex-col h-full min-h-0 bg-surface-base"
    :class="$attrs.class"
  >
    <div class="shrink-0 border-b border-border-muted px-6 py-4">
      <OuiText v-if="currentItem" size="xs" color="tertiary">
        Reading: {{ currentItem.label }}
      </OuiText>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto sidebar-scrollable">
      <nav class="px-6 pt-6 pb-20 space-y-2">
        <template v-for="section in visibleSections" :key="section.id">
          <div :class="section.id !== 'start' ? 'mt-4' : ''">
            <OuiText
              size="xs"
              transform="uppercase"
              class="tracking-wide px-2"
              color="tertiary"
            >
              {{ section.label }}
            </OuiText>
          </div>

          <AppNavigationLink
            v-for="item in section.items"
            :key="item.path"
            :to="item.path"
            :label="item.label"
            :icon="getIcon(item.path)"
            :exact-match="item.path === '/docs'"
            @navigate="handleNavigate"
          />
        </template>
      </nav>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from "vue";
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
import type { Component } from "vue";
import {
  docsNavItems,
  docsSectionLabels,
  getDocsCurrentItem,
} from "~/utils/docsNavigation";

const emit = defineEmits<{
  navigate: [];
}>();

const config = useConfig();
const route = useRoute();
const billingEnabled = computed(() => config.billingEnabled.value === true);

const currentItem = computed(() => getDocsCurrentItem(route.path));

const iconByPath: Record<string, Component> = {
  "/docs": BookOpenIcon,
  "/docs/getting-started": RocketLaunchIcon,
  "/docs/dashboard": HomeIcon,
  "/docs/deployments": RocketLaunchIcon,
  "/docs/gameservers": CubeIcon,
  "/docs/vps": ServerIcon,
  "/docs/databases": CircleStackIcon,
  "/docs/billing": CreditCardIcon,
  "/docs/organizations": BuildingOfficeIcon,
  "/docs/permissions": ShieldCheckIcon,
  "/docs/self-hosting": ServerIcon,
  "/docs/troubleshooting": QuestionMarkCircleIcon,
};

const availableItems = computed(() =>
  docsNavItems.filter(
    (item) => billingEnabled.value || item.path !== "/docs/billing"
  )
);

const visibleSections = computed(() =>
  (["start", "features", "management", "help"] as const)
    .map((sectionId) => ({
      id: sectionId,
      label: docsSectionLabels[sectionId],
      items: availableItems.value.filter((item) => item.section === sectionId),
    }))
    .filter((section) => section.items.length > 0)
);

const getIcon = (path: string) => iconByPath[path] || BookOpenIcon;

const handleNavigate = () => {
  emit("navigate");
};
</script>

<style scoped>
.sidebar-scrollable {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.sidebar-scrollable::-webkit-scrollbar {
  display: none;
}
</style>
