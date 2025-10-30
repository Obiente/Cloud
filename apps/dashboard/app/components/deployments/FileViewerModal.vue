<template>
  <OuiDialog v-model:open="isOpen" :title="fileName" size="4xl">
    <div v-if="isLoading" class="flex justify-center py-8">
      <OuiText color="secondary">Loading file...</OuiText>
    </div>

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
        ref="codeContainer"
        class="w-full h-[600px] overflow-auto rounded-lg bg-black p-4 font-mono text-sm"
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
import { ref, computed, watch, nextTick } from "vue";
import hljs from "highlight.js";
import "highlight.js/styles/github-dark.css";
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
const codeContainer = ref<HTMLElement | null>(null);
const isLoading = ref(false);
const error = ref("");
const fileContent = ref("");
const fileSize = ref(0);
const selectedLanguage = ref("");

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
    renderContent();
  } catch (err: any) {
    console.error("Failed to load file:", err);
    error.value = err.message || "Failed to load file content";
  } finally {
    isLoading.value = false;
  }
};

const renderContent = () => {
  if (!codeContainer.value) return;

  const lang = selectedLanguage.value || "plaintext";
  let highlighted: string;

  if (lang === "plaintext") {
    highlighted = escapeHtml(fileContent.value);
  } else {
    try {
      const result = hljs.highlight(fileContent.value, {
        language: lang,
        ignoreIllegals: true,
      });
      highlighted = result.value;
    } catch {
      highlighted = escapeHtml(fileContent.value);
    }
  }

  codeContainer.value.innerHTML = `<pre><code class="hljs language-${lang}">${highlighted}</code></pre>`;
};

const escapeHtml = (text: string) => {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
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
  renderContent();
});

</script>

