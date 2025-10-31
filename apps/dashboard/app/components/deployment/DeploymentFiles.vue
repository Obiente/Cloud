<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">File Browser</OuiText>
        <OuiFlex gap="sm">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="refreshFiles"
            :disabled="isLoading"
          >
            <ArrowPathIcon
              class="h-4 w-4"
              :class="{ 'animate-spin': isLoading }"
            />
            Refresh
          </OuiButton>
          <OuiButton variant="ghost" size="sm" @click="showUpload = !showUpload">
            Upload
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- File Upload Section -->
      <OuiCard v-if="showUpload" variant="outline">
        <OuiCardBody>
          <FileUploader
            :deployment-id="deploymentId"
            @uploaded="handleFilesUploaded"
          />
        </OuiCardBody>
      </OuiCard>

      <!-- Split Layout: Tree View + Editor -->
      <div class="flex gap-4 h-[600px]">
        <!-- Tree View Sidebar -->
        <div class="w-80 border border-border-default rounded-lg overflow-hidden flex flex-col">
          <!-- Source selector -->
          <div class="p-4 border-b border-border-default">
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold">File Sources</OuiText>
              <div class="flex flex-col gap-1">
                <!-- Container filesystem (only if running) -->
                <OuiButton
                  v-if="containerRunning"
                  variant="ghost"
                  size="sm"
                  :class="[
                    'w-full justify-start',
                    selectedSource === 'container' ? 'bg-primary/10 text-primary' : ''
                  ]"
                  @click="selectedSource = 'container'; loadFiles('/')"
                >
                  <ServerIcon class="h-4 w-4 mr-2" />
                  Container Filesystem
                </OuiButton>
                <!-- Volumes -->
                <OuiButton
                  v-for="volume in volumes"
                  :key="volume.name"
                  variant="ghost"
                  size="sm"
                  :class="[
                    'w-full justify-start',
                    selectedSource === `volume-${volume.name}` ? 'bg-primary/10 text-primary' : ''
                  ]"
                  @click="selectedSource = `volume-${volume.name}`; selectedVolumeName = volume.name; loadFiles('/')"
                >
                  <CubeIcon class="h-4 w-4 mr-2" />
                  {{ volume.name }}
                  <span class="ml-auto text-xs text-secondary">{{ volume.mountPoint }}</span>
                </OuiButton>
              </div>
            </OuiStack>
          </div>
          <div v-if="isLoading" class="flex justify-center py-8 flex-1">
            <OuiText color="secondary" size="sm">Loading files...</OuiText>
          </div>
          <div v-else class="flex-1 overflow-auto p-2">
            <TreeView.Root :collection="fileCollection">
              <TreeView.Tree>
                <TreeNode
                  v-for="(node, index) in fileCollection.rootNode.children"
                  :key="node.id"
                  :node="node"
                  :indexPath="[index]"
                  @click="handleNodeClick"
                />
              </TreeView.Tree>
            </TreeView.Root>
          </div>
        </div>

        <!-- Monaco Editor for File Viewing -->
        <div class="flex-1 border border-border-default rounded-lg overflow-hidden">
          <div
            v-if="!selectedFilePath"
            class="h-full flex items-center justify-center"
          >
            <OuiText color="secondary" size="sm">
              Select a file from the tree to view its contents
            </OuiText>
          </div>
          <div v-else class="h-full relative">
            <div
              v-if="isLoadingFile"
              class="absolute inset-0 flex items-center justify-center bg-surface-muted/50 z-10"
            >
              <OuiText color="secondary" size="sm">Loading file...</OuiText>
            </div>
            <div
              v-if="fileError"
              class="p-4 text-danger"
            >
              <OuiText color="danger" size="sm">{{ fileError }}</OuiText>
            </div>
            <div
              ref="editorContainer"
              class="w-full h-full"
            />
          </div>
        </div>
      </div>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import { ArrowPathIcon, ServerIcon, CubeIcon } from "@heroicons/vue/24/outline";
import { TreeView, createTreeCollection } from "@ark-ui/vue/tree-view";
import TreeNode from "./TreeNode.vue";
import FileUploader from "./FileUploader.vue";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  deploymentId: string;
  organizationId?: string;
}

interface FileNode {
  id: string;
  name: string;
  path: string;
  isDirectory: boolean;
  size?: number;
  children?: FileNode[];
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);

