import { afterEach, expect, it, vi } from "vitest";
import { createRefreshLoop } from "../app/utils/refreshLoop";

afterEach(() => vi.useRealTimers());
it("recovers from an initial outage without a reload and stops after disposal", async () => {
  vi.useFakeTimers();
  const fetch = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValue("fresh");
  const loop = createRefreshLoop(fetch, () => 1000);
  await vi.advanceTimersByTimeAsync(2000);
  expect(fetch).toHaveBeenCalledTimes(2);
  loop.stop();
  await vi.advanceTimersByTimeAsync(5000);
  expect(fetch).toHaveBeenCalledTimes(2);
});
it("does not overlap slow requests or reschedule after an in-flight unmount", async () => {
  vi.useFakeTimers();
  let finish!: () => void;
  const fetch = vi.fn(() => new Promise<void>(resolve => { finish = resolve; }));
  const loop = createRefreshLoop(fetch, () => 1000);
  const pending = loop.refresh();
  await loop.refresh();
  await vi.advanceTimersByTimeAsync(5000);
  expect(fetch).toHaveBeenCalledTimes(1);
  loop.stop(); finish(); await pending;
  expect(vi.getTimerCount()).toBe(0);
});
