<template>
  <NuxtLayout name="default">
    <OuiContainer>
      <OuiStack gap="lg">
        <OuiFlex align="center" justify="between">
          <OuiText size="2xl" weight="bold">Deployment {{ id }}</OuiText>
        </OuiFlex>
        <OuiFlex gap="md">
          <NuxtLink :to="`/deployments/${id}`">
            <OuiButton variant="ghost" :color="activeTab === 'overview' ? 'primary' : 'neutral'">Overview</OuiButton>
          </NuxtLink>
          <NuxtLink :to="`/deployments/${id}/logs`">
            <OuiButton variant="ghost" :color="activeTab === 'logs' ? 'primary' : 'neutral'">Logs</OuiButton>
          </NuxtLink>
        </OuiFlex>
        <slot />
      </OuiStack>
    </OuiContainer>
  </NuxtLayout>
</template>

<script setup lang="ts">
const route = useRoute();
const id = computed(() => route.params.id as string);
const activeTab = computed(() => {
  if (route.path.endsWith('/logs')) return 'logs';
  return 'overview';
});
</script>
