<template>
  <OuiStack gap="md">
    <!-- Query Editor Card -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center" class="w-full">
          <OuiFlex align="center" gap="md">
            <OuiText as="h3" size="sm" weight="semibold">Query Editor</OuiText>
            <!-- Query Tabs -->
            <div class="flex items-center gap-1 ml-2">
              <button
                v-for="(tab, idx) in queryTabs"
                :key="idx"
                class="group flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md transition-all duration-150"
                :class="
                  activeTabIdx === idx
                    ? 'bg-primary text-white shadow-sm'
                    : 'bg-surface-base text-secondary hover:text-primary hover:bg-interactive-hover border border-border-default'
                "
                @click="activeTabIdx = idx"
              >
                <CommandLineIcon class="h-3 w-3" />
                <span class="max-w-24 truncate">{{ tab.title }}</span>
                <button
                  v-if="queryTabs.length > 1"
                  class="ml-1 opacity-50 hover:opacity-100 transition-opacity"
                  :class="activeTabIdx === idx ? 'text-white/70 hover:text-white' : 'text-secondary hover:text-danger'"
                  @click.stop="closeTab(idx)"
                >
                  <XMarkIcon class="h-3 w-3" />
                </button>
              </button>
              <button
                class="p-1.5 text-secondary hover:text-primary rounded-md border border-dashed border-border-default hover:border-primary/30 hover:bg-interactive-hover transition-all"
                @click="addTab"
                title="New query tab"
              >
                <PlusIcon class="h-3.5 w-3.5" />
              </button>
            </div>
          </OuiFlex>

          <OuiFlex gap="sm" align="center">
            <!-- Quick Insert -->
            <div class="relative" @click.stop>
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="showSnippets = !showSnippets; showHistory = false"
              >
                <SparklesIcon class="h-3.5 w-3.5" />
                Snippets
              </OuiButton>
              <Transition name="dropdown">
                <div
                  v-if="showSnippets"
                  class="absolute right-0 top-full mt-1 w-64 bg-surface-overlay border border-border-default rounded-lg shadow-xl z-50 overflow-hidden"
                >
                  <div class="px-3 py-2 border-b border-border-default bg-surface-base">
                    <OuiText size="xs" weight="semibold" color="tertiary">Quick Insert</OuiText>
                  </div>
                  <div class="max-h-64 overflow-y-auto">
                    <button
                      v-for="snippet in sqlSnippets"
                      :key="snippet.label"
                      class="w-full text-left px-3 py-2 hover:bg-interactive-hover border-b border-border-default/50 last:border-0 transition-colors"
                      @click="insertSnippet(snippet.code)"
                    >
                      <OuiText size="xs" weight="medium">{{ snippet.label }}</OuiText>
                      <OuiText size="xs" color="tertiary" class="font-mono mt-0.5 truncate">{{ snippet.preview }}</OuiText>
                    </button>
                  </div>
                </div>
              </Transition>
            </div>

            <!-- History -->
            <div class="relative" @click.stop v-if="queryHistory.length > 0">
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="showHistory = !showHistory; showSnippets = false"
              >
                <ClockIcon class="h-3.5 w-3.5" />
                History
                <OuiBadge color="tertiary" size="xs" class="ml-1">{{ queryHistory.length }}</OuiBadge>
              </OuiButton>
              <Transition name="dropdown">
                <div
                  v-if="showHistory"
                  class="absolute right-0 top-full mt-1 w-96 bg-surface-overlay border border-border-default rounded-lg shadow-xl z-50 overflow-hidden"
                >
                  <div class="px-3 py-2 border-b border-border-default bg-surface-base flex items-center justify-between">
                    <OuiText size="xs" weight="semibold" color="tertiary">Query History</OuiText>
                    <button
                      class="text-xs text-secondary hover:text-danger transition-colors"
                      @click="clearHistory"
                    >
                      Clear all
                    </button>
                  </div>
                  <div class="max-h-72 overflow-y-auto">
                    <button
                      v-for="(item, idx) in queryHistory"
                      :key="idx"
                      class="w-full text-left px-3 py-2.5 hover:bg-interactive-hover border-b border-border-default/50 last:border-0 transition-colors group"
                      @click="loadFromHistory(item)"
                    >
                      <pre class="text-xs font-mono text-primary/80 whitespace-pre-wrap line-clamp-2">{{ item }}</pre>
                    </button>
                  </div>
                </div>
              </Transition>
            </div>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>

      <OuiCardBody class="p-0">
        <div class="query-workbench">
          <aside class="query-schema-rail">
            <div class="query-schema-header">
              <OuiText size="xs" weight="semibold" transform="uppercase" color="tertiary">
                Schema
              </OuiText>
              <OuiBadge color="tertiary" size="xs">{{ tables.length }}</OuiBadge>
            </div>
            <OuiInput
              v-model="schemaSearch"
              placeholder="Find table..."
              clearable
              size="sm"
              class="mb-2"
            >
              <template #prefix>
                <TableCellsIcon class="h-3.5 w-3.5 text-secondary" />
              </template>
            </OuiInput>
            <div class="query-schema-list">
              <div
                v-for="table in filteredSchemaTables"
                :key="table.name"
                class="query-schema-table"
              >
                <button
                  type="button"
                  class="query-schema-table-trigger"
                  @click="insertIdentifier(table.name)"
                >
                  <TableCellsIcon class="h-3.5 w-3.5 shrink-0 text-secondary" />
                  <span class="truncate font-mono">{{ table.name }}</span>
                  <span class="ml-auto text-[10px] text-text-tertiary">{{ table.columns.length }}</span>
                </button>
                <div class="query-schema-columns">
                  <button
                    v-for="column in table.columns.slice(0, 8)"
                    :key="column.name"
                    type="button"
                    class="query-schema-column"
                    @click="insertIdentifier(column.name)"
                  >
                    <span class="truncate">{{ column.name }}</span>
                    <span class="query-column-type">{{ column.dataType }}</span>
                  </button>
                </div>
              </div>
            </div>
          </aside>

          <section class="query-editor-pane">
            <!-- Monaco Editor -->
            <div class="relative border-b border-border-default" :style="{ height: editorHeight + 'px' }">
              <!-- Loading indicator -->
              <Transition name="fade">
                <div
                  v-if="editorLoading"
                  class="absolute inset-0 flex items-center justify-center bg-surface-base z-10"
                >
                  <div class="flex flex-col items-center gap-3">
                    <div class="w-6 h-6 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
                    <OuiText size="sm" color="tertiary">Loading editor...</OuiText>
                  </div>
                </div>
              </Transition>
              <OuiFileEditor
                ref="editorRef"
                v-model="activeTab.content"
                language="sql"
                :height="editorHeight + 'px'"
                :minimap="{ enabled: false }"
                :folding="false"
                container-class="w-full border-0 rounded-none"
                @vue:mounted="onEditorMounted"
              />
              <!-- Resize handle -->
              <div
                class="absolute bottom-0 left-0 right-0 h-1 cursor-row-resize bg-transparent hover:bg-primary/30 active:bg-primary/50 transition-colors"
                @mousedown="startResize"
              />
            </div>

            <!-- Toolbar -->
            <div class="query-editor-toolbar">
              <OuiFlex gap="md" align="center" wrap="wrap">
                <OuiFlex gap="xs" align="center">
                  <OuiText color="tertiary" size="xs">Limit:</OuiText>
                  <select
                    v-model="maxRows"
                    class="text-xs bg-surface-overlay border border-border-default rounded-md px-2 py-1 focus:outline-none focus:ring-1 focus:ring-primary/50"
                  >
                    <option value="100">100</option>
                    <option value="500">500</option>
                    <option value="1000">1,000</option>
                    <option value="5000">5,000</option>
                    <option value="10000">10,000</option>
                  </select>
                </OuiFlex>
                <div class="h-4 w-px bg-border-default hidden sm:block" />
                <OuiFlex gap="xs" align="center" class="text-secondary">
                  <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-surface-overlay border border-border-default rounded">Ctrl</kbd>
                  <span class="text-[10px]">+</span>
                  <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-surface-overlay border border-border-default rounded">Enter</kbd>
                  <OuiText size="xs" color="tertiary" class="ml-1">Execute</OuiText>
                </OuiFlex>
                <OuiText v-if="activeTab.content.trim()" size="xs" color="tertiary">
                  {{ activeTab.content.trim().split(/\s+/).length }} tokens
                </OuiText>
              </OuiFlex>

              <OuiFlex gap="sm" wrap="wrap">
                <OuiButton
                  v-if="activeTab.results"
                  variant="ghost"
                  color="secondary"
                  size="sm"
                  @click="exportResults('csv')"
                >
                  <ArrowDownTrayIcon class="h-3.5 w-3.5" />
                  CSV
                </OuiButton>
                <OuiButton
                  v-if="activeTab.results"
                  variant="ghost"
                  color="secondary"
                  size="sm"
                  @click="exportResults('json')"
                >
                  <ArrowDownTrayIcon class="h-3.5 w-3.5" />
                  JSON
                </OuiButton>
                <OuiButton
                  color="primary"
                  size="sm"
                  :loading="executing"
                  :disabled="!activeTab.content.trim()"
                  @click="executeCurrentQuery()"
                >
                  <PlayIcon class="h-3.5 w-3.5" />
                  Run Query
                </OuiButton>
              </OuiFlex>
            </div>
          </section>
        </div>
      </OuiCardBody>
    </OuiCard>

    <!-- Results Card -->
    <OuiCard v-if="activeTab.results || activeTab.error">
      <OuiCardHeader>
        <OuiFlex justify="between" align="center" class="w-full">
          <OuiFlex align="center" gap="sm">
            <template v-if="activeTab.results">
              <div class="flex items-center gap-2">
                <div class="w-2 h-2 rounded-full bg-success animate-pulse" />
                <OuiText weight="semibold">Results</OuiText>
              </div>
              <OuiBadge color="primary" size="sm">
                {{ formatNumber(activeTab.results.rowCount) }} rows
              </OuiBadge>
              <OuiBadge v-if="activeTab.results.truncated" color="warning" size="sm">
                Truncated
              </OuiBadge>
            </template>
            <template v-else-if="activeTab.error">
              <div class="flex items-center gap-2">
                <div class="w-2 h-2 rounded-full bg-danger" />
                <OuiText weight="semibold" color="danger">Error</OuiText>
              </div>
            </template>
          </OuiFlex>
          <OuiFlex v-if="activeTab.results" gap="sm" align="center">
            <OuiBadge color="tertiary" size="sm">
              <ClockIcon class="h-3 w-3 mr-1" />
              {{ activeTab.results.executionTimeMs }}ms
            </OuiBadge>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>

      <OuiCardBody class="p-0">
        <!-- Error display -->
        <div v-if="activeTab.error" class="p-4">
          <OuiAlert color="danger">
            <div class="flex items-start gap-3">
              <ExclamationTriangleIcon class="h-5 w-5 shrink-0 mt-0.5" />
              <div>
                <OuiText weight="semibold" class="mb-1">Query Failed</OuiText>
                <pre class="text-sm font-mono whitespace-pre-wrap opacity-90">{{ activeTab.error }}</pre>
              </div>
            </div>
          </OuiAlert>
        </div>

        <!-- Results table -->
        <div v-else-if="activeTab.results" class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead class="sticky top-0 z-10">
              <tr class="bg-surface-base border-b border-border-default">
                <th class="px-3 py-2.5 text-left text-xs font-medium text-secondary w-12">#</th>
                <th
                  v-for="col in activeTab.results.columns"
                  :key="col.name"
                  class="px-3 py-2.5 text-left font-medium text-xs cursor-pointer hover:bg-interactive-hover select-none transition-colors group"
                  @click="toggleSort(col.name)"
                >
                  <div class="flex items-center gap-1.5">
                    <span>{{ col.name }}</span>
                    <span class="text-secondary/60 font-normal text-[10px] uppercase">{{ col.dataType }}</span>
                    <ChevronUpDownIcon
                      v-if="sortColumn !== col.name"
                      class="h-3 w-3 text-secondary/40 opacity-0 group-hover:opacity-100 transition-opacity"
                    />
                    <ChevronUpIcon
                      v-else-if="sortDirection === 'ASC'"
                      class="h-3 w-3 text-primary"
                    />
                    <ChevronDownIcon
                      v-else
                      class="h-3 w-3 text-primary"
                    />
                  </div>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border-default/50">
              <tr
                v-for="(row, rowIdx) in sortedRows"
                :key="rowIdx"
                class="hover:bg-interactive-hover/50 transition-colors"
              >
                <td class="px-3 py-2 text-xs text-secondary font-mono">{{ Number(rowIdx) + 1 }}</td>
                <td
                  v-for="col in activeTab.results.columns"
                  :key="col.name"
                  class="px-3 py-2 font-mono text-xs max-w-xs cursor-pointer group"
                  :title="getCellValue(row, col.name)"
                  @click="showCellDetail(row, col)"
                >
                  <div class="flex items-center gap-1">
                    <span
                      v-if="row[col.name] === null"
                      class="text-secondary/50 italic"
                    >NULL</span>
                    <span
                      v-else-if="isJsonColumn(col.dataType)"
                      class="text-info truncate"
                    >{{ formatJsonPreview(row[col.name]) }}</span>
                    <span
                      v-else-if="isBooleanValue(row[col.name])"
                      class="inline-flex items-center"
                    >
                      <span
                        class="w-2 h-2 rounded-full mr-1.5"
                        :class="row[col.name] === 'true' || row[col.name] === true ? 'bg-success' : 'bg-secondary/30'"
                      />
                      {{ row[col.name] }}
                    </span>
                    <span v-else class="truncate">{{ row[col.name] }}</span>
                    <ClipboardIcon
                      class="h-3 w-3 text-secondary/30 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                    />
                  </div>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Empty state -->
          <div v-if="sortedRows.length === 0" class="py-12 text-center">
            <TableCellsIcon class="h-12 w-12 mx-auto text-secondary/30 mb-3" />
            <OuiText color="tertiary">No rows returned</OuiText>
          </div>
        </div>

        <!-- Truncation warning -->
        <OuiAlert v-if="activeTab.results?.truncated" color="warning" class="m-4 mt-0">
          <div class="flex items-center gap-2">
            <ExclamationTriangleIcon class="h-3.5 w-3.5" />
            <OuiText size="sm">Results truncated to {{ formatNumber(parseInt(maxRows)) }} rows. Increase the limit or add filters to see more data.</OuiText>
          </div>
        </OuiAlert>
      </OuiCardBody>
    </OuiCard>

    <!-- Cell Detail Modal -->
    <OuiDialog v-model:open="cellDetailOpen" title="Cell Value" size="lg">
      <div v-if="cellDetail" class="space-y-4">
        <div class="flex items-center gap-2 text-sm">
          <OuiBadge color="tertiary">{{ cellDetail.column.name }}</OuiBadge>
          <OuiText color="tertiary" size="xs">{{ cellDetail.column.dataType }}</OuiText>
        </div>
        <div class="relative">
          <pre
            class="text-sm font-mono bg-surface-base border border-border-default rounded-lg p-4 max-h-96 overflow-auto whitespace-pre-wrap break-all"
          >{{ formatCellForDisplay(cellDetail.value) }}</pre>
          <button
            class="absolute top-2 right-2 p-1.5 text-secondary hover:text-primary bg-surface-overlay border border-border-default rounded-md transition-colors"
            @click="copyCellValue"
            title="Copy to clipboard"
          >
            <ClipboardIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      <template #footer>
        <OuiButton variant="ghost" @click="cellDetailOpen = false">Close</OuiButton>
        <OuiButton color="primary" @click="copyCellValue">
          <ClipboardIcon class="h-3.5 w-3.5" />
          Copy Value
        </OuiButton>
      </template>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  PlayIcon,
  ClockIcon,
  ArrowDownTrayIcon,
  PlusIcon,
  XMarkIcon,
  SparklesIcon,
  CommandLineIcon,
  ExclamationTriangleIcon,
  ChevronUpDownIcon,
  ChevronUpIcon,
  ChevronDownIcon,
  ClipboardIcon,
  TableCellsIcon,
} from "@heroicons/vue/24/outline";
import { ref, computed, onMounted, onUnmounted, watch, nextTick, toRef } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { useDatabaseSchema } from "~/composables/useDatabaseSchema";

