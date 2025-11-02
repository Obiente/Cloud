<template>
  <OuiDialog
    v-model:open="dialogState.open"
    :title="dialogState.title"
    :description="dialogState.message"
    :close-on-interact-outside="dialogState.type === 'alert'"
    @update:open="handleOpenChange"
  >
    <!-- Prompt input field -->
    <OuiInput
      v-if="dialogState.type === 'prompt'"
      v-model="promptValue"
      :placeholder="dialogState.placeholder"
      class="mt-4"
      @keyup.enter="handlePromptConfirm"
    />

    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton
          v-if="dialogState.type === 'confirm' || dialogState.type === 'prompt'"
          @click="handleCancel"
          variant="ghost"
        >
          {{ dialogState.cancelLabel }}
        </OuiButton>
        <OuiButton
          @click="handleConfirmClick"
          :variant="dialogState.variant === 'danger' ? 'solid' : 'solid'"
          :color="dialogState.variant === 'danger' ? 'danger' : 'primary'"
          :disabled="dialogState.type === 'prompt' && !promptValue.trim()"
        >
          {{ dialogState.confirmLabel }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useDialog } from "~/composables/useDialog";
import OuiDialog from "./Dialog.vue";
import OuiButton from "./Button.vue";
import OuiFlex from "./Flex.vue";
import OuiInput from "./Input.vue";

const { dialogState, handleConfirm, handleCancel, handlePromptConfirm } = useDialog();
const promptValue = ref("");

// Watch for dialog opening to reset prompt value
watch(
  () => dialogState.value.open,
  (open) => {
    if (open && dialogState.value.type === "prompt") {
      promptValue.value = dialogState.value.defaultValue || "";
    }
  }
);

const handleConfirmClick = () => {
  if (dialogState.value.type === "prompt") {
    handlePromptConfirm(promptValue.value.trim());
  } else {
    handleConfirm();
  }
};

const handleOpenChange = (open: boolean) => {
  if (!open && dialogState.value.type === "confirm" && dialogState.value.resolve) {
    // If dialog is closed without clicking a button, treat as cancel
    handleCancel();
  } else if (!open && dialogState.value.type === "prompt" && dialogState.value.resolve) {
    // If prompt is closed, treat as cancel
    handleCancel();
  } else if (!open && dialogState.value.type === "alert" && dialogState.value.resolve) {
    // If alert is closed, treat as confirm
    handleConfirm();
  }
};
</script>

