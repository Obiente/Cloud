<template>
  <Menu.Root v-bind="$attrs" @open="emit('open')" @close="emit('close')">
    <Menu.Trigger as-child>
      <slot name="trigger" />
    </Menu.Trigger>
    <Menu.Positioner>
      <Menu.Content class="oui-context-menu">
        <slot />
      </Menu.Content>
    </Menu.Positioner>
  </Menu.Root>
</template>

<script setup lang="ts">
import { Menu } from "@ark-ui/vue/menu";

const emit = defineEmits<{
  (e: "open"): void;
  (e: "close"): void;
}>();
</script>

<style scoped>
.oui-context-menu {
  min-width: 180px;
  padding: 6px;
  border-radius: 8px;
  background: var(--oui-surface-overlay);
  border: 1px solid var(--oui-border-default);
  box-shadow: 0 15px 40px rgba(0, 0, 0, 0.2);
  color: var(--oui-text-primary);
  font-size: 13px;
}
</style>
