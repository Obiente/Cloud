<template>
  <Dialog.Root :open="open" @open-change="handleOpenChange">
    <Teleport to="body">
      <Dialog.Backdrop 
        class="
          fixed inset-0 z-40 bg-black/50 backdrop-blur-sm
          animate-in fade-in-0 duration-200
          data-[state=closed]:animate-out data-[state=closed]:fade-out-0
        " 
      />
      <Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <Dialog.Content
          class="
            card-base w-full max-w-lg max-h-[85vh] overflow-auto
            animate-in fade-in-0 zoom-in-95 duration-200
            data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95
          "
        >
          <div class="card-header flex items-center justify-between">
            <Dialog.Title class="card-title">
              {{ title }}
            </Dialog.Title>
            <Dialog.CloseTrigger class="p-1 hover:bg-hover rounded-md transition-colors">
              <XMarkIcon class="h-5 w-5 text-text-secondary" />
            </Dialog.CloseTrigger>
          </div>
          
          <div class="card-body">
            <Dialog.Description v-if="description" class="card-description mb-4">
              {{ description }}
            </Dialog.Description>
            
            <slot />
          </div>
          
          <div v-if="$slots.footer" class="card-footer flex justify-end space-x-3">
            <slot name="footer" />
          </div>
        </Dialog.Content>
      </Dialog.Positioner>
    </Teleport>
  </Dialog.Root>
</template>

<script setup lang="ts">
import { Dialog } from '@ark-ui/vue/dialog'
import { XMarkIcon } from '@heroicons/vue/24/outline'

interface Props {
  open?: boolean
  title: string
  description?: string
}

defineProps<Props>()

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

const handleOpenChange = (details: Dialog.OpenChangeDetails) => {
  emit('update:open', details.open)
}
</script>