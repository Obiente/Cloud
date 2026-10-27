import { computed, watch, type ComputedRef } from "vue";

/**
 * Optimized data fetching composable
 * Uses SSR for initial load, then client-side fetching for subsequent navigations
 * This provides fast initial page load (SSR) and instant navigation (client-side)
 * Navigation is always non-blocking - pages render immediately with loading states
 */
export function useClientFetch<T>(
  key: string | (() => string),
  fn: () => Promise<T>,
  options: {
    watch?: any[];
    immediate?: boolean;
    default?: () => T;
    server?: boolean;
    lazy?: boolean;
    preserveDataOnKeyChange?: boolean;
  } = {}
) {
  // Nuxt uses this option on the client too when restoring the SSR payload.
  // Keep it consistent across hydration and let Nuxt schedule later navigation.
  const shouldUseServer = options.server !== false;

  // Always use lazy mode for non-blocking navigation
  // Pages render immediately, data loads in background
  // If lazy is explicitly set to false, respect it (but this is not recommended)
  const shouldBeLazy = options.lazy !== false;

  const { preserveDataOnKeyChange, ...asyncDataOptions } = options;
  const asyncDataKey = typeof key === "function" ? computed(key) : key;

  const asyncData = useAsyncData<T>(
    asyncDataKey,
    fn,
    {
      ...asyncDataOptions,
      // Use server only on initial SSR load
      server: shouldUseServer,
      // Always lazy for non-blocking navigation
      lazy: shouldBeLazy,
    }
  );

  if (import.meta.client && typeof key === "function" && preserveDataOnKeyChange !== true) {
    const dynamicKey = asyncDataKey as ComputedRef<string>;
    watch(
      dynamicKey,
      (newKey, oldKey) => {
        if (newKey !== oldKey) {
          asyncData.data.value = options.default ? options.default() : (undefined as T);
        }
      },
      { flush: "sync" }
    );
  }

  return asyncData;
}
