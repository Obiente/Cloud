import { describe, expect, it, vi } from "vitest";
import { useResourceOperation } from "../app/composables/useResourceOperation";

describe("useResourceOperation", () => {
  it("tracks active operations and clears them when finished", () => {
    const operation = useResourceOperation();

    operation.beginOperation({
      kind: "start",
      label: "Starting resource",
      description: "Waiting for backend state.",
      failureMessage: "Failed to start resource",
    });

    expect(operation.isOperationActive.value).toBe(true);
    expect(operation.activeOperation.value?.kind).toBe("start");
    expect(operation.operationError.value).toBeNull();

    operation.finishOperation();

    expect(operation.isOperationActive.value).toBe(false);
    expect(operation.activeOperation.value).toBeNull();
  });

  it("keeps failure messages visible after an operation fails", () => {
    const operation = useResourceOperation();

    operation.beginOperation({
      kind: "stop",
      label: "Stopping resource",
      description: "Waiting for backend state.",
      failureMessage: "Failed to stop resource",
    });

    operation.failOperation("Docker host unavailable");

    expect(operation.isOperationActive.value).toBe(false);
    expect(operation.activeOperation.value).toBeNull();
    expect(operation.operationError.value).toBe("Docker host unavailable");
  });

  it("turns slow commands into visible timeout errors", async () => {
    vi.useFakeTimers();
    const operation = useResourceOperation({ timeoutMs: 1_000 });

    operation.beginOperation({
      kind: "restart",
      label: "Restarting resource",
      description: "Waiting for backend state.",
      failureMessage: "Failed to restart resource",
    });

    await vi.advanceTimersByTimeAsync(1_000);

    expect(operation.isOperationActive.value).toBe(false);
    expect(operation.operationError.value).toContain("Restarting resource is taking longer than expected");
    vi.useRealTimers();
  });
});
