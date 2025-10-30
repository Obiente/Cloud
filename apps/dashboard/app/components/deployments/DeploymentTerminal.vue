<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Interactive Terminal</OuiText>
        <OuiButton
          variant="ghost"
          size="sm"
          @click="connect"
          :disabled="isConnected"
        >
          Connect
        </OuiButton>
      </OuiFlex>

      <div
        ref="terminalContainer"
        class="bg-black text-green-400 p-4 rounded-xl text-xs font-mono overflow-auto"
        :style="{ height: '600px' }"
      >
        <div v-if="!isConnected" class="text-gray-500">
          Click "Connect" to start an interactive terminal session.
        </div>
        <div v-else>
          <div
            v-for="(line, idx) in terminalOutput"
            :key="idx"
            class="terminal-line"
          >
            {{ line }}
          </div>
          <div v-if="isWaitingInput" class="terminal-input-line">
            <span class="text-green-400">$ </span>
            <input
              ref="commandInput"
              v-model="command"
              @keydown.enter="executeCommand"
              class="bg-transparent text-green-400 outline-none flex-1"
              autofocus
            />
          </div>
        </div>
      </div>

      <OuiText size="xs" color="secondary">
        Note: Terminal access requires container exec API. This feature will be fully implemented when the backend API is ready.
      </OuiText>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue";

interface Props {
  deploymentId: string;
  organizationId: string;
}

const props = defineProps<Props>();

const terminalContainer = ref<HTMLElement | null>(null);
const commandInput = ref<HTMLInputElement | null>(null);
const isConnected = ref(false);
const isWaitingInput = ref(false);
const command = ref("");
const terminalOutput = ref<string[]>([]);

const connect = async () => {
  isConnected.value = true;
  isWaitingInput.value = true;
  terminalOutput.value = [
    `Connected to deployment ${props.deploymentId}`,
    "Type 'help' for available commands.",
    "",
  ];
  await nextTick();
  commandInput.value?.focus();
};

const executeCommand = async () => {
  if (!command.value.trim()) return;

  terminalOutput.value.push(`$ ${command.value}`);
  
  // TODO: Implement actual command execution via API
  terminalOutput.value.push(
    `Command execution not yet implemented. Would execute: ${command.value}`
  );
  
  command.value = "";
  await nextTick();
  scrollToBottom();
  commandInput.value?.focus();
};

const scrollToBottom = () => {
  if (terminalContainer.value) {
    terminalContainer.value.scrollTop = terminalContainer.value.scrollHeight;
  }
};

onMounted(() => {
  // Auto-connect could be enabled here
});
</script>

<style scoped>
.terminal-line {
  white-space: pre-wrap;
  word-break: break-word;
}
.terminal-input-line {
  display: flex;
  align-items: center;
}
</style>

