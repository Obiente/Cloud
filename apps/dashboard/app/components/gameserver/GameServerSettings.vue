<template>
  <OuiStack gap="lg">
    <OuiCard variant="default">
      <OuiCardBody>
        <OuiStack gap="lg">
          <OuiText as="h2" size="lg" weight="semibold" color="primary">
            Game Server Settings
          </OuiText>

          <form @submit.prevent="handleSave">
            <OuiStack gap="md">
              <!-- Name -->
              <OuiInput
                v-model="formData.name"
                label="Server Name"
                placeholder="my-game-server"
                required
                :disabled="isSaving"
              />

              <!-- Description -->
              <OuiTextarea
                v-model="formData.description"
                label="Description"
                placeholder="A brief description of your game server"
                :disabled="isSaving"
                :rows="3"
              />

              <!-- Resource Configuration -->
              <OuiStack gap="md">
                <OuiText size="sm" weight="semibold" color="primary">
                  Resource Configuration
                </OuiText>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <!-- Memory -->
                  <OuiInput
                    v-model="formData.memoryGB"
                    label="Memory (GB) - Max Limit"
                    type="number"
                    min="0.5"
                    max="32"
                    step="0.5"
                    placeholder="2"
                    required
                    :disabled="isSaving"
                    hint="Maximum memory allocation for the server"
                  />

                  <!-- CPU Cores -->
                  <OuiInput
                    v-model="formData.cpuCores"
                    label="vCPU Cores - Max Limit"
                    type="number"
                    min="0.25"
                    max="8"
                    step="0.25"
                    placeholder="1"
                    required
                    :disabled="isSaving"
                    hint="Maximum CPU cores allocation"
                  />
                </div>
              </OuiStack>

              <!-- Game-Specific Settings -->
              <OuiStack v-if="gameSpecificSettings.length > 0" gap="md">
                <OuiText size="sm" weight="semibold" color="primary">
                  Game-Specific Settings
                </OuiText>
                <OuiCard variant="outline" class="bg-surface-subtle/30">
                  <OuiCardBody>
                    <OuiStack gap="md">
                      <div
                        v-for="setting in gameSpecificSettings"
                        :key="setting.key"
                      >
                        <OuiInput
                          v-if="setting.type === 'input' || setting.type === 'number'"
                          v-model="setting.value"
                          :label="setting.label"
                          :placeholder="setting.placeholder"
                          :disabled="isSaving"
                          :hint="setting.hint"
                          :type="setting.type === 'number' ? 'number' : 'text'"
                          :min="setting.min"
                          :max="setting.max"
                          :step="setting.step"
                        />
                        <OuiSelect
                          v-else-if="setting.type === 'select'"
                          v-model="setting.value"
                          :label="setting.label"
                          :items="setting.options || []"
                          :disabled="isSaving"
                          :hint="setting.hint"
                        />
                      </div>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </OuiStack>

              <!-- Docker Configuration -->
              <OuiStack gap="md">
                <OuiText size="sm" weight="semibold" color="primary">
                  Docker Configuration
                </OuiText>

                <!-- Start Command -->
                <OuiInput
                  v-model="formData.startCommand"
                  label="Start Command"
                  placeholder="Leave empty to use default"
                  :disabled="isSaving"
                  hint="Custom command to start the server (optional)"
                />
              </OuiStack>

              <!-- Environment Variables -->
              <OuiStack gap="md">
                <OuiFlex justify="between" align="center">
                  <OuiText size="sm" weight="semibold" color="primary">
                    Environment Variables
                  </OuiText>
                  <OuiButton
                    variant="ghost"
                    size="sm"
                    @click="addEnvVar"
                    :disabled="isSaving"
                  >
                    Add Variable
                  </OuiButton>
                </OuiFlex>

                <div v-if="envVars.length === 0" class="text-center py-4">
                  <OuiText size="sm" color="secondary">
                    No environment variables configured
                  </OuiText>
                </div>

                <div v-else class="space-y-2">
                  <div
                    v-for="(envVar, index) in envVars"
                    :key="index"
                    class="flex gap-2 items-start"
                  >
                    <OuiInput
                      v-model="envVar.key"
                      placeholder="Variable name"
                      :disabled="isSaving"
                      class="flex-1"
                    />
                    <OuiInput
                      v-model="envVar.value"
                      placeholder="Variable value"
                      :disabled="isSaving"
                      class="flex-1"
                    />
                    <OuiButton
                      variant="ghost"
                      size="sm"
                      color="danger"
                      @click="removeEnvVar(index)"
                      :disabled="isSaving"
                    >
                      <TrashIcon class="h-4 w-4" />
                    </OuiButton>
                  </div>
                </div>
              </OuiStack>

              <!-- Save Button -->
              <OuiFlex justify="end" gap="sm">
                <OuiButton
                  variant="ghost"
                  @click="resetForm"
                  :disabled="isSaving || !hasChanges"
                >
                  Reset
                </OuiButton>
                <OuiButton
                  type="submit"
                  color="primary"
                  :loading="isSaving"
                  :disabled="!hasChanges"
                >
                  Save Changes
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </form>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Danger Zone -->
    <OuiCard variant="outline" class="border-danger/20">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiStack gap="sm">
            <OuiText as="h3" size="md" weight="semibold" color="danger">
              Danger Zone
            </OuiText>
            <OuiText size="sm" color="secondary">
              Irreversible and destructive actions
            </OuiText>
          </OuiStack>

          <OuiCard variant="outline" class="border-danger/30">
            <OuiCardBody>
              <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="primary">
                    Delete Game Server
                  </OuiText>
                  <OuiText size="xs" color="secondary">
                    Permanently delete this game server and all its data. This action cannot be undone.
                  </OuiText>
                </OuiStack>
                <OuiButton
                  variant="outline"
                  color="danger"
                  @click="$emit('delete')"
                >
                  Delete Server
                </OuiButton>
              </OuiFlex>
            </OuiCardBody>
          </OuiCard>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { TrashIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService, GameType } from "@obiente/proto";
