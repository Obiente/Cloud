<template>
  <div class="bg-background min-h-screen">
    <OuiContainer>
      <OuiStack gap="lg">
        <OuiText size="2xl" weight="semibold">Components</OuiText>
        <OuiCard>
          <OuiCardBody>
            <OuiFlex gap="md" wrap="wrap">
              <NuxtLink class="text-primary" to="/components/inputs"
                >Inputs</NuxtLink
              >
              <NuxtLink class="text-primary" to="/components/feedback"
                >Feedback</NuxtLink
              >
              <NuxtLink class="text-primary" to="/components/overlays"
                >Overlays</NuxtLink
              >
              <NuxtLink class="text-primary" to="/components/navigation"
                >Navigation</NuxtLink
              >
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
        <slot />
      </OuiStack>
    </OuiContainer>
  </div>
</template>

<script setup lang="ts">
  // components layout
</script>