const props = defineProps<{
  databaseId: string;
  databaseType: string;
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);
const editorRef = ref<any>(null);

// Schema for autocomplete
const { tables, views, functions, fetchSchema } = useDatabaseSchema(
  toRef(props, "databaseId")
);

// Query tabs
interface QueryTab {
  title: string;
  content: string;
  results: any;
  error: string | null;
}

const queryTabs = ref<QueryTab[]>([
  { title: "Query 1", content: "", results: null, error: null },
]);
const activeTabIdx = ref(0);
const activeTab = computed(() => queryTabs.value[activeTabIdx.value]!);

const maxRows = ref("1000");
const executing = ref(false);
const showHistory = ref(false);
const showSnippets = ref(false);
const editorLoading = ref(true);
const schemaSearch = ref("");

// SQL Snippets
const sqlSnippets = computed(() => {
  const tableList = tables.value.map(t => t.name).join(", ") || "table_name";
  const firstTable = tables.value[0]?.name || "table_name";
  return [
    { label: "Select All", preview: `SELECT * FROM ${firstTable}`, code: `SELECT * FROM ${firstTable} LIMIT 100;` },
    { label: "Select Count", preview: `SELECT COUNT(*) FROM...`, code: `SELECT COUNT(*) FROM ${firstTable};` },
    { label: "Select with Where", preview: "SELECT ... WHERE ...", code: `SELECT *\nFROM ${firstTable}\nWHERE column = 'value'\nLIMIT 100;` },
    { label: "Insert Row", preview: "INSERT INTO ...", code: `INSERT INTO ${firstTable} (column1, column2)\nVALUES ('value1', 'value2');` },
    { label: "Update Row", preview: "UPDATE ... SET ...", code: `UPDATE ${firstTable}\nSET column = 'new_value'\nWHERE id = 1;` },
    { label: "Delete Row", preview: "DELETE FROM ...", code: `DELETE FROM ${firstTable}\nWHERE id = 1;` },
    { label: "Create Table", preview: "CREATE TABLE ...", code: `CREATE TABLE new_table (\n  id SERIAL PRIMARY KEY,\n  name VARCHAR(255) NOT NULL,\n  created_at TIMESTAMPTZ DEFAULT NOW()\n);` },
    { label: "Join Tables", preview: "SELECT ... JOIN ...", code: `SELECT a.*, b.*\nFROM table_a a\nINNER JOIN table_b b ON a.id = b.a_id\nLIMIT 100;` },
  ];
});

// Query history
const historyKey = computed(() => `db-query-history-${props.databaseId}`);
const queryHistory = ref<string[]>([]);

const filteredSchemaTables = computed(() => {
  const q = schemaSearch.value.trim().toLowerCase();
  if (!q) return tables.value;

  return tables.value.filter((table) => {
    return table.name.toLowerCase().includes(q) ||
      table.columns.some((column) => column.name.toLowerCase().includes(q));
  });
});

// Sort state
const sortColumn = ref<string | null>(null);
const sortDirection = ref<"ASC" | "DESC">("ASC");

