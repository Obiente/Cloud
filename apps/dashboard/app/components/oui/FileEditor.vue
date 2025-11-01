<template>
  <div
    ref="editorContainer"
    :class="[containerClass, 'editor-container']"
  />
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick, computed } from "vue";
import { usePreferencesStore } from "~/stores/preferences";

interface ValidationError {
  line: number; // 1-based
  column: number; // 1-based
  message: string;
  severity: "error" | "warning";
  startLine?: number;
  endLine?: number;
  startColumn?: number;
  endColumn?: number;
}

interface Props {
  modelValue: string;
  language?: string;
  readOnly?: boolean;
  height?: string;
  fontSize?: number;
  wordWrap?: "off" | "on" | "wordWrapColumn" | "bounded";
  minimap?: { enabled: boolean };
  folding?: boolean;
  lineNumbers?: "on" | "off" | "relative" | "interval";
  containerClass?: string;
  // Editor-specific options
  formatOnPaste?: boolean;
  formatOnType?: boolean;
  bracketPairColorization?: { enabled: boolean };
  validationErrors?: ValidationError[]; // Inline validation errors
}

const props = withDefaults(defineProps<Props>(), {
  language: "plaintext",
  readOnly: false,
  height: "400px",
  fontSize: undefined,
  wordWrap: undefined,
  minimap: undefined,
  folding: true,
  lineNumbers: undefined,
  containerClass: "w-full rounded-xl border border-border-default overflow-hidden",
  formatOnPaste: false,
  formatOnType: false,
  bracketPairColorization: undefined,
  validationErrors: () => [],
});

const preferencesStore = usePreferencesStore();
const editorPreferences = computed(() => preferencesStore.editorPreferences);

// Use props if provided, otherwise fall back to preferences
const effectiveFontSize = computed(() => props.fontSize ?? editorPreferences.value.fontSize);
const effectiveWordWrap = computed(() => props.wordWrap ?? editorPreferences.value.wordWrap);
const effectiveMinimap = computed(() => props.minimap ?? { enabled: editorPreferences.value.minimap });
const effectiveLineNumbers = computed(() => props.lineNumbers ?? editorPreferences.value.lineNumbers);

const emit = defineEmits<{
  "update:modelValue": [value: string];
  "save": [];
  "change": [value: string];
}>();

const editorContainer = ref<HTMLElement | null>(null);
let editor: any = null;
let monaco: any = null;
let resizeObserver: ResizeObserver | null = null;

// Format the entire document
const formatDocument = async () => {
  if (!editor || props.readOnly) return;
  
  try {
    const action = editor.getAction("editor.action.formatDocument");
    if (action && action.isSupported()) {
      await action.run();
      // Emit change event after formatting
      const newValue = editor.getValue();
      emit("update:modelValue", newValue);
      emit("change", newValue);
    } else {
      console.debug("Format document action not available for language:", props.language);
    }
  } catch (err) {
    console.warn("Failed to format document:", err);
  }
};

// Format the selected text
const formatSelection = async () => {
  if (!editor || props.readOnly) return;
  
  const selection = editor.getSelection();
  if (!selection || selection.isEmpty()) {
    // No selection, format entire document instead
    await formatDocument();
    return;
  }
  
  try {
    const action = editor.getAction("editor.action.formatSelection");
    if (action && action.isSupported()) {
      await action.run();
      // Emit change event after formatting
      const newValue = editor.getValue();
      emit("update:modelValue", newValue);
      emit("change", newValue);
    } else {
      console.debug("Format selection action not available for language:", props.language);
    }
  } catch (err) {
    console.warn("Failed to format selection:", err);
  }
};

