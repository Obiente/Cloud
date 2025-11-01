<template>
  <Menu.Item
    class="relative flex w-full cursor-pointer select-none items-center justify-between gap-3 rounded-sm py-2 px-3 text-sm text-primary transition-colors duration-150 data-[highlighted]:bg-surface-raised data-[disabled]:cursor-not-allowed data-[disabled]:text-text-muted data-[disabled]:opacity-60"
    :value="value ?? ''"
    :disabled="disabled"
    @select="emit('select', $event)"
  >
    <OuiFlex align="center" gap="xs">
      <slot />
    </OuiFlex>
    <OuiText v-if="shortcut" size="xs" color="muted" class="ml-auto font-mono">
      {{ shortcut }}
    </OuiText>
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