// Cell detail modal
const cellDetailOpen = ref(false);
const cellDetail = ref<{ column: any; value: any } | null>(null);

// Editor resize
const editorHeight = ref(280);
let resizing = false;
let startY = 0;
let startHeight = 0;

function startResize(e: MouseEvent) {
  resizing = true;
  startY = e.clientY;
  startHeight = editorHeight.value;
  document.addEventListener("mousemove", onResize);
  document.addEventListener("mouseup", stopResize);
  e.preventDefault();
}

function onResize(e: MouseEvent) {
  if (!resizing) return;
  const delta = e.clientY - startY;
  editorHeight.value = Math.max(120, Math.min(600, startHeight + delta));
}

function stopResize() {
  resizing = false;
  document.removeEventListener("mousemove", onResize);
  document.removeEventListener("mouseup", stopResize);
}

// Tab management
function addTab() {
  const num = queryTabs.value.length + 1;
  queryTabs.value.push({
    title: `Query ${num}`,
    content: "",
    results: null,
    error: null,
  });
  activeTabIdx.value = queryTabs.value.length - 1;
}

function closeTab(idx: number) {
  if (queryTabs.value.length <= 1) return;
  queryTabs.value.splice(idx, 1);
  if (activeTabIdx.value >= queryTabs.value.length) {
    activeTabIdx.value = queryTabs.value.length - 1;
  }
}

// Update tab title from query content
watch(
  () => activeTab.value.content,
  (content) => {
    if (!content) return;
    const firstLine = content.split("\n")[0]!.trim().slice(0, 24);
    if (firstLine) {
      queryTabs.value[activeTabIdx.value]!.title = firstLine || `Query ${activeTabIdx.value + 1}`;
    }
  }
);

// Query execution
async function executeCurrentQuery(selectedOnly = false) {
  const tab = activeTab.value;
  let queryText = tab.content.trim();

  if (selectedOnly && editorRef.value) {
    const editor = editorRef.value.editor?.();
    if (editor) {
      const selection = editor.getSelection();
      if (selection && !selection.isEmpty()) {
        const model = editor.getModel();
        if (model) {
          queryText = model.getValueInRange(selection).trim();
        }
      }
    }
  }

  if (!queryText) return;

  executing.value = true;
  tab.error = null;
  tab.results = null;

  try {
    if (!organizationId.value) return;
    const res = await dbClient.executeQuery({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      query: queryText,
      maxRows: parseInt(maxRows.value) || 1000,
      timeoutSeconds: 30,
    });

    // Transform rows to objects
    const rows = ((res as any).rows || []).map((row: any) => {
      const obj: Record<string, any> = {};
      for (const cell of row.cells || []) {
        obj[cell.columnName] = cell.isNull ? null : cell.value;
      }
      return obj;
    });

    tab.results = {
      columns: (res as any).columns || [],
      rows,
      rowCount: (res as any).rowCount,
      truncated: (res as any).truncated,
      executionTimeMs: (res as any).executionTimeMs,
    };

    addToHistory(queryText);
    sortColumn.value = null;
    toast.success(`Query returned ${formatNumber(tab.results.rowCount)} rows`);
  } catch (err: unknown) {
    tab.error = (err as Error).message || "Query execution failed";
    toast.error("Query failed");
  } finally {
    executing.value = false;
  }
}

// Editor loading
function onEditorMounted() {
  // Give Monaco a moment to fully initialize
  setTimeout(() => {
    editorLoading.value = false;
  }, 300);
}

// Snippets
function insertSnippet(code: string) {
  activeTab.value.content = code;
  showSnippets.value = false;
}

function insertIdentifier(identifier: string) {
  const quoted = `"${identifier.replace(/"/g, "\"\"")}"`;
  const editor = editorRef.value?.editor?.();

  if (editor) {
    editor.trigger("keyboard", "type", { text: quoted });
    editor.focus();
    return;
  }

  activeTab.value.content = activeTab.value.content
    ? `${activeTab.value.content} ${quoted}`
    : quoted;
}

// History management
function addToHistory(query: string) {
  const history = queryHistory.value;
  const idx = history.indexOf(query);
  if (idx > -1) history.splice(idx, 1);
  history.unshift(query);
  if (history.length > 50) history.pop();
  try {
    localStorage.setItem(historyKey.value, JSON.stringify(history));
  } catch {
    // ignore
  }
}

function loadFromHistory(query: string) {
  activeTab.value.content = query;
  showHistory.value = false;
}

function loadHistory() {
  try {
    const stored = localStorage.getItem(historyKey.value);
    if (stored) {
      queryHistory.value = JSON.parse(stored);
    }
  } catch {
    // ignore
  }
}

function clearHistory() {
  queryHistory.value = [];
  localStorage.removeItem(historyKey.value);
  showHistory.value = false;
  toast.success("History cleared");
}

// Sort
function toggleSort(colName: string) {
  if (sortColumn.value === colName) {
    sortDirection.value = sortDirection.value === "ASC" ? "DESC" : "ASC";
  } else {
    sortColumn.value = colName;
    sortDirection.value = "ASC";
  }
}

const sortedRows = computed(() => {
  const results = activeTab.value?.results;
  if (!results?.rows) return [];
  if (!sortColumn.value) return results.rows;

  const col = sortColumn.value;
  const dir = sortDirection.value === "ASC" ? 1 : -1;
  return [...results.rows].sort((a: any, b: any) => {
    const va = a[col];
    const vb = b[col];
    if (va === null && vb === null) return 0;
    if (va === null) return 1;
    if (vb === null) return -1;
    if (va < vb) return -1 * dir;
    if (va > vb) return 1 * dir;
    return 0;
  });
});

// Cell operations
function getCellValue(row: any, colName: string): string {
  const val = row[colName];
  return val === null ? "NULL" : String(val);
}

function isJsonColumn(dataType: string): boolean {
  return dataType.toLowerCase().includes("json");
}

function isBooleanValue(val: any): boolean {
  return val === "true" || val === "false" || val === true || val === false;
}

function formatJsonPreview(val: any): string {
  try {
    const parsed = typeof val === "string" ? JSON.parse(val) : val;
    return JSON.stringify(parsed).slice(0, 50) + (JSON.stringify(parsed).length > 50 ? "..." : "");
  } catch {
    return String(val).slice(0, 50);
  }
}

function showCellDetail(row: any, column: any) {
  cellDetail.value = {
    column,
    value: row[column.name],
  };
  cellDetailOpen.value = true;
}

function formatCellForDisplay(val: any): string {
  if (val === null) return "NULL";
  try {
    const parsed = typeof val === "string" ? JSON.parse(val) : val;
    if (typeof parsed === "object") {
      return JSON.stringify(parsed, null, 2);
    }
  } catch {
    // not JSON
  }
  return String(val);
}

async function copyCellValue() {
  if (!cellDetail.value) return;
  try {
    await navigator.clipboard.writeText(formatCellForDisplay(cellDetail.value.value));
    toast.success("Copied to clipboard");
    cellDetailOpen.value = false;
  } catch {
    toast.error("Failed to copy");
  }
}

// Format helpers
function formatNumber(n: number): string {
  return new Intl.NumberFormat().format(n);
}

// Export
function exportResults(format: "csv" | "json") {
  const results = activeTab.value?.results;
  if (!results) return;

  let content: string;
  let filename: string;
  let mimeType: string;

  if (format === "csv") {
    const headers = results.columns.map((c: any) => c.name);
    const csvRows = [headers.join(",")];
    for (const row of results.rows) {
      const values = headers.map((h: string) => {
        const val = row[h];
        if (val === null) return "";
        const str = String(val);
        if (str.includes(",") || str.includes('"') || str.includes("\n")) {
          return `"${str.replace(/"/g, '""')}"`;
        }
        return str;
      });
      csvRows.push(values.join(","));
    }
    content = csvRows.join("\n");
    filename = `query-results-${Date.now()}.csv`;
    mimeType = "text/csv";
  } else {
    content = JSON.stringify(results.rows, null, 2);
    filename = `query-results-${Date.now()}.json`;
    mimeType = "application/json";
  }

  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
  toast.success(`Exported as ${format.toUpperCase()}`);
}

// Context-aware SQL autocomplete
async function registerAutocomplete() {
  await nextTick();
  const checkEditor = () => {
    if (!editorRef.value) return;
    const monacoInstance = editorRef.value.monaco?.();
    const editorInstance = editorRef.value.editor?.();
    if (!monacoInstance || !editorInstance) {
      setTimeout(checkEditor, 200);
      return;
    }

    // Keyboard shortcuts
    editorInstance.addCommand(
      monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyCode.Enter,
      () => executeCurrentQuery()
    );
    editorInstance.addCommand(
      monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyMod.Shift | monacoInstance.KeyCode.Enter,
      () => executeCurrentQuery(true)
    );

    registerSQLCompletionProvider(monacoInstance);
    setupValidation(monacoInstance, editorInstance);
  };
  setTimeout(checkEditor, 300);
}

let completionProviderDisposable: any = null;
let validationDisposable: any = null;

// Helper to find table by name (case-insensitive)
function findTable(name: string) {
  return tables.value.find(t => t.name.toLowerCase() === name.toLowerCase());
}

function findView(name: string) {
  return views.value.find(v => v.name.toLowerCase() === name.toLowerCase());
}

