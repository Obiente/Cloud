<template>
  <!--
    A labeled code/text block with a header bar and copy button.
    Used for connection strings, SSH commands, etc.

    <UiCodeBlock label="SSH Command" :value="sshInfo.sshProxyCommand" />
  -->
  <div class="rounded-lg border border-border-default overflow-hidden">
    <!-- Header bar -->
    <div class="flex items-center justify-between px-3 py-2 bg-surface-muted/30 border-b border-border-default">
      <OuiText size="xs" weight="medium">{{ label }}</OuiText>
      <button
        class="p-1 rounded text-tertiary hover:text-primary transition-colors"
        :title="`Copy ${label}`"
        @click="handleCopy"
      >
        <CheckIcon v-if="copied" class="h-3.5 w-3.5 text-success" />
        <ClipboardIcon v-else class="h-3.5 w-3.5" />
      </button>
    </div>
    <!-- Content -->
    <div class="px-3 py-2.5">
      <code :class="['text-xs font-mono text-secondary', breakAll ? 'break-all' : '', preWrap ? 'whitespace-pre-wrap' : '']">{{ value }}</code>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ClipboardIcon, CheckIcon } from "@heroicons/vue/24/outline";
import { useToast } from "~/composables/useToast";

const props = withDefaults(defineProps<{
  label: string;
  value: string;
  /** Break long single-line values (connection strings) */
  breakAll?: boolean;
  /** Preserve line breaks (multi-line instructions) */
  preWrap?: boolean;
}>(), {
  breakAll: false,
  preWrap: false,
});

const { toast } = useToast();
const copied = ref(false);

async function handleCopy() {
  try {
    await navigator.clipboard.writeText(props.value);
    copied.value = true;
    toast.success(`${props.label} copied`);
    setTimeout(() => (copied.value = false), 2000);
  } catch {
    toast.error("Failed to copy");
  }
}
</script>
