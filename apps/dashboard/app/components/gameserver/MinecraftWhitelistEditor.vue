<template>
  <div class="h-full overflow-y-auto p-6">
    <div class="max-w-4xl mx-auto">
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <div>
              <OuiText as="h2" size="lg" weight="semibold">
                {{ title }}
              </OuiText>
              <OuiText size="sm" color="secondary" class="mt-1">
                {{ description }}
              </OuiText>
            </div>
            <OuiButton
              variant="solid"
              size="sm"
              @click="showAddDialog = true"
            >
              <PlusIcon class="h-4 w-4 mr-2" />
              Add Player
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <!-- Add Player Dialog -->
            <OuiDialog 
              v-model:open="showAddDialog" 
              :title="`Add Player to ${title}`"
            >
              <form @submit.prevent="handleAddPlayer">
                <OuiStack gap="md">
                  <MinecraftPlayerCombobox
                    v-model="newPlayerName"
                    label="Player Name or UUID"
                    helper-text="Search for a player by username (minimum 3 characters). Press Enter to add."
                    placeholder="Type to search for a player..."
                    @select="handlePlayerSelect"
                  />
                  <OuiFlex gap="sm" justify="end">
                    <OuiButton 
                      type="button"
                      variant="ghost" 
                      @click="showAddDialog = false"
                    >
                      Cancel
                    </OuiButton>
                    <OuiButton 
                      type="submit"
                      variant="solid"
                    >
                      Add Player
                    </OuiButton>
                  </OuiFlex>
                </OuiStack>
              </form>
            </OuiDialog>

            <!-- List Table -->
            <div v-if="whitelist.length === 0" class="text-center py-12">
              <OuiText size="sm" color="secondary">
                {{ emptyMessage }}
              </OuiText>
            </div>
            <div v-else class="overflow-x-auto">
              <table class="w-full">
                <thead>
                  <tr class="border-b border-border-default">
                    <th class="text-left py-3 px-4">
                      <OuiText size="xs" weight="semibold" color="muted">
                        Player Name
                      </OuiText>
                    </th>
                    <th class="text-left py-3 px-4">
                      <OuiText size="xs" weight="semibold" color="muted">
                        UUID
                      </OuiText>
                    </th>
                    <th class="text-right py-3 px-4">
                      <OuiText size="xs" weight="semibold" color="muted">
                        Actions
                      </OuiText>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="(player, index) in whitelist"
                    :key="index"
                    class="border-b border-border-default hover:bg-surface-hover"
                  >
                    <td class="py-3 px-4">
                      <OuiFlex gap="sm" align="center">
                        <img
                          v-if="playerData.get(player.uuid || player.name || '')?.avatarUrl"
                          :src="playerData.get(player.uuid || player.name || '')?.avatarUrl"
                          :alt="playerData.get(player.uuid || player.name || '')?.name || 'Player'"
                          class="w-8 h-8 rounded"
                          @error="handleImageError"
                        />
                        <div
                          v-else
                          class="w-8 h-8 rounded bg-surface-elevated flex items-center justify-center"
                        >
                          <UserIcon class="h-4 w-4 text-text-tertiary" />
                        </div>
                        <div>
                          <OuiText size="sm" weight="medium">
                            {{ playerData.get(player.uuid || player.name || '')?.name || player.name || "Loading..." }}
                          </OuiText>
                          <OuiText
                            v-if="playerData.get(player.uuid || player.name || '')?.name && player.name"
                            size="xs"
                            color="muted"
                          >
                            {{ player.name }}
                          </OuiText>
                        </div>
                      </OuiFlex>
                    </td>
                    <td class="py-3 px-4">
                      <OuiText size="xs" color="muted" class="font-mono">
                        {{ player.uuid || "—" }}
                      </OuiText>
                    </td>
                    <td class="py-3 px-4 text-right">
                      <OuiButton
                        variant="ghost"
                        size="sm"
                        @click="handleRemovePlayer(index)"
                      >
                        <TrashIcon class="h-4 w-4" />
                      </OuiButton>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from "vue";