function cleanIdentifier(identifier: string) {
  return identifier.trim().replace(/^[`"']|[`"']$/g, "");
}

function hasCompleteRelationName(text: string) {
  const trimmed = text.trim();
  if (!trimmed || /,\s*$/.test(trimmed)) return false;

  const [relationName] = trimmed.split(/\s+/);
  if (!relationName) return false;

  const cleanName = cleanIdentifier(relationName);
  return !!findTable(cleanName) || !!findView(cleanName);
}

function getInsertableColumns(table: any) {
  return table.columns.filter((col: any) => {
    const dt = col.dataType.toLowerCase();
    const hasGeneratedDefault = !!col.defaultValue || dt.includes("serial") || dt.includes("auto_increment");
    return !col.isPrimaryKey || !hasGeneratedDefault;
  });
}

function getSnippetDefaultForType(dataType: string, colName: string, isPostgres: boolean) {
  const dt = dataType.toLowerCase();

  if (dt.includes("serial") || dt.includes("auto_increment")) return "DEFAULT";
  if (dt.includes("uuid")) return isPostgres ? "gen_random_uuid()" : "UUID()";
  if (dt.includes("timestamp") || dt.includes("datetime")) return "NOW()";
  if (dt.includes("date")) return "'2024-01-01'";
  if (dt.includes("time")) return "'12:00:00'";
  if (dt.includes("bool")) return "true";
  if (dt.includes("int") || dt.includes("numeric") || dt.includes("decimal") || dt.includes("real") || dt.includes("double") || dt.includes("float")) return "0";
  if (dt.includes("json")) return "'{}'";
  if (dt.includes("bytea")) return "E'\\\\x00'";
  if (dt.includes("blob")) return "X'00'";

  return `'${colName}_value'`;
}

function buildInsertSnippet(table: any, isPostgres: boolean) {
  const insertColumns = getInsertableColumns(table).slice(0, 8);
  const columns = insertColumns.length > 0 ? insertColumns : table.columns.slice(0, 8);
  const columnList = columns.map((col: any) => col.name).join(", ");
  const valueList = columns.map((col: any, idx: number) => {
    const hint = getSnippetDefaultForType(col.dataType, col.name, isPostgres);
    return `\${${idx + 1}:${hint}}`;
  }).join(", ");

  return `INSERT INTO ${table.name} (${columnList})\nVALUES (${valueList});`;
}

function buildUpdateSnippet(table: any, isPostgres: boolean) {
  const writableColumn = getInsertableColumns(table).find((col: any) => !col.isPrimaryKey) || table.columns.find((col: any) => !col.isPrimaryKey);
  const keyColumn = table.columns.find((col: any) => col.isPrimaryKey) || table.columns[0];
  const setHint = writableColumn ? getSnippetDefaultForType(writableColumn.dataType, writableColumn.name, isPostgres) : "value";
  const keyHint = keyColumn ? getSnippetDefaultForType(keyColumn.dataType, keyColumn.name, isPostgres) : "id";

  return `UPDATE ${table.name}\nSET ${writableColumn?.name || "column_name"} = \${1:${setHint}}\nWHERE ${keyColumn?.name || "id"} = \${2:${keyHint}};`;
}

function buildDeleteSnippet(table: any) {
  const keyColumn = table.columns.find((col: any) => col.isPrimaryKey) || table.columns[0];
  return `DELETE FROM ${table.name}\nWHERE ${keyColumn?.name || "id"} = \${1:value};`;
}

// Extract table name from INSERT INTO or UPDATE statement
function extractTargetTable(text: string): string | null {
  // INSERT INTO table_name
  const insertMatch = text.match(/INSERT\s+INTO\s+["'`]?(\w+)["'`]?/i);
  if (insertMatch && insertMatch[1]) return insertMatch[1];

  // UPDATE table_name
  const updateMatch = text.match(/UPDATE\s+["'`]?(\w+)["'`]?/i);
  if (updateMatch && updateMatch[1]) return updateMatch[1];

  return null;
}

// Extract columns specified in INSERT INTO table (col1, col2, ...)
function extractInsertColumns(text: string): string[] {
  const match = text.match(/INSERT\s+INTO\s+["'`]?\w+["'`]?\s*\(([^)]*)/i);
  if (!match || !match[1]) return [];
  return match[1].split(',').map((c: string) => c.trim().replace(/["'`]/g, '')).filter((c: string) => c);
}

// Check if cursor is inside parentheses after INSERT INTO table
function isInsideInsertColumns(textBeforeCursor: string): boolean {
  const match = textBeforeCursor.match(/INSERT\s+INTO\s+["'`]?\w+["'`]?\s*\([^)]*$/i);
  return !!match;
}

// Check if cursor is inside VALUES (...)
function isInsideValues(textBeforeCursor: string): boolean {
  const match = textBeforeCursor.match(/VALUES\s*\([^)]*$/i);
  return !!match;
}

// Check if cursor is after UPDATE table SET
function isAfterUpdateSet(textBeforeCursor: string): boolean {
  return /UPDATE\s+["'`]?\w+["'`]?\s+SET\s+[^;]*$/i.test(textBeforeCursor);
}

function registerSQLCompletionProvider(monacoInstance: any) {
  if (completionProviderDisposable) {
    completionProviderDisposable.dispose();
  }

  const isPostgres = props.databaseType === "1" || props.databaseType === "POSTGRESQL";

  // Strictly context-aware keyword mapping
  // Each context only shows keywords that are valid to follow it
  const contextKeywords: Record<string, string[]> = {
    // Start of statement - only statement starters
    START: ["SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE", "WITH", "EXPLAIN", "ANALYZE", "BEGIN", "COMMIT", "ROLLBACK"],

    // After SELECT - column expressions
    SELECT_COLUMNS: ["DISTINCT", "ALL", "*", "CASE", "WHEN", "FROM"],

    // Relation contexts only offer relations until one has actually been typed.
    FROM: [],
    AFTER_TABLE_SOURCE: ["WHERE", "INNER JOIN", "LEFT JOIN", "RIGHT JOIN", "FULL JOIN", "CROSS JOIN", "GROUP BY", "ORDER BY", "LIMIT", "OFFSET", "UNION", "INTERSECT", "EXCEPT"],

    // After JOIN
    JOIN: [],
    AFTER_JOIN_TABLE: ["ON", "USING"],

    // After ON - conditions
    ON: ["AND", "OR", "WHERE", "INNER", "LEFT", "RIGHT", "FULL", "CROSS", "JOIN", "ORDER", "GROUP"],

    // After WHERE - conditions
    WHERE: ["AND", "OR", "NOT", "IN", "BETWEEN", "LIKE", "ILIKE", "IS", "EXISTS", "ANY", "ALL", "SOME", "ORDER", "GROUP", "HAVING", "LIMIT"],

    // After ORDER BY
    ORDER_BY: ["ASC", "DESC", "NULLS", "LIMIT", "OFFSET"],

    // After GROUP BY
    GROUP_BY: ["HAVING", "ORDER", "LIMIT"],

    // After HAVING
    HAVING: ["AND", "OR", "ORDER", "LIMIT"],

    // After INSERT INTO
    INSERT_INTO: [], // Will suggest tables
    INSERT_TABLE_READY: ["VALUES", "DEFAULT VALUES", "SELECT"],

    // After INSERT INTO table
    INSERT_COLUMNS: [], // Will suggest columns

    // After VALUES
    VALUES: ["DEFAULT", "NULL"],

    // After UPDATE
    UPDATE: [], // Will suggest tables
    UPDATE_TABLE_READY: ["SET"],

    // After UPDATE table SET
    UPDATE_SET: ["WHERE"],

    // After DELETE
    DELETE: ["FROM"],

    // After DELETE FROM
    DELETE_FROM: ["WHERE", "RETURNING"],
    DELETE_FROM_READY: ["WHERE", "RETURNING"],

    // DDL contexts
    CREATE: ["TABLE", "INDEX", "VIEW", "SCHEMA", "DATABASE", "SEQUENCE", "TRIGGER", "FUNCTION", "PROCEDURE", "TYPE", "EXTENSION"],
    ALTER: ["TABLE", "INDEX", "VIEW", "SCHEMA", "DATABASE", "SEQUENCE", "COLUMN"],
    DROP: ["TABLE", "INDEX", "VIEW", "SCHEMA", "DATABASE", "SEQUENCE", "TRIGGER", "FUNCTION", "CONSTRAINT", "COLUMN"],
    TRUNCATE: [], // Will suggest tables
    TRUNCATE_TABLE_READY: ["CASCADE", "RESTART IDENTITY"],
  };

  const aggregateFunctions = ["COUNT", "SUM", "AVG", "MIN", "MAX", "ARRAY_AGG", "STRING_AGG", "JSON_AGG", "BOOL_AND", "BOOL_OR"];
  const scalarFunctions = isPostgres
    ? ["COALESCE", "NULLIF", "GREATEST", "LEAST", "LENGTH", "LOWER", "UPPER", "TRIM", "SUBSTRING", "REPLACE", "CONCAT", "CAST", "TO_CHAR", "TO_DATE", "TO_TIMESTAMP", "EXTRACT", "DATE_TRUNC", "NOW", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "GEN_RANDOM_UUID", "ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD", "FIRST_VALUE", "LAST_VALUE", "ABS", "CEIL", "FLOOR", "ROUND", "MOD", "POWER", "SQRT"]
    : ["COALESCE", "IFNULL", "NULLIF", "GREATEST", "LEAST", "LENGTH", "LOWER", "UPPER", "TRIM", "SUBSTRING", "REPLACE", "CONCAT", "CAST", "DATE_FORMAT", "STR_TO_DATE", "NOW", "CURDATE", "CURTIME", "DATEDIFF", "DATE_ADD", "DATE_SUB", "UUID", "ABS", "CEIL", "FLOOR", "ROUND", "MOD", "POWER", "SQRT"];

  // Built-in functions for VALUE context suggestions
  const builtinFunctions = isPostgres
    ? ["NOW()", "CURRENT_TIMESTAMP", "CURRENT_DATE", "GEN_RANDOM_UUID()", "DEFAULT"]
    : ["NOW()", "CURRENT_TIMESTAMP", "CURDATE()", "UUID()", "DEFAULT"];

  completionProviderDisposable = monacoInstance.languages.registerCompletionItemProvider("sql", {
    triggerCharacters: [".", " ", ",", "("],
    provideCompletionItems(model: any, position: any) {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      // Get full text and text before cursor
      const fullText = model.getValue();
      const textBeforeCursor = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      });
      const textBeforeCursorUpper = textBeforeCursor.toUpperCase();

      const lineText = model.getValueInRange({
        startLineNumber: position.lineNumber,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      });

      const suggestions: any[] = [];

      // Check for table.column context (highest priority)
      const dotMatch = lineText.match(/(\w+)\.\s*$/);
      if (dotMatch) {
        const tableName = dotMatch[1];
        const table = findTable(tableName);
        if (table) {
          return {
            suggestions: table.columns.map((col) => ({
              label: col.name,
              kind: monacoInstance.languages.CompletionItemKind.Field,
              detail: `${col.dataType}${col.isPrimaryKey ? " (PK)" : ""}${col.isNullable ? "" : " NOT NULL"}`,
              documentation: col.defaultValue ? `Default: ${col.defaultValue}` : undefined,
              insertText: col.name,
              sortText: "0" + col.name,
              range,
            })),
          };
        }
      }

      // INSERT INTO table_name ( ... ) - suggest columns
      if (isInsideInsertColumns(textBeforeCursorUpper)) {
        const tableName = extractTargetTable(textBeforeCursor);
        if (tableName) {
          const table = findTable(tableName);
          if (table) {
            const alreadyUsed = extractInsertColumns(textBeforeCursor).map(c => c.toLowerCase());
            for (const col of table.columns) {
              if (alreadyUsed.includes(col.name.toLowerCase())) continue;
              suggestions.push({
                label: col.name,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: `${col.dataType}${col.isPrimaryKey ? " (PK)" : ""}${col.isNullable ? " NULL" : " NOT NULL"}`,
                documentation: col.defaultValue ? `Default: ${col.defaultValue}` : undefined,
                insertText: col.name,
                sortText: col.isPrimaryKey ? "0" + col.name : "1" + col.name,
                range,
              });
            }
            return { suggestions };
          }
        }
      }

      // VALUES (...) - suggest value placeholders based on column types
      if (isInsideValues(textBeforeCursorUpper)) {
        const tableName = extractTargetTable(textBeforeCursor);
        if (tableName) {
          const table = findTable(tableName);
          if (table) {
            const insertCols = extractInsertColumns(textBeforeCursor);
            // Count how many values we've already entered
            const valuesMatch = textBeforeCursor.match(/VALUES\s*\(([^)]*$)/i);
            const valuesText = valuesMatch?.[1] ?? "";
            const valueCount = valuesText.split(',').filter((v: string) => v.trim()).length;

            // Determine which column we're filling
            const columnsToUse = insertCols.length > 0
              ? insertCols.map(c => table.columns.find(tc => tc.name.toLowerCase() === c.toLowerCase())).filter(Boolean)
              : table.columns;

            const currentColIndex = valueCount;
            const currentCol = columnsToUse[currentColIndex];

            if (currentCol) {
              const hint = getValueHintForType(currentCol.dataType, currentCol.name, isPostgres);
              suggestions.push({
                label: hint.label,
                kind: monacoInstance.languages.CompletionItemKind.Value,
                detail: `${currentCol.name} (${currentCol.dataType})`,
                insertText: hint.insertText,
                sortText: "0",
                range,
              });
            }

            // Also suggest functions appropriate for the type
            for (const fn of builtinFunctions) {
              suggestions.push({
                label: fn,
                kind: monacoInstance.languages.CompletionItemKind.Function,
                detail: "Built-in function",
                insertText: fn.includes("(") ? fn : fn + "()",
                sortText: "1" + fn,
                range,
              });
            }

            // NULL if nullable
            if (currentCol?.isNullable) {
              suggestions.push({
                label: "NULL",
                kind: monacoInstance.languages.CompletionItemKind.Constant,
                detail: "Null value",
                insertText: "NULL",
                sortText: "2",
                range,
              });
            }

            // DEFAULT
            suggestions.push({
              label: "DEFAULT",
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              detail: "Use column default",
              insertText: "DEFAULT",
              sortText: "3",
              range,
            });

            return { suggestions };
          }
        }
      }

      // UPDATE table SET ... - suggest columns
      if (isAfterUpdateSet(textBeforeCursorUpper)) {
        const tableName = extractTargetTable(textBeforeCursor);
        if (tableName) {
          const table = findTable(tableName);
          if (table) {
            for (const col of table.columns) {
              suggestions.push({
                label: col.name,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: col.dataType,
                insertText: `${col.name} = `,
                sortText: "0" + col.name,
                range,
              });
            }
            // WHERE keyword
            suggestions.push({
              label: "WHERE",
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: "WHERE ",
              sortText: "1WHERE",
              range,
            });
            return { suggestions };
          }
        }
      }

      // Determine context for other cases
      const context = determineContext(textBeforeCursorUpper);

      // Get valid keywords for this context (always an array, never undefined)
      const validKeywords: string[] = contextKeywords[context] ?? contextKeywords.START ?? [];

      // Context-specific suggestions
      switch (context) {
        case "START":
          // Real schema-backed statement snippets first, then bare statement starters.
          for (const t of tables.value.slice(0, 8)) {
            suggestions.push({
              label: `SELECT from ${t.name}`,
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: `Working SELECT for ${t.name}`,
              insertText: `SELECT *\nFROM ${t.name}\nLIMIT \${1:100};`,
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "0" + t.name,
              range,
            });
            suggestions.push({
              label: `INSERT into ${t.name}`,
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: `Working INSERT for ${t.name}`,
              insertText: buildInsertSnippet(t, isPostgres),
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "1" + t.name,
              range,
            });
            suggestions.push({
              label: `UPDATE ${t.name}`,
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: `Working UPDATE for ${t.name}`,
              insertText: buildUpdateSnippet(t, isPostgres),
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "2" + t.name,
              range,
            });
            suggestions.push({
              label: `DELETE from ${t.name}`,
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: `Working DELETE for ${t.name}`,
              insertText: buildDeleteSnippet(t),
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "3" + t.name,
              range,
            });
          }
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "9" + kw,
              range,
            });
          }
          break;

        case "SELECT_COLUMNS":
          // Columns, aggregate functions, scalar functions, and valid keywords
          for (const t of tables.value) {
            suggestions.push({
              label: `${t.name}.*`,
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: "All columns from table",
              insertText: `${t.name}.*`,
              sortText: "1" + t.name,
              range,
            });
            for (const col of t.columns) {
              suggestions.push({
                label: `${t.name}.${col.name}`,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: col.dataType,
                insertText: `${t.name}.${col.name}`,
                sortText: "2" + t.name + col.name,
                range,
              });
            }
          }
          // Aggregate functions
          for (const fn of aggregateFunctions) {
            suggestions.push({
              label: fn,
              kind: monacoInstance.languages.CompletionItemKind.Function,
              detail: "Aggregate function",
              insertText: fn + "($0)",
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "3" + fn,
              range,
            });
          }
          // Scalar functions
          for (const fn of scalarFunctions) {
            suggestions.push({
              label: fn,
              kind: monacoInstance.languages.CompletionItemKind.Function,
              detail: "Scalar function",
              insertText: fn + "($0)",
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "4" + fn,
              range,
            });
          }
          // Valid next keywords (FROM, DISTINCT, etc.)
          for (const kw of validKeywords) {
            if (kw === "*") {
              suggestions.push({
                label: "*",
                kind: monacoInstance.languages.CompletionItemKind.Keyword,
                detail: "All columns",
                insertText: "* ",
                sortText: "0*",
                range,
              });
            } else {
              suggestions.push({
                label: kw,
                kind: monacoInstance.languages.CompletionItemKind.Keyword,
                insertText: kw + " ",
                sortText: "5" + kw,
                range,
              });
            }
          }
          break;

        case "FROM":
        case "JOIN":
        case "TRUNCATE":
        case "DELETE_FROM":
          // Only real relations are valid immediately after FROM/JOIN/TRUNCATE.
          for (const t of tables.value) {
            suggestions.push({
              label: t.name,
              kind: monacoInstance.languages.CompletionItemKind.Struct,
              detail: `Table (${Number(t.rowCount)} rows)`,
              documentation: t.columns.map((c) => `${c.name}: ${c.dataType}`).join("\n"),
              insertText: t.name + " ",
              sortText: "0" + t.name,
              range,
            });
          }
          for (const v of views.value) {
            suggestions.push({
              label: v.name,
              kind: monacoInstance.languages.CompletionItemKind.Interface,
              detail: "View",
              insertText: v.name + " ",
              sortText: "1" + v.name,
              range,
            });
          }
          break;

        case "AFTER_TABLE_SOURCE":
        case "AFTER_JOIN_TABLE":
        case "DELETE_FROM_READY":
        case "TRUNCATE_TABLE_READY":
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "0" + kw,
              range,
            });
          }
          break;

        case "INSERT_INTO":
          // Only tables
          for (const t of tables.value) {
            suggestions.push({
              label: t.name,
              kind: monacoInstance.languages.CompletionItemKind.Struct,
              detail: `Table (${t.columns.length} columns)`,
              documentation: t.columns.map((c) => `${c.name}: ${c.dataType}${c.isNullable ? "" : " NOT NULL"}`).join("\n"),
              insertText: t.name + " ",
              sortText: "0" + t.name,
              range,
            });
          }
          break;

        case "INSERT_TABLE_READY": {
          const tableName = extractTargetTable(textBeforeCursor);
          const table = tableName ? findTable(tableName) : null;
          if (table) {
            const insertColumns = getInsertableColumns(table);
            suggestions.push({
              label: "(column list) VALUES",
              kind: monacoInstance.languages.CompletionItemKind.Snippet,
              detail: `Insert ${insertColumns.length || table.columns.length} columns into ${table.name}`,
              insertText: buildInsertSnippet(table, isPostgres).replace(/^INSERT\s+INTO\s+\S+\s*/i, ""),
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "0columns",
              range,
            });
          }
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "1" + kw,
              range,
            });
          }
          break;
        }

        case "UPDATE":
          // Only tables
          for (const t of tables.value) {
            suggestions.push({
              label: t.name,
              kind: monacoInstance.languages.CompletionItemKind.Struct,
              detail: `Table (${t.columns.length} columns)`,
              insertText: t.name + " SET ",
              sortText: "0" + t.name,
              range,
            });
          }
          break;

        case "UPDATE_TABLE_READY":
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "0" + kw,
              range,
            });
          }
          break;

        case "DELETE":
          // Only FROM keyword
          suggestions.push({
            label: "FROM",
            kind: monacoInstance.languages.CompletionItemKind.Keyword,
            insertText: "FROM ",
            sortText: "0",
            range,
          });
          break;

        case "WHERE":
        case "ON":
        case "HAVING":
          // Columns and valid condition keywords (no DDL keywords!)
          for (const t of tables.value) {
            for (const col of t.columns) {
              suggestions.push({
                label: `${t.name}.${col.name}`,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: col.dataType,
                insertText: `${t.name}.${col.name}`,
                sortText: "0" + t.name + col.name,
                range,
            });
          }
          }
          // Scalar functions for conditions
          for (const fn of scalarFunctions) {
            suggestions.push({
              label: fn,
              kind: monacoInstance.languages.CompletionItemKind.Function,
              detail: "Function",
              insertText: fn + "($0)",
              insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              sortText: "2" + fn,
              range,
            });
          }
          // Valid condition keywords
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "1" + kw,
              range,
            });
          }
          break;

        case "ORDER_BY":
          // Columns and ASC/DESC only
          for (const t of tables.value) {
            for (const col of t.columns) {
              suggestions.push({
                label: `${t.name}.${col.name}`,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: col.dataType,
                insertText: `${t.name}.${col.name}`,
                sortText: "0" + t.name + col.name,
                range,
              });
            }
          }
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "1" + kw,
              range,
            });
          }
          break;

        case "GROUP_BY":
          // Columns and HAVING only
          for (const t of tables.value) {
            for (const col of t.columns) {
              suggestions.push({
                label: `${t.name}.${col.name}`,
                kind: monacoInstance.languages.CompletionItemKind.Field,
                detail: col.dataType,
                insertText: `${t.name}.${col.name}`,
                sortText: "0" + t.name + col.name,
                range,
              });
            }
          }
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "1" + kw,
              range,
            });
          }
          break;

        case "CREATE":
        case "ALTER":
        case "DROP":
          // Only DDL object types
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "0" + kw,
              range,
            });
          }
          break;

        default:
          // For unhandled contexts, only show valid keywords from the map
          for (const kw of validKeywords) {
            suggestions.push({
              label: kw,
              kind: monacoInstance.languages.CompletionItemKind.Keyword,
              insertText: kw + " ",
              sortText: "0" + kw,
              range,
            });
          }
      }

      return { suggestions };
    },
  });
}

