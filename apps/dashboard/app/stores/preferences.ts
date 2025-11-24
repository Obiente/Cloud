import { defineStore } from "pinia";

const PREFERENCES_KEY = "obiente_preferences";

export interface EditorPreferences {
  wordWrap: "off" | "on" | "wordWrapColumn" | "bounded";
  tabSize: number;
  insertSpaces: boolean;
  fontSize: number;
  lineNumbers: "on" | "off" | "relative" | "interval";
  minimap: boolean;
  renderWhitespace: "none" | "boundary" | "selection" | "trailing" | "all";
}

export interface MetricsPreferences {
  timeframe: "10m" | "1h" | "24h" | "7d" | "30d" | "custom";
  customDateRange?: {
    start: string; // ISO string
    end: string; // ISO string
  };
}

interface Preferences {
  envVarsViewMode: "list" | "file";
  editor: EditorPreferences;
  metrics: MetricsPreferences;
  theme?: "dark" | "dark-purple";
  // Add more preferences here as needed
}

const defaultEditorPreferences: EditorPreferences = {
  wordWrap: "on",
  tabSize: 2,
  insertSpaces: true,
  fontSize: 14,
  lineNumbers: "on",
  minimap: true,
  renderWhitespace: "selection",
};

const defaultMetricsPreferences: MetricsPreferences = {
  timeframe: "24h",
};

const defaultPreferences: Preferences = {
  envVarsViewMode: "list",
  editor: defaultEditorPreferences,
  metrics: defaultMetricsPreferences,
};

