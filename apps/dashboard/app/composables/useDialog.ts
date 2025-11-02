import { ref } from "vue";

export interface AlertOptions {
  title?: string;
  message: string;
  confirmLabel?: string;
}

export interface ConfirmOptions {
  title?: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "default" | "danger";
}

export interface PromptOptions {
  title?: string;
  message: string;
  placeholder?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  defaultValue?: string;
}

interface DialogState {
  open: boolean;
  type: "alert" | "confirm" | "prompt" | null;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  variant: "default" | "danger";
  placeholder?: string;
  defaultValue?: string;
  resolve: ((value: boolean | string | null) => void) | null;
}

const dialogState = ref<DialogState>({
  open: false,
  type: null,
  title: "",
  message: "",
  confirmLabel: "OK",
  cancelLabel: "Cancel",
  variant: "default",
  placeholder: "",
  defaultValue: "",
  resolve: null,
});

export function useDialog() {
  const showAlert = (options: AlertOptions | string): Promise<void> => {
    return new Promise((resolve) => {
      const opts: AlertOptions =
        typeof options === "string" ? { message: options } : options;
      
      dialogState.value = {
        open: true,
        type: "alert",
        title: opts.title || "Alert",
        message: opts.message,
        confirmLabel: opts.confirmLabel || "OK",
        cancelLabel: "Cancel",
        variant: "default",
        resolve: () => {
          resolve();
        },
      };
    });
  };

  const showConfirm = (
    options: ConfirmOptions | string
  ): Promise<boolean> => {
    return new Promise((resolve) => {
      const opts: ConfirmOptions =
        typeof options === "string" ? { message: options } : options;
      
      dialogState.value = {
        open: true,
        type: "confirm",
        title: opts.title || "Confirm",
        message: opts.message,
        confirmLabel: opts.confirmLabel || "Confirm",
        cancelLabel: opts.cancelLabel || "Cancel",
        variant: opts.variant || "default",
        resolve: (value: boolean | string | null) => {
          resolve(value as boolean);
        },
      };
    });
  };

  const showPrompt = (options: PromptOptions): Promise<string | null> => {
    return new Promise((resolve) => {
      dialogState.value = {
        open: true,
        type: "prompt",
        title: options.title || "Input",
        message: options.message,
        confirmLabel: options.confirmLabel || "OK",
        cancelLabel: options.cancelLabel || "Cancel",
        variant: "default",
        placeholder: options.placeholder || "",
        defaultValue: options.defaultValue || "",
        resolve: (value: boolean | string | null) => {
          resolve(value as string | null);
        },
      };
    });
  };

  const handleConfirm = () => {
    if (dialogState.value.resolve) {
      dialogState.value.resolve(true);
      dialogState.value.resolve = null;
    }
    dialogState.value.open = false;
  };

  const handleCancel = () => {
    if (dialogState.value.resolve) {
      if (dialogState.value.type === "prompt") {
        dialogState.value.resolve(null);
      } else {
        dialogState.value.resolve(false);
      }
      dialogState.value.resolve = null;
    }
    dialogState.value.open = false;
  };

  const handleClose = () => {
    // For alert, close is same as confirm
    // For confirm/prompt, close is same as cancel
    if (dialogState.value.type === "alert") {
      handleConfirm();
    } else {
      handleCancel();
    }
  };

  const handlePromptConfirm = (value: string) => {
    if (dialogState.value.resolve) {
      dialogState.value.resolve(value);
      dialogState.value.resolve = null;
    }
    dialogState.value.open = false;
  };

  return {
    dialogState,
    showAlert,
    showConfirm,
    showPrompt,
    handleConfirm,
    handleCancel,
    handleClose,
    handlePromptConfirm,
  };
}

