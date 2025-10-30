import { defineStore } from "pinia";

type OrgSummary = {
  id: string;
  name?: string | null;
  slug?: string | null;
};

const ORGS_CACHE_KEY = "obiente_orgs_cache";
const SELECTED_ORG_KEY = "selectedOrgId";

export const useOrganizationsStore = defineStore("organizations", () => {
  const orgs = ref<OrgSummary[]>([]);
  const currentOrgId = ref<string>("");
  const hydrated = ref(false);
  const storageListenerRegistered = ref(false);

  const currentOrg = computed<OrgSummary | null>(() => {
    if (!currentOrgId.value) return null;
    return orgs.value.find((o) => o.id === currentOrgId.value) || null;
  });

  function setCurrentOrg(id: string | null) {
    if (!id) {
      currentOrgId.value = "";
      return;
    }
    currentOrgId.value = id;
  }

  function persist() {
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
      if (sel) {
        currentOrgId.value = sel;
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
