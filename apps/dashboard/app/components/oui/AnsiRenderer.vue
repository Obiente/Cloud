<template>
  <span class="ansi-renderer" v-html="renderedContent"></span>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Props {
  content: string;
}

const props = defineProps<Props>();

// ANSI escape code patterns
const ANSI_PATTERNS = {
  // Reset
  reset: /\x1b\[0m/g,
  // Bold
  bold: /\x1b\[1m/g,
  // Colors - foreground
  black: /\x1b\[30m/g,
  red: /\x1b\[31m/g,
  green: /\x1b\[32m/g,
  yellow: /\x1b\[33m/g,
  blue: /\x1b\[34m/g,
  magenta: /\x1b\[35m/g,
  cyan: /\x1b\[36m/g,
  white: /\x1b\[37m/g,
  // Bright colors
  brightBlack: /\x1b\[90m/g,
  brightRed: /\x1b\[91m/g,
  brightGreen: /\x1b\[92m/g,
  brightYellow: /\x1b\[93m/g,
  brightBlue: /\x1b\[94m/g,
  brightMagenta: /\x1b\[95m/g,
  brightCyan: /\x1b\[96m/g,
  brightWhite: /\x1b\[97m/g,
  // Background colors
  bgBlack: /\x1b\[40m/g,
  bgRed: /\x1b\[41m/g,
  bgGreen: /\x1b\[42m/g,
  bgYellow: /\x1b\[43m/g,
  bgBlue: /\x1b\[44m/g,
  bgMagenta: /\x1b\[45m/g,
  bgCyan: /\x1b\[46m/g,
  bgWhite: /\x1b\[47m/g,
  // Generic color codes (extended)
  color256: /\x1b\[38;5;(\d+)m/g,
  bgColor256: /\x1b\[48;5;(\d+)m/g,
  rgb: /\x1b\[38;2;(\d+);(\d+);(\d+)m/g,
  bgRgb: /\x1b\[48;2;(\d+);(\d+);(\d+)m/g,
};

const renderedContent = computed(() => {
  if (!props.content) return "";

  let html = props.content
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");

  // Apply ANSI codes in order
  // Reset
  html = html.replace(ANSI_PATTERNS.reset, "</span>");

  // Bold
  html = html.replace(ANSI_PATTERNS.bold, '<span class="ansi-bold">');

  // Standard colors
  html = html.replace(ANSI_PATTERNS.black, '<span class="ansi-black">');
  html = html.replace(ANSI_PATTERNS.red, '<span class="ansi-red">');
  html = html.replace(ANSI_PATTERNS.green, '<span class="ansi-green">');
  html = html.replace(ANSI_PATTERNS.yellow, '<span class="ansi-yellow">');
  html = html.replace(ANSI_PATTERNS.blue, '<span class="ansi-blue">');
  html = html.replace(ANSI_PATTERNS.magenta, '<span class="ansi-magenta">');
  html = html.replace(ANSI_PATTERNS.cyan, '<span class="ansi-cyan">');
  html = html.replace(ANSI_PATTERNS.white, '<span class="ansi-white">');

  // Bright colors
  html = html.replace(ANSI_PATTERNS.brightBlack, '<span class="ansi-bright-black">');
  html = html.replace(ANSI_PATTERNS.brightRed, '<span class="ansi-bright-red">');
  html = html.replace(ANSI_PATTERNS.brightGreen, '<span class="ansi-bright-green">');
  html = html.replace(ANSI_PATTERNS.brightYellow, '<span class="ansi-bright-yellow">');
  html = html.replace(ANSI_PATTERNS.brightBlue, '<span class="ansi-bright-blue">');
  html = html.replace(ANSI_PATTERNS.brightMagenta, '<span class="ansi-bright-magenta">');
  html = html.replace(ANSI_PATTERNS.brightCyan, '<span class="ansi-bright-cyan">');
  html = html.replace(ANSI_PATTERNS.brightWhite, '<span class="ansi-bright-white">');

  // Background colors
  html = html.replace(ANSI_PATTERNS.bgBlack, '<span class="ansi-bg-black">');
  html = html.replace(ANSI_PATTERNS.bgRed, '<span class="ansi-bg-red">');
  html = html.replace(ANSI_PATTERNS.bgGreen, '<span class="ansi-bg-green">');
  html = html.replace(ANSI_PATTERNS.bgYellow, '<span class="ansi-bg-yellow">');
  html = html.replace(ANSI_PATTERNS.bgBlue, '<span class="ansi-bg-blue">');
  html = html.replace(ANSI_PATTERNS.bgMagenta, '<span class="ansi-bg-magenta">');
  html = html.replace(ANSI_PATTERNS.bgCyan, '<span class="ansi-bg-cyan">');
  html = html.replace(ANSI_PATTERNS.bgWhite, '<span class="ansi-bg-white">');

  // Close any unclosed spans at the end
  const openSpans = (html.match(/<span/g) || []).length;
  const closeSpans = (html.match(/<\/span>/g) || []).length;
  const unclosed = openSpans - closeSpans;
  if (unclosed > 0) {
    html += "</span>".repeat(unclosed);
  }

  return html;
});
</script>

<style scoped>
.ansi-renderer {
  white-space: pre-wrap;
  word-break: break-word;
}

/* ANSI Color Styles */
.ansi-renderer :deep(.ansi-bold) {
  font-weight: 600;
}

.ansi-renderer :deep(.ansi-black) {
  color: #3a3a3a;
}

.ansi-renderer :deep(.ansi-red) {
  color: #ef4444;
}

.ansi-renderer :deep(.ansi-green) {
  color: #22c55e;
}

.ansi-renderer :deep(.ansi-yellow) {
  color: #eab308;
}

.ansi-renderer :deep(.ansi-blue) {
  color: #3b82f6;
}

.ansi-renderer :deep(.ansi-magenta) {
  color: #d946ef;
}

.ansi-renderer :deep(.ansi-cyan) {
  color: #06b6d4;
}

.ansi-renderer :deep(.ansi-white) {
  color: #f5f5f5;
}

.ansi-renderer :deep(.ansi-bright-black) {
  color: #525252;
}

.ansi-renderer :deep(.ansi-bright-red) {
  color: #f87171;
}

.ansi-renderer :deep(.ansi-bright-green) {
  color: #4ade80;
}

.ansi-renderer :deep(.ansi-bright-yellow) {
  color: #facc15;
}

.ansi-renderer :deep(.ansi-bright-blue) {
  color: #60a5fa;
}

.ansi-renderer :deep(.ansi-bright-magenta) {
  color: #e879f9;
}

.ansi-renderer :deep(.ansi-bright-cyan) {
  color: #22d3ee;
}

.ansi-renderer :deep(.ansi-bright-white) {
  color: #ffffff;
}

/* Background colors */
.ansi-renderer :deep(.ansi-bg-black) {
  background-color: #3a3a3a;
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-red) {
  background-color: rgba(239, 68, 68, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-green) {
  background-color: rgba(34, 197, 94, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-yellow) {
  background-color: rgba(234, 179, 8, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-blue) {
  background-color: rgba(59, 130, 246, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-magenta) {
  background-color: rgba(217, 70, 239, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-cyan) {
  background-color: rgba(6, 182, 212, 0.3);
  padding: 0 2px;
}

.ansi-renderer :deep(.ansi-bg-white) {
  background-color: rgba(245, 245, 245, 0.3);
  padding: 0 2px;
}
</style>

