import { computed, reactive, ref } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import {
  DeploymentService,
  GameServerService,
  DeleteContainerEntriesRequestSchema,
  RenameContainerEntryRequestSchema,
  CreateContainerEntryRequestSchema,
  WriteContainerFileRequestSchema,
  DeleteGameServerEntriesRequestSchema,
  RenameGameServerEntryRequestSchema,
  CreateGameServerEntryRequestSchema,
  WriteGameServerFileRequestSchema,
} from "@obiente/proto";
import type {
  ContainerFile,
  VolumeInfo,
  GameServerFile,
  GameServerVolumeInfo,
  RenameContainerEntryRequest,
  CreateContainerEntryRequest,
  WriteContainerFileRequest,
  RenameGameServerEntryRequest,
  CreateGameServerEntryRequest,
  WriteGameServerFileRequest,
} from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import type { ExplorerNode } from "../components/shared/fileExplorerTypes";

interface SourceState {
  type: "container" | "volume";
  volumeName?: string;
}

type ResourceType = "deployment" | "gameserver";

interface DeploymentExplorerOptions {
  type: "deployment";
  organizationId: string;
  deploymentId: string;
}

interface GameServerExplorerOptions {
  type: "gameserver";
  gameServerId: string;
}

type ExplorerOptions = DeploymentExplorerOptions | GameServerExplorerOptions;

type FileType = ContainerFile | GameServerFile;
type VolumeType = VolumeInfo | GameServerVolumeInfo;

type RenameEntryInput =
  | Partial<Omit<RenameContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>
  | Partial<Omit<RenameGameServerEntryRequest, "$typeName" | "$unknown" | "gameServerId">>;

type CreateEntryInput =
  | Partial<Omit<CreateContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>
  | Partial<Omit<CreateGameServerEntryRequest, "$typeName" | "$unknown" | "gameServerId">>;

type WriteFileInput =
  | Partial<Omit<WriteContainerFileRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>
  | Partial<Omit<WriteGameServerFileRequest, "$typeName" | "$unknown" | "gameServerId">>;

