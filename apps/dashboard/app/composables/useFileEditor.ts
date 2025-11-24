import { ref, computed, nextTick } from "vue";
import { useDialog } from "./useDialog";
import type { Ref } from "vue";

export type FileWriter = (path: string, content: string) => Promise<void>;

export function useFileEditor(
  writer: FileWriter,
  options: {
    fileContent: Ref<string>;
    originalFileContent: Ref<string>;
    currentFilePath: Ref<string | null>;
    onSaveSuccess?: () => void;
  }
) {
  const { fileContent, originalFileContent, currentFilePath, onSaveSuccess } = options;
  const dialog = useDialog();

  const isSaving = ref(false);
  const saveStatus = ref<"idle" | "saving" | "success" | "error">("idle");
  const saveErrorMessage = ref<string | null>(null);

  const hasUnsavedChanges = computed(() => {
    if (!currentFilePath.value) return false;
    return fileContent.value !== originalFileContent.value;
  });

  async function saveFile() {
    if (!currentFilePath.value) {
      console.warn("Cannot save: no file path");
      return;
    }
    if (isSaving.value) {
      console.log("Save already in progress, skipping");
      return;
    }

    console.log("Starting save for:", currentFilePath.value);
    isSaving.value = true;
    saveStatus.value = "saving";
    saveErrorMessage.value = null;

    await nextTick();

    try {
      await writer(currentFilePath.value, fileContent.value);

      console.log("File saved successfully");
      saveStatus.value = "success";
      originalFileContent.value = fileContent.value;

      if (onSaveSuccess) {
        onSaveSuccess();
      }

      // Reset status after 3 seconds
      setTimeout(() => {
        if (saveStatus.value === "success") {
          saveStatus.value = "idle";
        }
      }, 3000);
    } catch (err: any) {
      console.error("save file error:", err);
      saveStatus.value = "error";

      const errorMsg = err?.message || "Failed to save file. Please try again.";
      saveErrorMessage.value = errorMsg;

      // Show error message dialog after showing status
      setTimeout(async () => {
        dialog
          .showAlert({
            title: "Save Failed",
            message: errorMsg,
            confirmLabel: "OK",
          })
          .catch(() => {});

        // Reset status after showing dialog (5 seconds total)
        setTimeout(() => {
          if (saveStatus.value === "error") {
            console.log("Resetting save status from error to idle");
            saveStatus.value = "idle";
            saveErrorMessage.value = null;
          }
        }, 3000);
      }, 1000);
    } finally {
      isSaving.value = false;
    }
  }

  function reset() {
    isSaving.value = false;
    saveStatus.value = "idle";
    saveErrorMessage.value = null;
  }

  return {
    isSaving,
    saveStatus,
    saveErrorMessage,
    hasUnsavedChanges,
    saveFile,
    reset,
  };
}

