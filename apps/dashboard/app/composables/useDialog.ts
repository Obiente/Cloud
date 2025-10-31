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

interface DialogState {
  open: boolean;
  type: "alert" | "confirm" | null;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  variant: "default" | "danger";
  resolve: ((value: boolean) => void) | null;
}

const dialogState = ref<DialogState>({
  open: false,
  type: null,
  title: "",
  message: "",
  confirmLabel: "OK",
  cancelLabel: "Cancel",
  variant: "default",
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
        resolve: (confirmed: boolean) => {
          resolve(confirmed);
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
      dialogState.value.resolve(false);
      dialogState.value.resolve = null;
    }
    dialogState.value.open = false;
  };

  const handleClose = () => {
    // For alert, close is same as confirm
    // For confirm, close is same as cancel
    if (dialogState.value.type === "alert") {
      handleConfirm();
    } else {
      handleCancel();
    }
  };

  return {
    dialogState,
    showAlert,
    showConfirm,
    handleConfirm,
    handleCancel,
    handleClose,
  };
}