// Get value hint based on column data type
function getValueHintForType(dataType: string, colName: string, isPostgres: boolean): { label: string; insertText: string } {
  const dt = dataType.toLowerCase();

  if (dt.includes("serial") || dt.includes("auto_increment")) {
    return { label: "DEFAULT (auto)", insertText: "DEFAULT" };
  }
  if (dt.includes("uuid")) {
    return isPostgres
      ? { label: "gen_random_uuid()", insertText: "gen_random_uuid()" }
      : { label: "UUID()", insertText: "UUID()" };
  }
  if (dt.includes("timestamp") || dt.includes("datetime")) {
    return { label: "NOW()", insertText: "NOW()" };
  }
  if (dt.includes("date")) {
    return { label: "'2024-01-01'", insertText: "'${1:2024-01-01}'" };
  }
  if (dt.includes("time")) {
    return { label: "'12:00:00'", insertText: "'${1:12:00:00}'" };
  }
  if (dt.includes("bool")) {
    return { label: "true/false", insertText: "${1|true,false|}" };
  }
  if (dt.includes("int") || dt.includes("numeric") || dt.includes("decimal") || dt.includes("real") || dt.includes("double") || dt.includes("float")) {
    return { label: "0", insertText: "${1:0}" };
  }
  if (dt.includes("json")) {
    return { label: "'{}'", insertText: "'${1:{}}'" };
  }
  if (dt.includes("text") || dt.includes("varchar") || dt.includes("char")) {
    return { label: `'${colName}_value'`, insertText: `'\${1:${colName}_value}'` };
  }
  if (dt.includes("bytea") || dt.includes("blob")) {
    return { label: "E'\\\\x...'", insertText: "E'\\\\x${1:00}'" };
  }

  return { label: "'value'", insertText: "'${1:value}'" };
}