import {
  PlusIcon,
  TrashIcon,
  CheckCircleIcon,
  UserIcon,
} from "@heroicons/vue/24/outline";
import { useMinecraftPlayer } from "~/composables/useMinecraftPlayer";
import MinecraftPlayerCombobox from "./MinecraftPlayerCombobox.vue";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService, GameServerStatus } from "@obiente/proto";
import { useToast } from "~/composables/useToast";
import { useGameServerCommand } from "~/composables/useGameServerCommand";
import { useOrganizationsStore } from "~/stores/organizations";

interface WhitelistEntry {
  uuid: string;
  name: string;
}

interface Props {
  fileContent: string;
  title?: string;
  description?: string;
  emptyMessage?: string;
  gameServerId?: string;
  fileType?: "whitelist" | "ops";
}

interface Emits {
  (e: "save", content: string): void;
  (e: "reload"): void;
}

const props = withDefaults(defineProps<Props>(), {
  title: "Whitelist",
  description: "Manage players allowed to join your server",
  emptyMessage: "No players in whitelist. Add players to restrict server access.",
  fileType: "whitelist",
});

const emit = defineEmits<Emits>();

const whitelist = ref<WhitelistEntry[]>([]);
const showAddDialog = ref(false);
const newPlayerName = ref("");
const playerData = ref<Map<string, any>>(new Map());
const { getPlayerData, loadPlayers } = useMinecraftPlayer();
const isLoadingPlayers = ref(false);
const client = props.gameServerId ? useConnectClient(GameServerService) : null;
const { toast } = useToast();
const gameServerStatus = ref<GameServerStatus | null>(null);
const isServerRunning = computed(() => gameServerStatus.value === GameServerStatus.RUNNING);
const orgsStore = useOrganizationsStore();
const gameServerCommand = props.gameServerId ? useGameServerCommand({
  gameServerId: props.gameServerId,
  organizationId: orgsStore.currentOrgId || undefined,
}) : null;

