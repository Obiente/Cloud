<template>
  <OuiCardBody class="flex flex-col gap-4">
    <div class="flex justify-between items-center">
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

      <OuiFlex gap="sm" align="center" wrap="wrap">
        <OuiFlex gap="sm" align="center" v-if="source.type === 'container' && containers.length > 0">
          <OuiText size="xs" color="muted">Container:</OuiText>
          <OuiSelect
            :model-value="selectedServiceName || selectedContainerId || ''"
            :items="containerOptions"
            placeholder="Select container"
            style="min-width: 180px;"
            @update:model-value="handleContainerChange"
          />
        </OuiFlex>
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
            <OuiMenuItem
              value="new-folder"
              @select="() => handleCreate('directory')"
            >
              New Folder
            </OuiMenuItem>
            <OuiMenuItem
              value="new-symlink"
              @select="() => handleCreate('symlink')"
            >
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
        class="animate-[fade-in_0.2s_ease]"
      >
        <OuiCardBody>
          <FileUploader
            :deployment-id="deploymentId"
            :destination-path="currentDirectory"
            :volume-name="source.type === 'volume' ? source.volumeName : undefined"
            @uploaded="handleFilesUploaded"
          />
        </OuiCardBody>
      </OuiCard>
    </transition>

    <div
      class="grid grid-cols-[260px_1fr] gap-4 h-[calc(100vh-220px)] min-h-[calc(100vh-220px)] max-h-[calc(100vh-220px)] overflow-hidden"
    >
      <aside
        class="flex flex-col border border-border-default rounded-[10px] bg-surface-base overflow-hidden"
        aria-label="File tree"
      >
        <div class="p-3 border-b border-border-default">
          <OuiText
            size="xs"
            weight="semibold"
            class="uppercase tracking-[0.08em] text-[11px] mb-2"
            >Sources</OuiText
          >
          <nav class="flex flex-col gap-1.5">
                <button
              class="flex items-center gap-2 px-2.5 py-1.5 rounded-xl text-[13px] text-left transition-colors duration-150 text-text-secondary border-none bg-transparent cursor-pointer hover:bg-surface-hover hover:text-text-primary disabled:opacity-60 disabled:cursor-not-allowed"
              :class="{
                'bg-surface-selected text-text-primary':
                  explorer.source.type === 'container',
              }"
              :disabled="!containerRunning"
              @click="handleSwitchSource('container')"
            >
              <ServerIcon class="h-4 w-4" />
              <span>Container filesystem</span>
                </button>
                <button
                  v-for="volume in volumes"
                  :key="volume.name"
              class="flex items-center gap-2 px-2.5 py-1.5 rounded-xl text-[13px] text-left transition-colors duration-150 text-text-secondary border-none bg-transparent cursor-pointer hover:bg-surface-hover hover:text-text-primary"
              :class="{
                'bg-surface-selected text-text-primary':
                  explorer.source.volumeName === volume.name,
              }"
              @click="handleSwitchSource('volume', volume.name || '')"
            >
              <CubeIcon class="h-4 w-4" />
              <span>{{ volume.name }}</span>
              <span class="ml-auto text-[11px] text-text-tertiary">{{
                volume.mountPoint
              }}</span>
                </button>
          </nav>
              </div>

        <div class="flex-1 overflow-y-auto font-mono" role="tree">
          <div class="p-2">
            <div
              v-if="errorMessage"
              class="mb-2 p-2 rounded-xl bg-danger/10 border border-danger/30"
            >
              <div class="flex items-start gap-2">
                <ExclamationTriangleIcon
                  class="h-4 w-4 text-danger shrink-0 mt-0.5"
                />
                <div class="flex-1 min-w-0">
                  <OuiText
                    size="xs"
                    weight="semibold"
                    color="danger"
                    class="block mb-0.5"
                  >
                    Error loading files
                  </OuiText>
                  <OuiText size="xs" color="secondary" class="wrap-break-word">
                    {{ parseTreeError(errorMessage) }}
                  </OuiText>
          </div>
                <OuiButton
                  variant="ghost"
                  size="xs"
                  class="shrink-0 -mt-1 -mr-1"
                  @click="errorMessage = null"
                >
                  <XMarkIcon class="h-3.5 w-3.5" />
                </OuiButton>
          </div>
            </div>
            <template v-if="root.children.length === 0 && isLoadingTree">
            <OuiFlex direction="col" align="center" gap="sm" class="tree-empty">
              <ArrowPathIcon class="h-5 w-5 animate-spin" />
              <OuiText size="sm" color="secondary">Loading files…</OuiText>
            </OuiFlex>
          </template>
          <template v-else-if="root.children.length === 0">
            <OuiFlex direction="col" align="center" gap="sm" class="tree-empty">
              <OuiText size="sm" color="secondary">No files found</OuiText>
            </OuiFlex>
          </template>
          <template v-else>
            <TreeView.Root
              :collection="treeCollection"
              selection-mode="single"
              :selected-value="selectedPath ? [selectedPath] : []"
              class="file-tree-root"
            >
              <TreeView.Tree>
                <TreeNode
                  v-for="(child, idx) in root.children"
                  :key="child.id"
                  :node="child"
                  :indexPath="[idx]"
                  :selectedPath="selectedPath"
                  :allowEditing="allowEditing"
                  @toggle="(node, open) => handleToggle(node, open)"
                  @open="(node, options) => handleOpen(node, options)"
                  @action="handleContextAction"
                  @load-more="handleLoadMore"
                />
              </TreeView.Tree>
            </TreeView.Root>
          </template>
          </div>
        </div>
      </aside>

      <section
        class="flex flex-col border border-border-default rounded-[10px] bg-surface-base overflow-hidden min-h-0"
      >
        <header
          class="flex justify-between items-center gap-4 py-3 px-4 border-b border-border-default"
        >
          <div class="flex flex-col gap-1.5">
            <OuiText size="sm" weight="semibold">
              {{ currentNode?.name || "Preview" }}
            </OuiText>
            <OuiFlex gap="sm" align="center" class="flex-wrap">
              <span
                v-if="currentNode?.type === 'symlink'"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
              >
                <LinkIcon class="h-3.5 w-3.5" />
                {{ currentNode.symlinkTarget }}
              </span>
              <span
                v-if="currentNode?.mimeType"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >{{ currentNode.mimeType }}</span
              >
              <span
                v-if="currentNode?.owner"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >Owner: {{ currentNode.owner }}</span
              >
              <span
                v-if="currentNode?.group"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >Group: {{ currentNode.group }}</span
              >
              <span
                v-if="currentNode?.mode"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >Mode: {{ currentNode.mode.toString(8) }}</span
              >
              <span
                v-if="currentNode?.modifiedTime"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >Modified: {{ formatDatetime(currentNode.modifiedTime) }}</span
              >
              <span
                v-if="currentNode?.createdTime"
                class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[11px] rounded-xl bg-surface-subtle text-text-secondary"
                >Created: {{ formatDatetime(currentNode.createdTime) }}</span
              >
              <!-- Unsaved Changes Indicator -->
              <Transition name="fade">
                <span
                  v-if="hasUnsavedChanges && selectedPath && currentNode?.type === 'file'"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg border-2 border-warning/40 bg-warning/20 text-warning transition-all duration-200 shadow-md z-10"
                >
                  <OuiText size="xs" weight="semibold" color="warning">
                    Unsaved changes
                  </OuiText>
                </span>
              </Transition>
              <!-- Save Status Indicator -->
              <Transition name="fade">
                <span
                  v-if="saveStatus !== 'idle' && selectedPath"
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg border-2 transition-all duration-200 shadow-md z-10"
                  :data-status="saveStatus"
                  :data-test="'save-status-' + saveStatus"
                  :key="'save-status-' + saveStatus"
                  :class="{
                    'bg-success/20 text-success border-success/40': saveStatus === 'success',
                    'bg-danger/20 text-danger border-danger/40': saveStatus === 'error',
                    'bg-primary/20 text-primary border-primary/40': saveStatus === 'saving',
                  }"
                >
                  <ArrowPathIcon
                    v-if="saveStatus === 'saving'"
                    class="h-4 w-4 animate-spin"
                  />
                  <CheckCircleIcon
                    v-else-if="saveStatus === 'success'"
                    class="h-4 w-4"
                  />
                  <XCircleIcon
                    v-else-if="saveStatus === 'error'"
                    class="h-4 w-4"
                  />
                  <OuiText size="xs" weight="semibold" :color="saveStatus === 'success' ? 'success' : saveStatus === 'error' ? 'danger' : 'primary'">
                    {{
                      saveStatus === "saving"
                        ? "Saving..."
                        : saveStatus === "success"
                        ? "Saved"
                        : "Save Failed"
                    }}
                  </OuiText>
                </span>
              </Transition>
            </OuiFlex>
          </div>
          <OuiFlex gap="sm" align="center">
            <OuiCombobox
              v-if="currentNode?.type === 'file'"
              v-model="fileLanguage"
              :options="languageOptions"
              placeholder="Search language..."
              class="min-w-[180px] max-w-[250px]"
              size="sm"
            />
            <OuiButton
              v-if="
                selectedPath &&
                currentNode?.type === 'file' &&
                !fileError &&
                filePreviewType !== 'binary' &&
                filePreviewType !== 'image' &&
                filePreviewType !== 'video' &&
                filePreviewType !== 'audio' &&
                filePreviewType !== 'pdf'
              "
              variant="solid"
              size="sm"
              :disabled="isSaving"
              @click="handleSaveFile"
            >
              <DocumentArrowDownIcon class="h-4 w-4 mr-1.5" />
              <span>{{ isSaving ? 'Saving...' : 'Save' }}</span>
            </OuiButton>
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

        <div class="flex-1 relative min-h-0 overflow-hidden" role="tabpanel">
          <div
            v-if="!selectedPath"
            class="h-full flex items-center justify-center text-text-tertiary"
          >
            <OuiText size="sm" color="secondary"
              >Select a file to view its contents</OuiText
            >
            </div>
            <div
            v-else-if="fileError"
            class="h-full flex items-center justify-center p-8"
          >
            <div class="flex flex-col items-center gap-4 max-w-md text-center">
              <div
                class="flex items-center justify-center w-16 h-16 rounded-full bg-danger/10"
              >
                <ExclamationTriangleIcon class="h-8 w-8 text-danger" />
            </div>
              <div class="flex flex-col gap-2">
                <OuiText size="lg" weight="semibold" color="danger">
                  Unable to View File
                </OuiText>
                <OuiText size="sm" color="secondary">
                  {{ fileError }}
                </OuiText>
              </div>
              <OuiButton
                v-if="currentNode?.type === 'file'"
                variant="outline"
                size="sm"
                @click="handleDownload"
              >
                Download Instead
              </OuiButton>
            </div>
          </div>
          <!-- Media Preview (Images, Videos, Audio, PDF) -->
          <div
            v-else-if="
              selectedPath &&
              currentNode?.type === 'file' &&
              !fileError &&
              filePreviewType &&
              filePreviewType !== 'text' &&
              fileBlobUrl
            "
            class="h-full flex items-center justify-center p-8 bg-surface-base"
          >
            <div class="w-full h-full flex flex-col items-center justify-center gap-4">
              <!-- Image Preview -->
              <img
                v-if="filePreviewType === 'image'"
                :src="fileBlobUrl"
                :alt="currentNode?.name || 'Image preview'"
                class="max-w-full max-h-full object-contain rounded-lg shadow-lg"
                @error="handlePreviewError"
              />
              <!-- Video Preview -->
              <video
                v-else-if="filePreviewType === 'video'"
                :src="fileBlobUrl"
                controls
                class="max-w-full max-h-full rounded-lg shadow-lg"
                @error="handlePreviewError"
              >
                Your browser does not support the video tag.
              </video>
              <!-- Audio Preview -->
              <div
                v-else-if="filePreviewType === 'audio'"
                class="w-full max-w-md flex flex-col items-center gap-4 p-6 bg-surface-elevated rounded-lg border border-border-default"
              >
                <OuiText size="lg" weight="semibold">
                  {{ currentNode?.name || "Audio" }}
                </OuiText>
                <audio
                  :src="fileBlobUrl"
                  controls
                  class="w-full"
                  @error="handlePreviewError"
                >
                  Your browser does not support the audio tag.
                </audio>
          </div>
              <!-- PDF Preview -->
              <iframe
                v-else-if="filePreviewType === 'pdf'"
                :src="fileBlobUrl"
                class="w-full h-full border border-border-default rounded-lg"
                @error="handlePreviewError"
              />
              <!-- Binary/Unsupported -->
              <div
                v-else-if="filePreviewType === 'binary'"
                class="flex flex-col items-center gap-4 p-8 max-w-md text-center"
              >
                <div
                  class="flex items-center justify-center w-16 h-16 rounded-full bg-surface-elevated border-2 border-border-default"
                >
                  <DocumentIcon class="h-8 w-8 text-text-tertiary" />
        </div>
                <div class="flex flex-col gap-2">
                  <OuiText size="lg" weight="semibold">
                    Binary File
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    This file type cannot be previewed.
                    <template v-if="fileMetadata?.mimeType">
                      <br />
                      MIME type: {{ fileMetadata.mimeType }}
                    </template>
                  </OuiText>
      </div>
                <OuiButton variant="outline" size="sm" @click="handleDownload">
                  Download File
                </OuiButton>
              </div>
            </div>
          </div>
          <!-- Text Editor -->
          <OuiFileEditor
            v-else-if="
              selectedPath &&
              currentNode?.type === 'file' &&
              !fileError &&
              (filePreviewType === 'text' || filePreviewType === null)
            "
            :key="`editor-${selectedPath}-${editorRefreshKey}`"
            v-model="fileContent"
            :language="fileLanguage"
            :read-only="false"
            height="100%"
            :container-class="'w-full h-full border-0 overflow-hidden'"
            class="absolute inset-0"
            @save="handleSaveFile"
          />
          <div
            v-else
            class="h-full flex items-center justify-center text-text-tertiary"
          >
            <OuiText size="sm" color="secondary"
              >Select a file to view its contents</OuiText
            >
          </div>
        </div>
      </section>
    </div>
  </OuiCardBody>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
  import { useRoute, useRouter } from "vue-router";
  import {
    ArrowPathIcon,
    ServerIcon,
    CubeIcon,
    LinkIcon,
    ExclamationTriangleIcon,
    XMarkIcon,
    DocumentIcon,
    CheckCircleIcon,
    XCircleIcon,
    DocumentArrowDownIcon,
  } from "@heroicons/vue/24/outline";
  import { TreeView } from "@ark-ui/vue/tree-view";
  import {
    createTreeCollection,
    type TreeNode as ArkTreeNode,
  } from "@ark-ui/vue/collection";
