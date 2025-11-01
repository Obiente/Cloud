<template>
  <OuiDialog v-model:open="isOpen" :title="fileName" size="4xl">
    <OuiFlex v-if="isLoading" justify="center" class="py-8">
      <OuiText color="secondary">Loading file...</OuiText>
    </OuiFlex>

    <div v-else-if="error" class="p-4">
      <OuiText color="danger">{{ error }}</OuiText>
    </div>

    <div v-else class="relative">
      <!-- Toolbar -->
      <OuiFlex justify="between" align="center" class="mb-4 pb-2 border-b border-border-default">
        <OuiFlex gap="sm" align="center">
          <OuiText size="xs" color="secondary">{{ filePath }}</OuiText>
          <OuiText size="xs" color="secondary">•</OuiText>
          <OuiText size="xs" color="secondary">{{ formatSize(fileSize) }}</OuiText>
        </OuiFlex>
        <OuiFlex gap="sm">
          <OuiButton variant="ghost" size="sm" @click="copyContent">
            <DocumentDuplicateIcon class="h-4 w-4 mr-1" />
            Copy
          </OuiButton>
          <OuiSelect
            v-model="selectedLanguage"
            :items="languageOptions"
            size="sm"
            class="w-40"
          />
        </OuiFlex>
      </OuiFlex>

      <!-- File Content -->
      <div
        ref="editorContainer"
        class="w-full h-[600px] rounded-xl border border-border-default overflow-hidden"
      />
    </div>

    <template #footer>
      <OuiFlex justify="end">
        <OuiButton @click="isOpen = false">Close</OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from "vue";
import { DocumentDuplicateIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  deploymentId: string;
  filePath?: string;
  organizationId?: string;
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
});

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
}>();

const isOpen = computed({
  get: () => props.open,
  set: (val) => emit("update:open", val),
});

const orgsStore = useOrganizationsStore();
const organizationId = computed(() => props.organizationId || orgsStore.currentOrgId || "");

const client = useConnectClient(DeploymentService);
const editorContainer = ref<HTMLElement | null>(null);
const isLoading = ref(false);
const error = ref("");
const fileContent = ref("");
const fileSize = ref(0);
const selectedLanguage = ref("");
let editor: any = null;
let monaco: any = null;

const fileName = computed(() => {
  if (!props.filePath) return "File Viewer";
  return props.filePath.split("/").pop() || "File Viewer";
});

const filePath = computed(() => props.filePath || "");

const languageOptions = computed(() => {
  const ext = fileName.value.split(".").pop()?.toLowerCase() || "";
  const langMap: Record<string, string> = {
    js: "javascript",
    ts: "typescript",
    jsx: "javascript",
    tsx: "typescript",
    py: "python",
    go: "go",
    rs: "rust",
    java: "java",
    c: "c",
    cpp: "cpp",
    h: "c",
    hpp: "cpp",
    sh: "bash",
    yaml: "yaml",
    yml: "yaml",
    json: "json",
    xml: "xml",
    html: "html",
    css: "css",
    md: "markdown",
    sql: "sql",
    dockerfile: "dockerfile",
    makefile: "makefile",
    txt: "plaintext",
  };

  const detected = langMap[ext] || "plaintext";
  return [
    { label: "Auto-detect", value: detected },
    { label: "Plain Text", value: "plaintext" },
    { label: "JavaScript", value: "javascript" },
    { label: "TypeScript", value: "typescript" },
    { label: "Python", value: "python" },
    { label: "Go", value: "go" },
    { label: "Rust", value: "rust" },
    { label: "Java", value: "java" },
    { label: "YAML", value: "yaml" },
    { label: "JSON", value: "json" },
    { label: "HTML", value: "html" },
    { label: "CSS", value: "css" },
    { label: "Bash", value: "bash" },
    { label: "SQL", value: "sql" },
    { label: "Dockerfile", value: "dockerfile" },
  ];
});

const initEditor = async () => {
  if (typeof window === "undefined" || !editorContainer.value) return;

  try {
    if (!monaco) {
      const monacoModule = await import("monaco-editor");
      monaco = monacoModule;
      
      // Register OUI theme
      const { registerOUITheme } = await import("~/utils/monaco-theme");
      registerOUITheme(monaco);
    }

    if (editor) {
      editor.dispose();
    }

    editor = monaco.editor.create(editorContainer.value, {
      value: fileContent.value || "",
      language: selectedLanguage.value || "plaintext",
      theme: "oui-dark",
      automaticLayout: true,
      fontSize: 14,
      minimap: { enabled: true },
      scrollBeyondLastLine: false,
      wordWrap: "on",
      readOnly: false,
      lineNumbers: "on",
      renderWhitespace: "selection",
      folding: true,
      mouseWheelZoom: true, // Enable zoom with Ctrl+scroll (Cmd+scroll on Mac)
      accessibilitySupport: "on", // Enable accessibility features
    });
  } catch (err) {
    console.error("Failed to initialize Monaco Editor:", err);
  }
};

const loadFile = async () => {
  if (!props.filePath) return;

  isLoading.value = true;
  error.value = "";

  try {
    const res = await client.getContainerFile({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      path: props.filePath,
    });

    fileContent.value = res.content || "";
    fileSize.value = Number(res.size || 0);

    // Auto-detect language
    const ext = fileName.value.split(".").pop()?.toLowerCase() || "";
    const langMap: Record<string, string> = {
      js: "javascript",
      ts: "typescript",
      jsx: "javascript",
      tsx: "typescript",
      py: "python",
      go: "go",
      rs: "rust",
      java: "java",
      sh: "bash",
      yaml: "yaml",
      yml: "yaml",
      json: "json",
      html: "html",
      css: "css",
      md: "markdown",
      sql: "sql",
      dockerfile: "dockerfile",
    };
    selectedLanguage.value = langMap[ext] || "plaintext";

    await nextTick();
    if (editor) {
      editor.setValue(fileContent.value);
      monaco.editor.setModelLanguage(editor.getModel(), selectedLanguage.value || "plaintext");
    } else {
      await initEditor();
    }
  } catch (err: any) {
    console.error("Failed to load file:", err);
    error.value = err.message || "Failed to load file content";
  } finally {
    isLoading.value = false;
  }
};

const formatSize = (bytes: number | bigint) => {
  const size = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(fileContent.value);
    // TODO: Show toast notification
  } catch (err) {
    console.error("Failed to copy:", err);
  }
};

watch(() => props.filePath, () => {
  if (props.filePath && isOpen.value) {
    loadFile();
  }
});

watch(isOpen, (newVal) => {
  emit("update:open", newVal);
  if (newVal && props.filePath) {
    loadFile();
  }
});

watch(selectedLanguage, () => {
  if (editor && monaco) {
    monaco.editor.setModelLanguage(editor.getModel(), selectedLanguage.value || "plaintext");
  }
});

onUnmounted(() => {
  if (editor) {
    editor.dispose();
    editor = null;
  }
});

</script>