// Initialize Monaco Editor
const initEditor = async () => {
  // Guard against SSR and ensure container exists
  if (typeof window === "undefined") return;
  
  // Wait for container to be available
  await nextTick();
  
  // Double-check container is available and is an actual DOM element
  if (!editorContainer.value || !(editorContainer.value instanceof HTMLElement)) {
    return;
  }

  try {
    // Lazy load Monaco Editor
    if (!monaco) {
      const monacoModule = await import("monaco-editor");
      monaco = monacoModule;

      // Register OUI theme
      const { registerOUITheme } = await import("~/utils/monaco-theme");
      registerOUITheme(monaco);

      // Register dotenv language
      monaco.languages.register({ id: "dotenv" });
      monaco.languages.setMonarchTokensProvider("dotenv", {
        tokenizer: {
          root: [
            [/#.*$/, "comment"],                         // Comments
            [/^\s*[A-Z_][A-Z0-9_]*\s*=/, "key"],         // Keys
            [/".*?"/, "string"],                         // Double-quoted strings
            [/'.*?'/, "string"],                         // Single-quoted strings
            [/[^#=\s]+/, "value"],                        // Unquoted values
          ],
        },
      });
    }

    // Dispose existing editor if any
    if (editor) {
      editor.dispose();
      editor = null;
    }

    // Double-check container again before creating editor
    if (!editorContainer.value || !(editorContainer.value instanceof HTMLElement)) {
      console.warn("Editor container not available for Monaco initialization");
      return;
    }

    // Create editor instance with common configuration
    const editorOptions: any = {
      value: props.modelValue || "",
      language: props.language,
      theme: "oui-dark",
      automaticLayout: true,
      fontSize: effectiveFontSize.value,
      minimap: effectiveMinimap.value,
      scrollBeyondLastLine: false,
      wordWrap: effectiveWordWrap.value,
      readOnly: props.readOnly,
      formatOnPaste: props.formatOnPaste,
      formatOnType: props.formatOnType,
      tabSize: editorPreferences.value.tabSize,
      insertSpaces: editorPreferences.value.insertSpaces,
      lineNumbers: effectiveLineNumbers.value,
      renderWhitespace: editorPreferences.value.renderWhitespace,
      folding: props.folding,
      mouseWheelZoom: true, // Enable zoom with Ctrl+scroll (Cmd+scroll on Mac)
      accessibilitySupport: "on", // Enable accessibility features
      hover: {
        enabled: true,
        delay: 300,
        sticky: true,
      },
    };

    // Add optional bracket colorization
    if (props.bracketPairColorization) {
      editorOptions.bracketPairColorization = props.bracketPairColorization;
    }

    // Set container height if provided
    if (props.height && editorContainer.value) {
      editorContainer.value.style.height = props.height;
    }

    editor = monaco.editor.create(editorContainer.value, editorOptions);

    // Handle content changes
    editor.onDidChangeModelContent(() => {
      const newValue = editor.getValue();
      emit("update:modelValue", newValue);
      emit("change", newValue);
    });

    // Handle Ctrl+S / Cmd+S to save
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      emit("save");
    });

    // Format document: Shift+Alt+F (Windows/Linux) or Shift+Option+F (Mac)
    // This is Monaco's default shortcut, we'll ensure it works
    editor.addCommand(monaco.KeyMod.Shift | monaco.KeyMod.Alt | monaco.KeyCode.KeyF, async () => {
      await formatDocument();
    });

    // Format selection shortcut: Ctrl+K Ctrl+F (Windows/Linux) or Cmd+K Cmd+F (Mac)
    // Note: Monaco handles chord commands internally, but we can also trigger the action
    // Users can use Command Palette (F1) and search "Format Selection"
    // For convenience, we'll add an alternative: Ctrl+Shift+F (format selection if text selected, else format document)
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
      async () => {
        const selection = editor.getSelection();
        if (selection && !selection.isEmpty()) {
          await formatSelection();
        } else {
          await formatDocument();
        }
      }
    );

    // Handle resize
    resizeObserver = new ResizeObserver(() => {
      if (editor) {
        editor.layout();
      }
    });

    if (editorContainer.value) {
      resizeObserver.observe(editorContainer.value);
    }
  } catch (err) {
    console.error("Failed to initialize Monaco Editor:", err);
  }
};

// Watch for model value changes (external updates)
watch(
  () => props.modelValue,
  (newValue) => {
    if (editor && editor.getValue() !== newValue) {
      editor.setValue(newValue || "");
    }
  }
);

// Watch for container becoming available and initialize if needed
watch(
  () => editorContainer.value,
  async (newContainer) => {
    if (newContainer && newContainer instanceof HTMLElement && !editor) {
      await nextTick();
      await initEditor();
    }
  },
  { immediate: true }
);

// Watch for language changes
watch(
  () => props.language,
  (newLanguage) => {
    if (editor && monaco) {
      const model = editor.getModel();
      if (model) {
        monaco.editor.setModelLanguage(model, newLanguage);
      }
    }
  }
);