function determineContext(text: string): string {
  // Remove string literals and comments
  const cleaned = text.replace(/'[^']*'/g, "''").replace(/--[^\n]*/g, "").replace(/\/\*[\s\S]*?\*\//g, "");

  // Find all significant keywords
  const keywordRegex = /\b(SELECT|FROM|WHERE|JOIN|ON|AND|OR|ORDER\s+BY|GROUP\s+BY|HAVING|INSERT\s+INTO|INSERT|UPDATE|SET|DELETE|VALUES|CREATE|ALTER|DROP|TRUNCATE|LIMIT|OFFSET|RETURNING|INTO)\b/gi;
  const keywords = cleaned.match(keywordRegex);

  if (!keywords || keywords.length === 0) return "START";

  const lastKeyword = keywords[keywords.length - 1]!.toUpperCase().trim().replace(/\s+/g, " ");

  // SELECT context
  if (lastKeyword === "SELECT") {
    const afterSelect = cleaned.slice(cleaned.toUpperCase().lastIndexOf("SELECT"));
    if (!afterSelect.toUpperCase().includes("FROM")) return "SELECT_COLUMNS";
    return "FROM"; // We're past the SELECT columns
  }

  // FROM context
  if (lastKeyword === "FROM") {
    // Check if this is DELETE FROM
    const beforeFrom = cleaned.slice(0, cleaned.toUpperCase().lastIndexOf("FROM")).trim().toUpperCase();
    const afterFrom = cleaned.slice(cleaned.toUpperCase().lastIndexOf("FROM") + "FROM".length);
    if (beforeFrom.endsWith("DELETE")) {
      return hasCompleteRelationName(afterFrom) ? "DELETE_FROM_READY" : "DELETE_FROM";
    }
    return hasCompleteRelationName(afterFrom) ? "AFTER_TABLE_SOURCE" : "FROM";
  }

  // JOIN context
  if (lastKeyword.includes("JOIN")) {
    const joinIdx = cleaned.toUpperCase().lastIndexOf(lastKeyword);
    const afterJoin = cleaned.slice(joinIdx + lastKeyword.length);
    return hasCompleteRelationName(afterJoin) ? "AFTER_JOIN_TABLE" : "JOIN";
  }

  // ON context (after JOIN ... ON)
  if (lastKeyword === "ON") return "ON";

  // WHERE context
  if (lastKeyword === "WHERE") return "WHERE";

  // AND/OR in WHERE or HAVING
  if (lastKeyword === "AND" || lastKeyword === "OR") {
    // Check if we're in HAVING or WHERE
    const upperCleaned = cleaned.toUpperCase();
    const havingIdx = upperCleaned.lastIndexOf("HAVING");
    const whereIdx = upperCleaned.lastIndexOf("WHERE");
    if (havingIdx > whereIdx) return "HAVING";
    return "WHERE";
  }

  // ORDER BY
  if (lastKeyword === "ORDER BY") return "ORDER_BY";

  // GROUP BY
  if (lastKeyword === "GROUP BY") return "GROUP_BY";

  // HAVING
  if (lastKeyword === "HAVING") return "HAVING";

  // INSERT INTO
  if (lastKeyword === "INSERT INTO" || lastKeyword === "INSERT") {
    const insertIntoIdx = cleaned.toUpperCase().lastIndexOf("INSERT INTO");
    if (insertIntoIdx >= 0) {
      const afterInsertInto = cleaned.slice(insertIntoIdx + "INSERT INTO".length);
      return hasCompleteRelationName(afterInsertInto) ? "INSERT_TABLE_READY" : "INSERT_INTO";
    }
    return "INSERT_INTO";
  }

  // UPDATE
  if (lastKeyword === "UPDATE") {
    const afterUpdate = cleaned.slice(cleaned.toUpperCase().lastIndexOf("UPDATE") + "UPDATE".length);
    return hasCompleteRelationName(afterUpdate) ? "UPDATE_TABLE_READY" : "UPDATE";
  }

  // SET (in UPDATE)
  if (lastKeyword === "SET") return "UPDATE_SET";

  // DELETE
  if (lastKeyword === "DELETE") return "DELETE";

  // VALUES
  if (lastKeyword === "VALUES") return "VALUES";

  // DDL
  if (lastKeyword === "CREATE") return "CREATE";
  if (lastKeyword === "ALTER") return "ALTER";
  if (lastKeyword === "DROP") return "DROP";
  if (lastKeyword === "TRUNCATE") {
    const afterTruncate = cleaned.slice(cleaned.toUpperCase().lastIndexOf("TRUNCATE") + "TRUNCATE".length);
    return hasCompleteRelationName(afterTruncate) ? "TRUNCATE_TABLE_READY" : "TRUNCATE";
  }

  // LIMIT, OFFSET, RETURNING are terminal - suggest statement starters
  if (lastKeyword === "LIMIT" || lastKeyword === "OFFSET" || lastKeyword === "RETURNING") return "START";

  return "START";
}

