import { defineStore } from "pinia";

type OrgSummary = {
  id: string;
  name?: string | null;
  slug?: string | null;
  credits?: bigint | number;
};

const ORGS_CACHE_KEY = "obiente_orgs_cache";
const SELECTED_ORG_KEY = "selectedOrgId";
const SELECTED_ORG_COOKIE = "obiente_selected_org_id";

export const useOrganizationsStore = defineStore("organizations", () => {
  const orgs = ref<OrgSummary[]>([]);
  // Use cookie for SSR compatibility - available on both server and client
  const orgIdCookie = useCookie<string>(SELECTED_ORG_COOKIE, {
    default: () => "",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    httpOnly: false, // Needs to be accessible from client-side JS
    maxAge: 60 * 60 * 24 * 365, // 1 year
  });
  const currentOrgId = ref<string>(orgIdCookie.value || "");
  const hydrated = ref(false);
  const storageListenerRegistered = ref(false);

  const currentOrg = computed<OrgSummary | null>(() => {
    if (!currentOrgId.value) return null;
    return orgs.value.find((o) => o.id === currentOrgId.value) || null;
  });

  function setCurrentOrg(id: string | null) {
    if (!id) {
      currentOrgId.value = "";
      orgIdCookie.value = "";
      return;
    }
    currentOrgId.value = id;
    orgIdCookie.value = id;
  }

  function persist() {
    // Always update cookie (works on both server and client)
    if (currentOrgId.value) {
      orgIdCookie.value = currentOrgId.value;
    } else {
      orgIdCookie.value = "";
    }
    
    // Also persist to localStorage on client (if hydrated)
    if (!import.meta.client || !hydrated.value) return;
    try {
      localStorage.setItem(ORGS_CACHE_KEY, JSON.stringify(orgs.value));
      if (currentOrgId.value) {
        localStorage.setItem(SELECTED_ORG_KEY, currentOrgId.value);
      } else {
        localStorage.removeItem(SELECTED_ORG_KEY);
      }
    } catch {
      /* ignore */
    }
  }

  function hydrate() {
    if (hydrated.value || !import.meta.client) return;
    try {
      const raw = localStorage.getItem(ORGS_CACHE_KEY);
      const sel = localStorage.getItem(SELECTED_ORG_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as OrgSummary[];
        if (Array.isArray(parsed)) {
          orgs.value = parsed;
        }
      }
      // Prefer cookie over localStorage (cookie is source of truth for SSR)
      const cookieOrgId = orgIdCookie.value;
      if (cookieOrgId) {
        currentOrgId.value = cookieOrgId;
      } else if (sel) {
        currentOrgId.value = sel;
        // Sync to cookie if we got it from localStorage
        orgIdCookie.value = sel;
      }
    } catch {
      /* ignore */
    } finally {
      hydrated.value = true;
    }
  }

  function setOrganizations(list: Array<Partial<OrgSummary>>) {
    const normalized = list.map((org) => ({
      id: String(org.id),
      name: org.name ?? null,
      slug: org.slug ?? null,
      credits: org.credits !== undefined ? (typeof org.credits === 'bigint' ? Number(org.credits) : org.credits) : undefined,
    }));
    orgs.value = normalized;
    if (normalized.length && !normalized.find((o) => o.id === currentOrgId.value)) {
      setCurrentOrg(normalized[0]?.id || null);
    } else if (!normalized.length) {
      setCurrentOrg(null);
    }
    persist();
  }

  function switchOrganization(id: string) {
    if (!id) return;
    setCurrentOrg(id);
    persist();
    notifyOrganizationsUpdated();
  }

  function notifyOrganizationsUpdated() {
    if (!import.meta.client) return;
    localStorage.setItem("orgsUpdated", String(Date.now()));
  }

  function reset() {
    orgs.value = [];
    currentOrgId.value = "";
    orgIdCookie.value = "";
    hydrated.value = false;
    if (import.meta.client) {
      localStorage.removeItem(ORGS_CACHE_KEY);
      localStorage.removeItem(SELECTED_ORG_KEY);
    }
  }

  watch(
    () => [...orgs.value],
    () => {
      persist();
    },
    { deep: true }
  );

  watch(currentOrgId, () => {
    persist();
  });

  if (import.meta.client && !storageListenerRegistered.value) {
    hydrate();
    const handler = (event: StorageEvent) => {
      if (event.key === ORGS_CACHE_KEY) {
        const raw = event.newValue;
        if (raw) {
          try {
            const parsed = JSON.parse(raw) as OrgSummary[];
            if (Array.isArray(parsed)) {
              orgs.value = parsed;
            }
          } catch {
            /* ignore */
          }
        }
      }
      if (event.key === SELECTED_ORG_KEY) {
        setCurrentOrg(event.newValue || null);
      }
    };
    window.addEventListener("storage", handler);
    storageListenerRegistered.value = true;
  } else {
    hydrate();
  }

  return {
    orgs,
    currentOrgId,
    currentOrg,
    setOrganizations,
    switchOrganization,
    notifyOrganizationsUpdated,
    reset,
    hydrate,
  };
});
