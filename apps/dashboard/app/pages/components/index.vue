<template>
  <div>
    <OuiStack gap="md">
      <OuiText size="xl" weight="semibold">Oui Components Overview</OuiText>
      <OuiText>Browse groups:</OuiText>
      <OuiFlex gap="md">
        <NuxtLink class="text-primary" to="/components/inputs">Inputs</NuxtLink>
        <NuxtLink class="text-primary" to="/components/feedback">Feedback</NuxtLink>
        <NuxtLink class="text-primary" to="/components/overlays">Overlays</NuxtLink>
        <NuxtLink class="text-primary" to="/components/navigation">Navigation</NuxtLink>
      </OuiFlex>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'components' })
</script>
