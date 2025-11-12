<template>
  <OuiStack gap="sm" class="md:gap-md">
    <OuiTabs v-model="activeTab" :tabs="tabs" />
    <OuiCard variant="default">
      <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
        <template v-for="name in Object.keys(slots)" :key="name" #[name]>
          <slot :name="name" />
        </template>
      </OuiTabs>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, useSlots } from "vue";
import type { TabItem } from "~/components/oui/Tabs.vue";
import { useTabQuery } from "~/composables/useTabQuery";

const props = defineProps<{
  tabs: TabItem[];
  defaultTab?: string;
}>();

const slots = useSlots();

const activeTab = useTabQuery(
  computed(() => props.tabs),
  props.defaultTab
);

defineExpose({
  activeTab,
});
</script>


