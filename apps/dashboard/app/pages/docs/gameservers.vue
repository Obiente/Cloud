<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText as="h1" size="3xl" weight="bold" color="primary">
        Game Server Management
      </OuiText>
      <OuiText color="secondary" size="lg">
        Deploy and manage game servers for your multiplayer games with pay-as-you-go pricing
      </OuiText>
    </OuiStack>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Overview</OuiText>
        <OuiText size="sm" color="secondary">
          Game Servers provide dedicated infrastructure for hosting multiplayer game sessions
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Game Servers are dedicated Docker containers optimized for hosting multiplayer game sessions. 
            They feature pay-as-you-go pricing where you only pay when servers are running, making them 
            cost-effective compared to traditional game hosting providers that charge full price even when servers are idle.
          </OuiText>

          <OuiBox p="md" rounded="lg" class="bg-success/10 border border-success/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">
                Creating a New Game Server
              </OuiText>
              <OuiStack gap="xs" class="pl-4">
                <OuiText size="sm" color="secondary">
                  1. Navigate to Game Servers in the sidebar and click "New Game Server"
                </OuiText>
                <OuiText size="sm" color="secondary">
                  2. Enter a name and select a game type (Minecraft, Valheim, Rust, CS2, etc.)
                </OuiText>
                <OuiText size="sm" color="secondary">
                  3. Configure resources:
                </OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="sm" color="secondary">• Memory (default: 2GB)</OuiText>
                  <OuiText size="sm" color="secondary">• CPU cores (default: 1 core)</OuiText>
                  <OuiText size="sm" color="secondary">• Port (auto-assigned if not specified)</OuiText>
                </OuiStack>
                <OuiText size="sm" color="secondary">
                  4. Optionally configure:
                </OuiText>
                <OuiStack gap="xs" class="pl-4">
                  <OuiText size="sm" color="secondary">• Custom Docker image</OuiText>
                  <OuiText size="sm" color="secondary">• Custom start command</OuiText>
                  <OuiText size="sm" color="secondary">• Environment variables</OuiText>
                  <OuiText size="sm" color="secondary">• Server version (for versioned games like Minecraft)</OuiText>
                </OuiStack>
                <OuiText size="sm" color="secondary">
                  5. The container is created immediately, then start the server when ready
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>

          <OuiAccordion
            :items="gameServerAccordionItems"
            multiple
          >
            <template #trigger="{ item }">
              <OuiFlex align="center" gap="sm">
                <component v-if="item.icon" :is="item.icon" class="h-4 w-4" />
                <span>{{ item.label }}</span>
              </OuiFlex>
            </template>
            <template #content="{ item }">
              <div v-html="item.content"></div>
            </template>
          </OuiAccordion>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Supported Game Types</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiText size="sm" weight="semibold" color="primary">Minecraft</OuiText>
            <OuiStack gap="xs" class="mt-1">
              <OuiText size="xs" color="secondary">• Minecraft Java Edition</OuiText>
              <OuiText size="xs" color="secondary">• Minecraft Bedrock Edition</OuiText>
              <OuiText size="xs" color="secondary">• Supports version selection</OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiText size="sm" weight="semibold" color="primary">Survival Games</OuiText>
            <OuiStack gap="xs" class="mt-1">
              <OuiText size="xs" color="secondary">• Valheim</OuiText>
              <OuiText size="xs" color="secondary">• Rust</OuiText>
              <OuiText size="xs" color="secondary">• ARK: Survival Evolved</OuiText>
              <OuiText size="xs" color="secondary">• 7 Days to Die</OuiText>
              <OuiText size="xs" color="secondary">• Conan Exiles</OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiText size="sm" weight="semibold" color="primary">FPS Games</OuiText>
            <OuiStack gap="xs" class="mt-1">
              <OuiText size="xs" color="secondary">• Counter-Strike 2 (CS2)</OuiText>
              <OuiText size="xs" color="secondary">• Team Fortress 2 (TF2)</OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiText size="sm" weight="semibold" color="primary">Other Games</OuiText>
            <OuiStack gap="xs" class="mt-1">
              <OuiText size="xs" color="secondary">• Terraria</OuiText>
              <OuiText size="xs" color="secondary">• Factorio</OuiText>
              <OuiText size="xs" color="secondary">• Space Engineers</OuiText>
              <OuiText size="xs" color="secondary">• Custom Docker images</OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Game Server Features</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">File Management</OuiText>
              <OuiText size="sm" color="secondary">
                Browse, upload, edit, and delete files directly through the dashboard. Manage 
                server configuration files, mods, plugins, and world data without SSH access.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Interactive Terminal</OuiText>
              <OuiText size="sm" color="secondary">
                Access a full terminal session directly in your browser. Send console commands, 
                view real-time output, and manage your server interactively. If the server is stopped, 
                type "start" to start it.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Real-time Metrics</OuiText>
              <OuiText size="sm" color="secondary">
                Monitor CPU usage percentage, memory usage (current and limit), network traffic, 
                disk I/O, and player count in real-time. View historical metrics for analysis.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Logs</OuiText>
              <OuiText size="sm" color="secondary">
                View server logs with real-time streaming. Logs include console output, player 
                activity, errors, and other server events. Logs support ANSI color codes.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Console Commands</OuiText>
              <OuiText size="sm" color="secondary">
                Send commands to running game servers directly from the dashboard. Execute 
                server commands, manage players, and configure server settings remotely.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Player Tracking</OuiText>
              <OuiText size="sm" color="secondary">
                Track player count and max players for supported game types. Monitor server 
                population and activity metrics for capacity planning.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Persistent Storage</OuiText>
              <OuiText size="sm" color="secondary">
                Game server data is stored in persistent volumes. World data, configurations, 
                and mods persist across server restarts and updates.
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Usage Tracking</OuiText>
              <OuiText size="sm" color="secondary">
                Track CPU core-seconds, memory byte-seconds, bandwidth, and storage usage. 
                View usage metrics and estimated costs for billing purposes.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Game Server Statuses</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-secondary" />
              <OuiText size="sm" weight="medium" color="primary">CREATED</OuiText>
              <OuiText size="sm" color="secondary">• Server created but not started</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-warning" />
              <OuiText size="sm" weight="medium" color="primary">STARTING</OuiText>
              <OuiText size="sm" color="secondary">• Server container is starting</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-success" />
              <OuiText size="sm" weight="medium" color="primary">RUNNING</OuiText>
              <OuiText size="sm" color="secondary">• Server is running and accepting players</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-warning" />
              <OuiText size="sm" weight="medium" color="primary">STOPPING</OuiText>
              <OuiText size="sm" color="secondary">• Server is shutting down</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-secondary" />
              <OuiText size="sm" weight="medium" color="primary">STOPPED</OuiText>
              <OuiText size="sm" color="secondary">• Server has been stopped</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-warning" />
              <OuiText size="sm" weight="medium" color="primary">RESTARTING</OuiText>
              <OuiText size="sm" color="secondary">• Server is restarting</OuiText>
            </OuiFlex>
          </OuiBox>
          <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <OuiBox class="w-2 h-2 rounded-full bg-danger" />
              <OuiText size="sm" weight="medium" color="primary">FAILED</OuiText>
              <OuiText size="sm" color="secondary">• Server failed to start or crashed</OuiText>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>
        <OuiText as="h2" class="oui-card-title">Pricing Benefits</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiText>
            Game servers use pay-as-you-go pricing, meaning you only pay when servers are running. 
            This is significantly more cost-effective than traditional game hosting providers.
          </OuiText>
          <OuiBox p="md" rounded="lg" class="bg-primary/10 border border-primary/20">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold" color="primary">Cost Savings Example</OuiText>
              <OuiText size="sm" color="secondary">
                Traditional hosting providers charge $20-40/month even when your server is empty. 
                With Obiente Cloud, if your server runs 8 hours per day (240 hours/month), you pay 
                only for those 240 hours instead of 730 hours (24/7). This can save you 60-70% on 
                hosting costs.
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  CubeIcon,
  CommandLineIcon,
  ChartBarIcon,
  DocumentTextIcon,
} from "@heroicons/vue/24/outline";
import type { Component } from "vue";

