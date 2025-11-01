<template>
  <Menu.Item
    class="oui-context-menu-item"
    :value="value ?? ''"
    :disabled="disabled"
    @select="emit('select', $event)"
  >
    <span class="oui-context-menu-item__content">
      <slot />
    </span>
    <span v-if="shortcut" class="oui-context-menu-item__shortcut">{{ shortcut }}</span>
  </Menu.Item>
</template>

<script setup lang="ts">
import { Menu } from "@ark-ui/vue/menu";

const props = defineProps<{
  value?: string;
  disabled?: boolean;
  shortcut?: string;
}>();

const emit = defineEmits<{
  (e: "select", event: any): void;
}>();

const { value, disabled, shortcut } = props;
</script>

<style scoped>
.oui-context-menu-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  color: var(--oui-text-primary);
}

.oui-context-menu-item[data-disabled] {
  opacity: 0.55;
  cursor: not-allowed;
}

.oui-context-menu-item:not([data-disabled]):hover,
.oui-context-menu-item[data-highlighted] {
  background: var(--oui-surface-hover);
}

.oui-context-menu-item__shortcut {
  font-size: 11px;
  color: var(--oui-text-tertiary);
}
</style>
