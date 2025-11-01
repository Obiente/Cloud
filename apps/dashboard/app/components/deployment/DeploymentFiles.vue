<template>
  <OuiCardBody class="file-explorer">
    <div class="file-explorer__toolbar">
      <OuiFlex gap="sm" align="center">
        <OuiBreadcrumbs>
          <OuiBreadcrumbItem>
            <OuiBreadcrumbLink @click.prevent="handleBreadcrumbClick('/')"
              >Root</OuiBreadcrumbLink
            >
          </OuiBreadcrumbItem>
          <template v-for="crumb in breadcrumbs" :key="crumb.path">
            <OuiBreadcrumbSeparator />
            <OuiBreadcrumbItem>
              <OuiBreadcrumbLink
                :aria-current="crumb.path === selectedPath ? 'page' : undefined"
                @click.prevent="handleBreadcrumbClick(crumb.path)"
              >
                {{ crumb.name }}
              </OuiBreadcrumbLink>
            </OuiBreadcrumbItem>
          </template>
        </OuiBreadcrumbs>
      </OuiFlex>

      <OuiFlex gap="sm" align="center">
        <OuiMenu>
          <template #trigger>
            <OuiButton
              variant="ghost"
              size="sm"
              :disabled="!currentNode || currentNode.type !== 'directory'"
            >
              New
            </OuiButton>
          </template>
          <template #default>
            <OuiMenuItem value="new-file" @select="() => handleCreate('file')">
              New File
            </OuiMenuItem>
            <OuiMenuItem value="new-folder" @select="() => handleCreate('directory')">
              New Folder
            </OuiMenuItem>
            <OuiMenuItem value="new-symlink" @select="() => handleCreate('symlink')">
              New Symlink
            </OuiMenuItem>
          </template>
        </OuiMenu>
        <OuiButton
          variant="ghost"
          size="sm"
          :loading="isLoadingTree"
          @click="refreshRoot"
        >
          Refresh
        </OuiButton>
        <OuiButton variant="ghost" size="sm" @click="showUpload = !showUpload">
          Upload
        </OuiButton>
      </OuiFlex>
    </div>

    <transition name="fade">
      <OuiCard
        v-if="showUpload"
        variant="outline"
        class="file-explorer__uploader"
      >
        <OuiCardBody>
          <FileUploader
            :deployment-id="deploymentId"
            @uploaded="handleFilesUploaded"
          />
        </OuiCardBody>
      </OuiCard>
    </transition>

    <div class="file-explorer__content">
      <aside class="file-explorer__tree" aria-label="File tree">
        <div class="file-explorer__sources">
          <OuiText size="xs" weight="semibold" class="sources-title"
            >Sources</OuiText
          >
          <nav class="sources-nav">
            <button
              class="source-button"
              :class="{ 'is-active': explorer.source.type === 'container' }"
              :disabled="!containerRunning"
              @click="handleSwitchSource('container')"
            >
              <ServerIcon class="h-4 w-4" />
              <span>Container filesystem</span>
            </button>
            <button
              v-for="volume in volumes"
              :key="volume.name"
              class="source-button"
              :class="{
                'is-active': explorer.source.volumeName === volume.name,
              }"
              @click="handleSwitchSource('volume', volume.name || '')"
            >
              <CubeIcon class="h-4 w-4" />
              <span>{{ volume.name }}</span>
              <span class="muted">{{ volume.mountPoint }}</span>
            </button>
          </nav>
        </div>

        <div class="file-explorer__tree-scroll" role="tree">
          <template v-if="errorMessage">
            <OuiText color="danger" size="sm">{{ errorMessage }}</OuiText>
          </template>
          <template v-else-if="root.children.length === 0 && isLoadingTree">
            <OuiFlex
              direction="col"
              align="center"
              gap="sm"
              class="tree-empty"
            >
              <ArrowPathIcon class="h-5 w-5 animate-spin" />
              <OuiText size="sm" color="secondary">Loading files…</OuiText>
            </OuiFlex>
          </template>
          <template v-else-if="root.children.length === 0">
            <OuiFlex
              direction="col"
              align="center"
              gap="sm"
              class="tree-empty"
            >
              <OuiText size="sm" color="secondary">No files found</OuiText>
            </OuiFlex>
          </template>
          <template v-else>
            <TreeNode
              v-for="(child, idx) in root.children"
              :key="child.id"
              :node="child"
              :indexPath="[idx]"
              :selectedPath="selectedPath"
              :allowEditing="allowEditing"
              @toggle="handleToggle"
              @open="handleOpen"
              @action="handleContextAction"
              @load-more="handleLoadMore"
            />
          </template>
        </div>
      </aside>

      <section class="file-explorer__viewer">
        <header class="viewer-header">
          <div class="viewer-header__meta">
            <OuiText size="sm" weight="semibold">
              {{ currentNode?.name || "Preview" }}
            </OuiText>
            <OuiFlex gap="sm" align="center" class="viewer-meta">
              <span v-if="currentNode?.type === 'symlink'" class="badge">
                <LinkIcon class="h-3.5 w-3.5" />
                {{ currentNode.symlinkTarget }}
              </span>
              <span v-if="currentNode?.mimeType" class="badge">{{
                currentNode.mimeType
              }}</span>
              <span v-if="currentNode?.owner" class="badge"
                >Owner: {{ currentNode.owner }}</span
              >
              <span v-if="currentNode?.group" class="badge"
                >Group: {{ currentNode.group }}</span
              >
              <span v-if="currentNode?.mode" class="badge"
                >Mode: {{ currentNode.mode.toString(8) }}</span
              >
              <span v-if="currentNode?.modifiedTime" class="badge"
                >Modified: {{ formatDatetime(currentNode.modifiedTime) }}</span
              >
              <span v-if="currentNode?.createdTime" class="badge"
                >Created: {{ formatDatetime(currentNode.createdTime) }}</span
              >
            </OuiFlex>
          </div>
          <OuiFlex gap="sm" align="center">
            <OuiButton
              variant="ghost"
              size="sm"
              :disabled="!currentNode || currentNode.type !== 'file'"
              @click="handleDownload"
            >
              Download
            </OuiButton>
            <OuiMenu v-if="currentNode">
              <template #trigger>
                <OuiButton variant="ghost" size="sm">More</OuiButton>
              </template>
              <OuiMenuItem value="refresh" @select="handleRefreshSelection"
                >Refresh</OuiMenuItem
              >
              <OuiMenuItem
                value="rename"
                @select="() => currentNode && queueRename(currentNode)"
                >Rename</OuiMenuItem
              >
              <OuiMenuSeparator />
              <OuiMenuItem
                value="delete"
                @select="() => currentNode && queueDelete([currentNode.path])"
              >
                Delete
              </OuiMenuItem>
            </OuiMenu>
          </OuiFlex>
        </header>

        <div class="viewer-body" role="tabpanel">
          <div v-if="!selectedPath" class="viewer-empty">
            <OuiText size="sm" color="secondary"
              >Select a file to view its contents</OuiText
            >
          </div>
          <div v-else class="viewer-editor" ref="editorContainer"></div>
        </div>
      </section>
    </div>
  </OuiCardBody>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref, watch } from "vue";
  import {
    ArrowPathIcon,
    ServerIcon,
    CubeIcon,
    LinkIcon,
  } from "@heroicons/vue/24/outline";
  import TreeNode from "./TreeNode.vue";
  import { useFileExplorer } from "~/composables/useFileExplorer";
  import { useConnectClient } from "~/lib/connect-client";
  import { DeploymentService } from "@obiente/proto";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import type { CreateContainerEntryRequest } from "@obiente/proto";
  import { ContainerEntryType } from "@obiente/proto";

  const props = defineProps<{
    deploymentId: string;
    organizationId?: string;
    allowEditing?: boolean;
  }>();

  const monaco = ref<any>(null);
  const editor = ref<any>(null);
  const editorContainer = ref<HTMLDivElement | null>(null);
  const showUpload = ref(false);
  const hasMounted = ref(false);

  const explorer = useFileExplorer({
    organizationId: props.organizationId || "",
    deploymentId: props.deploymentId,
    allowEditing: props.allowEditing,
  });

  const {
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
    refreshRoot,
    loadChildren,
    findNode,
    deleteEntries,
    renameEntry,
    createEntry,
    getOrgId,
    setOrganizationId,
  } = explorer;

  const explorerClient = useConnectClient(DeploymentService);

  const allowEditing = props.allowEditing ?? true;

  const currentNode = computed(() => {
    if (!selectedPath.value) return null;
    return findNode(selectedPath.value) || null;
  });

  function handleSwitchSource(type: "container" | "volume", name?: string) {
    if (type === "container") {
      switchToContainer();
    } else if (name) {
      switchToVolume(name);
    }
    selectedPath.value = null;
    refreshRoot();
  }

  async function handleToggle(node: ExplorerNode) {
    if (node.isLoading) return;
    if (!node.hasLoaded || node.hasMore) {
      await loadChildren(
        node,
        node.hasMore ? node.nextCursor ?? undefined : undefined
      );
    }
    node.hasLoaded = true;
  }

  function handleOpen(node: ExplorerNode) {
    selectedPath.value = node.path;
    if (node.type === "directory") {
      handleToggle(node);
    } else {
      handleLoadFile(node);
    }
  }

  function handleLoadMore(node: ExplorerNode) {
    if (!node.hasMore || node.isLoading) return;
    loadChildren(node, node.nextCursor ?? undefined);
  }

  function handleContextAction(action: string, node: ExplorerNode) {
    switch (action) {
      case "open":
        handleOpen(node);
        break;
      case "open-editor":
        handleLoadFile(node);
        break;
      case "refresh":
        loadChildren(node);
        break;
      case "delete":
        if (!allowEditing) return;
        queueDelete([node.path]);
        break;
      case "rename":
        if (!allowEditing) return;
        queueRename(node);
        break;
      case "copy-path":
        navigator.clipboard
          ?.writeText(node.path)
          .catch((err) => console.error("copy path", err));
        break;
      case "new-file":
        if (!allowEditing) return;
        handleCreate("file");
        break;
      case "new-folder":
        if (!allowEditing) return;
        handleCreate("directory");
        break;
      case "new-symlink":
        if (!allowEditing) return;
        handleCreate("symlink");
        break;
    }
  }

  async function handleCreate(type: "file" | "directory" | "symlink") {
    if (!allowEditing) return;
    const parent = currentNode.value && currentNode.value.type === "directory" ? currentNode.value.path : "/";
    const name = prompt(`Name for new ${type}`);
    if (!name) return;
    let entryType = ContainerEntryType.FILE;
    if (type === "directory") entryType = ContainerEntryType.DIRECTORY;
    if (type === "symlink") entryType = ContainerEntryType.SYMLINK;

    const payload: Partial<CreateContainerEntryRequest> = {
      parentPath: parent,
      name,
      type: entryType,
      modeOctal: type === "directory" ? 0o755 : 0o644,
      volumeName: source.type === "volume" ? source.volumeName : undefined,
    };

    if (type === "symlink") {
      const target = prompt("Path to link to?")?.trim();
      if (!target) return;
      payload.template = target;
    }

    await createEntry(payload);
  }

  async function queueDelete(paths: string[]) {
    if (!allowEditing) return;
    if (!confirm(`Delete ${paths.length} item(s)?`)) return;
    await deleteEntries(paths);
    selectedPath.value = null;
  }

  async function queueRename(node: ExplorerNode) {
    if (!allowEditing) return;
    const target = prompt("New name", node.name);
    if (!target || target === node.name) return;
    const targetPath =
      `${node.parentPath === "/" ? "" : node.parentPath}/${target}` || target;
    await renameEntry({
      sourcePath: node.path,
      targetPath,
      overwrite: false,
      volumeName: source.type === "volume" ? source.volumeName : undefined,
    });
    selectedPath.value = targetPath;
  }

  async function handleLoadFile(node: ExplorerNode) {
    if (node.type !== "file") return;
    selectedPath.value = node.path;
    try {
      const res = await explorerClient.getContainerFile({
        organizationId: getOrgId(),
        deploymentId: props.deploymentId,
        path: node.path,
        volumeName: source.type === "volume" ? source.volumeName : undefined,
      });
      const content = res.content || "";
      const language = detectLanguage(node.path);
      await mountEditor();
      editor.value?.setValue(content);
      editor.value?.updateOptions({ readOnly: !allowEditing });
      if (monaco.value && editor.value) {
        monaco.value.editor.setModelLanguage(editor.value.getModel(), language);
      }
    } catch (err) {
      console.error("load file", err);
    }
  }

  function detectLanguage(path: string) {
    const ext = path.split(".").pop()?.toLowerCase();
    const map: Record<string, string> = {
      js: "javascript",
      ts: "typescript",
      jsx: "javascript",
      tsx: "typescript",
      py: "python",
      go: "go",
      rs: "rust",
      java: "java",
      sh: "shell",
      bash: "shell",
      zsh: "shell",
      fish: "shell",
      yaml: "yaml",
      yml: "yaml",
      json: "json",
      html: "html",
      css: "css",
      vue: "html",
      md: "markdown",
      sql: "sql",
    };
    return map[ext || ""] || "plaintext";
  }

  async function mountEditor() {
    if (!editorContainer.value) return;
    if (!monaco.value) {
      const monacoModule = await import("monaco-editor");
      monaco.value = monacoModule;
    }
    if (editor.value) {
      return;
    }
    editor.value = monaco.value.editor.create(editorContainer.value, {
      value: "",
      language: "plaintext",
      theme: "vs-dark",
      automaticLayout: true,
      minimap: { enabled: true },
      readOnly: !allowEditing,
    });
  }

  function handleBreadcrumbClick(path: string) {
    const node = explorer.findNode(path);
    if (node) handleOpen(node);
  }

  function handleRefreshSelection() {
    if (!currentNode.value) return;
    loadChildren(currentNode.value);
  }

  function handleDownload() {
    if (!currentNode.value || currentNode.value.type !== "file") return;
    alert("Download not yet implemented");
  }

  function formatDatetime(value?: string) {
    if (!value) return "";
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(value));
  }

  async function handleFilesUploaded() {
    showUpload.value = false;
    await refreshRoot();
  }

  watch(
    () => props.organizationId,
    async (newOrgId) => {
      setOrganizationId(newOrgId || "");
      if (!hasMounted.value || !newOrgId) return;
      await fetchVolumes();
      await refreshRoot();
    },
    { immediate: true }
  );

  onMounted(async () => {
    hasMounted.value = true;
    await fetchVolumes();
    await refreshRoot();
  });

  onUnmounted(() => {
    editor.value?.dispose();
    editor.value = null;
  });