definePageMeta({
  layout: "docs",
});

const gameServerAccordionItems = [
  {
    value: "resource-config",
    label: "Resource Configuration",
    icon: CubeIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Configure resources for your game server:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Memory:</strong> Configure RAM allocation (default: 2GB). More memory allows for larger worlds and more players.</li>
        <li><strong>CPU Cores:</strong> Set CPU cores (default: 1). More cores improve performance for CPU-intensive games.</li>
        <li><strong>Port:</strong> Game server port (auto-assigned starting from 25565 if not specified). Ports are automatically managed.</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Resources can be updated after creation, but the server will need to be restarted for changes to take effect.</p>
    `,
  },
  {
    value: "custom-config",
    label: "Custom Configuration",
    icon: CubeIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Customize your game server:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Custom Docker Image:</strong> Use a custom Docker image instead of the default for your game type</li>
        <li><strong>Custom Start Command:</strong> Override the default start command for your game server</li>
        <li><strong>Environment Variables:</strong> Set environment variables for server configuration</li>
        <li><strong>Server Version:</strong> Specify a server version for versioned games (e.g., Minecraft 1.20.1)</li>
        <li><strong>Description:</strong> Add a description to help identify your server</li>
      </ul>
    `,
  },
  {
    value: "file-management",
    label: "File Management",
    icon: DocumentTextIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Manage server files directly from the dashboard:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Browse files and directories in your server's persistent volume</li>
        <li>Upload files (configs, mods, plugins, world data)</li>
        <li>Edit files directly in the browser with syntax highlighting</li>
        <li>Delete files and directories</li>
        <li>View file permissions, ownership, and metadata</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Files are stored in persistent volumes that survive container restarts and updates.</p>
    `,
  },
  {
    value: "terminal-access",
    label: "Terminal Access",
    icon: CommandLineIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Interactive terminal access:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li>Access a full terminal session in your browser</li>
        <li>Send console commands directly to the game server</li>
        <li>View real-time output with ANSI color support</li>
        <li>If the server is stopped, type "start" in the terminal to start it</li>
        <li>Terminal supports resize commands for proper display</li>
      </ul>
      <p class="text-sm text-secondary mt-3">The terminal connects directly to the container, giving you full control over your server.</p>
    `,
  },
  {
    value: "monitoring",
    label: "Monitoring & Metrics",
    icon: ChartBarIcon as Component,
    content: `
      <p class="text-sm text-secondary mb-3">Comprehensive monitoring for game servers:</p>
      <ul class="list-disc list-inside space-y-1 text-sm text-secondary">
        <li><strong>Real-time Metrics:</strong> CPU usage percentage, memory usage/limit, network RX/TX, disk I/O</li>
        <li><strong>Player Metrics:</strong> Current player count and max players (for supported games)</li>
        <li><strong>Historical Metrics:</strong> View metrics over time to track performance trends</li>
        <li><strong>Usage Metrics:</strong> CPU core-seconds, memory byte-seconds, bandwidth, storage for billing</li>
        <li><strong>Logs:</strong> Real-time streaming logs with ANSI color support</li>
      </ul>
      <p class="text-sm text-secondary mt-3">Metrics are streamed in real-time and stored for historical analysis and billing.</p>
    `,
  },
];
</script>
