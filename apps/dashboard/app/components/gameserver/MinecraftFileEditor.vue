<template>
  <div class="h-full flex flex-col">
    <div v-if="isLoading" class="h-full flex items-center justify-center">
      <OuiStack gap="md" align="center">
        <OuiSpinner size="lg" />
        <OuiText size="sm" color="secondary">Loading file...</OuiText>
      </OuiStack>
    </div>
    <div v-else-if="error" class="h-full flex items-center justify-center p-8">
      <OuiStack gap="md" align="center" class="max-w-md text-center">
        <div class="flex items-center justify-center w-16 h-16 rounded-full bg-danger/10">
          <ExclamationTriangleIcon class="h-8 w-8 text-danger" />
        </div>
        <OuiText size="lg" weight="semibold" color="danger">
          Unable to Load File
        </OuiText>
        <OuiText size="sm" color="secondary">
          {{ error }}
        </OuiText>
        <OuiButton variant="outline" size="sm" @click="loadFile">
          <ArrowPathIcon class="h-4 w-4 mr-2" />
          Retry
        </OuiButton>
      </OuiStack>
    </div>
    <div class="h-full overflow-auto">
      <component
        :is="editorComponent"
        :file-content="fileContent"
        :game-server-id="gameServerId"
        v-bind="editorProps"
        @save="handleSave"
        @reload="handleReload"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { ExclamationTriangleIcon, ArrowPathIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import { useToast } from "~/composables/useToast";
import type { Component } from "vue";

interface Props {
  gameServerId: string;
  filePath: string;
  editorComponent: Component;
  editorProps?: Record<string, any>;
}

const props = defineProps<Props>();

const client = useConnectClient(GameServerService);
const { toast } = useToast();

const fileContent = ref("");
const isLoading = ref(true);
const isSaving = ref(false);
const error = ref<string | null>(null);

// Load file from game server
async function loadFile() {
  isLoading.value = true;
  error.value = null;

  try {
    // Get volumes for this game server
    const volumesRes = await client.listGameServerFiles({
      gameServerId: props.gameServerId,
      path: "/",
      listVolumes: true,
    });

    // Use the first volume (usually mounted at /data for game servers)
    // or default to container filesystem
    const volumeName = volumesRes.volumes?.[0]?.name;
    const mountPoint = volumesRes.volumes?.[0]?.mountPoint || "/data";

    // If using a volume and path starts with mount point, strip it
    // e.g., "/data/whitelist.json" -> "whitelist.json" when volume is mounted at /data
    let filePath = props.filePath;
    if (volumeName && filePath.startsWith(mountPoint)) {
      filePath = filePath.slice(mountPoint.length);
      // Ensure path starts with / for API
      if (!filePath.startsWith("/")) {
        filePath = "/" + filePath;
      }
    } else if (!filePath.startsWith("/")) {
      // Ensure path starts with / for API
      filePath = "/" + filePath;
    }

    const res = await client.getGameServerFile({
      gameServerId: props.gameServerId,
      path: filePath,
      volumeName: volumeName,
    });

    fileContent.value = res.content || "[]"; // Default to empty JSON array for JSON files
  } catch (err: any) {
    console.error("[MinecraftFileEditor] Failed to load file:", err);
    error.value = err?.message || "Failed to load file";
    // Set default content based on file type so editor can initialize
    if (props.filePath.endsWith('.json')) {
      fileContent.value = "[]";
    } else {
      fileContent.value = "";
    }
  } finally {
    isLoading.value = false;
  }
}

// Save file to game server
async function handleSave(content: string) {
  isSaving.value = true;
  error.value = null;

  try {
    // Get volume name (same as load)
    const volumesRes = await client.listGameServerFiles({
      gameServerId: props.gameServerId,
      path: "/",
      listVolumes: true,
    });
    const volumeName = volumesRes.volumes?.[0]?.name;
    const mountPoint = volumesRes.volumes?.[0]?.mountPoint || "/data";

    // If using a volume and path starts with mount point, strip it
    // e.g., "/data/whitelist.json" -> "whitelist.json" when volume is mounted at /data
    let filePath = props.filePath;
    if (volumeName && filePath.startsWith(mountPoint)) {
      filePath = filePath.slice(mountPoint.length);
      // Ensure path starts with / for API
      if (!filePath.startsWith("/")) {
        filePath = "/" + filePath;
      }
    } else if (!filePath.startsWith("/")) {
      // Ensure path starts with / for API
      filePath = "/" + filePath;
    }

    await client.writeGameServerFile({
      gameServerId: props.gameServerId,
      path: filePath,
      content: content,
      volumeName: volumeName,
    });

    fileContent.value = content; // Update local content
    toast.success("File saved successfully");
  } catch (err: any) {
    console.error("[MinecraftFileEditor] Failed to save file:", err);
    error.value = err?.message || "Failed to save file";
    toast.error("Failed to save file", err?.message);
  } finally {
    isSaving.value = false;
  }
}

// Load file on mount
onMounted(() => {
  loadFile();
});

// Reload if file path changes
watch(() => props.filePath, () => {
  loadFile();
});

// Handle reload event from editor component
async function handleReload() {
  // Reload the file to sync with server state
  await loadFile();
}

// Expose reload function for parent components
defineExpose({
  reload: loadFile,
});
</script>

