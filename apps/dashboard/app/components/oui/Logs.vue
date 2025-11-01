<template>
  <div class="oui-logs-container" :class="containerClass">
    <!-- Header -->
    <div v-if="title || $slots.actions" class="oui-logs-header">
      <div v-if="title" class="oui-logs-title">
        <slot name="title">
          <OuiText as="h3" size="md" weight="semibold">{{ title }}</OuiText>
        </slot>
      </div>
      <div v-if="$slots.actions" class="oui-logs-actions">
        <slot name="actions" />
      </div>
    </div>

    <!-- Controls at top -->
    <div v-if="$slots.footer || showTimestamps || showTailControls" class="oui-logs-controls-top">
      <slot name="footer">
        <div class="oui-logs-controls">
          <div v-if="showTailControls" class="oui-logs-control-group">
            <label class="oui-logs-control-label">Tail</label>
            <OuiInput
              :model-value="tailLines.toString()"
              type="number"
              :min="minTailLines"
              :max="maxTailLines"
              class="oui-logs-tail-input"
              size="sm"
              @update:model-value="handleTailChange"
            />
          </div>
          <OuiCheckbox
            v-if="showTimestamps"
            v-model="internalShowTimestamps"
            label="Show timestamps"
            class="oui-logs-checkbox"
          />
        </div>
      </slot>
    </div>

    <!-- Logs Display -->
    <div
      ref="logContainer"
      class="oui-logs-viewer"
      :class="viewerClass"
      :style="viewerStyle"
    >
      <!-- Empty State -->
      <div v-if="logs.length === 0 && !isLoading" class="oui-logs-empty">
        <slot name="empty">
          <OuiText color="secondary" size="sm" align="center">
            {{ emptyMessage }}
          </OuiText>
        </slot>
      </div>

      <!-- Loading State -->
      <div v-else-if="isLoading" class="oui-logs-loading">
        <slot name="loading">
          <OuiText color="secondary" size="sm" align="center">
            {{ loadingMessage }}
          </OuiText>
        </slot>
      </div>

      <!-- Log Lines -->
      <div v-else class="oui-logs-content-wrapper">
        <div class="oui-logs-content">
          <div
            v-for="(log, idx) in logs"
            :key="log.id || idx"
            class="oui-log-line"
            :class="getLogLineClass(log)"
          >
            <!-- Timestamp -->
            <span
              v-if="showTimestamps && log.timestamp"
              class="oui-log-timestamp"
            >
              {{ formatTimestamp(log.timestamp) }}
            </span>

            <!-- Log Level/Type Badge -->
            <span
              v-if="log.level || log.type"
              class="oui-log-level"
              :class="`oui-log-level-${(log.level || log.type || '').toLowerCase()}`"
            >
              {{ log.level || log.type }}
            </span>

            <!-- Log Content -->
            <span class="oui-log-content">
              <AnsiRenderer
                v-if="enableAnsi"
                :content="log.line || log.content || log.data || ''"
              />
              <span v-else>{{ log.line || log.content || log.data || '' }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import AnsiRenderer from "./AnsiRenderer.vue";

// Export LogEntry interface for use in other components
export interface LogEntry {
  id?: string | number;
  line?: string;
  content?: string;
  data?: string;
  timestamp?: string | Date;
  stderr?: boolean;
  level?: "info" | "warning" | "error" | "debug" | "trace";
  type?: string;
}

interface Props {
  logs: LogEntry[];
  title?: string;
  isLoading?: boolean;
  showTimestamps?: boolean;
  enableAnsi?: boolean;
  emptyMessage?: string;
  loadingMessage?: string;
  height?: string | number;
  autoScroll?: boolean;
  tailLines?: number;
  minTailLines?: number;
  maxTailLines?: number;
  showTailControls?: boolean;
  containerClass?: string;
  viewerClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  logs: () => [],
  isLoading: false,
  showTimestamps: true,
  enableAnsi: true,
  emptyMessage: "No logs available.",
  loadingMessage: "Loading logs...",
  height: "600px",
  autoScroll: true,
  tailLines: 200,
  minTailLines: 10,
  maxTailLines: 1000,
  showTailControls: false,
  containerClass: "",
  viewerClass: "",
});

const emit = defineEmits<{
  (e: "tail-change", value: number): void;
  (e: "update:showTimestamps", value: boolean): void;
}>();

const logContainer = ref<HTMLElement | null>(null);
const internalShowTimestamps = ref(props.showTimestamps);

watch(() => props.showTimestamps, (val) => {
  internalShowTimestamps.value = val;
});

watch(internalShowTimestamps, (val) => {
  emit("update:showTimestamps", val);
});

const viewerStyle = computed(() => {
  const height =
    typeof props.height === "number" ? `${props.height}px` : props.height;
  return { height };
});

const formatTimestamp = (ts: string | Date): string => {
  if (!ts) return "";
  const date = typeof ts === "string" ? new Date(ts) : ts;
  return date.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
};