// SQL validation with inline markers
function validateSQL(monacoInstance: any, model: any) {
  const markers: any[] = [];
  const content = model.getValue();

  // Validate INSERT statements
  validateInsertStatements(content, model, monacoInstance, markers);

  // Validate SELECT statements
  validateSelectStatements(content, model, monacoInstance, markers);

  // Validate UPDATE statements
  validateUpdateStatements(content, model, monacoInstance, markers);

  // Validate DELETE statements
  validateDeleteStatements(content, model, monacoInstance, markers);

  monacoInstance.editor.setModelMarkers(model, "sql-validation", markers);
}

// Validate INSERT statements
function validateInsertStatements(content: string, model: any, monacoInstance: any, markers: any[]) {
  const insertRegex = /INSERT\s+INTO\s+["'`]?(\w+)["'`]?\s*\(([^)]+)\)\s*VALUES\s*\(([^)]+)\)/gi;
  let match;

  while ((match = insertRegex.exec(content)) !== null) {
    const tableName = match[1];
    const columnsStr = match[2];
    const valuesStr = match[3];

    if (!tableName || !columnsStr || !valuesStr) continue;

    const table = findTable(tableName);

    if (!table) {
      addTableNotFoundMarker(monacoInstance, model, markers, tableName, match.index!, match[0]);
      continue;
    }

    const columns = columnsStr.split(',').map((c: string) => c.trim().replace(/["'`]/g, ''));
    const values = parseValues(valuesStr);

    // Check column count matches
    if (columns.length !== values.length) {
      const valuesStart = match.index + match[0].indexOf(valuesStr);
      const startPos = model.getPositionAt(valuesStart);
      const endPos = model.getPositionAt(valuesStart + valuesStr.length);
      markers.push({
        severity: monacoInstance.MarkerSeverity.Error,
        message: `Column count (${columns.length}) doesn't match value count (${values.length})`,
        startLineNumber: startPos.lineNumber,
        startColumn: startPos.column,
        endLineNumber: endPos.lineNumber,
        endColumn: endPos.column,
      });
    }

    // Check each column exists and types match
    for (let i = 0; i < columns.length; i++) {
      const colName = columns[i];
      const value = values[i];
      if (!colName) continue;

      const col = table.columns.find(c => c.name.toLowerCase() === colName.toLowerCase());

      if (!col) {
        const colStart = match.index! + match[0].indexOf(columnsStr) + columnsStr.indexOf(colName);
        const startPos = model.getPositionAt(colStart);
        const endPos = model.getPositionAt(colStart + colName.length);
        markers.push({
          severity: monacoInstance.MarkerSeverity.Error,
          message: `Column '${colName}' not found in table '${tableName}'`,
          startLineNumber: startPos.lineNumber,
          startColumn: startPos.column,
          endLineNumber: endPos.lineNumber,
          endColumn: endPos.column,
        });
        continue;
      }

      // Type validation
      if (value && value !== 'NULL' && value !== 'DEFAULT') {
        const typeError = checkTypeCompatibility(col.dataType, value);
        if (typeError) {
          const valuesStartIdx = match.index! + match[0].indexOf(valuesStr);
          let valueOffset = 0;
          for (let j = 0; j < i; j++) {
            valueOffset = valuesStr.indexOf(',', valueOffset) + 1;
          }
          const valueStart = valuesStartIdx + valueOffset;
          const valueLen = value.length;
          const startPos = model.getPositionAt(valueStart);
          const endPos = model.getPositionAt(valueStart + valueLen);
          markers.push({
            severity: monacoInstance.MarkerSeverity.Warning,
            message: `${typeError} for column '${colName}' (${col.dataType})`,
            startLineNumber: startPos.lineNumber,
            startColumn: startPos.column,
            endLineNumber: endPos.lineNumber,
            endColumn: endPos.column,
          });
        }
      }
    }
  }
}

// Validate SELECT statements
function validateSelectStatements(content: string, model: any, monacoInstance: any, markers: any[]) {
  // Match SELECT ... FROM table_name patterns
  const selectRegex = /SELECT\s+([\s\S]+?)\s+FROM\s+["'`]?(\w+)["'`]?(?:\s+(?:AS\s+)?["'`]?(\w+)["'`]?)?/gi;
  let match;

  while ((match = selectRegex.exec(content)) !== null) {
    const columnsStr = match[1];
    const tableName = match[2];
    const tableAlias = match[3];

    if (!tableName) continue;

    const table = findTable(tableName);

    if (!table) {
      addTableNotFoundMarker(monacoInstance, model, markers, tableName, match.index!, match[0]);
      continue;
    }

    // Parse columns (skip if *)
    if (columnsStr && columnsStr.trim() !== '*') {
      // Extract column references, being careful about aliases and functions
      const columnParts = splitSelectColumns(columnsStr);

      for (const colPart of columnParts) {
        // Skip expressions, functions, literals
        if (colPart.includes('(') || colPart.includes("'") || colPart.includes('"') || /^\d+$/.test(colPart.trim())) {
          continue;
        }

        // Handle table.column or alias.column
        let colName = colPart.trim();
        let targetTableName = tableName;

        if (colName.includes('.')) {
          const [prefix, col] = colName.split('.');
          if (prefix && col) {
            // Check if prefix matches table name or alias
            if (prefix.toLowerCase() === tableName.toLowerCase() ||
                (tableAlias && prefix.toLowerCase() === tableAlias.toLowerCase())) {
              colName = col;
            } else {
              // Might be referencing another table in a join - skip validation
              continue;
            }
          }
        }

        // Remove any alias (e.g., "col AS alias")
        const asMatch = colName.match(/^(\w+)\s+AS\s+/i);
        if (asMatch && asMatch[1]) {
          colName = asMatch[1];
        }

        // Skip if column is * or if it looks like an expression
        if (colName === '*' || !colName || /\s/.test(colName.trim())) {
          continue;
        }

        const col = table.columns.find(c => c.name.toLowerCase() === colName.toLowerCase());
        if (!col) {
          // Find position of this column in the SELECT clause
          const colIdx = content.indexOf(colPart, match.index);
          if (colIdx >= 0) {
            const startPos = model.getPositionAt(colIdx);
            const endPos = model.getPositionAt(colIdx + colPart.length);
            markers.push({
              severity: monacoInstance.MarkerSeverity.Error,
              message: `Column '${colName}' not found in table '${tableName}'`,
              startLineNumber: startPos.lineNumber,
              startColumn: startPos.column,
              endLineNumber: endPos.lineNumber,
              endColumn: endPos.column,
            });
          }
        }
      }
    }
  }
}

// Split SELECT columns, handling functions and aliases
function splitSelectColumns(columnsStr: string): string[] {
  const columns: string[] = [];
  let current = '';
  let parenDepth = 0;
  let inString = false;
  let stringChar = '';

  for (let i = 0; i < columnsStr.length; i++) {
    const char = columnsStr[i];
    const prevChar = i > 0 ? columnsStr[i - 1] : '';

    if (!inString && (char === "'" || char === '"')) {
      inString = true;
      stringChar = char;
      current += char;
    } else if (inString && char === stringChar && prevChar !== '\\') {
      inString = false;
      current += char;
    } else if (!inString && char === '(') {
      parenDepth++;
      current += char;
    } else if (!inString && char === ')') {
      parenDepth--;
      current += char;
    } else if (!inString && parenDepth === 0 && char === ',') {
      if (current.trim()) columns.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }

  if (current.trim()) columns.push(current.trim());
  return columns;
}

// Validate UPDATE statements
function validateUpdateStatements(content: string, model: any, monacoInstance: any, markers: any[]) {
  // Match UPDATE table_name SET col = val patterns
  const updateRegex = /UPDATE\s+["'`]?(\w+)["'`]?\s+SET\s+([\s\S]+?)(?:\s+WHERE|\s*;|\s*$)/gi;
  let match;

  while ((match = updateRegex.exec(content)) !== null) {
    const tableName = match[1];
    const setClause = match[2];

    if (!tableName || !setClause) continue;

    const table = findTable(tableName);

    if (!table) {
      addTableNotFoundMarker(monacoInstance, model, markers, tableName, match.index!, match[0]);
      continue;
    }

    // Parse SET clause assignments
    const assignments = splitSetClause(setClause);
    const setClauseStr = setClause; // Store for use in nested scope

    for (const assignment of assignments) {
      const eqIdx = assignment.indexOf('=');
      if (eqIdx < 0) continue;

      const colName = assignment.slice(0, eqIdx).trim().replace(/["'`]/g, '');
      const value = assignment.slice(eqIdx + 1).trim();

      const col = table.columns.find(c => c.name.toLowerCase() === colName.toLowerCase());
      const colIdx = setClauseStr.indexOf(colName);

      if (!col) {
        // Find position of column in SET clause
        const setStart = match.index! + match[0].indexOf(setClauseStr);
        if (colIdx >= 0) {
          const startPos = model.getPositionAt(setStart + colIdx);
          const endPos = model.getPositionAt(setStart + colIdx + colName.length);
          markers.push({
            severity: monacoInstance.MarkerSeverity.Error,
            message: `Column '${colName}' not found in table '${tableName}'`,
            startLineNumber: startPos.lineNumber,
            startColumn: startPos.column,
            endLineNumber: endPos.lineNumber,
            endColumn: endPos.column,
          });
        }
        continue;
      }

      // Type validation for the value
      if (value && value.toUpperCase() !== 'NULL' && value.toUpperCase() !== 'DEFAULT') {
        const typeError = checkTypeCompatibility(col.dataType, value);
        if (typeError) {
          const setStart = match.index! + match[0].indexOf(setClauseStr);
          const valueIdx = setClauseStr.indexOf(value, colIdx);
          if (valueIdx >= 0) {
            const startPos = model.getPositionAt(setStart + valueIdx);
            const endPos = model.getPositionAt(setStart + valueIdx + value.length);
            markers.push({
              severity: monacoInstance.MarkerSeverity.Warning,
              message: `${typeError} for column '${colName}' (${col.dataType})`,
              startLineNumber: startPos.lineNumber,
              startColumn: startPos.column,
              endLineNumber: endPos.lineNumber,
              endColumn: endPos.column,
            });
          }
        }
      }
    }
  }
}

// Split SET clause into individual assignments
function splitSetClause(setClause: string): string[] {
  const assignments: string[] = [];
  let current = '';
  let parenDepth = 0;
  let inString = false;
  let stringChar = '';

  for (let i = 0; i < setClause.length; i++) {
    const char = setClause[i];
    const prevChar = i > 0 ? setClause[i - 1] : '';

    if (!inString && (char === "'" || char === '"')) {
      inString = true;
      stringChar = char;
      current += char;
    } else if (inString && char === stringChar && prevChar !== '\\') {
      inString = false;
      current += char;
    } else if (!inString && char === '(') {
      parenDepth++;
      current += char;
    } else if (!inString && char === ')') {
      parenDepth--;
      current += char;
    } else if (!inString && parenDepth === 0 && char === ',') {
      if (current.trim()) assignments.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }

  if (current.trim()) assignments.push(current.trim());
  return assignments;
}

// Validate DELETE statements
function validateDeleteStatements(content: string, model: any, monacoInstance: any, markers: any[]) {
  // Match DELETE FROM table_name patterns
  const deleteRegex = /DELETE\s+FROM\s+["'`]?(\w+)["'`]?/gi;
  let match;

  while ((match = deleteRegex.exec(content)) !== null) {
    const tableName = match[1];

    if (!tableName) continue;

    const table = findTable(tableName);

    if (!table) {
      addTableNotFoundMarker(monacoInstance, model, markers, tableName, match.index!, match[0]);
    }
  }
}

// Helper to add table not found marker
function addTableNotFoundMarker(monacoInstance: any, model: any, markers: any[], tableName: string, matchIndex: number, matchStr: string) {
  const tableIdx = matchStr.indexOf(tableName);
  const startPos = model.getPositionAt(matchIndex + tableIdx);
  const endPos = model.getPositionAt(matchIndex + tableIdx + tableName.length);
  markers.push({
    severity: monacoInstance.MarkerSeverity.Error,
    message: `Table '${tableName}' not found`,
    startLineNumber: startPos.lineNumber,
    startColumn: startPos.column,
    endLineNumber: endPos.lineNumber,
    endColumn: endPos.column,
  });
}

// Parse VALUES clause, handling strings and nested parens
function parseValues(valuesStr: string): string[] {
  const values: string[] = [];
  let current = "";
  let inString = false;
  let stringChar = "";
  let parenDepth = 0;

  for (let i = 0; i < valuesStr.length; i++) {
    const char = valuesStr[i];
    const prevChar = i > 0 ? valuesStr[i - 1] : "";

    if (!inString && (char === "'" || char === '"')) {
      inString = true;
      stringChar = char;
      current += char;
    } else if (inString && char === stringChar && prevChar !== '\\') {
      inString = false;
      current += char;
    } else if (!inString && char === '(') {
      parenDepth++;
      current += char;
    } else if (!inString && char === ')') {
      parenDepth--;
      current += char;
    } else if (!inString && parenDepth === 0 && char === ',') {
      values.push(current.trim());
      current = "";
    } else {
      current += char;
    }
  }

  if (current.trim()) {
    values.push(current.trim());
  }

  return values;
}

// Check if value is compatible with column type
function checkTypeCompatibility(dataType: string, value: string): string | null {
  const dt = dataType.toLowerCase();
  const v = value.trim();

  // Skip function calls and expressions
  if (v.includes('(') || v.toUpperCase() === 'DEFAULT' || v.toUpperCase() === 'NULL') {
    return null;
  }

  const isStringLiteral = (v.startsWith("'") && v.endsWith("'")) || (v.startsWith('"') && v.endsWith('"'));
  const isNumeric = /^-?\d+(\.\d+)?$/.test(v);
  const isBool = /^(true|false)$/i.test(v);

  // Integer types
  if (dt.includes("int") || dt.includes("serial")) {
    if (isStringLiteral) return "Expected integer, got string";
    if (!isNumeric && !isBool) return "Expected integer value";
    if (v.includes('.')) return "Expected integer, got decimal";
  }

  // Decimal/numeric types
  if (dt.includes("decimal") || dt.includes("numeric") || dt.includes("real") || dt.includes("double") || dt.includes("float")) {
    if (isStringLiteral) return "Expected number, got string";
    if (!isNumeric) return "Expected numeric value";
  }

  // Boolean
  if (dt.includes("bool")) {
    if (!isBool && v !== '0' && v !== '1') return "Expected boolean (true/false)";
  }

  // String types
  if (dt.includes("varchar") || dt.includes("text") || dt.includes("char")) {
    if (!isStringLiteral && !v.toUpperCase().startsWith('CONCAT')) {
      return "Expected string literal (quoted)";
    }
  }

  // UUID
  if (dt.includes("uuid")) {
    if (isStringLiteral) {
      const uuidContent = v.slice(1, -1);
      if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(uuidContent)) {
        return "Invalid UUID format";
      }
    }
  }

  // Date/time
  if (dt.includes("date") && !dt.includes("update")) {
    if (isStringLiteral) {
      const dateContent = v.slice(1, -1);
      if (!/^\d{4}-\d{2}-\d{2}/.test(dateContent)) {
        return "Expected date format (YYYY-MM-DD)";
      }
    }
  }

  // JSON
  if (dt.includes("json")) {
    if (isStringLiteral) {
      const jsonContent = v.slice(1, -1);
      try {
        // Basic check - should start with { or [
        if (!jsonContent.trim().startsWith('{') && !jsonContent.trim().startsWith('[')) {
          return "Expected JSON object or array";
        }
      } catch {
        return "Invalid JSON";
      }
    }
  }

  return null;
}

// Setup validation on content change
function setupValidation(monacoInstance: any, editorInstance: any) {
  if (validationDisposable) {
    validationDisposable.dispose();
  }

  const model = editorInstance.getModel();
  if (!model) return;

  // Initial validation
  validateSQL(monacoInstance, model);

  // Validate on change with debounce
  let timeout: any = null;
  validationDisposable = model.onDidChangeContent(() => {
    if (timeout) clearTimeout(timeout);
    timeout = setTimeout(() => {
      validateSQL(monacoInstance, model);
    }, 500);
  });
}

// Close dropdowns on click outside
function onClickOutside(_e: MouseEvent) {
  showHistory.value = false;
  showSnippets.value = false;
}

onMounted(() => {
  loadHistory();
  fetchSchema();
  registerAutocomplete();
  document.addEventListener("click", onClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", onClickOutside);
  if (completionProviderDisposable) {
    completionProviderDisposable.dispose();
  }
  if (validationDisposable) {
    validationDisposable.dispose();
  }
});

watch(tables, () => {
  if (editorRef.value?.monaco?.() && editorRef.value?.editor?.()) {
    registerSQLCompletionProvider(editorRef.value.monaco());
    // Re-validate with updated schema
    const model = editorRef.value.editor().getModel();
    if (model) {
      validateSQL(editorRef.value.monaco(), model);
    }
  }
});
</script>

<style scoped>
.query-workbench {
  display: grid;
  grid-template-columns: minmax(13rem, 17rem) minmax(0, 1fr);
  min-height: 24rem;
  background: var(--oui-surface-base);
}

.query-schema-rail {
  min-width: 0;
  border-right: 1px solid var(--oui-border-default);
  background: color-mix(in srgb, var(--oui-surface-base) 86%, var(--oui-surface-overlay));
  padding: 0.75rem;
}

.query-schema-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.625rem;
}

.query-schema-list {
  display: flex;
  max-height: 25rem;
  flex-direction: column;
  gap: 0.375rem;
  overflow: auto;
  padding-right: 0.125rem;
}

.query-schema-table {
  border: 1px solid var(--oui-border-muted);
  border-radius: 0.5rem;
  background: var(--oui-surface-base);
  overflow: hidden;
}

.query-schema-table-trigger,
.query-schema-column {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 0.375rem;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.query-schema-table-trigger {
  padding: 0.5rem 0.625rem;
  font-size: 0.75rem;
}

.query-schema-table-trigger:hover,
.query-schema-column:hover {
  background: var(--oui-surface-hover);
}

.query-schema-columns {
  border-top: 1px solid var(--oui-border-muted);
  padding: 0.25rem;
}

.query-schema-column {
  justify-content: space-between;
  border-radius: 0.375rem;
  padding: 0.25rem 0.375rem 0.25rem 1.625rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  color: var(--oui-text-secondary);
}

.query-column-type {
  max-width: 5.5rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--oui-text-tertiary);
}

.query-editor-pane {
  min-width: 0;
}

.query-editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-bottom: 1px solid var(--oui-border-default);
  background: var(--oui-surface-base);
  padding: 0.625rem 1rem;
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 900px) {
  .query-workbench {
    grid-template-columns: 1fr;
  }

  .query-schema-rail {
    max-height: 16rem;
    border-right: 0;
    border-bottom: 1px solid var(--oui-border-default);
  }

  .query-editor-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
