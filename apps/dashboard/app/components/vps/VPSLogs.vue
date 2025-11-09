<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h3" size="sm" weight="semibold">Terminal Output</OuiText>
      <OuiFlex gap="sm" align="center">
        <OuiButton
          variant="ghost"
          size="sm"
          @click="clearLogs"
          class="gap-2"
        >
          <TrashIcon class="h-4 w-4" />
          Clear
        </OuiButton>
      </OuiFlex>
    </OuiFlex>

    <OuiLogs
      ref="logsComponent"
      :logs="formattedLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :show-tail-controls="false"
      :enable-ansi="true"
      :auto-scroll="true"
      empty-message="No output yet. Commands will appear here."
      loading-message="Connecting..."
      title="VPS Terminal"
    />

    <!-- Interactive Terminal -->
    <VPSTerminal
      :vps-id="props.vpsId"
      :organization-id="props.organizationId"
      @log-output="handleTerminalOutput"
    />

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
    <OuiText v-if="isConnected && !error" size="xs" color="success">
      ✓ Connected. Terminal output will appear here.
    </OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { TrashIcon } from "@heroicons/vue/24/outline";
import { useOrganizationsStore } from "~/stores/organizations";
import type { LogEntry } from "~/components/oui/Logs.vue";
import VPSTerminal from "~/components/vps/VPSTerminal.vue";

interface Props {
  vpsId: string;
  organizationId: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();

const logsComponent = ref<any>(null);
const isLoading = ref(false);
const isConnected = ref(false);
const error = ref<string | null>(null);
const showTimestamps = ref(true);

const logs = ref<LogEntry[]>([]);
let terminalOutputBuffer = "";

const formattedLogs = computed(() => logs.value);

// Handle output from terminal WebSocket
const handleTerminalOutput = (text: string) => {
  if (!text) return;
  
  // Add to buffer
  terminalOutputBuffer += text;
  
  // Split by newlines - keep incomplete line in buffer
  const lines = terminalOutputBuffer.split(/\r?\n/);
  
  // If buffer ends with newline, last element will be empty string
  // Otherwise, last element is an incomplete line that stays in buffer
  if (terminalOutputBuffer.endsWith("\n") || terminalOutputBuffer.endsWith("\r")) {
    terminalOutputBuffer = "";
  } else {
    terminalOutputBuffer = lines.pop() || ""; // Keep last incomplete line
  }
  
  // Add complete lines to logs (including empty lines)
  for (const line of lines) {
    // Add line even if empty (for proper spacing)
    logs.value.push({
      line: line || " ", // Use space for empty lines so they still render
      timestamp: new Date().toISOString(),
      level: undefined, // Terminal output doesn't have a log level
    });
    
    // Keep only last 10000 lines
    if (logs.value.length > 10000) {
      logs.value = logs.value.slice(-10000);
    }
  }
  
  // Also flush buffer if it gets too large (in case we never get a newline)
  if (terminalOutputBuffer.length > 1000) {
    logs.value.push({
      line: terminalOutputBuffer,
      timestamp: new Date().toISOString(),
    });
    terminalOutputBuffer = "";
    
    // Keep only last 10000 lines
    if (logs.value.length > 10000) {
      logs.value = logs.value.slice(-10000);
    }
  }
};

const clearLogs = () => {
  logs.value = [];
  terminalOutputBuffer = "";
  if (logsComponent.value) {
    logsComponent.value.scrollToTop?.();
  }
};
</script>