const getLogLineClass = (log: LogEntry): string => {
  const classes: string[] = [];

  // Error/Stderr styling
  if (log.stderr || log.level === "error") {
    classes.push("oui-log-line-error");
  } else if (log.level === "warning") {
    classes.push("oui-log-line-warning");
  } else if (log.level === "info") {
    classes.push("oui-log-line-info");
  } else if (log.level === "debug") {
    classes.push("oui-log-line-debug");
  }

  return classes.join(" ");
};

const handleTailChange = (val: string | number) => {
  const num = typeof val === "string" ? parseInt(val, 10) : val;
  if (!isNaN(num)) {
    emit("tail-change", num);
  }
};

const scrollToBottom = () => {
  if (props.autoScroll && logContainer.value) {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight;
      }
    });
  }
};

// Auto-scroll when logs change
watch(
  () => props.logs.length,
  () => {
    scrollToBottom();
  }
);

// Scroll on mount if needed
onMounted(() => {
  scrollToBottom();
});

defineExpose({
  scrollToBottom,
  scrollToTop: () => {
    if (logContainer.value) {
      logContainer.value.scrollTop = 0;
    }
  },
  clear: () => {
    // This is handled by parent component clearing logs array
  },
});
</script>

<style scoped>
.oui-logs-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.oui-logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.oui-logs-title {
  flex: 1;
}

.oui-logs-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.oui-logs-viewer {
  flex: 1;
  overflow: auto;
  background: var(--oui-surface-overlay, #1a1a1a);
  border: 1px solid var(--oui-border-default, rgba(255, 255, 255, 0.1));
  border-radius: 0.5rem;
  font-family: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas,
    "Courier New", monospace;
  font-size: 13px;
  line-height: 1.6;
  position: relative;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: auto;
}

.oui-logs-empty,
.oui-logs-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 2rem;
}

.oui-logs-content-wrapper {
  padding: 1rem;
  width: 100%;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: visible;
}

.oui-logs-content {
  width: 100%;
  max-width: 100%;
  min-width: min-content;
}

.oui-log-line {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.25rem 0;
  word-wrap: break-word;
  white-space: pre-wrap;
  transition: background-color 0.15s ease;
}

.oui-log-line:hover {
  background-color: rgba(255, 255, 255, 0.02);
}

.oui-log-timestamp {
  flex-shrink: 0;
  color: var(--oui-text-tertiary, #666);
  font-size: 11px;
  user-select: none;
  min-width: 90px;
}

.oui-log-level {
  flex-shrink: 0;
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  user-select: none;
  min-width: 60px;
  text-align: center;
}

.oui-log-level-error,
.oui-log-leveltype-error {
  background-color: rgba(239, 68, 68, 0.2);
  color: #fca5a5;
}

.oui-log-level-warning,
.oui-log-leveltype-warning {
  background-color: rgba(251, 191, 36, 0.2);
  color: #fcd34d;
}

.oui-log-level-info,
.oui-log-leveltype-info {
  background-color: rgba(59, 130, 246, 0.2);
  color: #93c5fd;
}

.oui-log-level-debug,
.oui-log-leveltype-debug {
  background-color: rgba(139, 92, 246, 0.2);
  color: #c4b5fd;
}

.oui-log-level-trace,
.oui-log-leveltype-trace {
  background-color: rgba(107, 114, 128, 0.2);
  color: #d1d5db;
}

.oui-log-content {
  flex: 1;
  min-width: 0;
  max-width: 100%;
  color: var(--oui-text-primary, #e5e5e5);
  overflow-wrap: break-word;
  word-break: break-word;
}

.oui-log-line-error .oui-log-content {
  color: #fca5a5;
}

.oui-log-line-warning .oui-log-content {
  color: #fcd34d;
}

.oui-log-line-info .oui-log-content {
  color: #93c5fd;
}

.oui-log-line-debug .oui-log-content {
  color: #c4b5fd;
}

/* Custom Scrollbar */
.oui-logs-viewer::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.oui-logs-viewer::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

.oui-logs-viewer::-webkit-scrollbar-thumb {
  background: var(--oui-border-default, rgba(255, 255, 255, 0.2));
  border-radius: 4px;
}

.oui-logs-viewer::-webkit-scrollbar-thumb:hover {
  background: var(--oui-border-strong, rgba(255, 255, 255, 0.3));
}

.oui-logs-controls-top {
  margin-bottom: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--oui-border-default, rgba(255, 255, 255, 0.1));
}

.oui-logs-controls {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  flex-wrap: wrap;
}

.oui-logs-control-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.oui-logs-control-label {
  font-size: 0.75rem;
  color: var(--oui-text-secondary, #999);
  font-weight: 500;
  user-select: none;
  white-space: nowrap;
}

.oui-logs-tail-input {
  width: 4rem;
}

.oui-logs-tail-input :deep(input) {
  text-align: center;
  font-size: 0.875rem;
  padding: 0.375rem 0.5rem;
  min-width: 0;
}

.oui-logs-tail-input :deep(input:focus) {
  border-color: var(--oui-border-strong, rgba(255, 255, 255, 0.2));
  box-shadow: 0 0 0 1px var(--oui-border-strong, rgba(255, 255, 255, 0.2));
  outline: none;
}

.oui-logs-checkbox {
  margin: 0;
}
</style>

