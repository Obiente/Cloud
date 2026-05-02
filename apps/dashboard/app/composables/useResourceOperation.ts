import { computed, getCurrentInstance, onUnmounted, ref } from "vue";

export type ResourceOperation = {
  kind: string;
  label: string;
  description: string;
  failureMessage: string;
};

export function useResourceOperation(options: { timeoutMs?: number } = {}) {
  const timeoutMs = options.timeoutMs ?? 90_000;
  const activeOperation = ref<ResourceOperation | null>(null);
  const operationError = ref<string | null>(null);
  const isOperationActive = computed(() => activeOperation.value !== null);
  let timeoutId: ReturnType<typeof setTimeout> | null = null;

  const clearOperationTimeout = () => {
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  };

  const finishOperation = () => {
    clearOperationTimeout();
    activeOperation.value = null;
  };

  const failOperation = (message: string) => {
    operationError.value = message;
    finishOperation();
  };

  const beginOperation = (operation: ResourceOperation) => {
    clearOperationTimeout();
    operationError.value = null;
    activeOperation.value = operation;
    timeoutId = setTimeout(() => {
      if (!activeOperation.value) return;
      operationError.value = `${activeOperation.value.label} is taking longer than expected. The page is still safe to refresh, and the latest backend state will be used when it responds.`;
      finishOperation();
    }, timeoutMs);
  };

  const getErrorMessage = (error: unknown, fallback: string) => {
    if (error instanceof Error && error.message) return error.message;
    return fallback;
  };

  if (getCurrentInstance()) {
    onUnmounted(clearOperationTimeout);
  }

  return {
    activeOperation,
    operationError,
    isOperationActive,
    beginOperation,
    finishOperation,
    failOperation,
    getErrorMessage,
  };
}