// Watch for read-only changes
watch(
  () => props.readOnly,
  (newReadOnly) => {
    if (editor) {
      editor.updateOptions({ readOnly: newReadOnly });
    }
  }
);

// Watch for editor preferences changes and update editor dynamically
watch(
  () => editorPreferences.value,
  (newPrefs) => {
    if (editor) {
      editor.updateOptions({
        fontSize: effectiveFontSize.value,
        wordWrap: effectiveWordWrap.value,
        minimap: effectiveMinimap.value,
        lineNumbers: effectiveLineNumbers.value,
        tabSize: newPrefs.tabSize,
        insertSpaces: newPrefs.insertSpaces,
        renderWhitespace: newPrefs.renderWhitespace,
      });
    }
  },
  { deep: true }
);

// Watch for validation errors and update markers
watch(
  () => props.validationErrors,
  (errors) => {
    if (!editor || !monaco) return;
    
    const model = editor.getModel();
    if (!model) return;

    if (!errors || errors.length === 0) {
      // Clear all markers
      monaco.editor.setModelMarkers(model, "validation", []);
      return;
    }

    // Convert validation errors to Monaco markers
    const markers = errors.map((err) => {
      const startLine = err.startLine || err.line;
      const endLine = err.endLine || err.line;
      const startColumn = err.startColumn || err.column;
      const endColumn = err.endColumn || err.column;

      // Build a formatted message with severity prefix for clarity
      const severityLabel = err.severity === "error" ? "Error" : "Warning";
      const formattedMessage = `[${severityLabel}] ${err.message}`;

      return {
        startLineNumber: startLine,
        startColumn: startColumn,
        endLineNumber: endLine,
        endColumn: endColumn,
        message: formattedMessage,
        severity: err.severity === "error" 
          ? monaco.MarkerSeverity.Error 
          : monaco.MarkerSeverity.Warning,
        source: "compose-validator",
        tags: err.severity === "warning" ? [monaco.MarkerTag.Unnecessary] : undefined,
      };
    });

    // Set markers on the model - Monaco will automatically show hover tooltips
    monaco.editor.setModelMarkers(model, "validation", markers);
    
    // Ensure hover is enabled for the editor
    if (editor && editor.updateOptions) {
      try {
        // Check if hover is enabled, and enable it if not
        // Monaco Editor API varies by version, so we'll just ensure it's enabled
        editor.updateOptions({ 
          hover: { 
            enabled: true, 
            delay: 300, 
            sticky: true 
          } 
        });
      } catch (err) {
        // Ignore errors if hover options can't be set (some Monaco versions handle this differently)
        console.debug("Could not set hover options:", err);
      }
    }
  },
  { deep: true, immediate: true }
);

onMounted(async () => {
  // Ensure we're on client side
  if (typeof window === "undefined") return;
  
  // Wait for multiple ticks to ensure DOM is fully ready
  await nextTick();
  await nextTick();
  
  // Check container is available before initializing
  if (!editorContainer.value || !(editorContainer.value instanceof HTMLElement)) {
    // If container not ready, wait a bit more and try again
    setTimeout(async () => {
      if (editorContainer.value && editorContainer.value instanceof HTMLElement && !editor) {
        await initEditor();
      }
    }, 100);
    return;
  }
  
  await initEditor();
});

onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  if (editor) {
    editor.dispose();
    editor = null;
  }
});

// Expose editor instance for advanced use cases
defineExpose({
  editor: () => editor,
  monaco: () => monaco,
  getValue: () => editor?.getValue() || "",
  setValue: (value: string) => {
    if (editor) {
      editor.setValue(value);
    }
  },
  formatDocument,
  formatSelection,
});
</script>

<style scoped>
/* Editor container - overflow-hidden is needed for rounded borders */
.editor-container {
  position: relative;
}
</style>

<style>
/* Global styles to ensure Monaco hover tooltips appear above all UI */
/* Monaco hover tooltips are rendered outside the container (appended to body),
   so they can escape overflow-hidden containers. We just need to ensure high z-index. */
.monaco-hover,
.monaco-hover-content,
.monaco-editor-hover,
.monaco-editor .monaco-hover,
.monaco-editor .monaco-hover-content {
  z-index: 99999 !important;
}

/* Ensure Monaco's hover widget container has high z-index */
.monaco-editor .monaco-hover-content-container {
  z-index: 99999 !important;
}
</style>