import { useToast } from "~/composables/useToast";
import type { GameServer } from "@obiente/proto";

interface Props {
  gameServer: GameServer | null;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  delete: [];
  saved: [];
}>();

const { toast } = useToast();
const client = useConnectClient(GameServerService);

const isSaving = ref(false);

// Form data
const formData = ref({
  name: "",
  description: "",
  memoryGB: "",
  cpuCores: "",
  startCommand: "",
});

const envVars = ref<Array<{ key: string; value: string }>>([]);

// Game-specific settings configuration
interface GameSetting {
  key: string;
  label: string;
  placeholder?: string;
  hint?: string;
  type: "input" | "select" | "number";
  value: string;
  options?: Array<{ label: string; value: string }>;
  min?: number;
  max?: number;
  step?: number;
}

// Game-specific settings reactive array
const gameSpecificSettings = ref<GameSetting[]>([]);

// Get game-specific settings based on game type
const initializeGameSpecificSettings = () => {
  if (!props.gameServer?.gameType) {
    gameSpecificSettings.value = [];
    return;
  }

  // Handle both number and GameType enum values
  const gameType = typeof props.gameServer.gameType === 'number' 
    ? props.gameServer.gameType as GameType 
    : props.gameServer.gameType;
  const settings: GameSetting[] = [];
  const currentEnvVars = props.gameServer.envVars || {};

  // Helper to get env var value
  const getEnvVar = (key: string, defaultValue = "") => {
    return currentEnvVars[key] || defaultValue;
  };

  switch (gameType) {
    case GameType.MINECRAFT:
    case GameType.MINECRAFT_JAVA:
      settings.push(
        {
          key: "server_version",
          label: "Minecraft Version",
          placeholder: "1.20.1",
          hint: "Minecraft server version (e.g., 1.20.1). Leave empty to use latest.",
          type: "input",
          value: props.gameServer.serverVersion || "",
        },
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "20",
          hint: "Maximum number of players allowed on the server",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "20"),
          min: 1,
          max: 1000,
        },
        {
          key: "DIFFICULTY",
          label: "Difficulty",
          hint: "Game difficulty level",
          type: "select",
          value: getEnvVar("DIFFICULTY", "easy"),
          options: [
            { label: "Peaceful", value: "peaceful" },
            { label: "Easy", value: "easy" },
            { label: "Normal", value: "normal" },
            { label: "Hard", value: "hard" },
          ],
        },
        {
          key: "MODE",
          label: "Game Mode",
          hint: "Default game mode for new players",
          type: "select",
          value: getEnvVar("MODE", "survival"),
          options: [
            { label: "Survival", value: "survival" },
            { label: "Creative", value: "creative" },
            { label: "Adventure", value: "adventure" },
            { label: "Spectator", value: "spectator" },
          ],
        },
        {
          key: "MOTD",
          label: "Server Message (MOTD)",
          placeholder: "A Minecraft Server",
          hint: "Message displayed in server list",
          type: "input",
          value: getEnvVar("MOTD", ""),
        },
        {
          key: "PVP",
          label: "PvP Enabled",
          hint: "Enable player vs player combat",
          type: "select",
          value: getEnvVar("PVP", "true"),
          options: [
            { label: "Enabled", value: "true" },
            { label: "Disabled", value: "false" },
          ],
        },
        {
          key: "TYPE",
          label: "Server Type",
          hint: "Minecraft server type/variant",
          type: "select",
          value: getEnvVar("TYPE", "VANILLA"),
          options: [
            { label: "Vanilla", value: "VANILLA" },
            { label: "Spigot", value: "SPIGOT" },
            { label: "Paper", value: "PAPER" },
            { label: "Forge", value: "FORGE" },
            { label: "Fabric", value: "FABRIC" },
          ],
        }
      );
      break;

    case GameType.MINECRAFT_BEDROCK:
      settings.push(
        {
          key: "server_version",
          label: "Minecraft Version",
          placeholder: "1.20.1",
          hint: "Minecraft Bedrock server version (e.g., 1.20.1). Leave empty to use latest.",
          type: "input",
          value: props.gameServer.serverVersion || "",
        },
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "10",
          hint: "Maximum number of players allowed on the server",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "10"),
          min: 1,
          max: 30,
        },
        {
          key: "DIFFICULTY",
          label: "Difficulty",
          hint: "Game difficulty level",
          type: "select",
          value: getEnvVar("DIFFICULTY", "easy"),
          options: [
            { label: "Peaceful", value: "peaceful" },
            { label: "Easy", value: "easy" },
            { label: "Normal", value: "normal" },
            { label: "Hard", value: "hard" },
          ],
        },
        {
          key: "GAMEMODE",
          label: "Game Mode",
          hint: "Default game mode for new players",
          type: "select",
          value: getEnvVar("GAMEMODE", "survival"),
          options: [
            { label: "Survival", value: "survival" },
            { label: "Creative", value: "creative" },
            { label: "Adventure", value: "adventure" },
          ],
        },
        {
          key: "ALLOW_LIST",
          label: "Allow List Enabled",
          hint: "Enable the allow list",
          type: "select",
          value: getEnvVar("ALLOW_LIST", "false"),
          options: [
            { label: "Enabled", value: "true" },
            { label: "Disabled", value: "false" },
          ],
        }
      );
      break;

    case GameType.VALHEIM:
      settings.push(
        {
          key: "SERVER_NAME",
          label: "Server Name",
          placeholder: "My Valheim Server",
          hint: "Name displayed in server list",
          type: "input",
          value: getEnvVar("SERVER_NAME", ""),
        },
        {
          key: "SERVER_PASS",
          label: "Server Password",
          placeholder: "Leave empty for no password",
          hint: "Password required to join the server",
          type: "input",
          value: getEnvVar("SERVER_PASS", ""),
        },
        {
          key: "SERVER_PUBLIC",
          label: "Public Server",
          hint: "Make server visible in public server list",
          type: "select",
          value: getEnvVar("SERVER_PUBLIC", "1"),
          options: [
            { label: "Public", value: "1" },
            { label: "Private", value: "0" },
          ],
        },
        {
          key: "WORLD_NAME",
          label: "World Name",
          placeholder: "Dedicated",
          hint: "Name of the world to load/create",
          type: "input",
          value: getEnvVar("WORLD_NAME", "Dedicated"),
        }
      );
      break;

    case GameType.TERRARIA:
      settings.push(
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "8",
          hint: "Maximum number of players",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "8"),
          min: 1,
          max: 16,
        },
        {
          key: "PASSWORD",
          label: "Server Password",
          placeholder: "Leave empty for no password",
          hint: "Password required to join",
          type: "input",
          value: getEnvVar("PASSWORD", ""),
        },
        {
          key: "DIFFICULTY",
          label: "Difficulty",
          hint: "World difficulty",
          type: "select",
          value: getEnvVar("DIFFICULTY", "normal"),
          options: [
            { label: "Normal", value: "normal" },
            { label: "Expert", value: "expert" },
            { label: "Master", value: "master" },
            { label: "Journey", value: "journey" },
          ],
        },
        {
          key: "WORLD_NAME",
          label: "World Name",
          placeholder: "World",
          hint: "Name of the world",
          type: "input",
          value: getEnvVar("WORLD_NAME", "World"),
        }
      );
      break;

    case GameType.RUST:
      settings.push(
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "50",
          hint: "Maximum number of players",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "50"),
          min: 1,
          max: 500,
        },
        {
          key: "SERVER_HOSTNAME",
          label: "Server Name",
          placeholder: "My Rust Server",
          hint: "Server name displayed in server list",
          type: "input",
          value: getEnvVar("SERVER_HOSTNAME", ""),
        },
        {
          key: "SERVER_DESCRIPTION",
          label: "Server Description",
          placeholder: "A Rust server",
          hint: "Server description",
          type: "input",
          value: getEnvVar("SERVER_DESCRIPTION", ""),
        },
        {
          key: "RUST_SERVER_STARTUP_ARGUMENTS",
          label: "Server Arguments",
          placeholder: "-batchmode -nographics +server.secure 1",
          hint: "Additional server startup arguments",
          type: "input",
          value: getEnvVar("RUST_SERVER_STARTUP_ARGUMENTS", ""),
        }
      );
      break;

    case GameType.CS2:
    case GameType.TF2:
      settings.push(
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "16",
          hint: "Maximum number of players",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "16"),
          min: 1,
          max: 64,
        },
        {
          key: "SRCDS_HOSTNAME",
          label: "Server Name",
          placeholder: "My Server",
          hint: "Server name displayed in server list",
          type: "input",
          value: getEnvVar("SRCDS_HOSTNAME", ""),
        },
        {
          key: "SRCDS_PW",
          label: "Server Password",
          placeholder: "Leave empty for no password",
          hint: "Password required to join",
          type: "input",
          value: getEnvVar("SRCDS_PW", ""),
        },
        {
          key: "SRCDS_RCONPW",
          label: "RCON Password",
          placeholder: "changeme",
          hint: "RCON password for remote administration",
          type: "input",
          value: getEnvVar("SRCDS_RCONPW", ""),
        }
      );
      break;

    case GameType.ARK:
      settings.push(
        {
          key: "MAX_PLAYERS",
          label: "Max Players",
          placeholder: "70",
          hint: "Maximum number of players",
          type: "number",
          value: getEnvVar("MAX_PLAYERS", "70"),
          min: 1,
          max: 100,
        },
        {
          key: "SERVER_ADMIN_PASSWORD",
          label: "Admin Password",
          placeholder: "changeme",
          hint: "Administrator password",
          type: "input",
          value: getEnvVar("SERVER_ADMIN_PASSWORD", ""),
        },
        {
          key: "SERVER_PASSWORD",
          label: "Server Password",
          placeholder: "Leave empty for no password",
          hint: "Password required to join",
          type: "input",
          value: getEnvVar("SERVER_PASSWORD", ""),
        },
        {
          key: "MAP",
          label: "Map",
          hint: "Which map to load",
          type: "select",
          value: getEnvVar("MAP", "TheIsland"),
          options: [
            { label: "The Island", value: "TheIsland" },
            { label: "The Center", value: "TheCenter" },
            { label: "Scorched Earth", value: "ScorchedEarth" },
            { label: "Ragnarok", value: "Ragnarok" },
            { label: "Aberration", value: "Aberration" },
            { label: "Extinction", value: "Extinction" },
            { label: "Valguero", value: "Valguero" },
            { label: "Crystal Isles", value: "CrystalIsles" },
            { label: "Genesis", value: "Genesis" },
            { label: "Genesis 2", value: "Genesis2" },
          ],
        }
      );
      break;

    case GameType.FACTORIO:
      settings.push(
        {
          key: "FACTORIO_SERVER_NAME",
          label: "Server Name",
          placeholder: "My Factorio Server",
          hint: "Server name",
          type: "input",
          value: getEnvVar("FACTORIO_SERVER_NAME", ""),
        },
        {
          key: "FACTORIO_SERVER_DESCRIPTION",
          label: "Server Description",
          placeholder: "A Factorio server",
          hint: "Server description",
          type: "input",
          value: getEnvVar("FACTORIO_SERVER_DESCRIPTION", ""),
        },
        {
          key: "FACTORIO_MAX_PLAYERS",
          label: "Max Players",
          placeholder: "4",
          hint: "Maximum number of players",
          type: "number",
          value: getEnvVar("FACTORIO_MAX_PLAYERS", "4"),
          min: 1,
          max: 100,
        },
        {
          key: "FACTORIO_PASSWORD",
          label: "Server Password",
          placeholder: "Leave empty for no password",
          hint: "Password required to join",
          type: "input",
          value: getEnvVar("FACTORIO_PASSWORD", ""),
        }
      );
      break;
  }

  gameSpecificSettings.value = settings;
};

