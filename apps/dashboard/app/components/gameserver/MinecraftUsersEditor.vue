<template>
    <OuiTabs v-model="activeSubTab" :tabs="subTabs" />
      <MinecraftFileEditor
        v-if="activeSubTab === 'whitelist'"
        :key="`whitelist-${gameServerId}`"
        :game-server-id="gameServerId"
        file-path="whitelist.json"
        :editor-component="MinecraftWhitelistEditor"
        :editor-props="{
          fileType: 'whitelist'
        }"
      />
      <MinecraftFileEditor
        v-else-if="activeSubTab === 'ops'"
        :key="`ops-${gameServerId}`"
        :game-server-id="gameServerId"
        file-path="ops.json"
        :editor-component="MinecraftWhitelistEditor"
        :editor-props="{
          title: 'Operators',
          description: 'Manage server operators with administrative privileges',
          emptyMessage: 'No operators configured. Add players to grant operator privileges.',
          fileType: 'ops'
        }"
      />
      <MinecraftFileEditor
        v-else-if="activeSubTab === 'banned-players'"
        :key="`banned-players-${gameServerId}`"
        :game-server-id="gameServerId"
        file-path="banned-players.json"
        :editor-component="MinecraftBannedPlayersEditor"
      />
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { UserGroupIcon, ShieldCheckIcon, UserMinusIcon } from "@heroicons/vue/24/outline";
import type { TabItem } from "~/components/oui/Tabs.vue";
import MinecraftWhitelistEditor from "./MinecraftWhitelistEditor.vue";
import MinecraftBannedPlayersEditor from "./MinecraftBannedPlayersEditor.vue";
import MinecraftFileEditor from "./MinecraftFileEditor.vue";

interface Props {
  gameServerId: string;
}

const props = defineProps<Props>();

const activeSubTab = ref("whitelist");

const subTabs = computed<TabItem[]>(() => [
  { id: "whitelist", label: "Whitelist", icon: UserGroupIcon },
  { id: "ops", label: "Operators", icon: ShieldCheckIcon },
  { id: "banned-players", label: "Banned Players", icon: UserMinusIcon },
]);
</script>

