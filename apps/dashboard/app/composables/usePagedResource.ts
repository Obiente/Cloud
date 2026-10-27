import { ref, shallowRef } from "vue";

export function usePagedResource<T extends { id: string }>(fetchPage: (offset: number, signal: AbortSignal) => Promise<{ items: T[]; total: number }>) {
  const items = shallowRef<T[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const error = shallowRef<Error | null>(null);
  let offset = 0;
  let generation = 0;
  let controller: AbortController | undefined;
  const loadMore = async () => {
    if (loading.value || (offset > 0 && offset >= total.value)) return;
    const current = generation;
    controller = new AbortController();
    loading.value = true;
    error.value = null;
    try {
      const page = await fetchPage(offset, controller.signal);
      if (current !== generation) return;
      if (page.items.length === 0 && page.total > offset) throw new Error("This page could not be loaded. Please refresh and try again.");
      const merged = new Map(items.value.map(item => [item.id, item]));
      for (const item of page.items) merged.set(item.id, item);
      items.value = [...merged.values()];
      offset += page.items.length;
      total.value = page.total;
    } catch (err) { if (current === generation) error.value = err instanceof Error ? err : new Error(String(err)); }
    finally { if (current === generation) loading.value = false; }
  };
  const dispose = () => { generation++; controller?.abort(); loading.value = false; };
  const refresh = () => { dispose(); offset = 0; items.value = []; total.value = 0; return loadMore(); };
  return { items, total, loading, error, loadMore, refresh, dispose };
}
