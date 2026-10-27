import { Code, ConnectError } from "@connectrpc/connect";

export async function runReconnectingStream<T>(
  signal: AbortSignal,
  connect: (signal: AbortSignal) => AsyncIterable<T>,
  receive: (value: T) => void,
  active: (value: boolean) => void,
  retryDelay = 1000,
) {
  let attempts = 0;
  while (!signal.aborted) {
    try {
      for await (const value of connect(signal)) {
        if (signal.aborted) return;
        attempts = 0;
        active(true);
        receive(value);
      }
    } catch (error) {
      if (signal.aborted) return;
      if (error instanceof ConnectError && [Code.PermissionDenied, Code.Unauthenticated, Code.NotFound].includes(error.code)) return;
    } finally { active(false); }
    if (signal.aborted) return;
    await new Promise<void>(resolve => {
      const done = () => { clearTimeout(timer); signal.removeEventListener("abort", done); resolve(); };
      const timer = setTimeout(done, Math.min(30000, retryDelay * 2 ** Math.min(attempts++, 5)));
      signal.addEventListener("abort", done, { once: true });
    });
  }
}
