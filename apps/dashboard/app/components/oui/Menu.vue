<template>
  <Menu.Root v-bind="$attrs" @open-change="handleOpenChange">
    <Menu.Trigger as-child>
      <slot name="trigger" />
    </Menu.Trigger>
    <template v-if="isMounted">
      <Teleport to="body">
        <Menu.Positioner>
          <Menu.Content
            class="z-50 min-w-[12rem] max-h-[300px] overflow-y-auto rounded-xl border border-border-default bg-surface-base shadow-md animate-in fade-in-0 zoom-in-95 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
          >
            <slot />
          </Menu.Content>
        </Menu.Positioner>
      </Teleport>
    </template>
    <template v-else>
      <!-- SSR fallback - render inline during SSR to avoid hydration mismatch -->
      <Menu.Positioner>
        <Menu.Content
          class="z-50 min-w-[12rem] max-h-[300px] overflow-y-auto rounded-xl border border-border-default bg-surface-base shadow-md hidden"
        >
          <slot />
        </Menu.Content>
      </Menu.Positioner>
    </template>
  </Menu.Root>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Menu } from "@ark-ui/vue/menu";

const isMounted = ref(false);

onMounted(() => {
  isMounted.value = true;
});

const emit = defineEmits<{
  (e: "open"): void;
  (e: "close"): void;
}>();

function handleOpenChange(details: { open: boolean }) {
  if (details.open) emit("open");
  else emit("close");
}
</script>