export const usePreferencesStore = defineStore("preferences", () => {
  const preferences = ref<Preferences>({ ...defaultPreferences });
  const hydrated = ref(false);

  function persist() {
    if (!import.meta.client || !hydrated.value) {
      console.log("[PreferencesStore] persist() skipped:", {
        isClient: import.meta.client,
        hydrated: hydrated.value,
      });
      return;
    }
    try {
      const serialized = JSON.stringify(preferences.value);
      console.log("[PreferencesStore] Persisting preferences:", serialized);
      localStorage.setItem(PREFERENCES_KEY, serialized);

      // Verify it was actually saved
      const verify = localStorage.getItem(PREFERENCES_KEY);
      if (verify) {
        const parsed = JSON.parse(verify);
        console.log("[PreferencesStore] Verified save - metrics timeframe:", parsed.metrics?.timeframe);
      }
    } catch (error) {
      console.error("[PreferencesStore] Failed to persist preferences:", error);
    }
  }

  function hydrate() {
    if (hydrated.value || !import.meta.client) {
      console.log("[PreferencesStore] hydrate() skipped:", { hydrated: hydrated.value, isClient: import.meta.client });
      return;
    }
    console.log("[PreferencesStore] Starting hydration...");
    try {
      const raw = localStorage.getItem(PREFERENCES_KEY);
      console.log("[PreferencesStore] Raw from localStorage:", raw ? "exists" : "empty");
      
      // Preserve current preferences (in case user made changes before hydration)
      const currentMetrics = preferences.value.metrics ? { ...preferences.value.metrics } : null;
      const currentEnvVarsMode = preferences.value.envVarsViewMode;
      
      if (raw) {
        const parsed = JSON.parse(raw) as Partial<Preferences>;
        console.log("[PreferencesStore] Parsed preferences:", JSON.stringify(parsed, null, 2));
        // Start with defaults, merge saved values, then preserve any current changes
        preferences.value = { ...defaultPreferences };
        // Merge top-level properties
        if (parsed.envVarsViewMode) {
          preferences.value.envVarsViewMode = parsed.envVarsViewMode;
        }
        if (parsed.theme) {
          preferences.value.theme = parsed.theme;
        }
        // Ensure nested objects are merged correctly (always merge to preserve defaults)
        preferences.value.metrics = {
          ...defaultMetricsPreferences,
          ...(parsed.metrics || {}),
        };
        preferences.value.editor = {
          ...defaultEditorPreferences,
          ...(parsed.editor || {}),
        };
        
        // Preserve any changes made before hydration (current values take priority)
        // Only preserve if current value differs from default (indicating user changed it)
        if (currentMetrics?.timeframe && currentMetrics.timeframe !== defaultMetricsPreferences.timeframe) {
          preferences.value.metrics.timeframe = currentMetrics.timeframe;
        }
        if (currentMetrics?.customDateRange) {
          preferences.value.metrics.customDateRange = currentMetrics.customDateRange;
        }
        if (currentEnvVarsMode && currentEnvVarsMode !== defaultPreferences.envVarsViewMode) {
          preferences.value.envVarsViewMode = currentEnvVarsMode;
        }
        
        console.log("[PreferencesStore] After merge - metrics:", JSON.stringify(preferences.value.metrics, null, 2));
        console.log("[PreferencesStore] After merge - timeframe:", preferences.value.metrics.timeframe);
      } else {
        // No saved preferences, but preserve any current changes
        preferences.value = { ...defaultPreferences };
        if (currentMetrics?.timeframe && currentMetrics.timeframe !== defaultMetricsPreferences.timeframe) {
          preferences.value.metrics.timeframe = currentMetrics.timeframe;
        }
        if (currentMetrics?.customDateRange) {
          preferences.value.metrics.customDateRange = currentMetrics.customDateRange;
        }
        if (currentEnvVarsMode && currentEnvVarsMode !== defaultPreferences.envVarsViewMode) {
          preferences.value.envVarsViewMode = currentEnvVarsMode;
        }
        console.log("[PreferencesStore] No saved preferences, using defaults");
      }
    } catch (error) {
      console.error("[PreferencesStore] Error during hydration:", error);
      preferences.value = { ...defaultPreferences };
    } finally {
      hydrated.value = true;
      console.log("[PreferencesStore] Hydration complete, hydrated =", hydrated.value);
      // Persist after hydration to save any changes made before hydration
      persist();
    }
  }

  function setEnvVarsViewMode(mode: "list" | "file") {
    preferences.value.envVarsViewMode = mode;
    persist();
  }

  function setEditorPreference<K extends keyof EditorPreferences>(
    key: K,
    value: EditorPreferences[K]
  ) {
    preferences.value.editor[key] = value;
    persist();
  }

  function setEditorPreferences(newPrefs: Partial<EditorPreferences>) {
    preferences.value.editor = { ...preferences.value.editor, ...newPrefs };
    persist();
  }

  function setMetricsPreference<K extends keyof MetricsPreferences>(
    key: K,
    value: MetricsPreferences[K]
  ) {
    // Ensure metrics object exists
    if (!preferences.value.metrics) {
      preferences.value.metrics = { ...defaultMetricsPreferences };
    }
    preferences.value.metrics[key] = value;
    console.log("[PreferencesStore] Set metrics preference:", key, value);
    console.log(
      "[PreferencesStore] Current metrics:",
      preferences.value.metrics
    );
    persist();
  }

  function setMetricsPreferences(newPrefs: Partial<MetricsPreferences>) {
    preferences.value.metrics = { ...preferences.value.metrics, ...newPrefs };
    persist();
  }

  function setTheme(theme: "dark" | "dark-purple") {
    preferences.value.theme = theme;
    persist();
  }

  // Set up watcher to persist changes (only after hydration to avoid infinite loops)
  let unwatch: (() => void) | null = null;
  
  const setupWatcher = () => {
    if (unwatch) return; // Already set up
    
    unwatch = watch(
      () => preferences.value,
      (newVal, oldVal) => {
        // Skip persistence during hydration to prevent infinite loops
        if (!hydrated.value) {
          return;
        }
        console.log("[PreferencesStore] Watcher fired, calling persist()", {
          hydrated: hydrated.value,
          newMetrics: newVal.metrics,
          oldMetrics: oldVal?.metrics,
        });
        persist();
      },
      { deep: true }
    );
  };

  // Defer hydration until after module initialization to avoid Vite transformation loops
  // Only hydrate on client side, and set up watcher after hydration completes
  if (import.meta.client) {
    // Use setTimeout to defer execution until after module is fully initialized
    // This prevents Vite from trying to transform the module while it's still being evaluated
    setTimeout(() => {
      hydrate();
      // Set up watcher after hydration completes
      setTimeout(() => {
        setupWatcher();
      }, 0);
    }, 0);
  }

  return {
    preferences,
    hydrated: computed(() => hydrated.value),
    envVarsViewMode: computed(() => preferences.value.envVarsViewMode),
    editorPreferences: computed(() => preferences.value.editor),
    metricsPreferences: computed(() => preferences.value.metrics),
    setEnvVarsViewMode,
    setEditorPreference,
    setEditorPreferences,
    setMetricsPreference,
    setMetricsPreferences,
    setTheme,
    hydrate,
  };
});
