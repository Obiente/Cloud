import { computed, reactive, ref } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import {
  DeploymentService,
  DeleteContainerEntriesRequestSchema,
  RenameContainerEntryRequestSchema,
  CreateContainerEntryRequestSchema,
  WriteContainerFileRequestSchema,
} from "@obiente/proto";
import type {
  ContainerFile,
  VolumeInfo,
  RenameContainerEntryRequest,
  CreateContainerEntryRequest,
  WriteContainerFileRequest,
} from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import type { ExplorerNode } from "../components/deployment/fileExplorerTypes";

interface SourceState {
  type: "container" | "volume";
  volumeName?: string;
}

interface ExplorerOptions {
  organizationId: string;
  deploymentId: string;
  allowEditing?: boolean;
}

type RenameEntryInput = Partial<
  Omit<RenameContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">
>;

type CreateEntryInput = Partial<
  Omit<CreateContainerEntryRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">
>;

type WriteFileInput = Partial<
  Omit<WriteContainerFileRequest, "$typeName" | "$unknown" | "organizationId" | "deploymentId">
>;

export function useFileExplorer(options: ExplorerOptions) {
  const client = useConnectClient(DeploymentService);
  const currentOrgId = ref(options.organizationId ?? "");
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
  const volumes = ref<VolumeInfo[]>([]);
  const containerRunning = ref(false);
  const selectedPath = ref<string | null>(null);
  const isLoadingTree = ref(false);
  const errorMessage = ref<string | null>(null);

  function setOrganizationId(id: string) {
    currentOrgId.value = id ?? "";
  }

  function getOrgId() {
    return currentOrgId.value ?? "";
  }

  async function fetchVolumes() {
    try {
      const orgId = getOrgId();
      if (!orgId) {
        console.warn("fetchVolumes: No organizationId available");
        volumes.value = [];
        containerRunning.value = false;
        switchToContainer();
        return;
      }
      const res = await client.listContainerFiles({
        organizationId: orgId,
        deploymentId: options.deploymentId,
        path: "/",
        listVolumes: true,
      });
      
      console.log("fetchVolumes: API response", res);
      console.log("fetchVolumes: res.volumes", res.volumes);
      console.log("fetchVolumes: res.volumes type", typeof res.volumes, Array.isArray(res.volumes));
      
      volumes.value = res.volumes ?? [];
      containerRunning.value = !!res.containerRunning;
      
      console.log("fetchVolumes: Found volumes", volumes.value.length, volumes.value);
      console.log("fetchVolumes: volumes.value after assignment", volumes.value);
      console.log("fetchVolumes: Container running", containerRunning.value);
      
      const firstVolume = volumes.value[0];
      if (!containerRunning.value && firstVolume) {
        switchToVolume(firstVolume.name ?? "");
      } else {
        source.type = "container";
        delete source.volumeName;
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

  function convertFile(file: ContainerFile): ExplorerNode {
    // Convert the file to an ExplorerNode, ensuring directories are correctly identified
    // Use explicit boolean check to handle any potential type coercion issues
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
      const orgId = getOrgId();
      if (!orgId) {
        node.children = [];
        node.hasMore = false;
        node.nextCursor = null;
        node.hasLoaded = true;
        return;
      }
      const res = await client.listContainerFiles({
        organizationId: orgId,
        deploymentId: options.deploymentId,
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
    } catch (err: any) {
      errorMessage.value = err?.message ?? "Failed to load files";
    } finally {
      node.isLoading = false;
    }
  }

  async function refreshRoot() {
    isLoadingTree.value = true;
    try {
      if (!getOrgId()) {
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
    if (!getOrgId()) return;
    const payload = create(DeleteContainerEntriesRequestSchema, {
      organizationId: getOrgId(),
      deploymentId: options.deploymentId,
      paths,
      volumeName: source.type === "volume" ? source.volumeName : undefined,
      recursive: true,
      force: true,
    });
    const res = await client.deleteContainerEntries(payload);
    if (!res.success && res.errors?.length) {
      throw new Error(res.errors.map((e) => e.message || "Unknown error").join("\n"));
    }
    await refreshRoot();
  }

  async function renameEntry(payload: RenameEntryInput) {
    if (!getOrgId()) return;
    const request = create(RenameContainerEntryRequestSchema, {
      ...payload,
      organizationId: getOrgId(),
      deploymentId: options.deploymentId,
    });
    const res = await client.renameContainerEntry(request);
    if (!res.success) {
      throw new Error("Rename failed");
    }
    await refreshRoot();
  }

  async function createEntry(payload: CreateEntryInput) {
    if (!getOrgId()) return;
    const request = create(CreateContainerEntryRequestSchema, {
      ...payload,
      organizationId: getOrgId(),
      deploymentId: options.deploymentId,
    });
    await client.createContainerEntry(request);
    await refreshRoot();
  }

  async function writeFile(payload: WriteFileInput) {
    if (!getOrgId()) return;
    const request = create(WriteContainerFileRequestSchema, {
      ...payload,
      organizationId: getOrgId(),
      deploymentId: options.deploymentId,
    });
    await client.writeContainerFile(request);
    await refreshRoot();
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

  return {
    root,
    volumes,
    source,
    containerRunning,
    selectedPath,
    breadcrumbs,
    errorMessage,
    isLoadingTree,
    fetchVolumes,
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
    setOrganizationId,
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
