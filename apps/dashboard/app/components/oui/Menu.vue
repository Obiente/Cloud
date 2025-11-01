<template>
  <Menu.Root v-bind="$attrs" @open-change="handleOpenChange">
    <Menu.Trigger as-child>
      <slot name="trigger" />
    </Menu.Trigger>
    <Teleport to="body">
      <Menu.Positioner>
        <Menu.Content
          class="z-50 min-w-[12rem] max-h-[300px] overflow-y-auto rounded-md border border-border-default bg-surface-overlay shadow-lg animate-in fade-in-0 zoom-in-95 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
        >
          <slot />
        </Menu.Content>
      </Menu.Positioner>
    </Teleport>
  </Menu.Root>
</template>

<script setup lang="ts">
import { Menu } from "@ark-ui/vue/menu";

const emit = defineEmits<{
  (e: "open"): void;
  (e: "close"): void;
}>();

function handleOpenChange(details: { open: boolean }) {
  if (details.open) emit("open");
  else emit("close");
}
</script>
