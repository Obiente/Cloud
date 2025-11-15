<template>
  <div class="h-full overflow-y-auto p-6">
    <div class="max-w-4xl mx-auto">
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <div>
              <OuiText as="h2" size="lg" weight="semibold">
                Banned Players
              </OuiText>
              <OuiText size="sm" color="secondary" class="mt-1">
                Manage players banned from your server
              </OuiText>
            </div>
            <OuiButton
              variant="solid"
              size="sm"
              @click="showAddDialog = true"
            >
              <PlusIcon class="h-4 w-4 mr-2" />
              Ban Player
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <!-- Add Ban Dialog -->
            <OuiDialog v-model:open="showAddDialog" title="Ban Player">
              <form @submit.prevent="handleAddBan">
                <OuiStack gap="md">
                  <MinecraftPlayerCombobox
                    v-model="newPlayerName"
                    label="Player Name or UUID"
                    helper-text="Search for a player by username (minimum 3 characters). Press Enter to ban."
                    placeholder="Type to search for a player..."
                    @select="handlePlayerSelect"
                  />
                  <OuiField label="Reason" hint="Reason for the ban">
                    <OuiInput
                      v-model="newBanReason"
                      placeholder="Banned by an operator"
                    />
                  </OuiField>
                  <OuiField label="Expires" hint="Leave empty for permanent ban">
                    <OuiInput
                      v-model="newBanExpires"
                      type="datetime-local"
                      placeholder="Permanent"
                    />
                  </OuiField>
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
                      Ban Player
                    </OuiButton>
                  </OuiFlex>
                </OuiStack>
              </form>
            </OuiDialog>

            <!-- Banned Players Table -->
            <div v-if="bannedPlayers.length === 0" class="text-center py-12">
              <OuiText size="sm" color="secondary">
                No players are banned.
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
                    <th class="text-left py-3 px-4">
                      <OuiText size="xs" weight="semibold" color="muted">
                        Reason
                      </OuiText>
                    </th>
                    <th class="text-left py-3 px-4">
                      <OuiText size="xs" weight="semibold" color="muted">
                        Expires
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
                    v-for="(player, index) in bannedPlayers"
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
                    <td class="py-3 px-4">
                      <OuiText size="sm" color="secondary">
                        {{ player.reason || "Banned by an operator" }}
                      </OuiText>
                    </td>
                    <td class="py-3 px-4">
                      <OuiText size="sm" color="secondary">
                        {{ formatExpires(player.expires) }}
                      </OuiText>
                    </td>
                    <td class="py-3 px-4 text-right">
                      <OuiButton
                        variant="ghost"
                        size="sm"
                        @click="handleRemoveBan(index)"
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
import { computed, ref, watch } from "vue";
import {
  PlusIcon,
  TrashIcon,
  UserIcon,
} from "@heroicons/vue/24/outline";
import { useMinecraftPlayer } from "~/composables/useMinecraftPlayer";
import MinecraftPlayerCombobox from "./MinecraftPlayerCombobox.vue";

interface BannedPlayer {
  uuid: string;
  name: string;
  created: string;
  source: string;
  expires: string;
  reason: string;
}

interface Props {
  fileContent: string;
  isSaving?: boolean;
}

interface Emits {
  (e: "save", content: string): void;
}

const props = defineProps<Props>();

const emit = defineEmits<Emits>();

const bannedPlayers = ref<BannedPlayer[]>([]);
const showAddDialog = ref(false);
const newPlayerName = ref("");
const newBanReason = ref("Banned by an operator");
const newBanExpires = ref("");
const playerData = ref<Map<string, any>>(new Map());
const { getPlayerData, loadPlayers } = useMinecraftPlayer();
const isLoadingPlayers = ref(false);

// Parse banned-players.json
function parseBannedPlayers(content: string): BannedPlayer[] {
  try {
    const parsed = JSON.parse(content);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

// Format banned players to JSON
function formatBannedPlayers(entries: BannedPlayer[]): string {
  return JSON.stringify(entries, null, 2);
}

// Format expires date
function formatExpires(expires: string): string {
  if (!expires || expires === "forever") return "Never";
  try {
    const date = new Date(expires);
    return date.toLocaleString();
  } catch {
    return expires;
  }
}

// Initialize from file content
watch(() => props.fileContent, (newContent) => {
  bannedPlayers.value = parseBannedPlayers(newContent);
  loadPlayerData();
}, { immediate: true });

// Load player data from Minecraft API
async function loadPlayerData() {
  if (bannedPlayers.value.length === 0) return;
  
  isLoadingPlayers.value = true;
  try {
    const identifiers = bannedPlayers.value
      .map((p) => p.uuid || p.name)
      .filter((id): id is string => !!id);
    
    const players = await loadPlayers(identifiers);
    playerData.value = players;
  } catch (error) {
    console.error("[BannedPlayersEditor] Failed to load player data:", error);
  } finally {
    isLoadingPlayers.value = false;
  }
}

function handleImageError(event: Event) {
  const img = event.target as HTMLImageElement;
  img.style.display = "none";
}

const selectedPlayer = ref<{ uuid: string; name: string } | null>(null);

function handlePlayerSelect(player: { uuid: string; name: string; avatarUrl?: string }) {
  selectedPlayer.value = player;
  newPlayerName.value = player.name;
}

async function handleAddBan() {
  const name = newPlayerName.value.trim();
  if (!name) return;

  const now = new Date().toISOString();
  const expires = newBanExpires.value
    ? new Date(newBanExpires.value).toISOString()
    : "forever";

  let entry: BannedPlayer;

  // If we have a selected player from the combobox, use that data
  if (selectedPlayer.value) {
    entry = {
      uuid: selectedPlayer.value.uuid,
      name: selectedPlayer.value.name,
      created: now,
      source: "Server",
      expires: expires,
      reason: newBanReason.value || "Banned by an operator",
    };
    // Cache the player data
    if (selectedPlayer.value.uuid) {
      const player = await getPlayerData(selectedPlayer.value.uuid);
      if (player) {
        playerData.value.set(selectedPlayer.value.uuid, player);
        playerData.value.set(selectedPlayer.value.name, player);
      }
    }
  } else {
    // Fallback: Check if it's a UUID or username
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    const isUUID = uuidRegex.test(name);

    if (isUUID) {
      entry = {
        uuid: name,
        name: "",
        created: now,
        source: "Server",
        expires: expires,
        reason: newBanReason.value || "Banned by an operator",
      };
      const player = await getPlayerData(name);
      if (player) {
        entry.name = player.name || "";
        playerData.value.set(name, player);
      }
    } else {
      entry = {
        uuid: "",
        name: name,
        created: now,
        source: "Server",
        expires: expires,
        reason: newBanReason.value || "Banned by an operator",
      };
      const player = await getPlayerData(name);
      if (player) {
        entry.uuid = player.uuid;
        playerData.value.set(player.uuid, player);
        playerData.value.set(name, player);
      }
    }
  }

  bannedPlayers.value.push(entry);
  // Auto-save when player is added
  const formatted = formatBannedPlayers(bannedPlayers.value);
  emit("save", formatted);
  
  newPlayerName.value = "";
  newBanReason.value = "Banned by an operator";
  newBanExpires.value = "";
  selectedPlayer.value = null;
  showAddDialog.value = false;
}

function handleRemoveBan(index: number) {
  bannedPlayers.value.splice(index, 1);
  // Auto-save when player is removed
  const formatted = formatBannedPlayers(bannedPlayers.value);
  emit("save", formatted);
}
</script>

