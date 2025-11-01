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

interface Preferences {
  envVarsViewMode: "list" | "file";
  editor: EditorPreferences;
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

const defaultPreferences: Preferences = {
  envVarsViewMode: "list",
  editor: defaultEditorPreferences,
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
    editorPreferences: computed(() => preferences.value.editor),
    setEnvVarsViewMode,
    setEditorPreference,
    setEditorPreferences,
    hydrate,
  };
});