// Initialize form from game server
const initializeForm = () => {
  if (!props.gameServer) return;

  formData.value = {
    name: props.gameServer.name || "",
    description: props.gameServer.description || "",
    memoryGB: props.gameServer.memoryBytes
      ? ((typeof props.gameServer.memoryBytes === 'bigint'
          ? Number(props.gameServer.memoryBytes)
          : Number(props.gameServer.memoryBytes)) / (1024 * 1024 * 1024)).toFixed(2)
      : "",
    cpuCores: props.gameServer.cpuCores?.toString() || "",
    startCommand: props.gameServer.startCommand || "",
  };

  // Initialize game-specific settings
  initializeGameSpecificSettings();

  // Convert env vars map to array (excluding game-specific ones)
  if (props.gameServer.envVars) {
    const gameSpecificKeys = new Set(gameSpecificSettings.value.map((s) => s.key));
    envVars.value = Object.entries(props.gameServer.envVars)
      .filter(([key]) => !gameSpecificKeys.has(key))
      .map(([key, value]) => ({
        key,
        value,
      }));
  } else {
    envVars.value = [];
  }
};

// Watch for game server changes
watch(() => props.gameServer, initializeForm, { immediate: true });

// Check if form has changes
const hasChanges = computed(() => {
  if (!props.gameServer) return false;

  const memoryBytes = parseFloat(formData.value.memoryGB) * 1024 * 1024 * 1024;
  const cpuCores = parseFloat(formData.value.cpuCores);

  // Check name
  if (formData.value.name !== props.gameServer.name) return true;

  // Check description
  if (formData.value.description !== (props.gameServer.description || "")) return true;

  // Check memory
  const currentMemoryBytes = typeof props.gameServer.memoryBytes === 'bigint'
    ? Number(props.gameServer.memoryBytes)
    : Number(props.gameServer.memoryBytes);
  if (Math.abs(memoryBytes - currentMemoryBytes) > 1000000) return true;

  // Check CPU cores
  if (cpuCores !== props.gameServer.cpuCores) return true;

  // Check start command
  if (formData.value.startCommand !== (props.gameServer.startCommand || "")) return true;

  // Check game-specific settings
  const currentEnvVars = props.gameServer.envVars || {};
  for (const setting of gameSpecificSettings.value) {
    if (setting.key === "server_version") {
      // Check server_version separately (it's not an env var)
      const currentServerVersion = props.gameServer.serverVersion || "";
      if (setting.value.trim() !== currentServerVersion) return true;
    } else {
      const currentValue = currentEnvVars[setting.key] || "";
      if (setting.value !== currentValue) return true;
    }
  }

  // Check env vars
  const formEnvVars: Record<string, string> = {};
  envVars.value.forEach(({ key, value }) => {
    if (key.trim()) {
      formEnvVars[key.trim()] = value.trim();
    }
  });

  // Merge game-specific settings into form env vars for comparison (excluding server_version)
  gameSpecificSettings.value.forEach((setting) => {
    if (setting.key !== "server_version" && setting.value.trim()) {
      formEnvVars[setting.key] = setting.value.trim();
    }
  });

  if (Object.keys(currentEnvVars).length !== Object.keys(formEnvVars).length) return true;

  for (const [key, value] of Object.entries(formEnvVars)) {
    if (currentEnvVars[key] !== value) return true;
  }

  return false;
});

