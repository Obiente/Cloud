import { onMounted, onUnmounted } from "vue";
import { createRefreshLoop } from "../utils/refreshLoop";

export function useAutoRefresh(refresh: () => Promise<unknown>, interval: number | (() => number) = 10000) {
  let loop: ReturnType<typeof createRefreshLoop> | undefined;
  const resume = () => { if (document.visibilityState === "visible") void loop?.refresh(); };
  onMounted(() => {
    loop = createRefreshLoop(refresh, () => typeof interval === "function" ? interval() : interval,
      () => document.visibilityState === "visible" && navigator.onLine);
    document.addEventListener("visibilitychange", resume);
    window.addEventListener("focus", resume);
    window.addEventListener("online", resume);
  });
  onUnmounted(() => {
    loop?.stop();
    document.removeEventListener("visibilitychange", resume);
    window.removeEventListener("focus", resume);
    window.removeEventListener("online", resume);
  });
}
