export function createRefreshLoop(fetch: () => Promise<unknown>, interval: () => number, enabled = () => true) {
  let stopped = false;
  let busy = false;
  let timer: ReturnType<typeof setTimeout>;
  const schedule = () => { if (!stopped) timer = setTimeout(refresh, Math.max(100, interval())); };
  const refresh = async () => {
    if (stopped || busy) return;
    clearTimeout(timer);
    busy = true;
    try { if (enabled()) await fetch(); }
    catch { /* The resource exposes its error; a failed request must not stop recovery. */ }
    finally { busy = false; schedule(); }
  };
  schedule();
  return { refresh, stop() { stopped = true; clearTimeout(timer); } };
}
