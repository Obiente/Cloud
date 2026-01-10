<template>
  <div class="preview-providers-wrapper h-full">
    <component 
      :is="component" 
      v-if="component && isReady" 
      :key="componentKey"
      @error="handleComponentError" 
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onErrorCaptured, provide, reactive, ref, watchEffect, nextTick } from "vue";
import { routerKey, routeLocationKey, type RouteLocationNormalizedLoaded, createRouter, createMemoryHistory } from "vue-router";
import { useNuxtApp } from "#app";
import { useOrganizationsStore } from "~/stores/organizations";
import type { 
  Deployment, 
  GameServer, 
  VPSInstance,
  GameServerUsageMetrics,
  VPSUsageMetrics,
  DeploymentUsageMetrics,
  Build,
} from "@obiente/proto";
import { BuildSchema, ListBuildsResponseSchema } from "@obiente/proto";
import {
  DeploymentStatus,
  BuildStatus,
  BuildStrategy,
  DeploymentType,
  Environment,
  GameServerStatus,
  GameType,
  VPSStatus,
} from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { timestamp } from "@obiente/proto/utils";
import { PREVIEW_CONNECT_KEY } from "~/lib/connect-client";

type PreviewMode = "deployment" | "game" | "vps";

interface LiveMetricsState {
  isStreaming: boolean;
  latestMetric: any;
  currentCpuUsage: number;
  currentMemoryUsage: number;
  currentNetworkRx: number;
  currentNetworkTx: number;
}

interface Props {
  component?: any;
  mode: PreviewMode;
  deploymentMock?: Partial<Deployment>;
  gameServerMock?: Partial<GameServer>;
  vpsMock?: Partial<VPSInstance>;
  gameUsageMock?: Partial<GameServerUsageMetrics>;
  gameMetricsMock?: LiveMetricsState;
  vpsMetricsMock?: LiveMetricsState;
}

const props = defineProps<Props>();

// Flag to control component rendering during cleanup
const isReady = ref(false);

// Computed key to force component recreation when props change
const componentKey = computed(() => {
  const id = props.mode === "deployment"
    ? props.deploymentMock?.id || "mock-deployment"
    : props.mode === "game"
    ? props.gameServerMock?.id || "mock-game-server"
    : props.vpsMock?.id || "mock-vps";
  return `${props.mode}-${id}-${Date.now()}`;
});

console.log('[PreviewProviders] Component initializing', {
  mode: props.mode,
  hasComponent: !!props.component,
  hasDeploymentMock: !!props.deploymentMock,
  hasGameServerMock: !!props.gameServerMock,
  hasVpsMock: !!props.vpsMock,
  componentKey: componentKey.value,
});

// Store original transport and WebSocket BEFORE setup
const originalTransport = (globalThis as { __OBIENTE_PREVIEW_CONNECT__?: unknown }).__OBIENTE_PREVIEW_CONNECT__ ?? null;
const originalWebSocket = globalThis.WebSocket;

console.log('[PreviewProviders] Stored originals', {
  hasOriginalTransport: !!originalTransport,
  hasOriginalWebSocket: !!originalWebSocket,
});

// Don't log the entire props object - it contains BigInt values that can't be stringified
const defaultId =
  props.mode === "deployment"
    ? props.deploymentMock?.id || "mock-deployment"
    : props.mode === "game"
    ? props.gameServerMock?.id || "mock-game-server"
    : props.vpsMock?.id || "mock-vps";

console.log('[PreviewProviders] Default ID calculated:', defaultId);

const pathForMode =
  props.mode === "deployment"
    ? `/deployments/${defaultId}`
    : props.mode === "game"
    ? `/gameservers/${defaultId}`
    : `/vps/${defaultId}`;

const route: RouteLocationNormalizedLoaded = reactive({
  path: pathForMode,
  name: "preview",
  params: {
    id: defaultId,
    buildId: "mock-build",
  },
  query: {},
  hash: "",
  fullPath: pathForMode,
  matched: [],
  redirectedFrom: undefined,
  meta: {},
} as RouteLocationNormalizedLoaded);

console.log('[PreviewProviders] Creating router for path:', pathForMode);

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: "/", name: "root", component: { template: "<div />" } },
    { path: "/deployments/:id", name: "deployment-id", component: { template: "<div />" } },
    { path: "/gameservers/:id", name: "gameserver-id", component: { template: "<div />" } },
    { path: "/vps/:id", name: "vps-id", component: { template: "<div />" } },
  ],
});

try {
  router.push(pathForMode);
  console.log('[PreviewProviders] Router pushed to:', pathForMode);
} catch (error) {
  console.error('[PreviewProviders] Error pushing router:', error);
}

provide(routeLocationKey, route);
provide(routerKey, router);
console.log('[PreviewProviders] Provided route and router contexts');

// Watch for prop changes and update route accordingly
let lastMode = props.mode;
let lastId = defaultId;

watchEffect(() => {
  try {
    const newId = props.mode === "deployment"
      ? props.deploymentMock?.id || "mock-deployment"
      : props.mode === "game"
      ? props.gameServerMock?.id || "mock-game-server"
      : props.vpsMock?.id || "mock-vps";
    
    const newPath = props.mode === "deployment"
      ? `/deployments/${newId}`
      : props.mode === "game"
      ? `/gameservers/${newId}`
      : `/vps/${newId}`;
    
    // If mode or ID changed, force component recreation
    if (props.mode !== lastMode || newId !== lastId) {
      console.log('[PreviewProviders] Mode/ID change detected - forcing component recreation', {
        oldMode: lastMode,
        newMode: props.mode,
        oldId: lastId,
        newId,
      });
      
      // Hide component briefly to force unmount
      isReady.value = false;
      
      lastMode = props.mode;
      lastId = newId;
      
      // Update route
      route.path = newPath;
      route.fullPath = newPath;
      route.params.id = newId;
      router.push(newPath);
      
      // Show component again after brief delay
      nextTick(() => {
        isReady.value = true;
        console.log('[PreviewProviders] Component recreated and ready');
      });
    } else if (route.path !== newPath) {
      console.log('[PreviewProviders] Route change detected', {
        oldPath: route.path,
        newPath,
      });
      route.path = newPath;
      route.fullPath = newPath;
      route.params.id = newId;
      router.push(newPath);
    }
  } catch (error) {
    console.error('[PreviewProviders] Error in watchEffect:', error);
  }
});

const orgStore = useOrganizationsStore();
if (!orgStore.currentOrgId) {
  orgStore.currentOrgId = "mock-org";
  console.log('[PreviewProviders] Set preview org ID');
} else {
  console.log('[PreviewProviders] Org ID already set:', orgStore.currentOrgId);
}

// Provide a mocked Connect transport so pages resolve data without network calls.
const nuxtApp = useNuxtApp();
console.log('[PreviewProviders] Got Nuxt app instance');

const emptyHeaders = () => new Headers();

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

// Helper to generate random timestamp in the past (between 1 and 30 days ago)
const randomPastTimestamp = () => {
  const daysAgo = 1 + Math.random() * 29; // 1-30 days ago
  const pastTime = Date.now() - (daysAgo * 24 * 60 * 60 * 1000);
  return timestamp(new Date(pastTime));
};

interface StreamOptions {
  signal?: AbortSignal;
  onHeader?: (h: Headers) => void;
  onTrailer?: (h: Headers) => void;
}

const makeStreamResponse = (
  gen: AsyncGenerator<any>,
  options?: StreamOptions
): AsyncIterableIterator<any> & { cancel: () => void; close: () => void } => {
  const iterator: AsyncIterator<any> = gen[Symbol.asyncIterator]
    ? gen[Symbol.asyncIterator]()
    : (gen as AsyncIterator<any>);

  const callOnTrailer = () => {
    if (options?.onTrailer) {
      options.onTrailer(emptyHeaders());
    }
  };

  let headerSent = false;

  const streamIter: AsyncIterableIterator<any> & { cancel: () => void; close: () => void } = {
    [Symbol.asyncIterator]() {
      return this;
    },
    async next() {
      if (!headerSent && options?.onHeader) {
        options.onHeader(emptyHeaders());
        headerSent = true;
      }
      const res = await iterator.next();
      if (res.done) {
        callOnTrailer();
      }
      const value =
        res && typeof res.value === "object" && res.value && "message" in (res.value as Record<string, unknown>)
          ? (res.value as { message: unknown }).message
          : res.value;
      
      return { done: res.done, value };
    },
    async return(value?: unknown) {
      callOnTrailer();
      if (typeof iterator.return === "function") {
        return iterator.return(value);
      }
      return { done: true, value: undefined };
    },
    cancel() {
      const signalLike = options?.signal as unknown;
      if (
        signalLike &&
        typeof signalLike === "object" &&
        "abort" in signalLike &&
        typeof (signalLike as { abort: () => void }).abort === "function"
      ) {
        (signalLike as { abort: () => void }).abort();
      }
      callOnTrailer();
    },
    close() {
      callOnTrailer();
    },
  };

  return streamIter;
};

// Helper to add logs to buffer (used by both unary and stream)
const addLog = (type: 'deployment' | 'gameserver' | 'vps' | 'build', message: string, stderr = false) => {
  if (!mockTransport._logBuffers) {
    mockTransport._logBuffers = {
      deployment: [],
      gameserver: [],
      vps: [],
      build: [],
    };
  }
  
  const log = type === 'deployment' || type === 'build'
    ? { line: message, timestamp: timestamp(new Date()), stderr, logLevel: 3 }
    : { line: message, timestamp: timestamp(new Date()), level: 3 };
  
  mockTransport._logBuffers[type].push(log);
  
  // Limit buffer size
  if (mockTransport._logBuffers[type].length > 200) {
    mockTransport._logBuffers[type].shift();
  }
  
  return log;
};

