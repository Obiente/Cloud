<template>
  <Toast.Root
    :class="[
      'bg-surface-overlay border rounded-lg shadow-lg p-4 min-w-[320px] max-w-[560px] sm:w-auto',
      'translate-[var(--x)] translate-y-[var(--y)] scale-[var(--scale)]',
      'z-[var(--z-index)] h-[var(--height)] opacity-[var(--opacity)]',
      'will-change-[translate,opacity,scale]',
      'transition-[translate,scale,opacity,height,box-shadow] duration-[400ms]',
      'ease-[cubic-bezier(0.21,1.02,0.73,1)]',
      'data-[state=closed]:transition-[translate,scale,opacity] data-[state=closed]:duration-[400ms]',
      'data-[state=closed]:ease-[cubic-bezier(0.06,0.71,0.55,1)]',
      'w-[calc(100%-2rem)] sm:w-auto',
      {
        'border-success/30 bg-success/5': type === 'success',
        'border-danger/30 bg-danger/5': type === 'error',
        'border-warning/30 bg-warning/5': type === 'warning',
        'border-primary/30 bg-primary/5': type === 'info',
      }
    ]"
    :style="{
      translate: 'var(--x) var(--y)',
      scale: 'var(--scale)',
      zIndex: 'var(--z-index)',
      height: 'var(--height)',
      opacity: 'var(--opacity)',
    }"
    :data-type="type"
  >
    <OuiFlex align="start" justify="between" gap="md" class="w-full">
      <OuiFlex v-if="icon || title || description" align="start" gap="sm" class="flex-1 min-w-0">
        <div v-if="icon" :class="[
          'shrink-0 mt-0.5 w-5 h-5',
          {
            'text-success': type === 'success',
            'text-danger': type === 'error',
            'text-warning': type === 'warning',
            'text-primary': type === 'info',
          }
        ]">
          <component :is="icon" class="w-full h-full" />
        </div>
        <OuiStack gap="xs" class="flex-1 min-w-0">
          <Toast.Title v-if="title" class="text-sm font-semibold text-foreground">
            {{ title }}
          </Toast.Title>
          <Toast.Description v-if="description" class="text-sm text-text-secondary">
            {{ description }}
          </Toast.Description>
        </OuiStack>
      </OuiFlex>
      <Toast.CloseTrigger class="shrink-0 p-1 rounded hover:bg-surface-muted text-text-secondary hover:text-foreground transition-colors">
        <XMarkIcon class="w-4 h-4" />
      </Toast.CloseTrigger>
    </OuiFlex>
  </Toast.Root>
</template>

<script setup lang="ts">
import { Toast } from "@ark-ui/vue/toast";
import { XMarkIcon } from "@heroicons/vue/24/outline";
import OuiFlex from "./Flex.vue";
import OuiStack from "./Stack.vue";

interface Props {
  title?: string;
  description?: string;
  type?: "success" | "error" | "warning" | "info";
  icon?: any;
  overlap?: boolean;
}

withDefaults(defineProps<Props>(), {
  type: "info",
  overlap: false,
});
</script>
