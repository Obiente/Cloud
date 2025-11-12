<template>
  <OuiCard variant="outline" class="border-border-default/50">
    <OuiCardBody class="p-3 md:p-6">
      <OuiFlex
        justify="between"
        align="start"
        wrap="wrap"
        gap="md"
        class="md:gap-lg md:items-center"
      >
        <OuiStack gap="sm" class="flex-1 min-w-0 md:gap-md">
          <OuiFlex align="center" gap="sm" wrap="wrap" class="md:gap-md">
            <OuiBox
              p="xs"
              rounded="lg"
              bg="accent-primary"
              :class="iconClass"
              class="bg-primary/10 ring-1 ring-primary/20 shrink-0 md:p-sm md:rounded-xl"
            >
              <component :is="icon" class="w-6 h-6 md:w-8 md:h-8 text-primary" />
            </OuiBox>
            <OuiStack gap="xs" class="min-w-0 flex-1 md:gap-none">
              <OuiFlex
                align="center"
                justify="between"
                gap="md"
                wrap="wrap"
                class="md:justify-start"
              >
                <OuiText
                  as="h1"
                  size="xl"
                  weight="bold"
                  truncate
                  class="md:text-2xl"
                >
                  {{ title }}
                </OuiText>
                <slot name="badges" />
              </OuiFlex>
              <OuiText size="xs" color="secondary" class="md:text-sm">
                <slot name="subtitle" />
              </OuiText>
            </OuiStack>
          </OuiFlex>
        </OuiStack>

        <OuiFlex
          gap="xs"
          wrap="wrap"
          class="w-full md:w-auto shrink-0 md:gap-sm md:flex-nowrap"
        >
          <slot name="actions" />
        </OuiFlex>
      </OuiFlex>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import type { Component } from "vue";

defineProps<{
  title: string;
  icon: Component;
  iconClass?: string;
}>();
</script>