// Add environment variable
const addEnvVar = () => {
  envVars.value.push({ key: "", value: "" });
};

// Remove environment variable
const removeEnvVar = (index: number) => {
  envVars.value.splice(index, 1);
};

// Reset form
const resetForm = () => {
  initializeForm();
};

// Save changes
const handleSave = async () => {
  if (!props.gameServer) return;

  isSaving.value = true;
  try {
    const memoryBytes = BigInt(Math.round(parseFloat(formData.value.memoryGB) * 1024 * 1024 * 1024));
    const cpuCores = parseFloat(formData.value.cpuCores);

    // Convert env vars array to map
    const envVarsMap: Record<string, string> = {};
    
    // Extract server_version separately (it's not an env var)
    let serverVersion: string | undefined = undefined;
    
    // Add game-specific settings
    gameSpecificSettings.value.forEach((setting) => {
      if (setting.key === "server_version") {
        // Handle server_version separately
        const versionValue = setting.value.trim();
        serverVersion = versionValue || undefined;
      } else if (setting.value.trim()) {
        envVarsMap[setting.key] = setting.value.trim();
      }
    });
    
    // Add manual environment variables
    envVars.value.forEach(({ key, value }) => {
      if (key.trim()) {
        envVarsMap[key.trim()] = value.trim();
      }
    });

    await client.updateGameServer({
      gameServerId: props.gameServer.id,
      name: formData.value.name,
      description: formData.value.description || undefined,
      memoryBytes,
      cpuCores,
      startCommand: formData.value.startCommand || undefined,
      envVars: envVarsMap,
      serverVersion,
    });

    toast.success("Game server settings updated successfully");
    emit("saved");
  } catch (error: any) {
    console.error("Failed to update game server:", error);
    toast.error(error?.message || "Failed to update game server settings");
  } finally {
    isSaving.value = false;
  }
};
</script>