</script>

<style scoped>
  .file-explorer {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .file-explorer__toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .file-explorer__content {
    display: grid;
    grid-template-columns: 260px minmax(0, 1fr);
    gap: 16px;
    min-height: 520px;
  }

  .file-explorer__tree {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--oui-border-default);
    border-radius: 10px;
    background: var(--oui-surface-base);
    overflow: hidden;
  }

  .file-explorer__sources {
    padding: 12px;
    border-bottom: 1px solid var(--oui-border-default);
  }

  .sources-title {
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 11px;
    margin-bottom: 8px;
  }

  .sources-nav {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .source-button {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: 6px;
    font-size: 13px;
    text-align: left;
    transition: background-color 0.12s ease, color 0.12s ease;
    color: var(--oui-text-secondary);
    border: none;
    background: transparent;
    cursor: pointer;
  }

  .source-button .muted {
    margin-left: auto;
    font-size: 11px;
    color: var(--oui-text-tertiary);
  }

  .source-button:hover:not(:disabled) {
    background: var(--oui-surface-hover);
    color: var(--oui-text-primary);
  }

  .source-button.is-active {
    background: var(--oui-surface-selected);
    color: var(--oui-text-primary);
  }

  .source-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .file-explorer__tree-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    font-family: var(--oui-font-mono);
  }

  .tree-empty {
    padding: 24px 8px;
    text-align: center;
  }

  .file-explorer__viewer {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--oui-border-default);
    border-radius: 10px;
    background: var(--oui-surface-base);
  }

  .viewer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--oui-border-default);
  }

  .viewer-header__meta {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .viewer-meta {
    flex-wrap: wrap;
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 6px;
    font-size: 11px;
    border-radius: 6px;
    background: var(--oui-surface-subtle);
    color: var(--oui-text-secondary);
  }

  .viewer-body {
    flex: 1;
    position: relative;
    min-height: 400px;
  }

  .viewer-empty {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--oui-text-tertiary);
  }

  .viewer-editor {
    position: absolute;
    inset: 0;
  }

  .file-explorer__uploader {
    animation: fade-in 0.2s ease;
  }

  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.2s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  @keyframes fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