const mockTransport: any = {
  unary: async (_service: any, method: any, _signal?: AbortSignal, _timeout?: number, header?: HeadersInit | unknown, input: any = {}) => {
    const methodName = String(_service?.name || method?.name || method?.localName || "").toLowerCase();
    console.log('[MockTransport.unary] Called', {
      methodName,
      hasInput: !!input,
      service: _service?.name,
    });

    try {
      const isHeadersInit = (value: unknown): value is HeadersInit => {
        if (!value) return false;
        if (value instanceof Headers) return true;
        if (typeof value === "string") return true;
        if (Array.isArray(value)) return true;
        return typeof value === "object" && "append" in (value as Record<string, unknown>);
      };

      if (!isHeadersInit(header) && header !== undefined && input !== undefined) {
        // Shift when header param is actually input
        input = header as unknown as Record<string, unknown>;
        header = undefined;
      }
      const name = methodName;

    // Helper to return mock responses
    const mockResponse = (message: any) => {
      try {
        const response = {
          message,
          header: emptyHeaders(),
          trailer: emptyHeaders()
        };
        console.log('[PreviewTransport.unary] Returning response for:', name);
        return response;
      } catch (error) {
        console.error('[PreviewTransport.unary] Error creating response:', error);
        throw error;
      }
    };

    const deploymentDefaults: Partial<Deployment> = {
      id: "mock-web-deployment",
      name: "Preview Deployment",
      domain: "preview.obiente.cloud",
      status: DeploymentStatus.RUNNING,
      healthStatus: "HEALTHY",
      lastDeployedAt: timestamp(new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000)),
      environment: Environment.PRODUCTION,
      buildTime: 120,
      port: 8080,
      image: "ghcr.io/preview/app:latest",
      createdAt: randomPastTimestamp(),
      repositoryUrl: "https://github.com/mock/repo",
      branch: "main",
      type: DeploymentType.NODE,
      buildStrategy: BuildStrategy.RAILPACK,
      customDomains: [],
      groups: [],
      storageUsage: BigInt(0),
      bandwidthUsage: BigInt(0),
    };

    // Initialize persistent resource storage
    if (!mockTransport._resources) {
      mockTransport._resources = {};
    }

    // Initialize or get deployment
    if (!mockTransport._resources.deployment) {
      mockTransport._resources.deployment = props.deploymentMock
        ? { ...deploymentDefaults, ...props.deploymentMock }
        : { ...deploymentDefaults };
    }
    const deployment = mockTransport._resources.deployment;

    const gameServerDefaults: Partial<GameServer> = {
      id: "mock-game-server",
      name: "Preview Game Server",
      status: GameServerStatus.RUNNING,
      port: 25565,
      cpuCores: 2,
      memoryBytes: BigInt(2 * 1024 * 1024 * 1024),
      organizationId: "mock-org",
      gameType: GameType.MINECRAFT_JAVA,
      createdAt: randomPastTimestamp(),
      updatedAt: timestamp(new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000)),
    };

    // Initialize or get game server
    if (!mockTransport._resources.gameServer) {
      mockTransport._resources.gameServer = props.gameServerMock
        ? { ...gameServerDefaults, ...props.gameServerMock }
        : { ...gameServerDefaults };
    }
    const gameServer = mockTransport._resources.gameServer;

    const vpsDefaults: Partial<VPSInstance> = {
      id: "mock-vps",
      name: "Preview VPS",
      description: "Preview-only instance",
      status: VPSStatus.RUNNING,
      region: "us-east-1",
      image: 0,
      size: "small",
      cpuCores: 1,
      memoryBytes: BigInt(2 * 1024 * 1024 * 1024),
      diskBytes: BigInt(20 * 1024 * 1024 * 1024),
      organizationId: "mock-org",
      ipv4Addresses: ["10.15.3.100"],
      ipv6Addresses: [],
      createdAt: randomPastTimestamp(),
      updatedAt: timestamp(new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000)),
    };

    // Initialize or get VPS
    if (!mockTransport._resources.vps) {
      mockTransport._resources.vps = props.vpsMock 
        ? { ...vpsDefaults, ...props.vpsMock } 
        : { ...vpsDefaults };
    }
    const vps = mockTransport._resources.vps;

    const sampleSSHKey = {
      id: "ssh-key-1",
      name: "Laptop Key",
      fingerprint: "SHA256:preview",
      createdAt: randomPastTimestamp(),
      updatedAt: timestamp(new Date(Date.now() - Math.random() * 24 * 60 * 60 * 1000)),
      key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIPREVIEWKEY",
    } as const;

    const sampleFirewallRule = {
      id: "fw-rule-1",
      pos: 1,
      action: 1,
      direction: 1,
      source: "0.0.0.0/0",
      dest: "",
      protocol: 1,
      dport: "22",
      iface: "vmbr0",
      comment: "Allow SSH",
      enable: true,
    } as const;

    const sampleFirewallOptions = {
      enable: true,
      policyIn: "ACCEPT",
      policyOut: "ACCEPT",
    } as const;

    // Type-safe build samples using protobuf Build type
    const createBuild = (data: Partial<Build> & { id: string; buildNumber: number; status: BuildStatus }): Build => create(BuildSchema, {
      id: data.id,
      deploymentId: "mock-web-deployment",
      organizationId: "mock-org",
      buildNumber: data.buildNumber,
      status: data.status,
      startedAt: data.startedAt || timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
      completedAt: data.completedAt,
      buildTime: data.buildTime || 0,
      triggeredBy: data.triggeredBy || "preview-user",
      repositoryUrl: data.repositoryUrl,
      branch: data.branch || "main",
      commitSha: data.commitSha,
      buildCommand: data.buildCommand,
      installCommand: data.installCommand,
      startCommand: data.startCommand,
      dockerfilePath: data.dockerfilePath,
      composeFilePath: data.composeFilePath,
      buildStrategy: data.buildStrategy || BuildStrategy.RAILPACK,
      imageName: data.imageName,
      composeYaml: data.composeYaml,
      size: data.size,
      error: data.error,
      createdAt: data.createdAt || timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
      updatedAt: data.updatedAt || timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
    });

    const buildSamples: Build[] = [
      createBuild({
        id: "build-1",
        buildNumber: 12,
        status: BuildStatus.BUILD_SUCCESS,
        branch: "main",
        commitSha: "a7f3c2e",
        buildTime: 95,
        size: "120 MB",
        repositoryUrl: "https://github.com/demo/app",
        buildCommand: "pnpm build",
        imageName: "obiente/demo-app:latest",
        buildStrategy: BuildStrategy.RAILPACK,
        startedAt: timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
        completedAt: timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000 + 95 * 1000)),
        createdAt: timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
        updatedAt: timestamp(new Date(Date.now() - 2 * 60 * 60 * 1000)),
      }),
      createBuild({
        id: "build-2",
        buildNumber: 11,
        status: BuildStatus.BUILD_SUCCESS,
        branch: "main",
        commitSha: "b8e4d3f",
        buildTime: 87,
        size: "118 MB",
        repositoryUrl: "https://github.com/demo/app",
        buildCommand: "pnpm build",
        imageName: "obiente/demo-app:v11",
        buildStrategy: BuildStrategy.RAILPACK,
        startedAt: timestamp(new Date(Date.now() - 6 * 60 * 60 * 1000)),
        completedAt: timestamp(new Date(Date.now() - 6 * 60 * 60 * 1000 + 87 * 1000)),
        createdAt: timestamp(new Date(Date.now() - 6 * 60 * 60 * 1000)),
        updatedAt: timestamp(new Date(Date.now() - 6 * 60 * 60 * 1000)),
      }),
      createBuild({
        id: "build-3",
        buildNumber: 10,
        status: BuildStatus.BUILD_FAILED,
        branch: "feature/api-updates",
        commitSha: "c9f5e4a",
        buildTime: 45,
        size: "0 MB",
        repositoryUrl: "https://github.com/demo/app",
        buildCommand: "pnpm build",
        imageName: "",
        buildStrategy: BuildStrategy.RAILPACK,
        error: "Build failed: TypeScript compilation errors",
        startedAt: timestamp(new Date(Date.now() - 12 * 60 * 60 * 1000)),
        completedAt: timestamp(new Date(Date.now() - 12 * 60 * 60 * 1000 + 45 * 1000)),
        createdAt: timestamp(new Date(Date.now() - 12 * 60 * 60 * 1000)),
        updatedAt: timestamp(new Date(Date.now() - 12 * 60 * 60 * 1000)),
      }),
      createBuild({
        id: "build-4",
        buildNumber: 9,
        status: BuildStatus.BUILD_SUCCESS,
        branch: "main",
        commitSha: "d1a6b7c",
        buildTime: 102,
        size: "115 MB",
        repositoryUrl: "https://github.com/demo/app",
        buildCommand: "pnpm build",
        imageName: "obiente/demo-app:v9",
        buildStrategy: BuildStrategy.RAILPACK,
        startedAt: timestamp(new Date(Date.now() - 24 * 60 * 60 * 1000)),
        completedAt: timestamp(new Date(Date.now() - 24 * 60 * 60 * 1000 + 102 * 1000)),
        createdAt: timestamp(new Date(Date.now() - 24 * 60 * 60 * 1000)),
        updatedAt: timestamp(new Date(Date.now() - 24 * 60 * 60 * 1000)),
      }),
      createBuild({
        id: "build-5",
        buildNumber: 8,
        status: BuildStatus.BUILD_SUCCESS,
        branch: "main",
        commitSha: "e2b7c8d",
        buildTime: 110,
        size: "125 MB",
        repositoryUrl: "https://github.com/demo/app",
        buildCommand: "pnpm build",
        imageName: "obiente/demo-app:v8",
        buildStrategy: BuildStrategy.RAILPACK,
        startedAt: timestamp(new Date(Date.now() - 3 * 24 * 60 * 60 * 1000)),
        completedAt: timestamp(new Date(Date.now() - 3 * 24 * 60 * 60 * 1000 + 110 * 1000)),
        createdAt: timestamp(new Date(Date.now() - 3 * 24 * 60 * 60 * 1000)),
        updatedAt: timestamp(new Date(Date.now() - 3 * 24 * 60 * 60 * 1000)),
      }),
    ];

    const makeUsage = () => {
      // Calculate usage based on actual current date
      const now = new Date();
      const daysElapsed = now.getDate(); // Day of month (1-31)
      const currentMonth = now.toLocaleString('en-US', { month: 'long', year: 'numeric' });
      const daysInMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
      
      const hoursElapsed = daysElapsed * 24;
      const secondsElapsed = hoursElapsed * 3600;
      
      // Simulate accumulated usage over the month
      const cpuCoreSeconds = BigInt(Math.floor(hoursElapsed * 3600 * 0.15)); // 15% avg CPU
      const memoryByteSeconds = BigInt(Math.floor(hoursElapsed * 3600 * 768 * 1024 * 1024)); // 768MB avg
      const bandwidthRxBytes = BigInt(Math.floor(hoursElapsed * 85_000)); // 85KB/s avg rx
      const bandwidthTxBytes = BigInt(Math.floor(hoursElapsed * 75_000)); // 75KB/s avg tx
      const diskReadBytes = BigInt(Math.floor(hoursElapsed * 11_000)); // 11KB/s avg read
      const diskWriteBytes = BigInt(Math.floor(hoursElapsed * 9_000)); // 9KB/s avg write
      const storageBytes = BigInt(Math.floor(15 * 1e9)); // 15GB storage
      
      // Cost calculations (example rates)
      // $0.04 per vCPU hour, $0.005 per GB-hour memory, $0.10 per GB bandwidth, $0.05 per GB-month storage
      const cpuCostCents = BigInt(Math.floor(Number(cpuCoreSeconds) / 3600 * 4)); // $0.04/hr
      const memoryCostCents = BigInt(Math.floor(Number(memoryByteSeconds) / (1024 * 1024 * 1024) / 3600 * 0.5)); // $0.005/GB-hr
      const bandwidthCostCents = BigInt(Math.floor(Number(bandwidthRxBytes + bandwidthTxBytes) / (1024 * 1024 * 1024) * 10)); // $0.10/GB
      const storageCostCents = BigInt(Math.floor(Number(storageBytes) / (1024 * 1024 * 1024) * 5)); // $0.05/GB-month
      const estimatedCostCents = cpuCostCents + memoryCostCents + bandwidthCostCents + storageCostCents;
      
      // Project to full month based on days in current month
      const projectionMultiplier = daysInMonth / daysElapsed;
      
      return {
        month: currentMonth,
        current: {
          cpuCoreSeconds,
          memoryByteSeconds,
          bandwidthRxBytes,
          bandwidthTxBytes,
          diskReadBytes,
          diskWriteBytes,
          storageBytes,
          requestCount: BigInt(Math.floor(hoursElapsed * 450)), // ~450 req/hr
          errorCount: BigInt(Math.floor(hoursElapsed * 0.5)), // ~0.5 errors/hr
          uptimeSeconds: BigInt(secondsElapsed),
          estimatedCostCents,
          cpuCostCents,
          memoryCostCents,
          bandwidthCostCents,
          storageCostCents,
        },
        estimatedMonthly: {
          cpuCoreSeconds: BigInt(Math.floor(Number(cpuCoreSeconds) * projectionMultiplier)),
          memoryByteSeconds: BigInt(Math.floor(Number(memoryByteSeconds) * projectionMultiplier)),
          bandwidthRxBytes: BigInt(Math.floor(Number(bandwidthRxBytes) * projectionMultiplier)),
          bandwidthTxBytes: BigInt(Math.floor(Number(bandwidthTxBytes) * projectionMultiplier)),
          diskReadBytes: BigInt(Math.floor(Number(diskReadBytes) * projectionMultiplier)),
          diskWriteBytes: BigInt(Math.floor(Number(diskWriteBytes) * projectionMultiplier)),
          storageBytes,
          requestCount: BigInt(Math.floor(hoursElapsed * 450 * projectionMultiplier)),
          errorCount: BigInt(Math.floor(hoursElapsed * 0.5 * projectionMultiplier)),
          uptimeSeconds: BigInt(daysInMonth * 86400),
          estimatedCostCents: BigInt(Math.floor(Number(estimatedCostCents) * projectionMultiplier)),
          cpuCostCents: BigInt(Math.floor(Number(cpuCostCents) * projectionMultiplier)),
          memoryCostCents: BigInt(Math.floor(Number(memoryCostCents) * projectionMultiplier)),
          bandwidthCostCents: BigInt(Math.floor(Number(bandwidthCostCents) * projectionMultiplier)),
          storageCostCents,
        },
      };
    };

    // Deployment mocks
    if (name.includes("deployment")) {
      console.log('[MockTransport.unary] ✅ Inside deployment block for:', name);
      if ((name.includes("getdeployment") && !name.includes("metrics") && !name.includes("usage") && !name.includes("health") && !name.includes("logs") && !name.includes("services") && !name.includes("routings") && !name.includes("compose")) || name === "deployment") {
        return { message: { deployment }, header: emptyHeaders(), trailer: emptyHeaders() };
      }
      if (name.includes("getdeploymenthealth")) {
        return mockResponse({
          healthStatus: deployment.healthStatus || "HEALTHY",
          healthMessage: deployment.healthMessage || "Healthy",
        });
      }
      if (name.includes("listdeployments")) {
        return mockResponse({
          deployments: [deployment],
          total: 1,
        });
      }
      if (name.includes("listdeploymentcontainers")) {
        return mockResponse({
          containers: [
            { containerId: "mock-container-1", serviceName: "web", status: "running" },
            { containerId: "mock-container-2", serviceName: "worker", status: "running" },
          ],
        });
      }
      if (name.includes("getdeploymentroutings") || name.includes("updatedeploymentroutings")) {
        const domain = deployment.domain || "preview.obiente.cloud";
        return mockResponse({
          rules: [
            {
              id: "mock-rule",
              domain,
              serviceName: "default",
              pathPrefix: "/",
              targetPort: deployment.port || 80,
              protocol: "https",
              sslEnabled: true,
              sslCertResolver: "letsencrypt",
            },
          ],
        });
      }
      if (name.includes("getdeploymentservices")) {
        return mockResponse({ services: [] });
      }
      if (name.includes("getdeploymentlogs") || name.includes("getdeploymentbuildlogs")) {
        return mockResponse({
          logs: [
            { line: "[demo] 🚀 Build started", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[demo] 📦 Cloning repository...", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[demo] 📥 Installing dependencies...", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[demo] 🔨 Running build command", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[demo] ✓ Build completed successfully", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[demo] 🐳 Docker image created", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
          ],
        });
      }
      if (name.includes("getdeploymentmetrics")) {
        console.log('[PreviewTransport.unary] 🎯 METRICS HANDLER REACHED');
        console.log('[PreviewTransport.unary] Request input:', {
          deploymentId: input?.deploymentId,
          organizationId: input?.organizationId,
          serviceName: input?.serviceName,
          containerId: input?.containerId,
          aggregate: input?.aggregate,
          startTime: input?.startTime,
          endTime: input?.endTime,
        });
        
        // Use time range from request or default to last 30 days
        const endTime = input?.endTime?.seconds ? Number(input.endTime.seconds) * 1000 : Date.now();
        const startTime = input?.startTime?.seconds ? Number(input.startTime.seconds) * 1000 : endTime - (30 * 24 * 3600000);
        const durationMs = endTime - startTime;
        const intervalMs = Math.max(60000, Math.floor(durationMs / 720)); // Max 720 points, min 1min interval
        const pointCount = Math.floor(durationMs / intervalMs);
        
        // Base values for realistic metrics
        const baseCpu = 15; // 15% base CPU
        const baseMemory = 768 * 1024 * 1024; // 768MB base memory
        const baseNetworkRx = 85_000; // 85KB/s
        const baseNetworkTx = 75_000; // 75KB/s
        const baseDiskRead = 12_000; // 12KB/s
        const baseDiskWrite = 9_000; // 9KB/s
        
        const metrics = Array.from({ length: pointCount }).map((_, idx) => {
          const progress = idx / pointCount;
          const time = startTime + idx * intervalMs;
          const hourOfDay = new Date(time).getHours();
          
          // Daily pattern: higher usage during business hours (9-17)
          const dailyMultiplier = 0.7 + 0.3 * Math.sin((hourOfDay - 6) * Math.PI / 12);
          
          // Weekly trend: slight increase over time
          const trendMultiplier = 1 + progress * 0.1;
          
          // Small random variation (±8%)
          const randomVariation = () => 0.92 + Math.random() * 0.16;
          
          return {
            timestamp: timestamp(new Date(time)),
            cpuUsagePercent: baseCpu * dailyMultiplier * trendMultiplier * randomVariation(),
            memoryUsageBytes: baseMemory * (0.95 + progress * 0.1) * randomVariation(),
            networkRxBytes: baseNetworkRx * dailyMultiplier * randomVariation(),
            networkTxBytes: baseNetworkTx * dailyMultiplier * randomVariation(),
            diskReadBytes: baseDiskRead * randomVariation(),
            diskWriteBytes: baseDiskWrite * randomVariation(),
          };
        });
        console.log('[PreviewTransport.unary] Generated', metrics.length, 'deployment metric data points for', {
          serviceName: input?.serviceName || 'aggregated',
          timeRange: `${new Date(startTime).toLocaleString()} - ${new Date(endTime).toLocaleString()}`,
          durationHours: (durationMs / 3600000).toFixed(1),
        });
        return mockResponse({ metrics });
      }
      if (name.includes("getdeploymentusage")) {
        return mockResponse(makeUsage());
      }
      if (name.includes("getdeploymentcompose") || name.includes("validatedeploymentcompose")) {
        return mockResponse({
          composeYaml:
            input?.composeYaml ||
            `version: '3.8'\n\nservices:\n  app:\n    image: ${deployment.image || "nginx"}\n    ports:\n      - "${deployment.port || 8080}:${deployment.port || 8080}"\n`,
          validationErrors: [],
          validationError: "",
        });
      }
      if (name.includes("updatedeploymentcompose")) {
        return mockResponse({
          validationErrors: [],
          validationError: "",
          deployment,
        });
      }
      if (name.includes("updatedeployment")) {
        console.log('[PreviewTransport.unary] Updating deployment with:', input);
        // Merge updates into deployment object and persist
        Object.assign(mockTransport._resources.deployment, input);
        return mockResponse({ deployment: mockTransport._resources.deployment, success: true });
      }
      if (name.includes("updatedeploymentenvvars")) {
        console.log('[PreviewTransport.unary] Updating deployment env vars');
        return mockResponse({ success: true });
      }
      if (name.includes("startdeployment") || name.includes("stopdeployment") || name.includes("triggerdeployment") || name.includes("restartdeployment")) {
        const action = name.includes("stop") ? 'stop' : name.includes("start") ? 'start' : name.includes("restart") ? 'restart' : 'trigger';
        console.log(`[MockTransport.unary] Deployment action: ${action}`);
        
        // Update state and add logs
        if (action === 'stop') {
          mockTransport._resourceStates.deployment.status = 'stopping';
          addLog('deployment', '[demo] 🛑 Shutdown signal received');
          addLog('deployment', '[demo] Closing active connections...');
          addLog('deployment', '[demo] ✓ All connections closed');
          addLog('deployment', '[demo] Saving application state...');
          addLog('deployment', '[demo] ✓ State saved successfully');
          addLog('deployment', '[demo] Shutting down gracefully...');
          addLog('deployment', '[demo] ✓ Application stopped');
          setTimeout(() => {
            mockTransport._resourceStates.deployment.status = 'stopped';
          }, 100);
        } else if (action === 'start') {
          mockTransport._resourceStates.deployment.status = 'starting';
          addLog('deployment', '[demo] 🚀 Starting application...');
          addLog('deployment', '[demo] Loading environment variables');
          addLog('deployment', '[demo] Connecting to database at postgres://***');
          addLog('deployment', '[demo] ✓ Database connection established');
          addLog('deployment', '[demo] Connecting to Redis cache at redis://***:6379');
          addLog('deployment', '[demo] ✓ Redis connection successful');
          addLog('deployment', '[demo] Initializing HTTP server on port 3000');
          addLog('deployment', '[demo] ✓ Server listening on http://0.0.0.0:3000');
          addLog('deployment', '[demo] ✅ Application started successfully');
          setTimeout(() => {
            mockTransport._resourceStates.deployment.status = 'running';
          }, 100);
        } else if (action === 'restart') {
          mockTransport._resourceStates.deployment.status = 'restarting';
          addLog('deployment', '[demo] 🔄 Restarting application...');
          addLog('deployment', '[demo] Gracefully stopping...');
          addLog('deployment', '[demo] ✓ Application stopped');
          addLog('deployment', '[demo] Starting application...');
          addLog('deployment', '[demo] ✓ Application started successfully');
          setTimeout(() => {
            mockTransport._resourceStates.deployment.status = 'running';
          }, 100);
        } else if (action === 'trigger') {
          // Trigger new build
          mockTransport._resourceStates.deployment.buildNumber += 1;
          const newBuildNumber = mockTransport._resourceStates.deployment.buildNumber;
          mockTransport._logBuffers.build = []; // Clear old build logs
          addLog('build', `[demo] 🚀 Build #${newBuildNumber} started`);
          addLog('build', '[demo] 📦 Cloning repository from https://github.com/demo/app');
          
          // Schedule build logs to be added over time
          setTimeout(() => addLog('build', '[demo] ✓ Repository cloned successfully'), 500);
          setTimeout(() => addLog('build', '[demo] 🔍 Detected Node.js project (package.json found)'), 1000);
          setTimeout(() => addLog('build', '[demo] 📥 Installing dependencies with pnpm...'), 1500);
          setTimeout(() => addLog('build', '[demo] ⠿ Resolving packages...'), 2000);
          setTimeout(() => addLog('build', '[demo] ✓ Dependencies installed (234 packages)'), 4000);
          setTimeout(() => addLog('build', '[demo] 🔨 Running build command: pnpm build'), 4500);
          setTimeout(() => addLog('build', '[demo] ⚡ Building with Vite...'), 5000);
          setTimeout(() => addLog('build', '[demo] ⠿ Transforming files...'), 5500);
          setTimeout(() => addLog('build', '[demo] ✓ 127 modules transformed'), 7000);
          setTimeout(() => addLog('build', '[demo] 📦 Creating production bundle...'), 7500);
          setTimeout(() => addLog('build', '[demo] ✓ Build completed in 12.3s'), 9000);
          setTimeout(() => addLog('build', '[demo] 📊 Bundle size: 420 KB (gzipped: 145 KB)'), 9500);
          setTimeout(() => addLog('build', '[demo] 🐳 Building Docker image...'), 10000);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 1/8: FROM node:20-alpine'), 10500);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 2/8: WORKDIR /app'), 11000);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 3/8: COPY package*.json ./'), 11500);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 4/8: RUN pnpm install --prod'), 12000);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 5/8: COPY dist ./dist'), 12500);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 6/8: EXPOSE 3000'), 13000);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 7/8: ENV NODE_ENV=production'), 13500);
          setTimeout(() => addLog('build', '[demo] ⠿ Step 8/8: CMD ["node", "dist/server.js"]'), 14000);
          setTimeout(() => addLog('build', '[demo] ✓ Docker image built successfully'), 15000);
          setTimeout(() => addLog('build', `[demo] 🏷️  Tagged as obiente/demo-app:v${newBuildNumber}`), 15500);
          setTimeout(() => addLog('build', '[demo] 📤 Pushing image to registry...'), 16000);
          setTimeout(() => addLog('build', '[demo] ✓ Image pushed successfully'), 18000);
          setTimeout(() => addLog('build', '[demo] ✅ Build completed successfully in 95s'), 18500);
          setTimeout(() => addLog('build', '[demo] 🎉 Deployment ready!'), 19000);
        }
        
        return mockResponse({ deployment });
      }
      if (name.includes("deletedeployment")) {
        return mockResponse({ success: true });
      }
      if (name.includes("listcontainerfiles")) {
        const path = input?.path || "/";
        const files = [
          { name: "src", path: `${path === "/" ? "" : path}/src`, isDirectory: true },
          { name: "logs", path: `${path === "/" ? "" : path}/logs`, isDirectory: true },
          { name: "README.md", path: `${path === "/" ? "" : path}/README.md`, isDirectory: false, size: 1280 },
        ];
        const volumes = [
          { name: "data", mountPoint: "/data", sizeBytes: 5 * 1e9 },
          { name: "logs", mountPoint: "/var/log", sizeBytes: 1 * 1e9 },
        ];
        return mockResponse({
          files,
          volumes,
          hasMore: false,
          nextCursor: "",
          containerRunning: true,
        });
      }
      if (name.includes("getcontainerfile")) {
        const path = input?.path || "/README.md";
        return mockResponse({
          content: `# Preview File\nPath: ${path}\nThis is a demo file in preview mode.`,
          encoding: "text",
          size: 256,
          metadata: { mimeType: "text/markdown" },
        });
      }
      if (
        name.includes("createcontainerentry") ||
        name.includes("renamecontainerentry") ||
        name.includes("deletecontainerentries") ||
        name.includes("writecontainerfile")
      ) {
        return mockResponse({ success: true, errors: [] });
      }
      if (name.includes("createdeploymentfilearchive")) {
        return mockResponse({ archiveResponse: { success: true, archivePath: "/tmp/demo-archive.zip", filesArchived: 3 } });
      }
      if (name.includes("extractdeploymentfile")) {
        return mockResponse({ success: true, filesExtracted: 5 });
      }
      if (name.includes("uploadcontainerfiles")) {
        return mockResponse({ success: true, filesUploaded: (input?.metadata?.files?.length || 1) });
      }
      if (name.includes("logs")) {
        return mockResponse({
          logs: [
            { line: "[preview] Application starting...", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
            { line: "[preview] Server listening on port 3000", timestamp: timestamp(new Date()), stderr: false, logLevel: 3 },
          ],
        });
      }
      if (name.includes("files")) {
        return mockResponse({
          files: [
            { name: "README.md", path: "/", type: "file", size: 1280 },
            { name: "logs", path: "/logs", type: "directory" },
          ],
        });
      }
      return mockResponse({});
    }

    // Build-related mocks (outside deployment block since method name is just "listbuilds")
    if (name.includes("listbuilds") || name.includes("getdeploymentbuilds")) {
      const responseMessage = create(ListBuildsResponseSchema, {
        builds: buildSamples,
        total: buildSamples.length,
      });
      console.log('[MockTransport] ListBuilds response:', {
        hasBuilds: !!responseMessage.builds,
        buildsLength: responseMessage.builds?.length,
        responseKeys: Object.keys(responseMessage),
      });
      return mockResponse(responseMessage);
    }
    if (name.includes("reverttobuild") || name.includes("deletebuild")) {
      return mockResponse({ success: true });
    }

    // Game server mocks
    if (name.includes("gameserver")) {
      // Check for specific methods BEFORE general ones
      if (name.includes("getgameservermetrics") || name.includes("metrics")) {
        console.log('[PreviewTransport.unary] Returning game server metrics');
        // Use time range from request or default to last 30 days
        const endTime = input?.endTime?.seconds ? Number(input.endTime.seconds) * 1000 : Date.now();
        const startTime = input?.startTime?.seconds ? Number(input.startTime.seconds) * 1000 : endTime - (30 * 24 * 3600000);
        const durationMs = endTime - startTime;
        const intervalMs = Math.max(60000, Math.floor(durationMs / 720)); // Max 720 points, min 1min interval
        const pointCount = Math.floor(durationMs / intervalMs);
        
        // Base values for realistic game server metrics
        const baseCpu = 22; // 22% base CPU (games are more CPU intensive)
        const baseMemory = 512 * 1024 * 1024; // 512MB base memory
        const baseNetworkRx = 120_000; // Higher network for multiplayer
        const baseNetworkTx = 100_000;
        const baseDiskRead = 15_000;
        const baseDiskWrite = 12_000;
        
        const metrics = Array.from({ length: pointCount }).map((_, idx) => {
          const progress = idx / pointCount;
          const time = startTime + idx * intervalMs;
          const hourOfDay = new Date(time).getHours();
          
          // Gaming pattern: peak evening hours (18-23)
          const isEveningPeak = hourOfDay >= 18 && hourOfDay <= 23;
          const dailyMultiplier = isEveningPeak ? 1.3 : (hourOfDay >= 12 && hourOfDay < 18) ? 1.1 : 0.7;
          
          // Small random variation (±8%)
          const randomVariation = () => 0.92 + Math.random() * 0.16;
          
          return {
            timestamp: timestamp(new Date(time)),
            cpuUsagePercent: baseCpu * dailyMultiplier * randomVariation(),
            memoryUsageBytes: baseMemory * (1 + dailyMultiplier * 0.2) * randomVariation(),
            networkRxBytes: baseNetworkRx * dailyMultiplier * randomVariation(),
            networkTxBytes: baseNetworkTx * dailyMultiplier * randomVariation(),
            diskReadBytes: baseDiskRead * randomVariation(),
            diskWriteBytes: baseDiskWrite * randomVariation(),
          };
        });
        console.log('[PreviewTransport.unary] Generated', metrics.length, 'metric data points');
        return mockResponse({ metrics, ...(props.gameMetricsMock || {}) });
      }
      if (name.includes("listgameservers")) {
        return mockResponse({
          gameServers: [gameServer],
          total: 1,
        });
      }
      if (name.includes("getgameserver") || name === "gameserver") {
        return mockResponse({ gameServer });
      }
      if (name.includes("usage")) {
        return mockResponse(props.gameUsageMock || makeUsage());
      }
      if (name.includes("listgameserverfiles")) {
        const path = input?.path || "/";
        return mockResponse({
          files: [
            { name: "world", path: `${path === "/" ? "" : path}/world`, isDirectory: true },
            { name: "server.properties", path: `${path === "/" ? "" : path}/server.properties`, isDirectory: false, size: 2048 },
          ],
          volumes: [{ name: "data", mountPoint: "/data" }],
          containerRunning: true,
          hasMore: false,
          nextCursor: "",
        });
      }
      if (name.includes("getgameserverfile")) {
        const path = input?.path || "/server.properties";
        return mockResponse({
          content: `# Demo game server file\nPath: ${path}\nSample configuration content`,
          encoding: "text",
          size: 512,
          metadata: { mimeType: "text/plain" },
        });
      }
      if (
        name.includes("creategameserverentry") ||
        name.includes("renamegameserverentry") ||
        name.includes("deletegameserverentries") ||
        name.includes("writegameserverfile")
      ) {
        return mockResponse({ success: true, errors: [] });
      }
      if (name.includes("creategameserverfilearchive")) {
        return mockResponse({ archiveResponse: { success: true, archivePath: "/tmp/game.zip", filesArchived: 4 } });
      }
      if (name.includes("extractgameserverfile")) {
        return mockResponse({ success: true, filesExtracted: 4 });
      }
      if (name.includes("uploadgameserverfiles")) {
        return mockResponse({ success: true, filesUploaded: (input?.metadata?.files?.length || 1) });
      }
      if (name.includes("searchgameserverfiles")) {
        return mockResponse({
          results: [
            {
              path: "/server.properties",
              name: "server.properties",
              isDirectory: false,
              size: 512,
            },
          ],
          totalFound: 1,
          hasMore: false,
        });
      }
      if (name.includes("getgameserverlogs")) {
        return mockResponse({
          logs: [
            { line: "[demo] Game server started", timestamp: timestamp(new Date()), level: 3 },
            { line: "[demo] Player joined", timestamp: timestamp(new Date()), level: 3 },
          ],
        });
      }
      if (
        name.includes("startgameserver") ||
        name.includes("stopgameserver") ||
        name.includes("restartgameserver") ||
        name.includes("deletegameserver")
      ) {
        const action = name.includes("stop") ? 'stop' : name.includes("start") ? 'start' : name.includes("restart") ? 'restart' : 'delete';
        console.log(`[MockTransport.unary] Game server action: ${action}`);
        
        if (action === 'stop') {
          mockTransport._resourceStates.gameserver.status = 'stopping';
          addLog('gameserver', '[demo] 🛑 Stopping game server...');
          addLog('gameserver', '[demo] Saving world data...');
          addLog('gameserver', '[demo] ✓ World saved successfully');
          addLog('gameserver', '[demo] Disconnecting players...');
          addLog('gameserver', '[demo] ✓ All players disconnected');
          addLog('gameserver', '[demo] ✓ Server stopped');
          setTimeout(() => {
            mockTransport._resourceStates.gameserver.status = 'stopped';
          }, 100);
        } else if (action === 'start') {
          mockTransport._resourceStates.gameserver.status = 'starting';
          addLog('gameserver', '[demo] 🚀 Starting game server...');
          addLog('gameserver', '[demo] Loading world data...');
          addLog('gameserver', '[demo] ✓ World loaded successfully');
          addLog('gameserver', '[demo] Initializing server on port 25565');
          addLog('gameserver', '[demo] ✓ Server started successfully');
          addLog('gameserver', '[demo] Server ready for connections');
          setTimeout(() => {
            mockTransport._resourceStates.gameserver.status = 'running';
          }, 100);
        } else if (action === 'restart') {
          mockTransport._resourceStates.gameserver.status = 'restarting';
          addLog('gameserver', '[demo] 🔄 Restarting game server...');
          addLog('gameserver', '[demo] Saving world...');
          addLog('gameserver', '[demo] ✓ Server stopped');
          addLog('gameserver', '[demo] Starting server...');
          addLog('gameserver', '[demo] ✓ Server restarted successfully');
          setTimeout(() => {
            mockTransport._resourceStates.gameserver.status = 'running';
          }, 100);
        }
        
        return mockResponse({ gameServer, success: true });
      }
      if (name.includes("updategameserver")) {
        console.log('[MockTransport.unary] Updating game server with:', input);
        // Merge updates into game server object and persist
        Object.assign(mockTransport._resources.gameServer, input);
        return mockResponse({ gameServer: mockTransport._resources.gameServer, success: true });
      }
      return mockResponse({});
    }

    // VPS mocks
    if (name.includes("vps")) {
      if (name.includes("getvps") || name.includes("getvpsinstance") || name.includes("superadmingetvps") || name === "vps") {
        return mockResponse({ vps });
      }
      if (name.includes("listvpsinstances") || name.includes("listvps")) {
        return mockResponse({
          vpsInstances: [vps],
          total: 1,
        });
      }
      if (name.includes("getvpsmetrics")) {
        // Use time range from request or default to last 30 days
        const endTime = input?.endTime?.seconds ? Number(input.endTime.seconds) * 1000 : Date.now();
        const startTime = input?.startTime?.seconds ? Number(input.startTime.seconds) * 1000 : endTime - (30 * 24 * 3600000);
        const durationMs = endTime - startTime;
        const intervalMs = Math.max(60000, Math.floor(durationMs / 720)); // Max 720 points, min 1min interval
        const pointCount = Math.floor(durationMs / intervalMs);
        
        // Base values for realistic VPS metrics (lower than app servers)
        const baseCpu = 12; // 12% base CPU
        const baseMemory = 350 * 1024 * 1024; // 350MB base memory
        const baseNetworkRx = 60_000;
        const baseNetworkTx = 50_000;
        const baseDiskUsed = 5 * 1024 * 1024 * 1024; // 5 GB used
        const totalDisk = 20 * 1024 * 1024 * 1024; // 20 GB total
        
        const metrics = Array.from({ length: pointCount }).map((_, idx) => {
          const progress = idx / pointCount;
          const time = startTime + idx * intervalMs;
          const hourOfDay = new Date(time).getHours();
          
          // Moderate daily pattern
          const dailyMultiplier = 0.8 + 0.2 * Math.sin((hourOfDay - 6) * Math.PI / 12);
          
          // Small random variation (±8%)
          const randomVariation = () => 0.92 + Math.random() * 0.16;
          
          return {
            timestamp: timestamp(new Date(time)),
            cpuUsagePercent: baseCpu * dailyMultiplier * randomVariation(),
            memoryUsedBytes: baseMemory * (0.98 + progress * 0.04) * randomVariation(),
            networkRxBytes: baseNetworkRx * dailyMultiplier * randomVariation(),
            networkTxBytes: baseNetworkTx * dailyMultiplier * randomVariation(),
            diskUsedBytes: baseDiskUsed * (1 + progress * 0.02) * randomVariation(),
            diskTotalBytes: totalDisk,
          };
        });
        return mockResponse({ metrics, ...(props.vpsMetricsMock || {}) });
      }
      if (name.includes("getvpsusage")) {
        return mockResponse(makeUsage());
      }
      if (name.includes("getvpsproxyinfo")) {
        return mockResponse({
          sshProxyCommand: `ssh -p 2222 root@${vps.id}@ssh.preview.obiente.cloud`,
          connectionInstructions: "Use the SSH command above to connect via the preview bastion.",
        });
      }
      if (name.includes("listvpsregions")) {
        return mockResponse({
          regions: [
            { id: "us-east-1", name: "US East 1", available: true },
            { id: "eu-west-1", name: "EU West 1", available: true },
          ],
        });
      }
      if (name.includes("listvps")) {
        return mockResponse({
          vpsInstances: [vps],
          total: 1,
        });
      }
      if (name.includes("startvps") || name.includes("stopvps") || name.includes("rebootvps")) {
        const action = name.includes("stop") ? 'stop' : name.includes("start") ? 'start' : 'reboot';
        console.log(`[MockTransport.unary] VPS action: ${action}`);
        
        if (action === 'stop') {
          mockTransport._resourceStates.vps.status = 'stopping';
          addLog('vps', '[demo] 🛑 Shutting down VPS...');
          addLog('vps', '[demo] Stopping services...');
          addLog('vps', '[demo] ✓ All services stopped');
          addLog('vps', '[demo] Syncing filesystem...');
          addLog('vps', '[demo] ✓ Filesystem synced');
          addLog('vps', '[demo] ✓ VPS powered off');
          setTimeout(() => {
            mockTransport._resourceStates.vps.status = 'stopped';
          }, 100);
        } else if (action === 'start') {
          mockTransport._resourceStates.vps.status = 'starting';
          addLog('vps', '[demo] 🚀 Starting VPS...');
          addLog('vps', '[demo] Powering on instance...');
          addLog('vps', '[demo] ✓ Instance powered on');
          addLog('vps', '[demo] Booting operating system...');
          addLog('vps', '[demo] ✓ System booted successfully');
          addLog('vps', '[demo] Starting services...');
          addLog('vps', '[demo] ✓ SSH service started on port 22');
          addLog('vps', '[demo] ✓ VPS is ready');
          setTimeout(() => {
            mockTransport._resourceStates.vps.status = 'running';
          }, 100);
        } else if (action === 'reboot') {
          mockTransport._resourceStates.vps.status = 'rebooting';
          addLog('vps', '[demo] 🔄 Rebooting VPS...');
          addLog('vps', '[demo] Sending reboot signal...');
          addLog('vps', '[demo] ✓ System shutting down');
          addLog('vps', '[demo] ✓ System starting up');
          addLog('vps', '[demo] ✓ VPS rebooted successfully');
          setTimeout(() => {
            mockTransport._resourceStates.vps.status = 'running';
          }, 100);
        }
        
        return mockResponse({ vps, success: true });
      }
      if (name.includes("updatevps")) {
        console.log('[MockTransport.unary] Updating VPS with:', input);
        // Merge updates into vps object and persist
        Object.assign(mockTransport._resources.vps, input);
        return mockResponse({ vps: mockTransport._resources.vps, success: true });
      }
      if (name.includes("reinitializevps")) {
        return mockResponse({ vps, rootPassword: "preview-root-password", message: "VPS reinitialized in preview" });
      }
      if (name.includes("deletevps")) {
        return mockResponse({ success: true });
      }
      if (name.includes("resetvpspassword")) {
        return mockResponse({ rootPassword: "preview-reset-password", message: "Password reset in preview" });
      }
      if (name.includes("listsshkeys")) {
        return mockResponse({ sshKeys: [sampleSSHKey] });
      }
      if (name.includes("updateshkey") || name.includes("addsshkey") || name.includes("createshkey")) {
        return mockResponse({ sshKey: sampleSSHKey });
      }
      if (name.includes("deletesshkey") || name.includes("removesshkey")) {
        return mockResponse({ success: true });
      }
      if (name.includes("getterminalkey")) {
        return mockResponse({
          fingerprint: "AA:BB:CC:DD:EE",
          createdAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
          updatedAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
        });
      }
      if (name.includes("rotateterminalkey") || name.includes("rotatebastionkey")) {
        return mockResponse({
          fingerprint: "FF:EE:DD:CC:BB",
          createdAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
          updatedAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
          message: "Key rotated (preview)",
        });
      }
      if (name.includes("removeterminalkey")) {
        return mockResponse({ success: true });
      }
      if (name.includes("getbastionkey")) {
        return mockResponse({
          fingerprint: "AA:BB:CC:BA:ST",
          createdAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
          updatedAt: { seconds: BigInt(Math.floor(Date.now() / 1000)), nanos: 0 },
        });
      }
      if (name.includes("getsshalias")) {
        return mockResponse({ alias: "preview-vps" });
      }
      if (name.includes("setsshalias")) {
        return mockResponse({ alias: input?.alias || "preview-vps", message: "SSH alias set (preview)" });
      }
      if (name.includes("removesshalias")) {
        return mockResponse({ message: "SSH alias removed (preview)" });
      }
      if (name.includes("listfirewallrules")) {
        return mockResponse({ rules: [sampleFirewallRule] });
      }
      if (name.includes("getfirewalloptions")) {
        return mockResponse({ options: sampleFirewallOptions });
      }
      if (name.includes("updatefirewalloptions")) {
        return mockResponse({ options: { ...sampleFirewallOptions, ...(input || {}) } });
      }
      if (name.includes("createfirewallrule") || name.includes("updatefirewallrule")) {
        return mockResponse({ rule: { ...sampleFirewallRule, ...(input?.rule || {}) } });
      }
      if (name.includes("deletefirewallrule")) {
        return mockResponse({ success: true });
      }
      if (name.includes("listvpsfiles")) {
        return mockResponse({
          files: [
            { name: "etc", path: "/etc", isDirectory: true },
            { name: "hosts", path: "/etc/hosts", isDirectory: false, size: 128 },
          ],
          hasMore: false,
          nextCursor: "",
        });
      }
      return mockResponse({});
    }

    if (name.includes("audit")) {
      console.log('[MockTransport.unary] Handling audit logs request', { input });
      const auditLogs = [
        {
          id: "audit-1",
          organizationId: "mock-org",
          action: "DEPLOY",
          service: "deployment",
          userId: "user-1",
          userName: "Preview User",
          userEmail: "preview@example.com",
          resourceType: "deployment",
          resourceId: deployment.id,
          responseStatus: 200,
          durationMs: 120,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000)),
          ipAddress: "127.0.0.1",
          requestData: JSON.stringify({ preview: true }),
          details: "Mock deployment action",
        },
        {
          id: "audit-2",
          organizationId: "mock-org",
          action: "SCALE",
          service: "gameserver",
          userId: "user-2",
          userEmail: "player@example.com",
          resourceType: "gameserver",
          resourceId: gameServer.id,
          responseStatus: 201,
          durationMs: 250,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 5 * 24 * 60 * 60 * 1000)),
          ipAddress: "192.168.1.10",
          requestData: JSON.stringify({ scale: 2 }),
          details: "Mock game server scale",
        },
        {
          id: "audit-3",
          organizationId: "mock-org",
          action: "UPDATE",
          service: "vps",
          userId: "user-3",
          userName: "Admin",
          resourceType: "vps",
          resourceId: vps.id,
          responseStatus: 500,
          durationMs: 320,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 3 * 24 * 60 * 60 * 1000)),
          ipAddress: "10.0.0.5",
          errorMessage: "Preview error message",
          requestData: JSON.stringify({ action: "preview" }),
          details: "Mock VPS update failed",
        },
        {
          id: "audit-4",
          organizationId: "mock-org",
          action: "CREATE",
          service: "deployment",
          userId: "user-1",
          userName: "Preview User",
          resourceType: "deployment",
          resourceId: deployment.id,
          responseStatus: 201,
          durationMs: 450,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 10 * 24 * 60 * 60 * 1000)),
          ipAddress: "127.0.0.1",
          requestData: JSON.stringify({ name: "New Deployment" }),
          details: "Created deployment",
        },
        {
          id: "audit-5",
          organizationId: "mock-org",
          action: "RESTART",
          service: "gameserver",
          userId: "user-2",
          userEmail: "player@example.com",
          resourceType: "gameserver",
          resourceId: gameServer.id,
          responseStatus: 200,
          durationMs: 180,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 2 * 24 * 60 * 60 * 1000)),
          ipAddress: "192.168.1.10",
          requestData: JSON.stringify({ restart: true }),
          details: "Restarted game server",
        },
        {
          id: "audit-6",
          organizationId: "mock-org",
          action: "START",
          service: "vps",
          userId: "user-3",
          userName: "Admin",
          resourceType: "vps",
          resourceId: vps.id,
          responseStatus: 200,
          durationMs: 240,
          createdAt: timestamp(new Date(Date.now() - Math.random() * 1 * 24 * 60 * 60 * 1000)),
          ipAddress: "10.0.0.5",
          requestData: JSON.stringify({ action: "start" }),
          details: "Started VPS instance",
        },
      ];

      const request = input || {};
      
      const matchesFilter = (log: typeof auditLogs[number]) => {
        const matchOrg = !request.organizationId || log.organizationId === request.organizationId;
        const matchService = !request.service || log.service === request.service;
        const matchAction = !request.action || log.action === request.action;
        const matchUser = !request.userId || log.userId === request.userId;
        const matchResourceType = !request.resourceType || log.resourceType === request.resourceType;
        const matchResourceId = !request.resourceId || log.resourceId === request.resourceId;

        let matchStatus = true;
        if (request.status) {
          if (request.status === "error") {
            matchStatus = log.responseStatus >= 400;
          } else {
            const statusNum = parseInt(String(request.status), 10);
            matchStatus = !isNaN(statusNum) ? log.responseStatus === statusNum : true;
          }
        }

        const matches = matchOrg && matchService && matchAction && matchUser && matchResourceType && matchResourceId && matchStatus;
        return matches;
      };

      const filteredLogs = auditLogs.filter(matchesFilter);
      
      // Log details about filtering
      if (filteredLogs.length === 0 && auditLogs.length > 0) {
        const firstLog = auditLogs[0];
        if (firstLog) {
          console.log('[MockTransport.unary] All audit logs filtered out. First log details:', {
            requestFilters: request,
            sampleLog: {
              organizationId: firstLog.organizationId,
              service: firstLog.service,
              action: firstLog.action,
              resourceType: firstLog.resourceType,
            },
          });
        }
      }
      
      console.log('[MockTransport.unary] Audit logs filtered', {
        totalLogs: auditLogs.length,
        filteredCount: filteredLogs.length,
        filters: request,
      });

      return mockResponse({
        auditLogs: filteredLogs,
        totalCount: filteredLogs.length,
        nextPageToken: "",
      });
    }

    if (name.includes("listorganizations")) {
      return mockResponse({
        organizations: [
          { id: "mock-org", name: "Preview Org" },
          { id: "another-org", name: "Another Org" },
        ],
      });
    }

    console.warn('[PreviewProvider] No mock matched for method:', name, 'returning empty');
    return mockResponse({});
    } catch (error) {
      console.error('[MockTransport.unary] Fatal error:', error, { methodName, name });
      return {
        message: {},
        header: emptyHeaders(),
        trailer: emptyHeaders()
      };
    }
  },
  stream: (method: any, signal?: AbortSignal, _timeout?: number, _header?: HeadersInit, input?: any) => {
    const name = String(method?.name || "").toLowerCase();
    console.log('[MockTransport.stream] Called for method:', name, { hasSignal: !!signal, hasInput: !!input });

    // Persistent log buffers for each resource type
    if (!mockTransport._logBuffers) {
      mockTransport._logBuffers = {
        deployment: [] as any[],
        gameserver: [] as any[],
        vps: [] as any[],
        build: [] as any[],
      };
    }

    // Track resource states
    if (!mockTransport._resourceStates) {
      mockTransport._resourceStates = {
        deployment: { status: 'running', buildNumber: 12 },
        gameserver: { status: 'running' },
        vps: { status: 'running' },
      };
    }

    try {
      const makeStream = async function* (messages: any[]) {
        console.log('[MockTransport.stream] makeStream starting with', messages.length, 'messages');
        try {
          for (const msg of messages) {
            if (signal?.aborted) {
              console.log('[MockTransport.stream] Stream aborted');
              break;
            }
            yield msg;
          }
          console.log('[MockTransport.stream] makeStream completed');
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeStream:', error);
          throw error;
        }
      };

      const makeLiveMetricsStream = async function* (factory: () => any) {
        console.log('[MockTransport.stream] makeLiveMetricsStream starting');
        let count = 0;
        try {
          while (count < 200) {
            if (signal?.aborted) {
              console.log('[MockTransport.stream] Metrics stream aborted at count', count);
              break;
            }
            const metric = factory();
            yield metric;
            count += 1;
            if (count % 10 === 0) {
              console.log('[MockTransport.stream] Metrics stream progress:', count);
            }
            await sleep(5000);
          }
          console.log('[MockTransport.stream] Metrics stream completed at count', count);
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeLiveMetricsStream:', error);
          throw error;
        }
      };

      const makeBuildLogsStream = async function* () {
        console.log('[MockTransport.stream] makeBuildLogsStream starting');
        const buffer = mockTransport._logBuffers.build;
        
        // First, yield all existing logs from buffer
        for (const log of buffer) {
          if (signal?.aborted) return;
          yield log;
        }
        
        const logMessages = [
          "[demo] 🚀 Build #12 started",
          "[demo] 📦 Cloning repository from https://github.com/demo/app",
          "[demo] ✓ Repository cloned successfully",
          "[demo] 🔍 Detected Node.js project (package.json found)",
          "[demo] 📥 Installing dependencies with pnpm...",
          "[demo] ⠿ Resolving packages...",
          "[demo] ✓ Dependencies installed (234 packages)",
          "[demo] 🔨 Running build command: pnpm build",
          "[demo] ⚡ Building with Vite...",
          "[demo] ⠿ Transforming files...",
          "[demo] ✓ 127 modules transformed",
          "[demo] 📦 Creating production bundle...",
          "[demo] ✓ Build completed in 12.3s",
          "[demo] 📊 Bundle size: 420 KB (gzipped: 145 KB)",
          "[demo] 🐳 Building Docker image...",
          "[demo] ⠿ Step 1/8: FROM node:20-alpine",
          "[demo] ⠿ Step 2/8: WORKDIR /app",
          "[demo] ⠿ Step 3/8: COPY package*.json ./",
          "[demo] ⠿ Step 4/8: RUN pnpm install --prod",
          "[demo] ⠿ Step 5/8: COPY dist ./dist",
          "[demo] ⠿ Step 6/8: EXPOSE 3000",
          "[demo] ⠿ Step 7/8: ENV NODE_ENV=production",
          "[demo] ⠿ Step 8/8: CMD [\"node\", \"dist/server.js\"]",
          "[demo] ✓ Docker image built successfully",
          "[demo] 🏷️  Tagged as obiente/demo-app:latest",
          "[demo] 📤 Pushing image to registry...",
          "[demo] ✓ Image pushed successfully",
          "[demo] ✅ Build completed successfully in 95s",
          "[demo] 🎉 Deployment ready!",
        ];
        
        // If buffer is empty, add initial logs
        if (buffer.length === 0) {
          for (const message of logMessages) {
            if (signal?.aborted) return;
            
            const log = { line: message, timestamp: timestamp(new Date()), stderr: false, logLevel: 3 };
            buffer.push(log);
            yield log;
            
            // Vary timing based on log type
            if (message.includes('Installing') || message.includes('Building') || message.includes('Pushing')) {
              await sleep(800);
            } else if (message.includes('Step')) {
              await sleep(400);
            } else {
              await sleep(200);
            }
          }
          
          // Limit buffer size
          if (buffer.length > 200) {
            buffer.splice(0, buffer.length - 200);
          }
        }
      };

      const makeDeploymentLogsStream = async function* () {
        console.log('[MockTransport.stream] makeDeploymentLogsStream starting');
        const buffer = mockTransport._logBuffers.deployment;
        
        // First, yield all existing logs from buffer
        for (const log of buffer) {
          if (signal?.aborted) return;
          yield log;
        }
        
        const logMessages = [
          "[demo] Application starting...",
          "[demo] Loading environment variables",
          "[demo] Connecting to database at postgres://***",
          "[demo] ✓ Database connection established",
          "[demo] Connecting to Redis cache at redis://***:6379",
          "[demo] ✓ Redis connection successful",
          "[demo] Initializing HTTP server on port 3000",
          "[demo] ✓ Server listening on http://0.0.0.0:3000",
          "[demo] GET /health 200 - 12ms",
          "[demo] GET /api/users 200 - 45ms",
          "[demo] POST /api/auth/login 200 - 89ms",
          "[demo] Background job: Processing email queue (5 items)",
          "[demo] ✓ Email queue processed successfully",
          "[demo] GET /api/deployments 200 - 34ms",
          "[demo] WebSocket connection opened from 192.168.1.42",
          "[demo] Cache hit for key: user:profile:1234",
          "[demo] PUT /api/settings 200 - 67ms",
          "[demo] Scheduled task: Database backup started",
          "[demo] ✓ Database backup completed (142MB)",
          "[demo] GET /metrics 200 - 8ms",
          "[demo] Health check passed - all systems operational",
        ];
        
        let index = buffer.length;
        try {
          while (index < 500) {
            if (signal?.aborted) break;
            
            const message = logMessages[index % logMessages.length];
            const log = { line: message, timestamp: timestamp(new Date()), stderr: false, logLevel: 3 };
            
            buffer.push(log);
            yield log;
            
            index += 1;
            await sleep(2000 + Math.random() * 3000);
            
            // Limit buffer size
            if (buffer.length > 200) {
              buffer.shift();
            }
          }
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeDeploymentLogsStream:', error);
        }
      };

      const makeGameServerLogsStream = async function* () {
        console.log('[MockTransport.stream] makeGameServerLogsStream starting');
        const buffer = mockTransport._logBuffers.gameserver;
        
        // First, yield all existing logs from buffer
        for (const log of buffer) {
          if (signal?.aborted) return;
          yield log;
        }
        
        const logMessages = [
          "[demo] Game server initializing...",
          "[demo] Loading world data...",
          "[demo] Server started on port 25565",
          "[demo] Player 'Steve' joined the game",
          "[demo] Player 'Alex' joined the game",
          "[demo] Autosaving world...",
          "[demo] World saved successfully",
          "[demo] Player 'Steve' left the game",
          "[demo] Spawning entities in chunk [12, 45]",
          "[demo] Player 'Alex' achieved advancement [Stone Age]",
          "[demo] Server TPS: 20.0 (healthy)",
          "[demo] Memory usage: 1.2GB / 4GB",
          "[demo] Active players: 2",
          "[demo] Loaded chunks: 234",
        ];
        
        let index = buffer.length;
        try {
          while (index < 500) {
            if (signal?.aborted) break;
            
            const message = logMessages[index % logMessages.length];
            const log = { line: message, timestamp: timestamp(new Date()), level: 3 };
            
            buffer.push(log);
            yield log;
            
            index += 1;
            await sleep(2000 + Math.random() * 3000);
            
            // Limit buffer size
            if (buffer.length > 200) {
              buffer.shift();
            }
          }
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeGameServerLogsStream:', error);
        }
      };

      const makeVPSLogsStream = async function* () {
        console.log('[MockTransport.stream] makeVPSLogsStream starting');
        const buffer = mockTransport._logBuffers.vps;
        
        // First, yield all existing logs from buffer
        for (const log of buffer) {
          if (signal?.aborted) return;
          yield log;
        }
        
        const logMessages = [
          "[demo] VPS instance started",
          "[demo] System initialization complete",
          "[demo] SSH service running on port 22",
          "[demo] Firewall rules applied",
          "[demo] CPU usage: 12%",
          "[demo] Available memory: 3.2GB",
          "[demo] Network interface: eth0 up",
          "[demo] Disk I/O: nominal",
          "[demo] Security updates available: 0",
          "[demo] Uptime: 2 days 4 hours",
          "[demo] System load average: 0.45",
          "[demo] Active connections: 3",
        ];
        
        let index = buffer.length;
        try {
          while (index < 500) {
            if (signal?.aborted) break;
            
            const message = logMessages[index % logMessages.length];
            const log = { line: message, timestamp: timestamp(new Date()), level: 3 };
            
            buffer.push(log);
            yield log;
            
            index += 1;
            await sleep(4000 + Math.random() * 4000);
            
            // Limit buffer size
            if (buffer.length > 200) {
              buffer.shift();
            }
          }
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeVPSLogsStream:', error);
        }
      };

      const makeLiveAuditStream = async function* () {
        console.log('[MockTransport.stream] makeLiveAuditStream starting');
        let i = 0;
        try {
          while (i < 100) {
            i += 1;
            if (signal?.aborted) {
              console.log('[MockTransport.stream] Audit stream aborted at', i);
              break;
            }
            yield {
              auditLogs: [
                {
                  id: `audit-${Date.now()}-${i}`,
                  action: "PREVIEW",
                  service: "preview",
                  userId: "preview-user",
                  userName: "Preview User",
                  resourceType: "preview",
                  resourceId: "mock",
                  responseStatus: 200,
                  durationMs: 50,
                  createdAt: timestamp(new Date()),
                  ipAddress: "127.0.0.1",
                  requestData: JSON.stringify({ i }),
                  details: "Mock audit event from preview stream",
                },
              ],
              totalCount: 1,
            };
            await sleep(5000);
          }
          console.log('[MockTransport.stream] Audit stream completed');
        } catch (error) {
          console.error('[MockTransport.stream] Error in makeLiveAuditStream:', error);
          throw error;
        }
      };

    // Create proper StreamResponse structure as expected by Connect
    const createStreamResponse = (gen: AsyncGenerator<any>) => {
      try {
        const response = {
          stream: true as const,
          service: method?.parent || { typeName: 'preview', methods: {} },
          method,
          header: emptyHeaders(),
          trailer: emptyHeaders(),
          message: makeStreamResponse(gen, { signal })
        };
        console.log('[MockTransport.stream] Created stream response for method:', name);
        return response;
      } catch (error) {
        console.error('[MockTransport.stream] Error creating stream response:', error);
        throw error;
      }
    };

    // Deployment streams
    if (name.includes("deployment")) {
      console.log('[MockTransport.stream] Handling deployment stream:', name);
      if (name.includes("buildlogs")) {
        console.log('[MockTransport.stream] Creating build logs stream');
        return Promise.resolve(createStreamResponse(makeBuildLogsStream()));
      }
      if (name.includes("logs")) {
        console.log('[MockTransport.stream] Creating deployment logs stream');
        return Promise.resolve(createStreamResponse(makeDeploymentLogsStream()));
      }
      if (name.includes("metrics")) {
        console.log('[MockTransport.stream] Creating deployment metrics stream');
        return Promise.resolve(createStreamResponse(
          makeLiveMetricsStream(() => ({
            timestamp: timestamp(new Date()),
            cpuUsagePercent: 10 + Math.round(Math.random() * 15),
            memoryUsageBytes: 512 * 1024 * 1024 + Math.round(Math.random() * 512 * 1024 * 1024),
            networkRxBytes: 90_000 + Math.round(Math.random() * 60_000),
            networkTxBytes: 70_000 + Math.round(Math.random() * 50_000),
            diskReadBytes: 10_000 + Math.round(Math.random() * 5_000),
            diskWriteBytes: 8_000 + Math.round(Math.random() * 5_000),
          }))
        ));
      }
    }

    // Game server streams
    if (name.includes("gameserver")) {
      console.log('[MockTransport.stream] Handling game server stream:', name);
      if (name.includes("logs")) {
        console.log('[MockTransport.stream] Creating game server logs stream');
        return Promise.resolve(createStreamResponse(makeGameServerLogsStream()));
      }
      if (name.includes("metrics")) {
        console.log('[MockTransport.stream] Creating game server metrics stream');
        return Promise.resolve(createStreamResponse(
          makeLiveMetricsStream(() => ({
            timestamp: timestamp(new Date()),
            cpuUsagePercent: 15 + Math.round(Math.random() * 20),
            memoryUsageBytes: 30 * 1024 * 1024 + Math.round(Math.random() * 20 * 1024 * 1024),
            networkRxBytes: 120_000 + Math.round(Math.random() * 80_000),
            networkTxBytes: 80_000 + Math.round(Math.random() * 60_000),
            diskReadBytes: 12_000 + Math.round(Math.random() * 6_000),
            diskWriteBytes: 10_000 + Math.round(Math.random() * 6_000),
          }))
        ));
      }
    }

    // VPS streams
    if (name.includes("vps")) {
      console.log('[MockTransport.stream] Handling VPS stream:', name);
      if (name.includes("logs")) {
        console.log('[MockTransport.stream] Creating VPS logs stream');
        return Promise.resolve(createStreamResponse(makeVPSLogsStream()));
      }
      if (name.includes("metrics")) {
        console.log('[MockTransport.stream] Creating VPS metrics stream');
        const baseMemory = 20 * 1024 * 1024; // 20 MB base
        const baseDisk = 5 * 1024 * 1024 * 1024; // 5 GB base
        const totalDisk = 20 * 1024 * 1024 * 1024; // 20 GB total
        return Promise.resolve(createStreamResponse(
          makeLiveMetricsStream(() => ({
            timestamp: timestamp(new Date()),
            cpuUsagePercent: 8 + Math.round(Math.random() * 12),
            memoryUsedBytes: baseMemory + Math.round(Math.random() * 15 * 1024 * 1024),
            networkRxBytes: 40_000 + Math.round(Math.random() * 40_000),
            networkTxBytes: 30_000 + Math.round(Math.random() * 30_000),
            diskUsedBytes: baseDisk + Math.round(Math.random() * 2 * 1024 * 1024 * 1024),
            diskTotalBytes: totalDisk,
          }))
        ));
      }
    }

    // Audit streams
    if (name.includes("audit")) {
      console.log('[MockTransport.stream] Creating audit stream');
      return Promise.resolve(createStreamResponse(makeLiveAuditStream()));
    }

    console.warn('[MockTransport.stream] No match for method:', name, '- returning empty stream');
    return Promise.resolve(createStreamResponse(makeStream([])));
    } catch (error) {
      console.error('[MockTransport.stream] Fatal error in stream:', error, { name });
      // Return a minimal valid stream response
      return Promise.resolve({
        stream: true as const,
        service: method?.parent || { typeName: 'preview', methods: {} },
        method,
        header: emptyHeaders(),
        trailer: emptyHeaders(),
        message: (async function* () { })() as any
      });
    }
  },
};

