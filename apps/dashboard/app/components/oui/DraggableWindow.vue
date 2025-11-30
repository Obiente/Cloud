<template>
  <FloatingPanel.Root
    v-model:open="isOpen"
    :draggable="draggable"
    :resizable="resizable"
    :close-on-escape="closeOnEscape"
    :default-position="defaultPosition"
    :persist-rect="persistRect"
    :strategy="strategy"
    :id="id"
  >
    <Teleport to="body">
      <FloatingPanel.Positioner :class="['z-[9999]', positionerClass]">
        <FloatingPanel.Content
          :class="[
            'z-[9999] rounded-xl bg-surface-base border border-border-default shadow-2xl',
            'flex flex-col',
            'min-w-[400px] min-h-[300px] max-w-[90vw] max-h-[90vh]',
            minimized ? 'hidden' : '',
            contentClass
          ]"
          :style="{
            width: size?.width ? `${size.width}px` : undefined,
            height: size?.height ? `${size.height}px` : undefined,
          }"
          :aria-label="title || 'Window'"
          :aria-describedby="description ? `${id}-description` : undefined"
          role="dialog"
          v-bind="$attrs"
        >
          <FloatingPanel.DragTrigger v-if="draggable">
            <FloatingPanel.Header
              :class="[
                'flex items-center justify-between px-4 py-2 bg-surface-raised border-b border-border-default',
                'cursor-move select-none',
                headerClass
              ]"
              :aria-label="`${title || 'Window'} header, drag to move`"
            >
              <slot name="header">
                <OuiFlex gap="sm" align="center" class="flex-1 min-w-0">
                  <OuiText v-if="title" as="h3" size="sm" weight="semibold" truncate>
                    {{ title }}
                  </OuiText>
                  <OuiText
                    v-if="description"
                    :id="`${id}-description`"
                    size="xs"
                    color="secondary"
                    class="sr-only"
                  >
                    {{ description }}
                  </OuiText>
                </OuiFlex>
                <OuiFlex gap="xs">
                  <OuiButton
                    v-if="minimizable"
                    variant="ghost"
                    size="xs"
                    class="!p-1"
                    :aria-label="minimized ? 'Restore window' : 'Minimize window'"
                    @click="handleMinimize"
                  >
                    <MinusIcon class="w-4 h-4" />
                  </OuiButton>
                  <FloatingPanel.CloseTrigger v-if="showClose">
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      class="!p-1"
                      aria-label="Close window"
                      @click="handleClose"
                    >
                      <XMarkIcon class="w-4 h-4" />
                    </OuiButton>
                  </FloatingPanel.CloseTrigger>
                </OuiFlex>
              </slot>
            </FloatingPanel.Header>
          </FloatingPanel.DragTrigger>

          <FloatingPanel.Body :class="['flex-1 overflow-auto min-h-0', bodyClass]">
            <slot />
          </FloatingPanel.Body>

          <div v-if="$slots.footer" :class="['border-t border-border-default bg-surface-raised px-4 py-3 shrink-0', footerClass]">
            <slot name="footer" />
          </div>
        </FloatingPanel.Content>
      </FloatingPanel.Positioner>

      <!-- Minimized state -->
      <div
        v-if="minimized && minimizable"
        class="fixed z-40 bg-surface-raised border border-border-default rounded-t-lg shadow-lg px-4 py-2 cursor-pointer"
        :style="{
          left: `${minimizedPosition.x}px`,
          top: `${minimizedPosition.y}px`,
          width: '300px',
        }"
        :aria-label="`${title || 'Window'} minimized, click to restore`"
        role="button"
        tabindex="0"
        @click="handleRestore"
        @keydown.enter="handleRestore"
        @keydown.space.prevent="handleRestore"
      >
        <OuiFlex justify="between" align="center">
          <OuiText size="sm" weight="semibold" truncate>
            {{ title || 'Window' }}
          </OuiText>
          <OuiButton
            variant="ghost"
            size="xs"
            @click.stop="handleClose"
            class="!p-1"
            aria-label="Close window"
          >
            <XMarkIcon class="w-4 h-4" />
          </OuiButton>
        </OuiFlex>
      </div>
    </Teleport>
  </FloatingPanel.Root>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { FloatingPanel } from "@ark-ui/vue/floating-panel";
import { MinusIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { useUniqueId } from "~/composables/useId";

interface Props {
  modelValue: boolean;
  title?: string;
  description?: string;
  draggable?: boolean;
  resizable?: boolean;
  minimizable?: boolean;
  closeOnEscape?: boolean;
  showClose?: boolean;
  defaultPosition?: { x: number; y: number };
  size?: { width: number; height: number };
  persistRect?: boolean;
  strategy?: "absolute" | "fixed";
  contentClass?: string;
  headerClass?: string;
  bodyClass?: string;
  footerClass?: string;
  positionerClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  draggable: true,
  resizable: false,
  minimizable: true,
  closeOnEscape: true,
  showClose: true,
  persistRect: true,
  strategy: "fixed",
  contentClass: "",
  headerClass: "",
  bodyClass: "p-4",
  footerClass: "",
  positionerClass: "",
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
  minimize: [];
  restore: [];
}>();

const id = useUniqueId();

const isOpen = computed({
  get: () => props.modelValue,
  set: (v: boolean) => {
    emit("update:modelValue", v);
  },
});

const minimized = ref(false);
const minimizedPosition = ref({ x: 100, y: 100 });

watch(
  () => props.modelValue,
  (val, oldVal) => {
    if (!val && oldVal) {
      emit("close");
    }
  }
);

function handleClose() {
  emit("update:modelValue", false);
  emit("close");
}

function handleMinimize() {
  minimized.value = true;
  emit("minimize");
}

function handleRestore() {
  minimized.value = false;
  emit("restore");
}

defineOptions({
  inheritAttrs: false,
});
</script>

