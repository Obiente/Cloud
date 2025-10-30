<template>
  <NuxtLayout name="default">
    <OuiContainer>
      <OuiStack gap="lg">
        <OuiText size="2xl" weight="bold">Admin</OuiText>
        <OuiFlex gap="md">
          <NuxtLink to="/admin/quotas">
            <OuiButton variant="ghost" :color="activeTab === 'quotas' ? 'primary' : 'neutral'">Quotas</OuiButton>
          </NuxtLink>
          <NuxtLink to="/admin/roles">
            <OuiButton variant="ghost" :color="activeTab === 'roles' ? 'primary' : 'neutral'">Roles</OuiButton>
          </NuxtLink>
          <NuxtLink to="/admin/bindings">
            <OuiButton variant="ghost" :color="activeTab === 'bindings' ? 'primary' : 'neutral'">Bindings</OuiButton>
          </NuxtLink>
        </OuiFlex>
        <slot />
      </OuiStack>
    </OuiContainer>
  </NuxtLayout>
</template>

<script setup lang="ts">
const route = useRoute();
const activeTab = computed(() => {
  if (route.path.endsWith('/quotas')) return 'quotas';
  if (route.path.endsWith('/roles')) return 'roles';
  if (route.path.endsWith('/bindings')) return 'bindings';
  return '';
});
</script>
