import { afterEach, expect, it, vi } from "vitest";
import { runReconnectingStream } from "../app/utils/reconnectingStream";
afterEach(() => vi.useRealTimers());
it("reconnects after an interrupted stream and propagates cancellation", async () => {
  vi.useFakeTimers();
  const controller = new AbortController();
  let attempts = 0;
  const receive = vi.fn();
  const run = runReconnectingStream(controller.signal, async function* (signal) {
    expect(signal).toBe(controller.signal);
    attempts++;
    if (attempts === 1) throw new Error("connection lost");
    yield 42;
  }, receive, vi.fn());
  await vi.advanceTimersByTimeAsync(1000);
  expect(attempts).toBe(2);
  expect(receive).toHaveBeenCalledWith(42);
  controller.abort();
  await run;
  await vi.advanceTimersByTimeAsync(60000);
  expect(attempts).toBe(2);
  expect(vi.getTimerCount()).toBe(0);
});
