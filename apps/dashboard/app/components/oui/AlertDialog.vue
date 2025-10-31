<template>
  <OuiDialog
    v-model:open="dialogState.open"
    :title="dialogState.title"
    :description="dialogState.message"
    :close-on-interact-outside="dialogState.type === 'alert'"
    @update:open="handleOpenChange"
  >
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton
          v-if="dialogState.type === 'confirm'"
          @click="handleCancel"
          variant="ghost"
        >
          {{ dialogState.cancelLabel }}
        </OuiButton>
        <OuiButton
          @click="handleConfirm"
          :variant="dialogState.variant === 'danger' ? 'solid' : 'solid'"
          :color="dialogState.variant === 'danger' ? 'danger' : 'primary'"
        >
          {{ dialogState.confirmLabel }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { useDialog } from "~/composables/useDialog";
import OuiDialog from "./Dialog.vue";
import OuiButton from "./Button.vue";
import OuiFlex from "./Flex.vue";

const { dialogState, handleConfirm, handleCancel } = useDialog();

const handleOpenChange = (open: boolean) => {
  if (!open && dialogState.value.type === "confirm" && dialogState.value.resolve) {
    // If dialog is closed without clicking a button, treat as cancel
    handleCancel();
  } else if (!open && dialogState.value.type === "alert" && dialogState.value.resolve) {
    // If alert is closed, treat as confirm
    handleConfirm();
  }
};
</script>