import TreeNode from "./TreeNode.vue";
  import { useFileExplorer } from "~/composables/useFileExplorer";
  import { useDeploymentContainerQuery } from "~/composables/useDeploymentContainerQuery";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import type { CreateContainerEntryRequest } from "@obiente/proto";
  import { ContainerEntryType } from "@obiente/proto";
  import OuiFileEditor from "~/components/oui/FileEditor.vue";
  import OuiCombobox from "~/components/oui/Combobox.vue";
  import OuiSelect from "~/components/oui/Select.vue";
  import { useDialog } from "~/composables/useDialog";

  const props = defineProps<{
  deploymentId: string;
  organizationId?: string;
    allowEditing?: boolean;
  }>();

  const route = useRoute();
  const router = useRouter();

  const showUpload = ref(false);
  const hasMounted = ref(false);
  const isInitializingFromQuery = ref(false); // Flag to prevent circular updates during query param initialization
  const isLoadingFile = ref(false); // Track if a file load is in progress
  let currentFileLoadController: AbortController | null = null; // AbortController for cancelling pending requests
  const fileContent = ref("");
  const originalFileContent = ref(""); // Track original content to detect changes
  const fileLanguage = ref("plaintext");
  const currentFilePath = ref<string | null>(null);
  const fileError = ref<string | null>(null);
  const fileMetadata = ref<{ mimeType?: string; encoding?: string; size?: number } | null>(null);
  const fileBlobUrl = ref<string | null>(null);
  const filePreviewType = ref<"text" | "image" | "video" | "audio" | "pdf" | "binary" | null>(null);
  const editorRefreshKey = ref(0); // Force editor refresh when reloading file
  
  // Track if file has unsaved changes
  const hasUnsavedChanges = computed(() => {
    if (!currentFilePath.value) return false;
    return fileContent.value !== originalFileContent.value;
  });

  // Comprehensive list of Monaco-supported languages
  const languageOptions = computed(() => {
    const autoDetected = currentNode.value
      ? detectLanguage(currentNode.value.path)
      : "plaintext";
    return [
      { label: `Auto (${autoDetected})`, value: autoDetected },
      { label: "Plain Text", value: "plaintext" },
      { label: "JavaScript", value: "javascript" },
      { label: "TypeScript", value: "typescript" },
      { label: "Python", value: "python" },
      { label: "Java", value: "java" },
      { label: "C", value: "c" },
      { label: "C++", value: "cpp" },
      { label: "C#", value: "csharp" },
      { label: "Go", value: "go" },
      { label: "Rust", value: "rust" },
      { label: "Ruby", value: "ruby" },
      { label: "PHP", value: "php" },
      { label: "Swift", value: "swift" },
      { label: "Kotlin", value: "kotlin" },
      { label: "Scala", value: "scala" },
      { label: "Dart", value: "dart" },
      { label: "HTML", value: "html" },
      { label: "CSS", value: "css" },
      { label: "SCSS", value: "scss" },
      { label: "Less", value: "less" },
      { label: "SASS", value: "sass" },
      { label: "JSON", value: "json" },
      { label: "YAML", value: "yaml" },
      { label: "XML", value: "xml" },
      { label: "Markdown", value: "markdown" },
      { label: "Shell Script", value: "shell" },
      { label: "Bash", value: "bash" },
      { label: "PowerShell", value: "powershell" },
      { label: "SQL", value: "sql" },
      { label: "MySQL", value: "mysql" },
      { label: "PostgreSQL", value: "pgsql" },
      { label: "Dockerfile", value: "dockerfile" },
      { label: "Makefile", value: "makefile" },
      { label: "INI", value: "ini" },
      { label: "TOML", value: "toml" },
      { label: "Properties", value: "properties" },
      { label: "LaTeX", value: "latex" },
      { label: "R", value: "r" },
      { label: "Razor", value: "razor" },
      { label: "Lua", value: "lua" },
      { label: "Perl", value: "perl" },
      { label: "CoffeeScript", value: "coffeescript" },
      { label: "F#", value: "fsharp" },
      { label: "Haskell", value: "haskell" },
      { label: "Elixir", value: "elixir" },
      { label: "Erlang", value: "erlang" },
      { label: "OCaml", value: "ocaml" },
      { label: "MATLAB", value: "matlab" },
      { label: "Objective-C", value: "objective-c" },
      { label: "Pascal", value: "pascal" },
      { label: "VB.NET", value: "vb" },
      { label: "Batch", value: "bat" },
      { label: "Diff", value: "diff" },
      { label: "Log", value: "log" },
      { label: "Groovy", value: "groovy" },
      { label: "Handlebars", value: "handlebars" },
      { label: "Jade", value: "jade" },
      { label: "Pug", value: "pug" },
      { label: "Svelte", value: "svelte" },
      { label: "Vue", value: "vue" },
      { label: "Structured Text", value: "st" },
      { label: "ABAP", value: "abap" },
      { label: "Apex", value: "apex" },
      { label: "Azure CLI", value: "azcli" },
      { label: "Bicep", value: "bicep" },
      { label: "Cameligo", value: "cameligo" },
      { label: "Clojure", value: "clojure" },
      { label: "CSP", value: "csp" },
      { label: "Cypher", value: "cypher" },
      { label: "ECMAScript", value: "ecmascript" },
      { label: "Flow9", value: "flow9" },
      { label: "FreeMarker", value: "freemarker2" },
      { label: "GraphQL", value: "graphql" },
      { label: "HCL", value: "hcl" },
      { label: "HTML (Eex)", value: "html-eex" },
      { label: "JavaScript React", value: "javascriptreact" },
      { label: "Liquid", value: "liquid" },
      { label: "Lua", value: "lua" },
      { label: "M3", value: "m3" },
      { label: "MDX", value: "mdx" },
      { label: "Mips", value: "mips" },
      { label: "MSDAX", value: "msdax" },
      { label: "Pascaligo", value: "pascaligo" },
      { label: "Pligi", value: "plsql" },
      { label: "Redis", value: "redis" },
      { label: "Redshift", value: "redshift" },
      { label: "REST", value: "restructuredtext" },
      { label: "SB", value: "sb" },
      { label: "Scheme", value: "scheme" },
      { label: "SOP", value: "solidity" },
      { label: "SOP", value: "sophia" },
      { label: "SPARQL", value: "sparql" },
      { label: "System Verilog", value: "systemverilog" },
      { label: "Tcl", value: "tcl" },
      { label: "Twig", value: "twig" },
      { label: "TypeScript React", value: "typescriptreact" },
      { label: "Verilog", value: "verilog" },
      { label: "Wgsl", value: "wgsl" },
      { label: "XQuery", value: "xquery" },
      { label: "YAML", value: "yaml" },
      { label: "Zig", value: "zig" },
      { label: "DotENV", value: "dotenv" },
    ];
  });

  const explorer = useFileExplorer({
    organizationId: props.organizationId || "",
    deploymentId: props.deploymentId,
    allowEditing: props.allowEditing,
  });

  const {
    root,
    volumes: volumesRef,
    source,
    containerRunning,
    selectedPath,
    breadcrumbs,
    errorMessage,
    isLoadingTree,
    containers,
    selectedContainerId,
    selectedServiceName,
    fetchVolumes,
    loadContainers,
    setContainer,
    switchToVolume,
    switchToContainer,
    refreshRoot,
    loadChildren,
    findNode,
    deleteEntries,
    renameEntry,
    createEntry,
    writeFile,
    getOrgId,
    setOrganizationId,
  } = explorer;

  // Use composable for container query management
  const containerQuery = useDeploymentContainerQuery(
    props.deploymentId,
    props.organizationId
  );

  // Sync with composable - watch for container changes from query params
  watch(
    containerQuery.selectedContainer,
    (container) => {
      if (container) {
        setContainer(container.containerId, container.serviceName);
      }
    }
  );

  // Ensure volumes is reactive for template - access via computed
  const volumes = computed(() => volumesRef.value || []);

  type ExplorerTreeNode = ArkTreeNode & {
    value: ExplorerNode | null;
    children?: ExplorerTreeNode[];
  };

  const treeCollection = computed(() => {
    const visit = (
      nodes: ExplorerNode[] | undefined,
      parentId: string | null,
      acc: ExplorerTreeNode[]
    ) => {
      if (!nodes?.length) return;
      for (const node of nodes) {
        const rawSegment =
          node.name ||
          (node.path ? node.path.split("/").filter(Boolean).pop() : "") ||
          node.id ||
          `node-${acc.length}`;
        const segment =
          rawSegment?.split("/").filter(Boolean).join("-") || rawSegment;
        const nodeId = parentId
          ? `${parentId}/${segment || rawSegment}`
          : segment || rawSegment;
        const treeNode: ExplorerTreeNode = {
          id: nodeId,
          parentId: parentId ?? undefined,
          value: node,
          isBranch: node.type === "directory" || !!node.children?.length,
          isLeaf: node.type !== "directory" && !node.children?.length,
  children: [],
        };
        acc.push(treeNode);
        if (node.children?.length) {
          visit(node.children, nodeId, treeNode.children!);
        }
      }
    };

    const items: ExplorerTreeNode[] = [];
    visit(root.children, "ROOT", items);

    return createTreeCollection({
      rootNode: {
        id: "ROOT",
        value: null,
        children: items,
      },
  });
});

  const explorerClient = useConnectClient(DeploymentService);
  const dialog = useDialog();

  const allowEditing = props.allowEditing ?? true;
  const isSaving = ref(false);
  const saveStatus = ref<"idle" | "saving" | "success" | "error">("idle");
  const saveErrorMessage = ref<string | null>(null);

  const currentNode = computed(() => {
    if (!selectedPath.value) return null;
    return findNode(selectedPath.value) || null;
  });

  const currentDirectory = computed(() => {
    if (currentNode.value?.type === "directory") {
      return currentNode.value.path || "/";
    }
    if (currentNode.value?.type === "file") {
      // Use parent directory for files
      const parent = findNode(currentNode.value.parentPath || "/");
      return parent?.path || "/";
    }
    return explorer.root.path || "/";
  });

  function handleSwitchSource(type: "container" | "volume", name?: string) {
    if (type === "container") {
      switchToContainer();
    } else if (name) {
      switchToVolume(name);
    }
    selectedPath.value = null;
    currentFilePath.value = null;
    fileContent.value = "";
    fileLanguage.value = "plaintext";
    updateFileQueryParam(null); // Clear file query param
    refreshRoot();
  }

  async function handleToggle(node: ExplorerNode, open: boolean) {
    if (node.isLoading) return;

    node.isExpanded = open;

    if (open) {
      if (!node.hasLoaded || node.hasMore) {
        await loadChildren(
          node,
          node.hasMore ? node.nextCursor ?? undefined : undefined
        );
      }
      node.hasLoaded = true;
      selectedPath.value = node.path;
    }
  }

  async function handleOpen(
    node: ExplorerNode,
    options?: { ensureExpanded?: boolean }
  ) {
    selectedPath.value = node.path;
    if (node.type === "directory") {
      if (options?.ensureExpanded && !node.isExpanded) {
        await handleToggle(node, true);
      }
    } else {
      await handleLoadFile(node);
    }
  }

  function handleLoadMore(node: ExplorerNode) {
    if (!node.hasMore || node.isLoading) return;
    loadChildren(node, node.nextCursor ?? undefined);
  }

  function handleContextAction(action: string, node: ExplorerNode) {
    switch (action) {
      case "open":
        handleOpen(node, { ensureExpanded: node.type === "directory" });
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
    if (!allowEditing) {
      await dialog.showAlert({
        title: "Editing Disabled",
        message: "File editing is not enabled for this deployment.",
      });
      return;
    }

    // Determine parent directory - use current directory if it's a directory, otherwise use parent of current file
    const parent =
      currentNode.value && currentNode.value.type === "directory"
        ? currentNode.value.path
        : currentDirectory.value || "/";

    // Get name from user using dialog
    const nameResult = await dialog.showPrompt({
      title: `Create New ${type === "directory" ? "Folder" : type === "file" ? "File" : "Symlink"}`,
      message: `Enter a name for the new ${type}:`,
      placeholder: `new-${type}-name`,
      confirmLabel: "Create",
      cancelLabel: "Cancel",
    });

    if (!nameResult || !nameResult.trim()) {
      return; // User cancelled or entered empty name
    }

    const name = nameResult.trim();

    // Validate name (basic validation)
    if (name.includes("/") || name.includes("\\")) {
      await dialog.showAlert({
        title: "Invalid Name",
        message: "Name cannot contain path separators.",
      });
      return;
    }

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

    // For symlinks, get the target path
    if (type === "symlink") {
      const targetResult = await dialog.showPrompt({
        title: "Create Symlink",
        message: "Enter the target path for the symlink:",
        placeholder: "/path/to/target",
        confirmLabel: "Create",
        cancelLabel: "Cancel",
      });
      
      if (!targetResult || !targetResult.trim()) {
        return; // User cancelled
      }
      payload.template = targetResult.trim();
    }

    try {
      await createEntry(payload);
      
      // Refresh the parent directory to show the new entry
      const parentNode = findNode(parent);
      if (parentNode && parentNode.type === "directory") {
        await loadChildren(parentNode);
      } else {
        // Fallback to refreshing root
        await refreshRoot();
      }

      // Show success message
      await dialog.showAlert({
        title: "Created",
        message: `Successfully created ${type}: ${name}`,
        confirmLabel: "OK",
      });
    } catch (err: any) {
      console.error("Failed to create entry:", err);
      await dialog.showAlert({
        title: "Creation Failed",
        message: err?.message || `Failed to create ${type}. Please try again.`,
        confirmLabel: "OK",
      });
    }
  }

  async function queueDelete(paths: string[]) {
    if (!allowEditing) return;
    if (!confirm(`Delete ${paths.length} item(s)?`)) return;
    await deleteEntries(paths);
    selectedPath.value = null;
    updateFileQueryParam(null); // Clear file query param when closing file
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

  function parseTreeError(errorMessage: string): string {
    // Check for device file errors
    if (
      errorMessage.includes("/dev/") ||
      errorMessage.includes("/proc/") ||
      errorMessage.includes("/sys/")
    ) {
      const pathMatch = errorMessage.match(/(\/dev\/|\/proc\/|\/sys\/)[^\s"]+/);
      if (pathMatch) {
        const sysPath = pathMatch[0];
        if (sysPath.startsWith("/dev/")) {
          return `Cannot access device file: ${sysPath}. Device files are special system files.`;
        }
        if (sysPath.startsWith("/proc/") || sysPath.startsWith("/sys/")) {
          return `Cannot access system file: ${sysPath}. System directories may have restricted access.`;
        }
      }
    }

    // Check for permission errors
    if (
      errorMessage.includes("permission") ||
      errorMessage.includes("Permission") ||
      errorMessage.includes("EACCES")
    ) {
      return "Permission denied. You don't have access to this location.";
    }

    // Check for not found errors
    if (
      errorMessage.includes("not found") ||
      errorMessage.includes("ENOENT") ||
      errorMessage.includes("No such file")
    ) {
      return "Directory not found. It may have been deleted or moved.";
    }

    // Check for container stopped errors
    if (
      errorMessage.includes("container is stopped") ||
      errorMessage.includes("container is not running")
    ) {
      if (errorMessage.includes("Use volume_name") || errorMessage.includes("start the container")) {
        return "Container is not running. To access files, either start the container or use a volume (which can be accessed even when containers are stopped).";
      }
      return "Container is not running. Please start it to access the filesystem, or use a volume for persistent storage.";
    }
    
    // Check for command failures
    if (
      errorMessage.includes("command") &&
      errorMessage.includes("failed") &&
      errorMessage.includes("exit code")
    ) {
      const pathMatch = errorMessage.match(/["']([^"']+)["']/);
      if (pathMatch) {
        return `Unable to list files in: ${pathMatch[1]}. This location may not be accessible.`;
      }
      return "Unable to list files. The location may be restricted or inaccessible.";
    }
    
    // Check for "failed to list files" errors
    if (errorMessage.includes("failed to list files")) {
      const pathMatch = errorMessage.match(/failed to list files in ["']([^"']+)["']/);
      if (pathMatch) {
        return `Unable to list files in: ${pathMatch[1]}. This location may not be accessible or the container may not be running.`;
      }
      return "Unable to list files. The container may not be running or the location may be inaccessible.";
    }

    // Generic error - try to extract meaningful part
    if (errorMessage.includes("[internal]")) {
      const match = errorMessage.match(/\[internal\]\s*(.+?)(?:\s+NuxtJS|$)/);
      if (match && match[1]) {
        return `Unable to access: ${match[1].trim()}`;
      }
    }

    // Default user-friendly message
    return errorMessage.length > 150
      ? `${errorMessage.substring(0, 150)}...`
      : errorMessage;
  }

  function parseFileError(err: any): string {
    const errorMessage = err?.message || String(err) || "Unknown error";

    // Check for device file errors
    if (
      errorMessage.includes("/dev/") ||
      errorMessage.includes("command") ||
      errorMessage.includes("exit code")
    ) {
      const devMatch = errorMessage.match(/\/dev\/[^\s"]+/);
      if (devMatch) {
        const devFile = devMatch[0];
        return `Cannot view device file: ${devFile}. Device files are special system files that cannot be read as text.`;
      }
      if (errorMessage.includes("command") && errorMessage.includes("failed")) {
        return "Unable to read this file. It may be a binary file, device file, or have restricted permissions.";
      }
    }

    // Check for permission errors
    if (
      errorMessage.includes("permission") ||
      errorMessage.includes("Permission") ||
      errorMessage.includes("EACCES")
    ) {
      return "Permission denied. You don't have read access to this file.";
    }

    // Check for not found errors
    if (
      errorMessage.includes("not found") ||
      errorMessage.includes("ENOENT") ||
      errorMessage.includes("No such file")
    ) {
      return "File not found. It may have been deleted or moved.";
    }

    // Check for too large errors
    if (
      errorMessage.includes("too large") ||
      errorMessage.includes("file size")
    ) {
      return "File is too large to display. Please download it instead.";
    }

    // Check for binary file indicators
    if (errorMessage.includes("binary") || errorMessage.includes("Binary")) {
      return "This appears to be a binary file and cannot be displayed as text.";
    }

    // Generic error - try to extract meaningful part
    if (errorMessage.includes("[internal]")) {
      const match = errorMessage.match(/\[internal\]\s*(.+?)(?:\s+NuxtJS|$)/);
      if (match && match[1]) {
        return `Unable to read file: ${match[1].trim()}`;
      }
    }

    // Default user-friendly message
    return `Unable to load file: ${errorMessage}`;
  }

  function isLikelyUnviewableFile(path: string): {
    unviewable: boolean;
    reason?: string;
  } {
    const lowerPath = path.toLowerCase();

    // Device files
    if (lowerPath.startsWith("/dev/")) {
      return {
        unviewable: true,
        reason:
          "Device files are special system files that cannot be viewed as text",
      };
    }

    // Block devices, character devices, sockets, pipes
    if (lowerPath.includes("/proc/") || lowerPath.includes("/sys/")) {
      return {
        unviewable: true,
        reason:
          "System files in /proc or /sys are typically not viewable as text",
      };
    }

    // Binary file extensions (common ones)
    const binaryExts = [
      ".bin",
      ".exe",
      ".so",
      ".dylib",
      ".dll",
      ".app",
      ".deb",
      ".rpm",
      ".pkg",
      ".dmg",
    ];
    if (binaryExts.some((ext) => lowerPath.endsWith(ext))) {
      return {
        unviewable: true,
        reason:
          "This appears to be a binary file and cannot be displayed as text",
      };
    }

    return { unviewable: false };
  }

  async function handleLoadFile(node: ExplorerNode) {
    if (node.type !== "file") return;

    // Cancel any pending file load request
    if (currentFileLoadController) {
      currentFileLoadController.abort();
      currentFileLoadController = null;
    }

    // Prevent concurrent loads of the same file
    if (isLoadingFile.value && currentFilePath.value === node.path) {
      console.log("[handleLoadFile] Already loading this file, skipping duplicate request");
      return;
    }

    // Clean up previous blob URL when switching files
    if (fileBlobUrl.value) {
      URL.revokeObjectURL(fileBlobUrl.value);
      fileBlobUrl.value = null;
    }
    filePreviewType.value = null;
    fileMetadata.value = null;

    // Check if file is likely unviewable before attempting to load
    const unviewableCheck = isLikelyUnviewableFile(node.path);
    if (unviewableCheck.unviewable) {
      fileError.value = unviewableCheck.reason || "This file cannot be viewed";
      fileContent.value = "";
      fileLanguage.value = "plaintext";
      selectedPath.value = node.path;
      currentFilePath.value = node.path;
      return;
    }

    // Check for unsaved changes before switching files
    if (hasUnsavedChanges.value && currentFilePath.value) {
      const confirmed = await dialog.showConfirm({
        title: "Unsaved Changes",
        message: `You have unsaved changes in ${currentFilePath.value.split("/").pop()}. Open another file?`,
        confirmLabel: "Discard & Open",
        cancelLabel: "Cancel",
      });
      if (!confirmed) return;
    }
    
    selectedPath.value = node.path;
    currentFilePath.value = node.path;
    fileError.value = null; // Clear previous errors
    saveStatus.value = "idle"; // Reset save status when switching files
    saveErrorMessage.value = null;
    // Reset original content - will be set when file loads
    originalFileContent.value = "";

    // Update query parameter to track the open file
    updateFileQueryParam(node.path);

    // Create new AbortController for this request
    const abortController = new AbortController();
    currentFileLoadController = abortController;
    isLoadingFile.value = true;

    try {
      // Store the request path to verify it's still the current file after load
      const requestPath = node.path;
      
      const res = await explorerClient.getContainerFile({
        organizationId: getOrgId(),
        deploymentId: props.deploymentId,
        path: node.path,
        volumeName: source.type === "volume" ? source.volumeName : undefined,
      });

      // Verify this request is still valid (file hasn't changed during load)
      if (currentFilePath.value !== requestPath) {
        console.log("[handleLoadFile] File changed during load, discarding stale response");
        return;
      }

      // Store metadata
      fileMetadata.value = {
        mimeType: res.metadata?.mimeType,
        encoding: res.encoding || "text",
        size: Number(res.size || 0),
      };

      // Determine preview type based on MIME type or file extension
      const mimeType = res.metadata?.mimeType || "";
      const previewType = detectPreviewType(node.path, mimeType);
      filePreviewType.value = previewType;

      if (previewType === "text") {
        // Text file - show in editor
        const content = res.content || "";
        fileContent.value = content;
        originalFileContent.value = content; // Store original content
        editorRefreshKey.value++; // Force editor to refresh with new content
        fileLanguage.value = detectLanguage(node.path);
        // Clean up any existing blob URL
        if (fileBlobUrl.value) {
          URL.revokeObjectURL(fileBlobUrl.value);
          fileBlobUrl.value = null;
        }
      } else {
        // Media file - create blob URL for preview
        fileContent.value = ""; // Clear text content
        fileLanguage.value = "plaintext";

        // Create blob from content
        let blob: Blob;
        if (res.encoding === "base64") {
          // Convert base64 to binary
          const binaryString = atob(res.content);
          const bytes = new Uint8Array(binaryString.length);
          for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
          }
          blob = new Blob([bytes], {
            type: mimeType || "application/octet-stream",
          });
        } else {
          // Text content (shouldn't happen for media, but handle it)
          blob = new Blob([res.content], {
            type: mimeType || "text/plain",
          });
        }

        // Create object URL for preview
        if (fileBlobUrl.value) {
          URL.revokeObjectURL(fileBlobUrl.value);
        }
        fileBlobUrl.value = URL.createObjectURL(blob);
      }

      fileError.value = null; // Clear error on success
    } catch (err: any) {
      // Don't show error if request was aborted (cancelled)
      if (err?.name === "AbortError" || err?.message?.includes("aborted")) {
        console.log("[handleLoadFile] Request was aborted");
        return;
      }
      
      console.error("load file", err);
      fileError.value = parseFileError(err);
      fileContent.value = "";
      fileLanguage.value = "plaintext";
      filePreviewType.value = null;
      fileMetadata.value = null;
      if (fileBlobUrl.value) {
        URL.revokeObjectURL(fileBlobUrl.value);
        fileBlobUrl.value = null;
      }
    } finally {
      // Reset loading state and clear abort controller
      isLoadingFile.value = false;
      if (currentFileLoadController === abortController) {
        currentFileLoadController = null;
      }
    }
  }

  function detectPreviewType(
    path: string,
    mimeType?: string
  ): "text" | "image" | "video" | "audio" | "pdf" | "binary" {
    // Check MIME type first (most reliable)
    if (mimeType) {
      if (mimeType.startsWith("image/")) return "image";
      if (mimeType.startsWith("video/")) return "video";
      if (mimeType.startsWith("audio/")) return "audio";
      if (mimeType === "application/pdf") return "pdf";
      // Text MIME types
      if (
        mimeType.startsWith("text/") ||
        mimeType.includes("json") ||
        mimeType.includes("xml") ||
        mimeType.includes("javascript") ||
        mimeType.includes("css") ||
        mimeType.includes("html")
      ) {
        return "text";
      }
      // If it's a binary MIME type, return binary
      if (
        mimeType.startsWith("application/") &&
        !mimeType.includes("json") &&
        !mimeType.includes("xml") &&
        mimeType !== "application/pdf"
      ) {
        return "binary";
      }
    }

    // Fallback to file extension
    const ext = path.split(".").pop()?.toLowerCase() || "";
    const imageExts = [
      "jpg",
      "jpeg",
      "png",
      "gif",
      "webp",
      "svg",
      "bmp",
      "ico",
      "tiff",
      "tif",
    ];
    const videoExts = ["mp4", "webm", "ogg", "mov", "avi", "mkv", "flv", "wmv"];
    const audioExts = [
      "mp3",
      "wav",
      "ogg",
      "aac",
      "flac",
      "m4a",
      "wma",
      "opus",
    ];
    // Common text file paths (especially system files without extensions)
    const commonTextPaths = [
      "/etc/",
      "/etc/profile",
      "/etc/passwd",
      "/etc/group",
      "/etc/hosts",
      "/etc/fstab",
      "/etc/resolv.conf",
      "/etc/ssh/",
      "/etc/nginx/",
      "/etc/apache/",
      "/var/log/",
      "/home/",
      "/root/",
      "/opt/",
      "/usr/local/",
      ".env",
      ".gitignore",
      ".dockerignore",
      "Dockerfile",
      "docker-compose",
      "Makefile",
      "README",
      "CHANGELOG",
      "LICENSE",
    ];
    
    // Check if path matches common text file patterns
    const lowerPath = path.toLowerCase();
    if (commonTextPaths.some(pattern => lowerPath.includes(pattern.toLowerCase()))) {
      return "text";
    }
    
    const textExts = [
      "txt",
      "md",
      "json",
      "yaml",
      "yml",
      "xml",
      "html",
      "htm",
      "css",
      "js",
      "jsx",
      "ts",
      "tsx",
      "py",
      "go",
      "rs",
      "java",
      "c",
      "cpp",
      "h",
      "hpp",
      "sh",
      "bash",
      "zsh",
      "fish",
      "sql",
      "log",
      "conf",
      "config",
      "ini",
      "env",
      "dockerfile",
      "makefile",
      "gitignore",
      "gitattributes",
      "editorconfig",
      "prettierrc",
      "eslintrc",
    ];

    if (imageExts.includes(ext)) return "image";
    if (videoExts.includes(ext)) return "video";
    if (audioExts.includes(ext)) return "audio";
    if (ext === "pdf") return "pdf";
    if (textExts.includes(ext)) return "text";

    // Default to binary for unknown types
    return "binary";
  }

  function handlePreviewError() {
    // If preview fails, show error and allow download
    fileError.value = "Failed to load preview. The file may be corrupted or unsupported.";
    filePreviewType.value = "binary";
  }

  async function handleSaveFile() {
    if (!currentFilePath.value) {
      console.warn("Cannot save: no file path");
      return;
    }
    if (isSaving.value) {
      console.log("Save already in progress, skipping");
      return; // Prevent double-saving
    }

    console.log("Starting save for:", currentFilePath.value);
    console.log("Current saveStatus before save:", saveStatus.value);
    isSaving.value = true;
    saveStatus.value = "saving";
    console.log("Save status set to 'saving':", saveStatus.value);
    saveErrorMessage.value = null;
    
    // Force Vue to update by using nextTick
    await nextTick();
    console.log("After nextTick, saveStatus:", saveStatus.value);

    try {
      await writeFile({
        path: currentFilePath.value,
        content: fileContent.value,
        volumeName: source.type === "volume" ? source.volumeName : undefined,
      });

      console.log("File saved successfully");
      saveStatus.value = "success";
      // Update original content to match saved content
      originalFileContent.value = fileContent.value;

      // Reset status after 3 seconds (status indicator shows "Saved")
      setTimeout(() => {
        if (saveStatus.value === "success") {
          saveStatus.value = "idle";
        }
      }, 3000);
    } catch (err: any) {
      console.error("save file error:", err);
      saveStatus.value = "error";
      
      const errorMsg = err?.message || "Failed to save file. Please try again.";
      saveErrorMessage.value = errorMsg;

      // Show error message dialog after showing status
      setTimeout(async () => {
        dialog.showAlert({
          title: "Save Failed",
          message: errorMsg,
          confirmLabel: "OK",
        }).catch(() => {});

        // Reset status after showing dialog (5 seconds total)
        setTimeout(() => {
          if (saveStatus.value === "error") {
            console.log("Resetting save status from error to idle");
            saveStatus.value = "idle";
            saveErrorMessage.value = null;
          }
        }, 3000);
      }, 1000); // Show dialog after 1 second
    } finally {
      isSaving.value = false;
    }
  }

  function detectLanguage(path: string) {
    // Get the filename (last segment of path)
    const filename = path.split("/").pop()?.toLowerCase() || "";

    // Check for dotfiles by full basename first
    const dotfileMap: Record<string, string> = {
      ".bashrc": "shell",
      ".bash_profile": "shell",
      ".bash_logout": "shell",
      ".profile": "shell",
      ".zshrc": "shell",
      ".zshenv": "shell",
      ".zprofile": "shell",
      ".zlogin": "shell",
      ".zlogout": "shell",
      ".fish": "shell",
      ".config/fish/config.fish": "shell",
      ".gitignore": "gitignore",
      ".gitconfig": "gitconfig",
      ".dockerignore": "dockerignore",
      ".env": "dotenv",
      ".env.local": "dotenv",
      ".env.production": "dotenv",
      ".env.development": "dotenv",
      ".vimrc": "vim",
      ".vim": "vim",
      ".editorconfig": "plaintext",
      ".prettierrc": "json",
      ".eslintrc": "json",
      ".eslintrc.json": "json",
      ".eslintrc.js": "javascript",
      ".eslintrc.cjs": "javascript",
      ".eslintrc.mjs": "javascript",
      ".stylelintrc": "json",
      ".babelrc": "json",
    };

    // Check dotfile map first
    if (dotfileMap[filename]) {
      return dotfileMap[filename];
    }

    // For paths like .config/fish/config.fish, check the last segment
    if (filename.includes(".")) {
      const lastPart = filename.split("/").pop() || filename;
      if (dotfileMap[lastPart]) {
        return dotfileMap[lastPart];
      }
    }

    // Then check by file extension
    const ext = filename.split(".").pop()?.toLowerCase();
    const extMap: Record<string, string> = {
      // JavaScript/TypeScript
    js: "javascript",
      mjs: "javascript",
      cjs: "javascript",
      jsx: "javascriptreact",
    ts: "typescript",
      tsx: "typescriptreact",
      // Python
    py: "python",
      pyw: "python",
      pyi: "python",
      pyx: "python",
      // Go
    go: "go",
      // Rust
    rs: "rust",
      // Java
    java: "java",
      class: "java",
      jar: "java",
      // C/C++
      c: "c",
      h: "c",
      cpp: "cpp",
      cxx: "cpp",
      cc: "cpp",
      hpp: "cpp",
      hxx: "cpp",
      // C#
      cs: "csharp",
      csx: "csharp",
      // Shell scripts
      sh: "shell",
      bash: "shell",
      zsh: "shell",
      fish: "shell",
      ksh: "shell",
      csh: "shell",
      tcsh: "shell",
      // Web technologies
    html: "html",
      htm: "html",
      xhtml: "html",
    css: "css",
      scss: "scss",
      sass: "sass",
      less: "less",
      styl: "stylus",
      // Data formats
      json: "json",
      json5: "json",
      jsonc: "json",
      yaml: "yaml",
      yml: "yaml",
      xml: "xml",
      xsd: "xml",
      xsl: "xml",
      xslt: "xml",
      // Markup/Markdown
    md: "markdown",
      markdown: "markdown",
      mdown: "markdown",
      mkd: "markdown",
      mkdn: "markdown",
      rst: "restructuredtext",
      // SQL
    sql: "sql",
      mysql: "mysql",
      pgsql: "pgsql",
      // PHP
      php: "php",
      php3: "php",
      php4: "php",
      php5: "php",
      phtml: "php",
      // Ruby
      rb: "ruby",
      rbx: "ruby",
      gemspec: "ruby",
      rake: "ruby",
      // Swift
      swift: "swift",
      // Kotlin
      kt: "kotlin",
      kts: "kotlin",
      // Scala
      scala: "scala",
      sc: "scala",
      // Dart
      dart: "dart",
      // Lua
      lua: "lua",
      // Perl
      pl: "perl",
      pm: "perl",
      t: "perl",
      // R
      r: "r",
      R: "r",
      // PowerShell
      ps1: "powershell",
      psd1: "powershell",
      psm1: "powershell",
      // Batch
      bat: "bat",
      cmd: "bat",
      // Docker/Container
    dockerfile: "dockerfile",
      dockerignore: "dockerignore",
      // Build tools
      makefile: "makefile",
      make: "makefile",
      mk: "makefile",
      // Config files
      ini: "ini",
      cfg: "ini",
      toml: "toml",
      properties: "properties",
      conf: "plaintext",
      config: "plaintext",
      // Templates
      hbs: "handlebars",
      handlebars: "handlebars",
      mustache: "handlebars",
      jade: "jade",
      pug: "pug",
      twig: "twig",
      // Frontend frameworks
      vue: "vue",
      svelte: "svelte",
      // GraphQL
      graphql: "graphql",
      gql: "graphql",
      // LaTeX
      tex: "latex",
      latex: "latex",
      // Clojure
      clj: "clojure",
      cljs: "clojure",
      cljc: "clojure",
      edn: "clojure",
      // CoffeeScript
      coffee: "coffeescript",
      cson: "coffeescript",
      // F#
      fs: "fsharp",
      fsi: "fsharp",
      fsx: "fsharp",
      // Haskell
      hs: "haskell",
      lhs: "haskell",
      // Elixir
      ex: "elixir",
      exs: "elixir",
      // Erlang
      erl: "erlang",
      hrl: "erlang",
      // OCaml
      ml: "ocaml",
      mli: "ocaml",
      // MATLAB/Objective-C (.m files - default to MATLAB, user can override)
      m: "matlab",
      // Objective-C
      mm: "objective-c",
      M: "objective-c",
      // Pascal
      pas: "pascal",
      p: "pascal",
      pp: "pascal",
      // Groovy
      groovy: "groovy",
      gvy: "groovy",
      // Diff
      diff: "diff",
      patch: "diff",
      // Logs
      log: "log",
      // Plain text
      txt: "plaintext",
      text: "plaintext",
      // Zig
      zig: "zig",
      // Solidity
      sol: "solidity",
      // SystemVerilog
      sv: "systemverilog",
      svh: "systemverilog",
      // Verilog
      v: "verilog",
      vh: "verilog",
      // TCL
      tcl: "tcl",
      // Liquid
      liquid: "liquid",
      // MDX
      mdx: "mdx",
      // HCL (Terraform)
      tf: "hcl",
      tfvars: "hcl",
      hcl: "hcl",
      // Bicep
      bicep: "bicep",
      // Other
      lock: "plaintext",
      gitignore: "gitignore",
    };

    return extMap[ext || ""] || "plaintext";
  }

  function handleBreadcrumbClick(path: string) {
    const node = explorer.findNode(path);
    if (node) handleOpen(node);
  }

  async function handleRefreshSelection() {
    if (!currentNode.value) return;
    
    // If it's a file, reload the file content
    if (currentNode.value.type === "file" && currentFilePath.value) {
      // Check for unsaved changes
      if (hasUnsavedChanges.value) {
        const confirmed = await dialog.showConfirm({
          title: "Unsaved Changes",
          message: "You have unsaved changes. Refreshing will discard them. Continue?",
          confirmLabel: "Discard & Refresh",
          cancelLabel: "Cancel",
        });
        if (!confirmed) return;
      }
      
      // Cancel any pending requests first
      if (currentFileLoadController) {
        currentFileLoadController.abort();
        currentFileLoadController = null;
      }
      
      // Force a fresh load by incrementing the editor refresh key first
      // This ensures any cached content is cleared before loading
      editorRefreshKey.value++;
      
      // Clear current content to show we're loading fresh data
      fileContent.value = "";
      originalFileContent.value = "";
      
      // Small delay to ensure any pending requests are cancelled and UI updates
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Reload the file
      await handleLoadFile(currentNode.value);
    } else {
      // For directories, just reload children
      await loadChildren(currentNode.value);
    }
  }

  async function handleDownload() {
    if (!currentNode.value || currentNode.value.type !== "file") return;

    try {
      // Fetch file content
      const res = await explorerClient.getContainerFile({
        organizationId: getOrgId(),
      deploymentId: props.deploymentId,
        path: currentNode.value.path,
        volumeName: source.type === "volume" ? source.volumeName : undefined,
      });

      // Get file name from path
      const fileName =
        currentNode.value.name ||
        currentNode.value.path.split("/").pop() ||
        "download";

      // Create blob from content
      // If encoding is base64, decode it first
      let blob: Blob;
      if (res.encoding === "base64") {
        // Convert base64 to binary
        const binaryString = atob(res.content);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          bytes[i] = binaryString.charCodeAt(i);
        }
        blob = new Blob([bytes], {
          type: res.metadata?.mimeType || "application/octet-stream",
        });
      } else {
        // Text content
        blob = new Blob([res.content], {
          type: res.metadata?.mimeType || "text/plain",
        });
      }

      // Create download URL
      const url = URL.createObjectURL(blob);

      // Create temporary anchor element and trigger download
      const link = document.createElement("a");
      link.href = url;
      link.download = fileName;
      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
  } catch (err: any) {
      console.error("Failed to download file:", err);
      // Show user-friendly error
      const errorMsg = parseFileError(err);
      alert(`Failed to download file: ${errorMsg}`);
    }
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

  const containerOptions = computed(() => {
    const options: Array<{ label: string; value: string }> = [
      { label: "Default (first container)", value: "" }
    ];
    containers.value.forEach(container => {
      const label = container.serviceName 
        ? `${container.serviceName} (${container.containerId.substring(0, 12)})`
        : container.containerId.substring(0, 12);
      const value = container.serviceName || container.containerId;
      options.push({ label, value });
    });
    return options;
  });

  const selectedContainerLabel = computed(() => {
    if (!selectedServiceName.value && !selectedContainerId.value) {
      return "Default (first container)";
    }
    const container = containers.value.find(c => 
      (c.serviceName || c.containerId) === (selectedServiceName.value || selectedContainerId.value)
    );
    if (container) {
      return container.serviceName || container.containerId.substring(0, 12);
    }
    return "Unknown";
  });

    function handleContainerChange(value: string) {
    if (!value) {
      setContainer(undefined, undefined);
    } else {
      const container = containers.value.find(c => 
        (c.serviceName || c.containerId) === value
      );
      if (container) {
        setContainer(container.containerId, container.serviceName);
      } else {
        setContainer(value, undefined);
      }
    }
    selectedPath.value = null;
    currentFilePath.value = null;
    fileContent.value = "";
    fileLanguage.value = "plaintext";
    updateFileQueryParam(null); // Clear file query param
    refreshRoot();
  }

  // Helper function to update file query parameter
  function updateFileQueryParam(filePath: string | null) {
    if (filePath) {
      // Update query param with the file path (URL-encoded)
      router.replace({
        query: {
          ...route.query,
          tab: "files", // Ensure we're on the files tab
          file: encodeURIComponent(filePath),
        },
      });
    } else {
      // Clear file query param when no file is selected
      const query = { ...route.query };
      delete query.file;
      router.replace({ query });
    }
  }

  // Helper function to open file from path (used for query param loading)
  async function openFileFromPath(filePath: string) {
    // Wait for tree to be loaded
    if (!root.hasLoaded) {
      await refreshRoot();
    }

    // Find the node in the tree
    const node = findNode(filePath);
    if (node && node.type === "file") {
      await handleLoadFile(node);
    } else {
      // If node not found, try loading parent directories recursively
      const pathParts = filePath.split("/").filter(Boolean);
      let currentPath = "";
      
      // Load all parent directories up to the file's parent
      for (const part of pathParts.slice(0, -1)) {
        currentPath = currentPath + "/" + part;
        const dirNode = findNode(currentPath || "/");
        if (dirNode && dirNode.type === "directory" && !dirNode.hasLoaded) {
          await loadChildren(dirNode);
        }
      }

      // Try finding the file node again
      const fileNode = findNode(filePath);
      if (fileNode && fileNode.type === "file") {
        await handleLoadFile(fileNode);
      } else {
        console.warn(`File not found in tree: ${filePath}`);
      }
    }
  }

onMounted(async () => {
    hasMounted.value = true;
    // Set organization ID if provided
    if (props.organizationId) {
      setOrganizationId(props.organizationId);
    }
    // Always try to fetch volumes when mounted (will skip if no orgId)
    await loadContainers();
    await fetchVolumes();
    await refreshRoot();

    // Check for file query parameter and open the file
    const fileParam = route.query.file;
    if (typeof fileParam === "string") {
      try {
        isInitializingFromQuery.value = true; // Prevent query param updates during init
        const filePath = decodeURIComponent(fileParam);
        await openFileFromPath(filePath);
      } catch (err) {
        console.error("Failed to open file from query param:", err);
      } finally {
        isInitializingFromQuery.value = false;
      }
    }
});

onUnmounted(() => {
    // Clean up blob URL on unmount
    if (fileBlobUrl.value) {
      URL.revokeObjectURL(fileBlobUrl.value);
      fileBlobUrl.value = null;
    }
  });

  watch(
    () => props.organizationId,
    async (newOrgId) => {
      if (!hasMounted.value) return;
      const orgId = newOrgId || "";
      setOrganizationId(orgId);
      // Fetch volumes when orgId changes (after mount)
      if (orgId) {
        await loadContainers();
        await fetchVolumes();
        await refreshRoot();
      }
    }
  );

  // Watch for file query parameter changes (e.g., back/forward navigation, shared links)
  watch(
    () => route.query.file,
    async (fileParam) => {
      if (!hasMounted.value) return; // Skip during initial mount (handled in onMounted)
      
      // Only react to query param changes if it's different from current file
      const currentFileFromQuery = typeof fileParam === "string" 
        ? decodeURIComponent(fileParam) 
        : null;
      
      if (currentFileFromQuery && currentFileFromQuery !== currentFilePath.value) {
        try {
          await openFileFromPath(currentFileFromQuery);
        } catch (err) {
          console.error("Failed to open file from query param change:", err);
        }
      } else if (!fileParam && currentFilePath.value) {
        // Query param was cleared, but we still have a file open - don't close it automatically
        // The user might have cleared it manually
      }
    }
  );

  // Update query param when selectedPath changes (but avoid circular updates)
  watch(
    selectedPath,
    (newPath) => {
      // Skip if we're initializing from query param to prevent circular updates
      if (isInitializingFromQuery.value) return;
      
      // Only update if the query param doesn't match (to avoid circular updates from query watcher)
      const currentFileFromQuery = typeof route.query.file === "string"
        ? decodeURIComponent(route.query.file)
        : null;
      
      if (newPath !== currentFileFromQuery) {
        updateFileQueryParam(newPath);
      }
    }
  );

  // Expose refresh method for parent component
  defineExpose({
    refreshRoot,
  });
</script>

<style scoped>
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
