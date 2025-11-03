<template>
  <FloatingPanel.Root
    v-model:open="open"
    :draggable="draggable"
    :resizable="resizable"
    :close-on-escape="closeOnEscape"
    :default-position="defaultPosition"
    :persist-rect="persistRect"
    :strategy="strategy"
  >
    <Teleport to="body">
      <FloatingPanel.Positioner class="z-[9999]">
        <FloatingPanel.Content
          :class="[
            'z-[9999] rounded-xl bg-surface-overlay border border-border-muted shadow-2xl',
            'min-w-[320px] max-w-[90vw] max-h-[90vh]',
            'flex flex-col',
            'fixed',
            contentClass
          ]"
        >
          <FloatingPanel.DragTrigger v-if="draggable">
            <FloatingPanel.Header
              :class="[
                'p-4 border-b border-border-muted cursor-move select-none',
                headerClass
              ]"
            >
              <slot name="header">
                <OuiFlex justify="between" align="center">
                  <OuiStack gap="xs" class="flex-1">
                    <OuiText v-if="title" as="h3" size="lg" weight="semibold">
                      {{ title }}
                    </OuiText>
                    <OuiText v-if="description" as="p" size="xs" color="secondary">
                      {{ description }}
                    </OuiText>
                  </OuiStack>
                  <FloatingPanel.CloseTrigger v-if="showClose">
                    <OuiButton variant="ghost" size="xs" @click="handleClose">
                      Close
                    </OuiButton>
                  </FloatingPanel.CloseTrigger>
                </OuiFlex>
              </slot>
            </FloatingPanel.Header>
          </FloatingPanel.DragTrigger>

          <FloatingPanel.Body :class="['flex-1 overflow-auto', bodyClass]">
            <slot />
          </FloatingPanel.Body>

          <div v-if="$slots.footer" :class="['p-4 border-t border-border-muted', footerClass]">
            <slot name="footer" />
          </div>
        </FloatingPanel.Content>
      </FloatingPanel.Positioner>
    </Teleport>
  </FloatingPanel.Root>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { FloatingPanel } from "@ark-ui/vue/floating-panel";

interface Props {
  modelValue: boolean;
  title?: string;
  description?: string;
  draggable?: boolean;
  resizable?: boolean;
  closeOnEscape?: boolean;
  showClose?: boolean;
  defaultPosition?: { x: number; y: number };
  persistRect?: boolean;
  strategy?: "absolute" | "fixed";
  contentClass?: string;
  headerClass?: string;
  bodyClass?: string;
  footerClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  draggable: true,
  resizable: false,
  closeOnEscape: true,
  showClose: true,
  persistRect: true,
  strategy: "fixed",
  contentClass: "",
  headerClass: "",
  bodyClass: "p-4",
  footerClass: "",
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
}>();

// Use direct prop access for v-model:open
const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => {
    emit("update:modelValue", v);
  },
});

// Emit close event when panel closes
watch(
  () => props.modelValue,
  (val, oldVal) => {
    if (!val && oldVal) {
      emit("close");
    }
  }
);

const handleClose = () => {
  emit("update:modelValue", false);
  emit("close");
};
</script>

