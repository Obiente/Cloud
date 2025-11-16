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
  } = {}
) {
  // Check if we're in a client-side navigation context
  // On initial SSR, use server-side for fast first paint
  // On client navigation, use client-side for instant navigation
  const isInitialSSR = import.meta.server;
  const isClientNav = import.meta.client && !isInitialSSR;
  
  // Determine if we should use server-side fetching
  // - Use server on initial SSR (unless explicitly disabled)
  // - Use client on navigation (faster, no server round-trip)
  const shouldUseServer = options.server !== false && isInitialSSR;
  
  // Always use lazy mode for non-blocking navigation
  // Pages render immediately, data loads in background
  // If lazy is explicitly set to false, respect it (but this is not recommended)
  const shouldBeLazy = options.lazy !== false;
  
  return useAsyncData<T>(
    typeof key === 'function' ? key() : key,
    fn,
    {
      ...options,
      // Use server only on initial SSR load
      server: shouldUseServer,
      // Always lazy for non-blocking navigation
      lazy: shouldBeLazy,
    }
  );
}