// Parse whitelist.json
function parseWhitelist(content: string): WhitelistEntry[] {
  try {
    const parsed = JSON.parse(content);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

// Format whitelist to JSON
function formatWhitelist(entries: WhitelistEntry[]): string {
  return JSON.stringify(entries, null, 2);
}

// Load game server status
async function loadGameServerStatus() {
  if (!props.gameServerId || !client) return;
  
  try {
    const res = await client.getGameServer({
      gameServerId: props.gameServerId,
    });
    if (res.gameServer) {
      gameServerStatus.value = res.gameServer.status;
    }
  } catch (error) {
    console.error("[WhitelistEditor] Failed to load game server status:", error);
  }
}

// Initialize from file content
watch(() => props.fileContent, (newContent) => {
  whitelist.value = parseWhitelist(newContent);
  loadPlayerData();
}, { immediate: true });

// Load server status on mount
onMounted(() => {
  if (props.gameServerId) {
    loadGameServerStatus();
  }
});

// Load player data from Minecraft API
async function loadPlayerData() {
  if (whitelist.value.length === 0) return;
  
  isLoadingPlayers.value = true;
  try {
    const identifiers = whitelist.value
      .map((p) => p.uuid || p.name)
      .filter((id): id is string => !!id);
    
    const players = await loadPlayers(identifiers);
    playerData.value = players;
  } catch (error) {
    console.error("[WhitelistEditor] Failed to load player data:", error);
  } finally {
    isLoadingPlayers.value = false;
  }
}

function handleImageError(event: Event) {
  const img = event.target as HTMLImageElement;
  img.style.display = "none";
}

const selectedPlayer = ref<{ uuid: string; name: string } | null>(null);

function handlePlayerSelect(player: { uuid: string; name: string; avatarUrl?: string; label?: string; value?: string }) {
  selectedPlayer.value = player;
  newPlayerName.value = player.name;
}

async function handleAddPlayer() {
  const name = newPlayerName.value.trim();
  if (!name) return;

  let entry: WhitelistEntry;
  let playerName = name;

  // If we have a selected player from the combobox, use that data
  if (selectedPlayer.value) {
    entry = {
      uuid: selectedPlayer.value.uuid,
      name: selectedPlayer.value.name,
    };
    playerName = selectedPlayer.value.name;
    // Cache the player data
    if (selectedPlayer.value.uuid) {
      const player = await getPlayerData(selectedPlayer.value.uuid);
      if (player) {
        playerData.value.set(selectedPlayer.value.uuid, player);
        playerData.value.set(selectedPlayer.value.name, player);
        playerName = player.name || playerName;
      }
    }
  } else {
    // Fallback: Check if it's a UUID or username
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    const isUUID = uuidRegex.test(name);

    if (isUUID) {
      // Add by UUID, try to fetch name
      entry = {
        uuid: name,
        name: "",
      };
      const player = await getPlayerData(name);
      if (player) {
        entry.name = player.name || "";
        playerName = player.name || name;
        playerData.value.set(name, player);
      }
    } else {
      // Add by name, try to fetch UUID
      entry = {
        uuid: "",
        name: name,
      };
      const player = await getPlayerData(name);
      if (player) {
        entry.uuid = player.uuid;
        playerName = player.name || name;
        playerData.value.set(player.uuid, player);
        playerData.value.set(name, player);
      }
    }
  }

  // If server is running, use command instead of file write
  if (isServerRunning.value && props.gameServerId && gameServerCommand) {
    // Use the player name for the command - prefer the fetched name, fallback to input
    const commandPlayerName = playerName || entry.name || name;
    
    if (!commandPlayerName || commandPlayerName.trim() === "") {
      toast.error("Cannot add player: player name is required");
      return;
    }
    
    try {
      // Construct the command - Minecraft commands use exact username
      const command = props.fileType === "ops" 
        ? `op ${commandPlayerName.trim()}`
        : `whitelist add ${commandPlayerName.trim()}`;
      
      console.log("[WhitelistEditor] Sending command via WebSocket:", command);
      
      // Send command via WebSocket
      await gameServerCommand.sendCommand(command);
      
      toast.success(`Player added to ${props.title.toLowerCase()}`);
      
      // Reload file immediately - commands are sent via WebSocket and take effect instantly
      emit("reload");
    } catch (error: any) {
      console.error("[WhitelistEditor] Failed to send command:", error);
      toast.error(`Failed to add player: ${error?.message || "Unknown error"}`);
      return;
    }
  } else {
    // Server not running, update local state and save file
    whitelist.value.push(entry);
    // Auto-save when server is stopped
    const formatted = formatWhitelist(whitelist.value);
    emit("save", formatted);
  }

  newPlayerName.value = "";
  selectedPlayer.value = null;
  showAddDialog.value = false;
}

async function handleRemovePlayer(index: number) {
  const player = whitelist.value[index];
  if (!player) return;

  // Get player name - prefer the name from playerData, fallback to entry name
  const playerName = playerData.value.get(player.uuid || player.name || '')?.name || player.name;
  
  if (!playerName || playerName.trim() === "") {
    toast.error("Cannot remove player: player name is required");
    return;
  }
  
  // If server is running, use command instead of file write
  if (isServerRunning.value && props.gameServerId && gameServerCommand) {
    try {
      // Construct the command - Minecraft commands use exact username
      const command = props.fileType === "ops" 
        ? `deop ${playerName.trim()}`
        : `whitelist remove ${playerName.trim()}`;
      
      console.log("[WhitelistEditor] Sending command via WebSocket:", command);
      
      // Send command via WebSocket
      await gameServerCommand.sendCommand(command);
      
      toast.success(`Player removed from ${props.title.toLowerCase()}`);
      
      // Reload file immediately - commands are sent via WebSocket and take effect instantly
      emit("reload");
    } catch (error: any) {
      console.error("[WhitelistEditor] Failed to send command:", error);
      toast.error(`Failed to remove player: ${error?.message || "Unknown error"}`);
      return;
    }
  } else {
    // Server not running, update local state and save file
    whitelist.value.splice(index, 1);
    // Auto-save when server is stopped
    const formatted = formatWhitelist(whitelist.value);
    emit("save", formatted);
  }
}

</script>

