import { defineStore } from "pinia";

const PREFERENCES_KEY = "obiente_preferences";

interface Preferences {
  envVarsViewMode: "list" | "file";
  // Add more preferences here as needed
}

const defaultPreferences: Preferences = {
  envVarsViewMode: "list",
};

export const usePreferencesStore = defineStore("preferences", () => {
  const preferences = ref<Preferences>({ ...defaultPreferences });
  const hydrated = ref(false);

  function persist() {
    if (!import.meta.client || !hydrated.value) return;
    try {
      localStorage.setItem(PREFERENCES_KEY, JSON.stringify(preferences.value));
    } catch {
      /* ignore */
    }
  }

  function hydrate() {
    if (hydrated.value || !import.meta.client) return;
    try {
      const raw = localStorage.getItem(PREFERENCES_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as Partial<Preferences>;
        // Merge with defaults to ensure all properties exist
        preferences.value = { ...defaultPreferences, ...parsed };
      }
    } catch {
      /* ignore */
    } finally {
      hydrated.value = true;
    }
  }

  function setEnvVarsViewMode(mode: "list" | "file") {
    preferences.value.envVarsViewMode = mode;
    persist();
  }

  // Watch for changes and persist
  watch(
    () => preferences.value,
    () => {
      persist();
    },
    { deep: true }
  );

  // Hydrate on initialization
  if (import.meta.client) {
    hydrate();
  }

  return {
    preferences,
    envVarsViewMode: computed(() => preferences.value.envVarsViewMode),
    setEnvVarsViewMode,
    hydrate,
  };
});

