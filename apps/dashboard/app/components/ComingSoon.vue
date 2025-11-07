<template>
  <OuiContainer size="full" py="xl">
    <OuiStack gap="xl" align="center" justify="center" class="min-h-[60vh]">
      <!-- Icon -->
      <OuiBox
        bg="surface-raised"
        border="1"
        borderColor="muted"
        class="w-24 h-24 rounded-3xl flex items-center justify-center"
      >
        <component :is="icon" class="h-12 w-12 text-accent-primary" />
      </OuiBox>

      <!-- Title -->
      <OuiStack gap="sm" align="center" class="max-w-2xl">
        <OuiText as="h1" size="4xl" weight="bold" color="primary" align="center">
          {{ title }}
        </OuiText>
        <OuiText size="lg" color="secondary" align="center">
          {{ description }}
        </OuiText>
      </OuiStack>

      <!-- Coming Soon Badge -->
      <OuiCard variant="outline" class="border-accent-primary/30 bg-accent-primary/5">
        <OuiCardBody>
          <OuiFlex align="center" gap="md">
            <ClockIcon class="h-5 w-5 text-accent-primary" />
            <OuiText size="lg" weight="semibold" color="primary">
              Coming Soon
            </OuiText>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Additional Info -->
      <OuiBox class="max-w-xl">
        <OuiText size="sm" color="muted" align="center">
          {{ additionalInfo }}
        </OuiText>
      </OuiBox>

      <!-- Back Button -->
      <OuiButton variant="ghost" size="md" @click="navigateTo('/dashboard')">
        <OuiFlex align="center" gap="sm">
          <ArrowLeftIcon class="h-4 w-4" />
          <OuiText>Back to Dashboard</OuiText>
        </OuiFlex>
      </OuiButton>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ClockIcon, ArrowLeftIcon } from "@heroicons/vue/24/outline";
import type { Component } from "vue";

interface Props {
  title: string;
  description: string;
  icon: Component;
  additionalInfo?: string;
}

withDefaults(defineProps<Props>(), {
  additionalInfo: "We're working hard to bring you this feature. Stay tuned for updates!",
});
</script>

