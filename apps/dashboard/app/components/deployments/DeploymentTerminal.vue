<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Interactive Terminal</OuiText>
        <OuiButton variant="ghost" size="sm" @click="connectTerminal" :disabled="isConnected || isLoading">
          {{ isLoading ? "Connecting..." : "Connect" }}
        </OuiButton>
      </OuiFlex>

      <OuiText size="sm" color="secondary">
        Access an interactive terminal session to run commands directly in your container.
      </OuiText>

      <div
        ref="terminalRef"
        class="w-full h-96 rounded-lg bg-black p-4 font-mono text-sm overflow-auto"
      />

      <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>

      <OuiText v-if="isConnected" size="xs" color="success">
        ✓ Terminal connected. Type commands to interact with your container.
      </OuiText>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  deploymentId: string;
  organizationId?: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => props.organizationId || orgsStore.currentOrgId || "");

const client = useConnectClient(DeploymentService);
const terminalRef = ref<HTMLElement | null>(null);
const isConnected = ref(false);
const isLoading = ref(false);
const error = ref("");
let terminalStream: any = null;

const connectTerminal = async () => {
  isLoading.value = true;
  error.value = "";

  try {
    if (!terminalRef.value) {
      throw new Error("Terminal element not found");
    }

    // Create bidirectional stream
    terminalStream = client.streamTerminal();

    // Send initial connection message
    await terminalStream.send({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      input: new Uint8Array(0),
      cols: 80,
      rows: 24,
    });

    // Handle output from container
    terminalStream.onMessage((output: any) => {
      if (terminalRef.value) {
        const text = new TextDecoder().decode(output.output);
        terminalRef.value.textContent += text;
        terminalRef.value.scrollTop = terminalRef.value.scrollHeight;
      }
      if (output.exit) {
        isConnected.value = false;
        error.value = "Terminal session ended";
      }
    });

    // Handle input from user
    const handleKeyPress = (e: KeyboardEvent) => {
      if (!isConnected.value || !terminalStream) return;
      
      const input = new TextEncoder().encode(e.key);
      terminalStream.send({
        organizationId: organizationId.value,
        deploymentId: props.deploymentId,
        input: Array.from(input),
        cols: 80,
        rows: 24,
      });
    };

    terminalRef.value.addEventListener("keypress", handleKeyPress);
    terminalRef.value.setAttribute("contenteditable", "true");
    terminalRef.value.focus();

    isConnected.value = true;
  } catch (err: any) {
    console.error("Failed to connect terminal:", err);
    error.value = err.message || "Failed to connect terminal. Please try again.";
  } finally {
    isLoading.value = false;
  }
};

onUnmounted(() => {
  if (terminalStream) {
    terminalStream.close();
  }
});
</script>
