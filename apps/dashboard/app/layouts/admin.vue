<template>
  <NuxtLayout name="default">
    <OuiContainer>
      <OuiStack gap="lg">
        <OuiText size="2xl" weight="bold">Admin</OuiText>
        <nav aria-label="Organization administration" class="flex flex-wrap gap-4">
          <OuiButton v-for="tab in tabs" :key="tab.id" :as="NuxtLink" :to="`/admin/${tab.id}`"
            variant="ghost" :color="activeTab === tab.id ? 'primary' : 'neutral'"
            :aria-current="activeTab === tab.id ? 'page' : undefined">{{ tab.label }}</OuiButton>
        </nav>
        <slot />
      </OuiStack>
    </OuiContainer>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { NuxtLink } from "#components";
const tabs = [{id:"quotas",label:"Quotas"},{id:"roles",label:"Roles"},{id:"bindings",label:"Bindings"}];
const route = useRoute();
const activeTab = computed(() => {
  if (route.path.endsWith('/quotas')) return 'quotas';
  if (route.path.endsWith('/roles')) return 'roles';
  if (route.path.endsWith('/bindings')) return 'bindings';
  return '';
});
</script>
