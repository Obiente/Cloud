<template>
  <OuiBox 
    v-if="!inline"
    :class="containerClass"
    :p="padding"
    border="1"
    borderColor="muted"
    rounded="xl"
    class="bg-surface-base overflow-hidden relative"
  >
    <div class="flex items-center justify-between gap-2 mb-2" v-if="showHeader">
      <OuiText size="xs" weight="medium" color="secondary" v-if="language">
        {{ languageLabel }}
      </OuiText>
      <OuiButton
        v-if="copyable"
        variant="ghost"
        size="xs"
        @click="copyCode"
        class="ml-auto"
      >
        <ClipboardIcon class="h-4 w-4" />
      </OuiButton>
    </div>
    <OuiButton
      v-else-if="copyable"
      variant="ghost"
      size="xs"
      @click="copyCode"
      :class="[
        'absolute right-2 z-10',
        isSingleLine ? 'top-1/2 -translate-y-1/2' : 'top-2'
      ]"
    >
      <ClipboardIcon class="h-4 w-4" />
    </OuiButton>
    <pre 
      :class="preClass"
      class="text-xs font-mono whitespace-pre overflow-x-auto m-0"
    ><code 
        ref="codeRef"
        :class="`language-${language || 'plaintext'}`"
        class="block"
      >{{ trimmedCode }}</code></pre>
  </OuiBox>
  <code
    v-else
    ref="codeRef"
    :class="`language-${language || 'plaintext'} text-xs font-mono bg-surface-subtle px-1.5 py-0.5 rounded`"
    class="inline"
  >{{ trimmedCode }}</code>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from "vue";
import { ClipboardIcon } from "@heroicons/vue/24/outline";
import { useToast } from "~/composables/useToast";

interface Props {
  code: string;
  language?: string;
  padding?: "xs" | "sm" | "md" | "lg" | "xl";
  copyable?: boolean;
  showHeader?: boolean;
  containerClass?: string;
  inline?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  language: undefined,
  padding: "md",
  copyable: true,
  showHeader: false,
  containerClass: "",
  inline: false,
});

const codeRef = ref<HTMLElement | null>(null);
const { toast } = useToast();

// Trim code to remove trailing whitespace
const trimmedCode = computed(() => {
  // Remove all trailing whitespace including newlines and spaces
  return props.code.replace(/[\s\n\r]+$/, '');
});

// Check if code is single line
const isSingleLine = computed(() => {
  return !trimmedCode.value.includes('\n');
});

const languageLabel = computed(() => {
  if (!props.language) return "Code";
  
  const labels: Record<string, string> = {
    bash: "Bash",
    sh: "Shell",
    yaml: "YAML",
    yml: "YAML",
    json: "JSON",
    javascript: "JavaScript",
    js: "JavaScript",
    typescript: "TypeScript",
    ts: "TypeScript",
    python: "Python",
    py: "Python",
    go: "Go",
    rust: "Rust",
    rs: "Rust",
    java: "Java",
    html: "HTML",
    css: "CSS",
    sql: "SQL",
    dockerfile: "Dockerfile",
    plaintext: "Plain Text",
  };
  
  return labels[props.language.toLowerCase()] || props.language.toUpperCase();
});

const preClass = computed(() => {
  return props.containerClass || "";
});

const highlightCode = async () => {
  await nextTick();
  if (typeof window === "undefined" || !(window as any).hljs) return;
  
  if (codeRef.value) {
    const element = codeRef.value;
    // Set the trimmed code content before highlighting
    element.textContent = trimmedCode.value;
    // Remove existing highlighting
    element.classList.remove("hljs");
    // Re-highlight
    (window as any).hljs.highlightElement(element);
  }
};

const copyCode = async () => {
  try {
    await navigator.clipboard.writeText(trimmedCode.value);
    toast.success("Code copied to clipboard");
  } catch (err) {
    console.error("Failed to copy code:", err);
    toast.error("Failed to copy code");
  }
};

// Highlight when component mounts or code/language changes
onMounted(() => {
  highlightCode();
});

watch(() => [props.code, props.language], () => {
  highlightCode();
});
</script>

<style scoped>
/* Ensure code blocks take full width */
pre {
  margin: 0;
  padding: 0;
}

code {
  display: block;
  width: 100%;
  margin-bottom: 0;
}

/* Override highlight.js padding - remove vertical padding, keep horizontal */
:deep(.hljs) {
  padding-top: 0 !important;
  padding-bottom: 0 !important;
  margin: 0 !important;
  margin-bottom: 0 !important;
  display: block;
}
</style>