export function useFileExplorer(options: ExplorerOptions) {
  const isDeployment = options.type === "deployment";
  const deploymentClient = isDeployment ? useConnectClient(DeploymentService) : null;
  const gameServerClient = !isDeployment ? useConnectClient(GameServerService) : null;
  const client = isDeployment ? deploymentClient : gameServerClient;
  
  const currentOrgId = ref(isDeployment ? (options as DeploymentExplorerOptions).organizationId ?? "" : "");
  
  const root = reactive<ExplorerNode>({
    id: "ROOT",
    name: "",
    path: "/",
    parentPath: "",
    type: "directory",
    children: [],
    isLoading: false,
    hasLoaded: false,
    hasMore: false,
    nextCursor: null,
    isExpanded: true,
  });

  const source = reactive<SourceState>({ type: "container" });
  const volumes = ref<VolumeType[]>([]);
  const containerRunning = ref(false);
  const selectedPath = ref<string | null>(null);
  const isLoadingTree = ref(false);
  const errorMessage = ref<string | null>(null);
  
  // Container selection (only for deployments)
  const selectedContainerId = ref<string>("");
  const selectedServiceName = ref<string>("");
  const containers = ref<Array<{ containerId: string; serviceName?: string; status?: string }>>([]);

  function setOrganizationId(id: string) {
    if (isDeployment) {
      currentOrgId.value = id ?? "";
    }
  }

  function getOrgId() {
    return isDeployment ? currentOrgId.value ?? "" : "";
  }

  async function fetchVolumes() {
    try {
      if (isDeployment) {
        const orgId = getOrgId();
        if (!orgId) {
          console.warn("fetchVolumes: No organizationId available");
          volumes.value = [];
          containerRunning.value = false;
          switchToContainer();
          return;
        }
        const res = await deploymentClient!.listContainerFiles({
          organizationId: orgId,
          deploymentId: (options as DeploymentExplorerOptions).deploymentId,
          path: "/",
          listVolumes: true,
        });
        
        volumes.value = (res.volumes ?? []) as VolumeType[];
        containerRunning.value = !!res.containerRunning;
        
        const firstVolume = volumes.value[0] as VolumeInfo | undefined;
        if (!containerRunning.value && firstVolume) {
          switchToVolume(firstVolume.name ?? "");
        } else {
          source.type = "container";
          delete source.volumeName;
        }
      } else {
        const res = await gameServerClient!.listGameServerFiles({
          gameServerId: (options as GameServerExplorerOptions).gameServerId,
          path: "/",
          listVolumes: true,
        });

        volumes.value = (res.volumes ?? []) as VolumeType[];
        containerRunning.value = !!res.containerRunning;

        // For game servers, prefer volume view (where server files are stored)
        // Default to the first volume if available, otherwise use container filesystem
        const firstVolume = volumes.value[0] as GameServerVolumeInfo | undefined;
        if (firstVolume) {
          // Check if this volume is mounted at /data (standard for game servers)
          const isDataVolume = firstVolume.mountPoint === "/data" || 
                               (volumes.value.length === 1 && firstVolume.mountPoint?.includes("data"));
          if (isDataVolume || !containerRunning.value) {
          switchToVolume(firstVolume.name ?? "");
          } else {
            source.type = "container";
            delete source.volumeName;
          }
        } else {
          source.type = "container";
          delete source.volumeName;
        }
      }
    } catch (err) {
      console.error("fetchVolumes error:", err);
      volumes.value = [];
      containerRunning.value = false;
    }
  }

  function switchToVolume(name: string) {
    source.type = "volume";
    source.volumeName = name;
  }

  function switchToContainer() {
    source.type = "container";
    delete source.volumeName;
  }

  function convertFile(file: FileType): ExplorerNode {
    const isDirectory = Boolean(file.isDirectory);
    const isSymlink = Boolean(file.isSymlink);

    const type: ExplorerNode["type"] = isDirectory
      ? "directory"
      : isSymlink ? "symlink" : "file";
    const size = file.size !== undefined && file.size !== null ? Number(file.size) : undefined;
    return {
      id: file.path,
      name: file.name || file.path.split("/").pop() || file.path,
      path: file.path,
      parentPath: file.path.split("/").slice(0, -1).join("/") || "/",
      type,
      symlinkTarget: file.symlinkTarget || undefined,
      size,
      owner: file.owner || undefined,
      group: file.group || undefined,
      mode: file.modeOctal ?? undefined,
      mimeType: file.mimeType || undefined,
      modifiedTime: timestampToIso(file.modifiedTime),
      createdTime: timestampToIso(file.createdTime),
      volumeName: file.volumeName || undefined,
      children: [],
      isLoading: false,
      hasLoaded: false,
      hasMore: false,
      nextCursor: null,
      isExpanded: false,
    };
  }

  function findNode(path: string, current: ExplorerNode = root): ExplorerNode | null {
    if (current.path === path) return current;
    if (!current.children) return null;
    for (const child of current.children) {
      const match = findNode(path, child);
      if (match) return match;
    }
    return null;
  }

  async function loadChildren(node: ExplorerNode, cursor?: string) {
    node.isLoading = true;
    errorMessage.value = null;
    try {
      if (isDeployment) {
        const orgId = getOrgId();
        if (!orgId) {
          node.children = [];
          node.hasMore = false;
          node.nextCursor = null;
          node.hasLoaded = true;
          return;
        }
        const res = await deploymentClient!.listContainerFiles({
          organizationId: orgId,
          deploymentId: (options as DeploymentExplorerOptions).deploymentId,
          path: node.path,
          cursor,
          pageSize: 200,
          volumeName: source.type === "volume" ? source.volumeName : undefined,
          containerId: source.type === "container" && selectedContainerId.value ? selectedContainerId.value : undefined,
          serviceName: source.type === "container" && selectedServiceName.value ? selectedServiceName.value : undefined,
        });
        containerRunning.value = !!res.containerRunning;
        const children = (res.files ?? []).map(convertFile);
        if (!cursor) {
          node.children = sortNodes(children);
        } else {
          node.children = node.children ? [...node.children, ...children] : [...children];
          sortNodes(node.children);
        }
        node.hasMore = !!res.hasMore;
        node.nextCursor = res.nextCursor ?? null;
        node.hasLoaded = true;
      } else {
        const res = await gameServerClient!.listGameServerFiles({
          gameServerId: (options as GameServerExplorerOptions).gameServerId,
          path: node.path,
          cursor,
          pageSize: 200,
          volumeName: source.type === "volume" ? source.volumeName : undefined,
        });
        containerRunning.value = !!res.containerRunning;
        const children = (res.files ?? []).map(convertFile);
        if (!cursor) {
          node.children = sortNodes(children);
        } else {
          node.children = node.children ? [...node.children, ...children] : [...children];
          sortNodes(node.children);
        }
        node.hasMore = !!res.hasMore;
        node.nextCursor = res.nextCursor ?? null;
        node.hasLoaded = true;
      }
    } catch (err: any) {
      errorMessage.value = err?.message ?? "Failed to load files";
    } finally {
      node.isLoading = false;
    }
  }

  async function refreshRoot() {
    isLoadingTree.value = true;
    try {
      if (isDeployment && !getOrgId()) {
        root.children = [];
        root.hasMore = false;
        root.nextCursor = null;
        root.hasLoaded = false;
        return;
      }
      await loadChildren(root);
    } finally {
      isLoadingTree.value = false;
    }
  }

  async function deleteEntries(paths: string[]) {
    if (isDeployment && !getOrgId()) return;
    
    // Optimistically remove files from tree immediately for better UX
    const deletedNodes: Array<{ node: ExplorerNode | null; parent: ExplorerNode | null }> = [];
    for (const path of paths) {
      const node = findNode(path);
      if (node) {
        const parentPath = node.parentPath || "/";
        const parent = findNode(parentPath);
        if (parent && parent.children) {
          parent.children = parent.children.filter(child => child.path !== path);
          deletedNodes.push({ node, parent });
        }
      }
    }

    try {
      if (isDeployment) {
        const payload = create(DeleteContainerEntriesRequestSchema, {
          organizationId: getOrgId(),
          deploymentId: (options as DeploymentExplorerOptions).deploymentId,
          paths,
          volumeName: source.type === "volume" ? source.volumeName : undefined,
          recursive: true,
          force: true,
        });
        const res = await deploymentClient!.deleteContainerEntries(payload);
        if (!res.success && res.errors?.length) {
          // Revert optimistic update on error
          for (const { node, parent } of deletedNodes) {
            if (node && parent && parent.children) {
              parent.children.push(node);
              sortNodes(parent.children);
            }
          }
          throw new Error(res.errors.map((e) => e.message || "Unknown error").join("\n"));
        }
      } else {
        const payload = create(DeleteGameServerEntriesRequestSchema, {
          gameServerId: (options as GameServerExplorerOptions).gameServerId,
          paths,
          volumeName: source.type === "volume" ? source.volumeName : undefined,
          recursive: true,
          force: true,
        });
        const res = await gameServerClient!.deleteGameServerEntries(payload);
        if (!res.success && res.errors?.length) {
          // Revert optimistic update on error
          for (const { node, parent } of deletedNodes) {
            if (node && parent && parent.children) {
              parent.children.push(node);
              sortNodes(parent.children);
            }
          }
          throw new Error(res.errors.map((e) => e.message || "Unknown error").join("\n"));
        }
      }

      // Refresh parent directories to ensure consistency
      const refreshedParents = new Set<string>();
      for (const { parent } of deletedNodes) {
        if (parent && !refreshedParents.has(parent.path)) {
          refreshedParents.add(parent.path);
          await loadChildren(parent);
        }
      }
    } catch (error) {
      // If error and optimistic update was reverted, refresh all affected parents
      const refreshedParents = new Set<string>();
      for (const { parent } of deletedNodes) {
        if (parent && !refreshedParents.has(parent.path)) {
          refreshedParents.add(parent.path);
          await loadChildren(parent);
        }
      }
      throw error;
    }
  }

  async function renameEntry(payload: RenameEntryInput) {
    if (isDeployment && !getOrgId()) return;
    
    if (isDeployment) {
      const request = create(RenameContainerEntryRequestSchema, {
        ...(payload as Partial<Omit<RenameContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>),
        organizationId: getOrgId(),
        deploymentId: (options as DeploymentExplorerOptions).deploymentId,
      });
      const res = await deploymentClient!.renameContainerEntry(request);
      if (!res.success) {
        throw new Error("Rename failed");
      }
    } else {
      const request = create(RenameGameServerEntryRequestSchema, {
        ...(payload as Partial<Omit<RenameGameServerEntryRequest, "$typeName" | "$unknown" | "gameServerId">>),
        gameServerId: (options as GameServerExplorerOptions).gameServerId,
      });
      const res = await gameServerClient!.renameGameServerEntry(request);
      if (!res.success) {
        throw new Error("Rename failed");
      }
    }

    // Refresh the parent directory instead of entire root
    const sourcePath = (payload as any).sourcePath;
    if (sourcePath) {
      const parentPath = sourcePath.substring(0, sourcePath.lastIndexOf("/")) || "/";
      const parent = findNode(parentPath);
      if (parent) {
        await loadChildren(parent);
      } else {
        // Fallback to root if parent not found
        await refreshRoot();
      }
    } else {
      await refreshRoot();
    }
  }

  async function createEntry(payload: CreateEntryInput) {
    if (isDeployment && !getOrgId()) return;
    
    if (isDeployment) {
      const request = create(CreateContainerEntryRequestSchema, {
        ...(payload as Partial<Omit<CreateContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>),
        organizationId: getOrgId(),
        deploymentId: (options as DeploymentExplorerOptions).deploymentId,
      });
      await deploymentClient!.createContainerEntry(request);
    } else {
      const request = create(CreateGameServerEntryRequestSchema, {
        ...(payload as Partial<Omit<CreateGameServerEntryRequest, "$typeName" | "$unknown" | "gameServerId">>),
        gameServerId: (options as GameServerExplorerOptions).gameServerId,
      });
      await gameServerClient!.createGameServerEntry(request);
    }

    // Refresh the parent directory instead of entire root
    const parentPath = (payload as any).parentPath || "/";
    const parent = findNode(parentPath);
    if (parent) {
      await loadChildren(parent);
    } else {
      // Fallback to root if parent not found
      await refreshRoot();
    }
  }

  async function writeFile(payload: WriteFileInput) {
    if (isDeployment && !getOrgId()) return;
    
    if (isDeployment) {
      const request = create(WriteContainerFileRequestSchema, {
        ...(payload as Partial<Omit<WriteContainerFileRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">>),
        organizationId: getOrgId(),
        deploymentId: (options as DeploymentExplorerOptions).deploymentId,
      });
      await deploymentClient!.writeContainerFile(request);
    } else {
      const request = create(WriteGameServerFileRequestSchema, {
        ...(payload as Partial<Omit<WriteGameServerFileRequest, "$typeName" | "$unknown" | "gameServerId">>),
        gameServerId: (options as GameServerExplorerOptions).gameServerId,
      });
      await gameServerClient!.writeGameServerFile(request);
    }
    // Don't refresh the tree after saving - it causes the file to close
    // The file was just saved, so we know it exists. Metadata refresh can happen on next manual refresh.
    // This preserves the selectedPath and keeps the file open.
  }

  const breadcrumbs = computed(() => {
    if (!selectedPath.value) return [];
    const segments = selectedPath.value.split(/\//).filter(Boolean);
    const parts = [] as Array<{ name: string; path: string }>;
    let current = "";
    for (const segment of segments) {
      current = `${current}/${segment}`;
      parts.push({ name: segment, path: current || "/" });
    }
    return parts;
  });

  // Load containers for deployments
  async function loadContainers() {
    if (!isDeployment) return;
    
    try {
      const orgId = getOrgId();
      if (!orgId) {
        containers.value = [];
        return;
      }
      const res = await (deploymentClient as any).listDeploymentContainers({
        deploymentId: (options as DeploymentExplorerOptions).deploymentId,
        organizationId: orgId,
      });
      
      if (res?.containers) {
        containers.value = res.containers.map((c: any) => ({
          containerId: c.containerId,
          serviceName: c.serviceName || undefined,
          status: c.status,
        }));
      }
    } catch (err) {
      console.error("Failed to load containers:", err);
      containers.value = [];
    }
  }

  function setContainer(containerId?: string, serviceName?: string) {
    if (!isDeployment) return;
    selectedContainerId.value = containerId || "";
    selectedServiceName.value = serviceName || "";
  }

  return {
    root,
    volumes,
    source,
    containerRunning,
    selectedPath,
    breadcrumbs,
    errorMessage,
    isLoadingTree,
    // Container selection (only populated for deployments)
    containers: isDeployment ? computed(() => containers.value) : computed(() => []),
    selectedContainerId: isDeployment ? computed(() => selectedContainerId.value) : computed(() => ""),
    selectedServiceName: isDeployment ? computed(() => selectedServiceName.value) : computed(() => ""),
    fetchVolumes,
    loadContainers: isDeployment ? loadContainers : () => Promise.resolve(),
    setContainer: isDeployment ? setContainer : () => {},
    switchToVolume,
    switchToContainer,
    loadChildren,
    refreshRoot,
    findNode,
    deleteEntries,
    renameEntry,
    createEntry,
    writeFile,
    getOrgId,
    setOrganizationId: isDeployment ? setOrganizationId : () => {},
  };
}

function timestampToIso(value: any): string | undefined {
  if (!value) return undefined;
  if (typeof value === "string") return value;
  if (value instanceof Date) return value.toISOString();
  if (typeof value === "object") {
    if (typeof value.toDate === "function") {
      return value.toDate().toISOString();
    }
    if (value.seconds !== undefined) {
      const millis = Number(value.seconds) * 1000 + Math.floor((value.nanos ?? 0) / 1e6);
      return new Date(millis).toISOString();
    }
  }
  return undefined;
}

function sortNodes(nodes: ExplorerNode[]): ExplorerNode[] {
  return nodes.sort((a, b) => {
    if (a.type === "directory" && b.type !== "directory") return -1;
    if (a.type !== "directory" && b.type === "directory") return 1;
    return a.name.localeCompare(b.name, undefined, { sensitivity: "base" });
  });
}

