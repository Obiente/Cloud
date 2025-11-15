import { ref } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";

interface MinecraftPlayer {
  uuid: string;
  name?: string;
  avatarUrl?: string;
  loaded: boolean;
}

const playerCache = new Map<string, MinecraftPlayer>();

/**
 * Fetches Minecraft player data via ConnectRPC (proxies Mojang API to avoid CORS)
 */
export function useMinecraftPlayer() {
  const client = useConnectClient(GameServerService);

  /**
   * Get player UUID from username
   */
  async function getUUIDFromUsername(username: string): Promise<string | null> {
    try {
      const response = await client.getMinecraftPlayerUUID({
        username,
      });

      if (!response.uuid) {
        return null;
      }

      return response.uuid;
    } catch (error) {
      console.error("[MinecraftPlayer] Failed to get UUID:", error);
      return null;
    }
  }

  /**
   * Get player profile (name, avatar) from UUID
   */
  async function getPlayerProfile(uuid: string): Promise<MinecraftPlayer | null> {
    // Check cache first
    if (playerCache.has(uuid)) {
      return playerCache.get(uuid)!;
    }

    try {
      const response = await client.getMinecraftPlayerProfile({
        uuid,
      });

      if (!response.uuid) {
        // If not found, return basic info
        const player: MinecraftPlayer = {
          uuid,
          loaded: false,
        };
        playerCache.set(uuid, player);
        return player;
      }

      const player: MinecraftPlayer = {
        uuid: response.uuid,
        name: response.name || undefined,
        avatarUrl: response.avatarUrl || undefined,
        loaded: true,
      };

      playerCache.set(uuid, player);
      return player;
    } catch (error) {
      console.error("[MinecraftPlayer] Failed to get profile:", error);
      const player: MinecraftPlayer = {
        uuid,
        loaded: false,
      };
      playerCache.set(uuid, player);
      return player;
    }
  }

  /**
   * Get player data (name and avatar) from UUID or username
   */
  async function getPlayerData(identifier: string): Promise<MinecraftPlayer | null> {
    // Check if it's a UUID (with or without dashes)
    const uuidRegex = /^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$/i;
    const isUUID = uuidRegex.test(identifier);

    if (isUUID) {
      // Ensure UUID has dashes for consistency
      const uuidWithDashes = identifier.replace(/-/g, "").replace(
        /^([0-9a-f]{8})([0-9a-f]{4})([0-9a-f]{4})([0-9a-f]{4})([0-9a-f]{12})$/i,
        "$1-$2-$3-$4-$5"
      );
      return getPlayerProfile(uuidWithDashes);
    } else {
      // It's a username, get UUID first
      const uuid = await getUUIDFromUsername(identifier);
      if (!uuid) {
        return null;
      }
      return getPlayerProfile(uuid);
    }
  }

  /**
   * Batch load multiple players
   */
  async function loadPlayers(identifiers: string[]): Promise<Map<string, MinecraftPlayer>> {
    const results = new Map<string, MinecraftPlayer>();

    // Load in parallel (with rate limiting)
    const batchSize = 5;
    for (let i = 0; i < identifiers.length; i += batchSize) {
      const batch = identifiers.slice(i, i + batchSize);
      const promises = batch.map(async (id) => {
        const player = await getPlayerData(id);
        if (player) {
          results.set(id, player);
        }
      });

      await Promise.all(promises);

      // Small delay between batches to avoid rate limiting
      if (i + batchSize < identifiers.length) {
        await new Promise((resolve) => setTimeout(resolve, 200));
      }
    }

    return results;
  }

  /**
   * Clear player cache
   */
  function clearCache() {
    playerCache.clear();
  }

  return {
    getPlayerData,
    getPlayerProfile,
    getUUIDFromUsername,
    loadPlayers,
    clearCache,
  };
}

