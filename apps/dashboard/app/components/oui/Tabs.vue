<template>
  <Tabs.Root v-model="modelValue">
    <Tabs.List class="flex gap-2 border-b border-border-default">
      <slot name="triggers" />
    </Tabs.List>
    <slot />
  </Tabs.Root>
</template>

<script setup lang="ts">
import { Tabs } from '@ark-ui/vue/tabs'

const modelValue = defineModel<string>({ default: '' })
</script>