const client = useConnectClient(DeploymentService);
const isLoading = ref(false);
const showUpload = ref(false);
const selectedFilePath = ref("");
const isLoadingFile = ref(false);
const fileError = ref("");
const editorContainer = ref<HTMLElement | null>(null);
let editor: any = null;
let monaco: any = null;

// Volume and source selection
const volumes = ref<Array<{ name: string; mountPoint: string; source: string; isPersistent: boolean }>>([]);
const selectedSource = ref<string>("container"); // "container" or "volume-{name}"
const selectedVolumeName = ref<string>("");
const containerRunning = ref(false);

// File tree structure
const fileTree = ref<FileNode>({
  id: "ROOT",
  name: "",
  path: "/",
  isDirectory: true,
  children: [],
});

const fileCollection = computed(() => {
  return createTreeCollection<FileNode>({
    nodeToValue: (node) => node.id,
    nodeToString: (node) => node.name,
    rootNode: fileTree.value,
  });
});

// Build tree structure from flat file list
const buildTree = (files: any[], basePath: string = "/"): FileNode[] => {
  const treeMap = new Map<string, FileNode>();

  for (const file of files) {
    const pathParts = file.path
      .replace(basePath === "/" ? "" : basePath, "")
      .split("/")
      .filter(Boolean);

    let currentPath = basePath === "/" ? "" : basePath;
    let parent: FileNode | null = null;

    for (let i = 0; i < pathParts.length; i++) {
      const part = pathParts[i];
      const isLast = i === pathParts.length - 1;
      currentPath += (currentPath === "" ? "" : "/") + part;

      const nodeId = currentPath;
      if (!treeMap.has(nodeId)) {
        const node: FileNode = {
          id: nodeId,
          name: part,
          path: currentPath,
          isDirectory: !isLast || file.isDirectory,
          size: isLast && !file.isDirectory ? file.size : undefined,
          children: [],
        };
        treeMap.set(nodeId, node);

        if (parent) {
          if (!parent.children) parent.children = [];
          parent.children.push(node);
        }
      }

      parent = treeMap.get(nodeId)!;
    }
  }

  return Array.from(treeMap.values())
    .filter((node) => {
      const pathParts = node.path.split("/").filter(Boolean);
      return pathParts.length === 1;
    })
    .sort((a, b) => {
      // Directories first, then files, then alphabetical
      if (a.isDirectory !== b.isDirectory) {
        return a.isDirectory ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
};

// Load volumes list
const loadVolumes = async () => {
  try {
    const res = await client.listContainerFiles({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      path: "/",
      listVolumes: true,
    });
    volumes.value = res.volumes || [];
    containerRunning.value = res.containerRunning || false;
    
    // Auto-select first volume if container is not running
    const firstVolume = volumes.value[0];
    if (!containerRunning.value && firstVolume) {
      selectedSource.value = `volume-${firstVolume.name}`;
      selectedVolumeName.value = firstVolume.name;
    } else if (containerRunning.value) {
      selectedSource.value = "container";
    } else if (firstVolume) {
      selectedSource.value = `volume-${firstVolume.name}`;
      selectedVolumeName.value = firstVolume.name;
    }
  } catch (error) {
    console.error("Failed to load volumes:", error);
  }
};

const loadFiles = async (path: string = "/") => {
  isLoading.value = true;
  try {
    // Determine if we're loading from a volume or container
    const isVolume = selectedSource.value.startsWith("volume-");
    const volumeName = isVolume ? selectedVolumeName.value : undefined;
    
    const res = await client.listContainerFiles({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      path: path,
      volumeName: volumeName,
    });

    const files = res.files || [];
    containerRunning.value = res.containerRunning || false;
    
    if (path === "/") {
      // Root level - rebuild entire tree
      fileTree.value = {
        id: "ROOT",
        name: "",
        path: "/",
        isDirectory: true,
        children: buildTree(files, "/"),
      };
    } else {
      // Load directory contents and merge into tree
      const newNodes = buildTree(files, path);
      updateTreeNodes(path, newNodes);
    }
  } catch (error) {
    console.error("Failed to load files:", error);
    fileTree.value = {
      id: "ROOT",
      name: "",
      path: "/",
      isDirectory: true,
      children: [],
    };
  } finally {
    isLoading.value = false;
  }
};

// Update tree nodes at a specific path
const updateTreeNodes = (path: string, nodes: FileNode[]) => {
  const pathParts = path.split("/").filter(Boolean);
  let current: FileNode | undefined = fileTree.value;

  for (const part of pathParts) {
    if (!current?.children) return;
    current = current.children.find((c) => c.name === part);
    if (!current) return;
  }

  if (current) {
    current.children = nodes.sort((a, b) => {
      if (a.isDirectory !== b.isDirectory) {
        return a.isDirectory ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
  }
};

const refreshFiles = () => {
  loadFiles("/");
};

const handleNodeClick = async (event: { path: string; isDirectory: boolean }) => {
  if (event.isDirectory) {
    // Load directory contents
    await loadFiles(event.path);
    // Expand the node in the tree (collection will handle this)
  } else {
    // Load and display file
    await loadFile(event.path);
  }
};

const initEditor = async () => {
  if (typeof window === "undefined" || !editorContainer.value) return;

  try {
    if (!monaco) {
      const monacoModule = await import("monaco-editor");
      monaco = monacoModule;
      
      // Register OUI theme
      const { registerOUITheme } = await import("~/utils/monaco-theme");
      registerOUITheme(monaco);
    }

    if (editor) {
      editor.dispose();
    }

    editor = monaco.editor.create(editorContainer.value, {
      value: "",
      language: "plaintext",
      theme: "oui-dark",
      automaticLayout: true,
      fontSize: 14,
      minimap: { enabled: true },
      scrollBeyondLastLine: false,
      wordWrap: "on",
      readOnly: true,
      lineNumbers: "on",
      renderWhitespace: "selection",
      folding: true,
      mouseWheelZoom: true, // Enable zoom with Ctrl+scroll (Cmd+scroll on Mac)
    });
  } catch (err) {
    console.error("Failed to initialize Monaco Editor:", err);
  }
};

const getLanguageFromPath = (path: string): string => {
  const ext = path.split(".").pop()?.toLowerCase() || "";
  const langMap: Record<string, string> = {
    js: "javascript",
    ts: "typescript",
    jsx: "javascript",
    tsx: "typescript",
    py: "python",
    go: "go",
    rs: "rust",
    java: "java",
    sh: "bash",
    yaml: "yaml",
    yml: "yaml",
    json: "json",
    html: "html",
    css: "css",
    md: "markdown",
    sql: "sql",
    dockerfile: "dockerfile",
    vue: "html",
  };
  return langMap[ext] || "plaintext";
};

const loadFile = async (path: string) => {
  selectedFilePath.value = path;
  isLoadingFile.value = true;
  fileError.value = "";

  try {
    // Determine if file is in a volume or container
    const isVolume = selectedSource.value.startsWith("volume-");
    const volumeName = isVolume ? selectedVolumeName.value : undefined;
    
    const res = await client.getContainerFile({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      path: path,
      volumeName: volumeName,
    });

    const content = res.content || "";
    const language = getLanguageFromPath(path);

    await nextTick();
    if (!editor) {
      await initEditor();
    }

    if (editor && monaco) {
      editor.setValue(content);
      monaco.editor.setModelLanguage(editor.getModel(), language);
    }
  } catch (err: any) {
    console.error("Failed to load file:", err);
    fileError.value = err.message || "Failed to load file content";
  } finally {
    isLoadingFile.value = false;
  }
};

const handleFilesUploaded = async (files: File[]) => {
  showUpload.value = false;
  console.log("Files uploaded:", files.map(f => f.name));
  
  // Refresh multiple times with delays to catch async processing
  for (let i = 0; i < 3; i++) {
    await new Promise((resolve) => setTimeout(resolve, 1000 * (i + 1)));
    await loadFiles("/");
  }
  
  // Also try refreshing current directory if we're not at root
  const currentDir = getCurrentDirectory();
  if (currentDir && currentDir !== "/") {
    await loadFiles(currentDir);
  }
};

// Helper to get the currently visible directory
const getCurrentDirectory = (): string => {
  // Try to determine current directory from the tree state
  // For now, just return "/" as we always show root
  return "/";
};

onMounted(async () => {
  await nextTick();
  await loadVolumes();
  await loadFiles("/");
  await initEditor();
});

onUnmounted(() => {
  if (editor) {
    editor.dispose();
    editor = null;
  }
});
</script>