// Stub WebSocket to avoid real terminal connections in preview
class MockWebSocket extends EventTarget {
  // WebSocket constants
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  
  readonly CONNECTING = 0;
  readonly OPEN = 1;
  readonly CLOSING = 2;
  readonly CLOSED = 3;
  
  url: string;
  readyState = 1; // OPEN
  #timer: number | null = null;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(url: string) {
    super();
    this.url = url;
    console.log('[MockWebSocket] Created for URL:', url);
    
    // Simulate connection opening
    setTimeout(() => {
      const openEvent = new Event("open");
      this.dispatchEvent(openEvent);
      if (this.onopen) this.onopen(openEvent);
      console.log('[MockWebSocket] Connection opened');
    }, 100);
  }

  send(data: any) {
    console.log('[MockWebSocket] Send called with:', data);
    
    try {
      const message = JSON.parse(data);
      console.log('[MockWebSocket] Parsed message:', message);
      
      // Handle init message - respond with connected
      if (message.type === 'init') {
        setTimeout(() => {
          this.sendMessage({
            type: 'connected',
            message: 'Demo terminal connected successfully'
          });
          
          // Send welcome message as output
          this.sendOutput('\r\nConnecting to demo terminal...\r\n');
          this.sendOutput('Welcome to Obiente Cloud Interactive Demo\r\n');
          this.sendOutput("Type 'help' to see available commands\r\n\r\n");
          this.sendOutput('demo@obiente:~$ ');
        }, 50);
        return;
      }
      
      // Handle input message - process terminal commands
      if (message.type === 'input' && message.input) {
        console.log('[MockWebSocket] Received input:', message.input);
        const input = new Uint8Array(message.input);
        const text = new TextDecoder().decode(input);
        console.log('[MockWebSocket] Decoded input text:', JSON.stringify(text));
        this.handleTerminalInput(text);
        return;
      }
      
      // Handle resize - just acknowledge
      if (message.type === 'resize') {
        console.log('[MockWebSocket] Terminal resized to:', message.cols, 'x', message.rows);
        return;
      }
      
    } catch (err) {
      console.error('[MockWebSocket] Failed to parse message:', err);
    }
  }

  #commandBuffer = '';
  #commandHistory: string[] = [];
  #historyIndex = -1;
  #cursorPosition = 0;
  #githubCache: Map<string, any> = new Map();
  #currentPath = '/apps/dashboard';
  
  async fetchGithubContent(path: string): Promise<any> {
    const cacheKey = path;
    if (this.#githubCache.has(cacheKey)) {
      return this.#githubCache.get(cacheKey);
    }
    
    try {
      const apiUrl = `https://api.github.com/repos/Obiente/cloud/contents${path}`;
      const response = await fetch(apiUrl, {
        headers: {
          'Accept': 'application/vnd.github.v3+json',
        }
      });
      
      if (!response.ok) {
        console.error('[Terminal] GitHub API error:', response.status);
        return null;
      }
      
      const data = await response.json();
      this.#githubCache.set(cacheKey, data);
      return data;
    } catch (error) {
      console.error('[Terminal] Failed to fetch from GitHub:', error);
      return null;
    }
  }
  
  handleTerminalInput(input: string) {
    // Handle escape sequences (arrow keys, etc.)
    if (input.startsWith('\x1b[')) {
      const code = input.slice(2);
      
      // Arrow Up - previous command in history
      if (code === 'A') {
        if (this.#commandHistory.length > 0 && this.#historyIndex < this.#commandHistory.length - 1) {
          this.#historyIndex++;
          const cmd = this.#commandHistory[this.#commandHistory.length - 1 - this.#historyIndex];
          if (cmd !== undefined) this.replaceCurrentLine(cmd);
        }
        return;
      }
      
      // Arrow Down - next command in history
      if (code === 'B') {
        if (this.#historyIndex > 0) {
          this.#historyIndex--;
          const cmd = this.#commandHistory[this.#commandHistory.length - 1 - this.#historyIndex];
          if (cmd !== undefined) this.replaceCurrentLine(cmd);
        } else if (this.#historyIndex === 0) {
          this.#historyIndex = -1;
          this.replaceCurrentLine('');
        }
        return;
      }
      
      // Arrow Left - move cursor left
      if (code === 'D') {
        if (this.#cursorPosition > 0) {
          this.#cursorPosition--;
          this.sendOutput('\x1b[D');
        }
        return;
      }
      
      // Arrow Right - move cursor right
      if (code === 'C') {
        if (this.#cursorPosition < this.#commandBuffer.length) {
          this.#cursorPosition++;
          this.sendOutput('\x1b[C');
        }
        return;
      }
      
      return; // Ignore other escape sequences
    }
    
    // Handle Tab - autocomplete
    if (input === '\t') {
      this.handleTabComplete();
      return;
    }
    
    // Handle Enter
    if (input === '\r' || input === '\n') {
      this.sendOutput('\r\n');
      const command = this.#commandBuffer.trim();
      this.#commandBuffer = '';
      this.#cursorPosition = 0;
      this.#historyIndex = -1;
      
      if (command) {
        // Add to history (avoid duplicates of last command)
        if (this.#commandHistory.length === 0 || this.#commandHistory[this.#commandHistory.length - 1] !== command) {
          this.#commandHistory.push(command);
          // Keep history to reasonable size
          if (this.#commandHistory.length > 100) {
            this.#commandHistory.shift();
          }
        }
        this.executeCommand(command);
      } else {
        this.sendOutput('demo@obiente:~$ ');
      }
      return;
    }
    
    // Handle Backspace
    if (input === '\x7f' || input === '\b') {
      if (this.#cursorPosition > 0) {
        // Remove character at cursor position
        this.#commandBuffer = 
          this.#commandBuffer.slice(0, this.#cursorPosition - 1) +
          this.#commandBuffer.slice(this.#cursorPosition);
        this.#cursorPosition--;
        
        // Redraw the line from cursor position
        const remaining = this.#commandBuffer.slice(this.#cursorPosition);
        this.sendOutput('\b' + remaining + ' \b');
        // Move cursor back to correct position
        for (let i = 0; i < remaining.length; i++) {
          this.sendOutput('\b');
        }
      }
      return;
    }
    
    // Handle Ctrl+C
    if (input === '\x03') {
      this.#commandBuffer = '';
      this.#cursorPosition = 0;
      this.sendOutput('^C\r\ndemo@obiente:~$ ');
      return;
    }
    
    // Handle Ctrl+L (clear screen)
    if (input === '\x0c') {
      this.sendOutput('\x1b[2J\x1b[H');
      this.sendOutput('demo@obiente:~$ ' + this.#commandBuffer);
      return;
    }
    
    // Regular character - insert at cursor position
    if (input >= ' ' && input <= '~') {
      this.#commandBuffer = 
        this.#commandBuffer.slice(0, this.#cursorPosition) +
        input +
        this.#commandBuffer.slice(this.#cursorPosition);
      this.#cursorPosition++;
      
      // Echo character and redraw rest of line if needed
      const remaining = this.#commandBuffer.slice(this.#cursorPosition);
      this.sendOutput(input + remaining);
      // Move cursor back to correct position
      for (let i = 0; i < remaining.length; i++) {
        this.sendOutput('\b');
      }
    }
  }
  
  replaceCurrentLine(newCommand: string) {
    // Clear current line
    this.sendOutput('\r\x1b[K');
    this.sendOutput('demo@obiente:~$ ');
    this.#commandBuffer = newCommand;
    this.#cursorPosition = newCommand.length;
    this.sendOutput(newCommand);
  }
  
  handleTabComplete() {
    const parts = this.#commandBuffer.split(/\s+/);
    const currentPart = parts[parts.length - 1] || '';
    
    // Command completion (if first word)
    if (parts.length === 1) {
      const commands = ['ls', 'cat', 'pwd', 'whoami', 'echo', 'date', 'uname', 'uptime', 'free', 'df', 'ps', 'top', 'env', 'help', 'clear', 'exit'];
      const matches = commands.filter(cmd => cmd.startsWith(currentPart));
      
      if (matches.length === 1 && matches[0]) {
        // Complete the command
        const completion = matches[0].slice(currentPart.length);
        this.#commandBuffer += completion + ' ';
        this.#cursorPosition = this.#commandBuffer.length;
        this.sendOutput(completion + ' ');
      } else if (matches.length > 1) {
        // Show all matches
        this.sendOutput('\r\n' + matches.join('  ') + '\r\n');
        this.sendOutput('demo@obiente:~$ ' + this.#commandBuffer);
      }
      return;
    }
    
    // File/directory completion - fetch from GitHub
    this.handleFileCompletion(currentPart);
  }
  
  async handleFileCompletion(currentPart: string) {
    // Determine the directory to search in
    const lastSlash = currentPart.lastIndexOf('/');
    const dirPath = lastSlash >= 0 ? currentPart.substring(0, lastSlash) : '';
    const partialName = lastSlash >= 0 ? currentPart.substring(lastSlash + 1) : currentPart;
    
    const searchPath = dirPath ? `${this.#currentPath}/${dirPath}` : this.#currentPath;
    const data = await this.fetchGithubContent(searchPath);
    
    if (!data || !Array.isArray(data)) {
      // No completions available
      return;
    }
    
    // Filter files/dirs that match the partial name
    const items = data.map((item: any) => {
      const name = item.type === 'dir' ? item.name + '/' : item.name;
      return dirPath ? `${dirPath}/${name}` : name;
    });
    
    const matches = items.filter((item: string) => {
      const itemName = item.split('/').pop() || '';
      return itemName.toLowerCase().startsWith(partialName.toLowerCase());
    });
    
    if (matches.length === 1 && matches[0]) {
      // Complete the filename
      const completion = matches[0].slice(currentPart.length);
      
      // Clear what user typed and replace with completion
      for (let i = 0; i < this.#commandBuffer.length; i++) {
        this.sendOutput('\b');
      }
      
      const parts = this.#commandBuffer.split(/\s+/);
      parts[parts.length - 1] = matches[0];
      this.#commandBuffer = parts.join(' ');
      this.#cursorPosition = this.#commandBuffer.length;
      
      // Redraw the prompt with completed command
      this.sendOutput('\r\x1b[K');
      this.sendOutput('demo@obiente:~$ ' + this.#commandBuffer);
    } else if (matches.length > 1) {
      // Show all matches
      this.sendOutput('\r\n' + matches.join('  ') + '\r\n');
      this.sendOutput('demo@obiente:~$ ' + this.#commandBuffer);
    }
  }
  
  async handleLsCommand(args: string[]) {
    const target = args.length > 0 && args[0] ? args[0].replace(/\/+$/, '') : '';
    const path = target ? `${this.#currentPath}/${target}` : this.#currentPath;
    
    const data = await this.fetchGithubContent(path);
    
    if (!data) {
      this.sendOutput(`ls: cannot access '${target || '.'}': Failed to fetch from GitHub\r\n`);
      this.sendOutput('demo@obiente:~$ ');
      return;
    }
    
    if (Array.isArray(data)) {
      // Directory listing
      const items = data.map(item => item.type === 'dir' ? item.name + '/' : item.name);
      this.sendOutput(items.join('  ') + '\r\n');
    } else if (data.type === 'file') {
      // Single file - just show the name
      this.sendOutput(data.name + '\r\n');
    } else {
      this.sendOutput(`ls: cannot access '${target}': Not a directory\r\n`);
    }
    
    this.sendOutput('demo@obiente:~$ ');
  }
  
  async handleCatCommand(args: string[]) {
    if (!args[0]) {
      this.sendOutput('cat: missing operand\r\n');
      this.sendOutput('demo@obiente:~$ ');
      return;
    }
    
    const filePath = `${this.#currentPath}/${args[0]}`;
    const data = await this.fetchGithubContent(filePath);
    
    if (!data) {
      this.sendOutput(`cat: ${args[0]}: Failed to fetch from GitHub\r\n`);
      this.sendOutput('demo@obiente:~$ ');
      return;
    }
    
    if (data.type === 'dir') {
      this.sendOutput(`cat: ${args[0]}: Is a directory\r\n`);
      this.sendOutput('demo@obiente:~$ ');
      return;
    }
    
    if (data.type === 'file' && data.content) {
      try {
        // GitHub API returns base64 encoded content
        const content = atob(data.content);
        // Convert to terminal-friendly format (replace \n with \r\n)
        const terminalContent = content.replace(/\n/g, '\r\n');
        this.sendOutput(terminalContent);
        if (!terminalContent.endsWith('\r\n')) {
          this.sendOutput('\r\n');
        }
      } catch (error) {
        this.sendOutput(`cat: ${args[0]}: Failed to decode file content\r\n`);
      }
    } else {
      this.sendOutput(`cat: ${args[0]}: No such file or directory\r\n`);
    }
    
    this.sendOutput('demo@obiente:~$ ');
  }
  
  executeCommand(command: string) {
    const parts = command.split(/\s+/);
    const cmd = parts[0];
    const args = parts.slice(1);

    switch (cmd) {
      case 'ls':
      case 'dir':
        this.handleLsCommand(args);
        break;
      
      case 'pwd':
        this.sendOutput(`/home/demo${this.#currentPath}\r\n`);
        break;
      
      case 'whoami':
        this.sendOutput('demo\r\n');
        break;
      
      case 'cat':
        this.handleCatCommand(args);
        break;
      
      case 'echo':
        this.sendOutput(`${args.join(' ')}\r\n`);
        break;
      
      case 'date':
        this.sendOutput(`${new Date().toUTCString()}\r\n`);
        break;
      
      case 'uname':
        if (args.includes('-a')) {
          this.sendOutput('Linux demo-container 5.15.0 #1 SMP x86_64 GNU/Linux\r\n');
        } else {
          this.sendOutput('Linux\r\n');
        }
        break;
      
      case 'uptime':
        this.sendOutput('14:32:05 up 2 days, 3:45, 1 user, load average: 0.52, 0.58, 0.61\r\n');
        break;
      
      case 'free':
        this.sendOutput('              total        used        free      shared  buff/cache   available\r\n');
        this.sendOutput('Mem:        8192000     3456000     2048000      128000     2688000     4224000\r\n');
        this.sendOutput('Swap:       2048000      512000     1536000\r\n');
        break;
      
      case 'df':
        this.sendOutput('Filesystem     1K-blocks      Used Available Use% Mounted on\r\n');
        this.sendOutput('/dev/sda1       51474912  15728640  33105024  33% /\r\n');
        this.sendOutput('tmpfs            4096000         0   4096000   0% /dev/shm\r\n');
        break;
      
      case 'ps':
        if (args.includes('aux') || args.includes('-ef')) {
          this.sendOutput('USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND\r\n');
          this.sendOutput('root         1  0.1  0.2  12345  6789 ?        Ss   12:00   0:01 /sbin/init\r\n');
          this.sendOutput('demo       123  0.3  1.5 234567 45678 ?        Sl   12:01   0:15 node server.js\r\n');
          this.sendOutput('demo       456  0.0  0.1  12345  3456 pts/0    Ss   14:30   0:00 /bin/bash\r\n');
        } else {
          this.sendOutput('  PID TTY          TIME CMD\r\n');
          this.sendOutput('  456 pts/0    00:00:00 bash\r\n');
          this.sendOutput('  789 pts/0    00:00:00 ps\r\n');
        }
        break;
      
      case 'top':
        this.sendOutput('top - 14:32:05 up 2 days,  3:45,  1 user,  load average: 0.52, 0.58, 0.61\r\n');
        this.sendOutput('Tasks: 145 total,   1 running, 144 sleeping,   0 stopped,   0 zombie\r\n');
        this.sendOutput('%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.1 id,  0.3 wa,  0.0 hi,  0.3 si,  0.0 st\r\n');
        this.sendOutput("Press 'q' to quit (demo terminal)\r\n");
        break;
      
      case 'env':
        this.sendOutput('PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\r\n');
        this.sendOutput('HOME=/home/demo\r\n');
        this.sendOutput('USER=demo\r\n');
        this.sendOutput('NODE_ENV=production\r\n');
        this.sendOutput('PORT=3000\r\n');
        break;
      
      case 'help':
        this.sendOutput('Available commands in this demo:\r\n');
        this.sendOutput('  ls [dir]     - list files in directory\r\n');
        this.sendOutput('  cat <file>   - display file contents\r\n');
        this.sendOutput('  pwd, whoami, echo, date, uname, uptime\r\n');
        this.sendOutput('  free, df, ps, top, env, clear, exit\r\n');
        this.sendOutput('\r\nThis is a demo terminal running in preview mode.\r\n');
        break;
      
      case 'clear':
        this.sendOutput('\x1b[2J\x1b[H'); // ANSI clear screen
        break;
      
      case 'exit':
        this.sendOutput('Closing demo terminal...\r\n');
        setTimeout(() => this.close(), 500);
        return;
      
      default:
        this.sendOutput(`${cmd}: command not found\r\n`);
        break;
    }
    
    // Send prompt for next command
    this.sendOutput('demo@obiente:~$ ');
  }
  
  sendMessage(message: any) {
    const event = new MessageEvent('message', {
      data: JSON.stringify(message)
    });
    this.dispatchEvent(event);
    if (this.onmessage) this.onmessage(event);
  }
  
  sendOutput(text: string) {
    const encoder = new TextEncoder();
    const data = encoder.encode(text);
    this.sendMessage({
      type: 'output',
      data: Array.from(data)
    });
  }

  close() {
    console.log('[MockWebSocket] Close called');
    this.readyState = 3; // CLOSED
    if (this.#timer !== null) {
      clearInterval(this.#timer);
      this.#timer = null;
    }
    const closeEvent = new CloseEvent('close');
    this.dispatchEvent(closeEvent);
    if (this.onclose) this.onclose(closeEvent);
  }

  override addEventListener(type: string, listener: EventListenerOrEventListenerObject | null, options?: boolean | AddEventListenerOptions) {
    if (!listener) return;
    return super.addEventListener(type, listener, options);
  }

  override removeEventListener(type: string, listener?: EventListenerOrEventListenerObject | null, options?: boolean | EventListenerOptions) {
    if (!listener) return;
    return super.removeEventListener(type, listener, options);
  }
}

// Install mock transport immediately (before component mounts) so child components can use it
try {
  const previewGlobal = globalThis as typeof globalThis & { __OBIENTE_PREVIEW_CONNECT__?: unknown };
  previewGlobal.__OBIENTE_PREVIEW_CONNECT__ = mockTransport;
  globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  // Scope preview transport to this provider's subtree via injection
  provide(PREVIEW_CONNECT_KEY, mockTransport);
  console.log('[PreviewProviders] Mock transport and WebSocket installed successfully');
  
  // Mark as ready after a small delay to ensure everything is set up
  nextTick(() => {
    isReady.value = true;
    console.log('[PreviewProviders] Component marked as ready');
  });
} catch (error) {
  console.error('[PreviewProviders] Error installing mock transport:', error);
}

onBeforeUnmount(() => {
  try {
    console.log('[PreviewProviders] Cleanup starting for instance:', instanceId, 'mode:', props.mode);
    
    // First, mark as not ready to hide component
    isReady.value = false;
    console.log('[PreviewProviders] Component marked as not ready');
    
    // Give component time to unmount before restoring globals
    setTimeout(() => {
      // Restore original transport and WebSocket
      const globalWithPreview = globalThis as typeof globalThis & { __OBIENTE_PREVIEW_CONNECT__?: unknown };
      if (originalTransport) {
        globalWithPreview.__OBIENTE_PREVIEW_CONNECT__ = originalTransport;
        console.log('[PreviewProviders] Restored original transport');
      } else {
        delete globalWithPreview.__OBIENTE_PREVIEW_CONNECT__;
        console.log('[PreviewProviders] Deleted mock transport');
      }
      if (originalWebSocket) {
        globalThis.WebSocket = originalWebSocket;
        console.log('[PreviewProviders] Restored original WebSocket');
      }
      console.log('[PreviewProviders] Mock transport cleanup completed');
    }, 0);
  } catch (error) {
    console.error('[PreviewProviders] Error during cleanup:', error);
  }
});

// Capture any errors from child components
onErrorCaptured((err, instance, info) => {
  console.error('[PreviewProviders] Error captured from child component:', {
    error: err,
    message: err?.message,
    stack: err?.stack,
    componentName: instance?.$options?.name || instance?.$?.type?.name,
    errorInfo: info,
  });
  // Return false to prevent error from propagating
  return false;
});

// Handle component errors
const handleComponentError = (error: any) => {
  console.error('[PreviewProviders] Component error event:', error);
};

// Track instance ID for debugging
const instanceId = `preview-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
console.log('[PreviewProviders] Instance created:', instanceId, 'mode:', props.mode);

console.log('[PreviewProviders] Setup complete - component ready to render');
</script>
