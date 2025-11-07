<template>
  <!-- Only render dialog when mounted on client to avoid hydration mismatches -->
  <ClientOnly>
  <Dialog.Root 
    :open="open" 
    :closeOnInteractOutside="closeOnInteractOutside !== false"
    @open-change="handleOpenChange"
  >
      <Teleport to="body">
        <Dialog.Backdrop
          class="fixed inset-0 z-40 bg-background/80 backdrop-blur-sm animate-in fade-in-0 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0"
        />
        <Dialog.Positioner
          class="fixed inset-0 z-50 flex items-center justify-center p-4"
        >
          <Dialog.Content
            class="w-full max-w-lg max-h-[85vh] flex flex-col animate-in fade-in-0 zoom-in-95 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
          >
            <OuiCard variant="raised" class="shadow-2xl flex flex-col flex-1 min-h-0">
              <OuiCardHeader class="flex items-center justify-between flex-shrink-0">
                <Dialog.Title>
                  <OuiText v-if="title" as="h2" size="lg" weight="semibold" color="primary">
                    {{ title }}
                  </OuiText>
                  <span v-else class="sr-only">Dialog</span>
                </Dialog.Title>
                <Dialog.CloseTrigger>
                  <OuiButton
                    variant="ghost"
                    size="sm"
                    class="hover:bg-surface-hover transition-colors"
                    aria-label="Close dialog"
                  >
                    <XMarkIcon class="h-5 w-5" />
                  </OuiButton>
                </Dialog.CloseTrigger>
              </OuiCardHeader>

              <OuiCardBody class="flex-1 overflow-auto min-h-0">
                <Dialog.Description v-if="description" class="mb-4">
                  <OuiText color="secondary">
                    {{ description }}
                  </OuiText>
                </Dialog.Description>

                <slot />
              </OuiCardBody>

              <OuiCardFooter
                v-if="$slots.footer"
                class="flex justify-end space-x-3 flex-shrink-0"
              >
                <slot name="footer" />
              </OuiCardFooter>
            </OuiCard>
          </Dialog.Content>
        </Dialog.Positioner>
      </Teleport>
  </Dialog.Root>
  </ClientOnly>
</template>

<script setup lang="ts">
import { Dialog } from "@ark-ui/vue/dialog";
import { XMarkIcon } from "@heroicons/vue/24/outline";
import OuiCard from "./Card.vue";
import OuiCardHeader from "./CardHeader.vue";
import OuiCardBody from "./CardBody.vue";
import OuiCardFooter from "./CardFooter.vue";
import OuiText from "./Text.vue";
import OuiButton from "./Button.vue";

interface Props {
  open?: boolean;
  title: string;
  description?: string;
  closeOnInteractOutside?: boolean;
}

defineProps<Props>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const handleOpenChange = (details: Dialog.OpenChangeDetails) => {
  emit("update:open", details.open);
};
</script>
