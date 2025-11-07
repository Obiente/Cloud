import { useOrganizationsStore } from "~/stores/organizations";
import { useAuth } from "~/composables/useAuth";

/**
 * Composable to get the current organization ID that works during SSR.
 * Uses cookie as the source of truth (available on both server and client).
 * 
 * @returns A ref containing the current organization ID (empty string if none selected)
 * 
 * @example
 * ```ts
 * const orgId = useOrganizationId();
 * 
 * // Use in cache key
 * const { data } = await useAsyncData(
 *   () => `my-data-${orgId.value}`,
 *   async () => {
 *     const id = orgId.value;
 *     // Fetch data...
 *   }
 * );
 * ```
 */
export function useOrganizationId() {
  const orgsStore = useOrganizationsStore();
  const auth = useAuth();
  
  // Use cookie as source of truth (works on both server and client for SSR)
  const orgIdCookie = useCookie<string>("obiente_selected_org_id", {
    default: () => "",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    httpOnly: false, // Needs to be accessible from client-side JS
    maxAge: 60 * 60 * 24 * 365, // 1 year
  });

  // On client, sync cookie with store if they differ
  if (import.meta.client) {
    const storeOrgId = orgsStore.currentOrgId || auth.currentOrganizationId || "";
    const cookieOrgId = orgIdCookie.value || "";
    
    // If store has org ID but cookie doesn't, sync cookie
    if (storeOrgId && !cookieOrgId) {
      orgIdCookie.value = storeOrgId;
    }
    // If cookie has org ID but store doesn't, sync store
    else if (cookieOrgId && !storeOrgId) {
      orgsStore.switchOrganization(cookieOrgId);
    }
  }

  // Return the cookie value (source of truth for SSR compatibility)
  return computed(() => orgIdCookie.value || "");
}

