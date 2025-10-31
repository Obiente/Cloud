<template>
  <OuiStack gap="lg">
    <OuiText size="xl" weight="semibold">Navigation</OuiText>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Tabs</OuiText>
        <div class="mt-2">
          <OuiTabs v-model="tab" :tabs="tabs">
            <template #one>
              <OuiText>Tab One Content</OuiText>
            </template>
            <template #two>
              <OuiText>Tab Two Content</OuiText>
            </template>
            <template #three>
              <OuiText>Tab Three Content</OuiText>
            </template>
          </OuiTabs>
        </div>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
  import type { TabItem } from "~/components/oui/Tabs.vue";

  definePageMeta({ layout: "components" });

  const tab = ref("one");

  const tabs: TabItem[] = [
    { id: "one", label: "One" },
    { id: "two", label: "Two" },
    { id: "three", label: "Three" },
  ];
</script>
