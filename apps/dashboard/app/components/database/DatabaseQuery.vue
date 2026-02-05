<template>
  <OuiStack gap="md">
    <!-- Query Tabs -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center" class="w-full">
          <OuiFlex align="center" gap="sm">
            <OuiText as="h3" size="lg" weight="semibold">Query Editor</OuiText>
            <!-- Tab bar -->
            <OuiFlex gap="xs" class="ml-4">
              <button
                v-for="(tab, idx) in queryTabs"
                :key="idx"
                class="px-3 py-1 text-xs rounded-md border transition-colors"
                :class="
                  activeTabIdx === idx
                    ? 'bg-primary/10 border-primary/30 text-primary'
                    : 'bg-transparent border-border-default text-secondary hover:text-primary hover:border-primary/20'
                "
                @click="activeTabIdx = idx"
              >
                {{ tab.title }}
                <span
                  v-if="queryTabs.length > 1"
                  class="ml-1.5 text-secondary hover:text-danger cursor-pointer"
                  @click.stop="closeTab(idx)"
                >&times;</span>
              </button>
              <button
                class="px-2 py-1 text-xs text-secondary hover:text-primary rounded-md border border-dashed border-border-default hover:border-primary/20 transition-colors"
                @click="addTab"
              >
                +
              </button>
            </OuiFlex>
          </OuiFlex>
          <OuiFlex gap="sm" align="center">
            <!-- History dropdown -->
            <div class="relative" v-if="queryHistory.length > 0">
              <OuiButton
                variant="ghost"
                color="secondary"
                size="sm"
                @click="showHistory = !showHistory"
              >
                <ClockIcon class="h-4 w-4" />
                History
              </OuiButton>
              <div
                v-if="showHistory"
                class="absolute right-0 top-full mt-1 w-96 max-h-64 overflow-y-auto bg-surface-overlay border border-border-default rounded-lg shadow-lg z-50"
              >
                <button
                  v-for="(item, idx) in queryHistory"
                  :key="idx"
                  class="w-full text-left px-3 py-2 text-xs font-mono hover:bg-interactive-hover border-b border-border-default last:border-0 truncate"
                  @click="loadFromHistory(item)"
                >
                  {{ item }}
                </button>
              </div>
            </div>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <!-- Monaco Editor -->
        <div class="relative" :style="{ height: editorHeight + 'px' }">
          <OuiFileEditor
            ref="editorRef"
            v-model="activeTab.content"
            language="sql"
            :height="editorHeight + 'px'"
            :minimap="{ enabled: false }"
            :folding="false"
            container-class="w-full border-0 rounded-none"
          />
          <!-- Resize handle -->
          <div
            class="absolute bottom-0 left-0 right-0 h-1.5 cursor-row-resize bg-transparent hover:bg-primary/20 transition-colors"
            @mousedown="startResize"
          />
        </div>

        <!-- Toolbar -->
        <OuiFlex
          justify="between"
          align="center"
          class="px-4 py-2 border-t border-border-default bg-surface-base"
        >
          <OuiFlex gap="sm" align="center">
            <OuiText color="secondary" size="xs">
              Max rows:
            </OuiText>
            <OuiInput
              v-model="maxRows"
              type="number"
              size="sm"
              class="w-20"
              :min="1"
              :max="10000"
            />
            <OuiText color="secondary" size="xs" class="hidden md:block">
              Ctrl+Enter to execute &middot; Ctrl+Shift+Enter for selection
            </OuiText>
          </OuiFlex>
          <OuiFlex gap="sm">
            <OuiButton
              variant="ghost"
              color="secondary"
              size="sm"
              :disabled="!activeTab.content.trim()"
              @click="exportResults"
              v-if="activeTab.results"
            >
              <ArrowDownTrayIcon class="h-4 w-4" />
              Export CSV
            </OuiButton>
            <OuiButton
              color="primary"
              size="sm"
              :loading="executing"
              :disabled="!activeTab.content.trim()"
              @click="executeCurrentQuery"
            >
              <PlayIcon class="h-4 w-4" />
              Execute
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Results -->
    <OuiCard v-if="activeTab.results">
      <OuiCardHeader>
        <OuiFlex justify="between" align="center" class="w-full">
          <OuiFlex align="center" gap="sm">
            <OuiText as="h3" size="sm" weight="semibold">Results</OuiText>
            <OuiBadge v-if="activeTab.results.truncated" color="warning" size="xs">
              Truncated
            </OuiBadge>
          </OuiFlex>
          <OuiText color="secondary" size="xs">
            {{ activeTab.results.rowCount }} row(s) in {{ activeTab.results.executionTimeMs }}ms
          </OuiText>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border-default bg-surface-base">
                <th
                  v-for="col in activeTab.results.columns"
                  :key="col.name"
                  class="px-3 py-2 text-left font-medium text-xs cursor-pointer hover:bg-interactive-hover select-none"
                  @click="toggleSort(col.name)"
                >
                  <OuiFlex align="center" gap="xs">
                    <span>{{ col.name }}</span>
                    <span class="text-secondary font-normal">{{ col.dataType }}</span>
                    <span v-if="sortColumn === col.name" class="text-primary">
                      {{ sortDirection === 'ASC' ? '↑' : '↓' }}
                    </span>
                  </OuiFlex>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, rowIdx) in sortedRows"
                :key="rowIdx"
                class="border-b border-border-default/50 hover:bg-interactive-hover/50"
              >
                <td
                  v-for="col in activeTab.results.columns"
                  :key="col.name"
                  class="px-3 py-1.5 font-mono text-xs whitespace-nowrap max-w-xs truncate cursor-pointer"
                  :title="getCellValue(row, col.name)"
                  @click="copyCell(row, col.name)"
                >
                  <span
                    v-if="row[col.name] === null"
                    class="text-secondary italic"
                  >NULL</span>
                  <span v-else>{{ row[col.name] }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <OuiAlert v-if="activeTab.results.truncated" color="warning" class="m-4">
          Results were truncated. Only the first {{ maxRows }} rows are shown.
        </OuiAlert>
      </OuiCardBody>
    </OuiCard>

    <!-- Error -->
    <OuiAlert v-if="activeTab.error" color="danger">
      <OuiText weight="semibold">Query Error</OuiText>
      <OuiText size="sm">{{ activeTab.error }}</OuiText>
    </OuiAlert>
  </OuiStack>
</template>

<script setup lang="ts">
import {
  PlayIcon,
  ClockIcon,
  ArrowDownTrayIcon,
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

const maxRows = ref<string>("1000");
const executing = ref(false);
const showHistory = ref(false);

// Query history (per database)
const historyKey = computed(() => `db-query-history-${props.databaseId}`);
const queryHistory = ref<string[]>([]);

// Sort state for results
const sortColumn = ref<string | null>(null);
const sortDirection = ref<"ASC" | "DESC">("ASC");

// Editor resize
const editorHeight = ref(250);
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
  editorHeight.value = Math.max(100, Math.min(600, startHeight + delta));
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
    const firstLine = content.split("\n")[0]!.trim().slice(0, 30);
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
    const rows = (res.rows || []).map((row: any) => {
      const obj: Record<string, any> = {};
      for (const cell of row.cells || []) {
        obj[cell.columnName] = cell.isNull ? null : cell.value;
      }
      return obj;
    });

    tab.results = {
      columns: res.columns || [],
      rows,
      rowCount: res.rowCount,
      truncated: res.truncated,
      executionTimeMs: res.executionTimeMs,
    };

    // Add to history
    addToHistory(queryText);
    sortColumn.value = null;
    toast.success("Query executed successfully");
  } catch (err: any) {
    tab.error = err.message || "Query execution failed";
    toast.error("Query execution failed", err.message);
  } finally {
    executing.value = false;
  }
}

// History management
function addToHistory(query: string) {
  const history = queryHistory.value;
  // Remove duplicates
  const idx = history.indexOf(query);
  if (idx > -1) history.splice(idx, 1);
  // Add to front
  history.unshift(query);
  // Keep max 50
  if (history.length > 50) history.pop();
  // Persist
  try {
    localStorage.setItem(historyKey.value, JSON.stringify(history));
  } catch {
    // ignore storage errors
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

async function copyCell(row: any, colName: string) {
  const val = getCellValue(row, colName);
  try {
    await navigator.clipboard.writeText(val);
    toast.success("Copied to clipboard");
  } catch {
    // ignore
  }
}

// Export CSV
function exportResults() {
  const results = activeTab.value?.results;
  if (!results) return;

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

  const blob = new Blob([csvRows.join("\n")], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `query-results-${Date.now()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

// Register autocomplete and keybindings after Monaco is available
async function registerAutocomplete() {
  await nextTick();
  // Wait for editor to be ready
  const checkEditor = () => {
    if (!editorRef.value) return;
    const monacoInstance = editorRef.value.monaco?.();
    const editorInstance = editorRef.value.editor?.();
    if (!monacoInstance || !editorInstance) {
      setTimeout(checkEditor, 200);
      return;
    }

    // Register Ctrl+Enter to execute
    editorInstance.addCommand(
      monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyCode.Enter,
      () => executeCurrentQuery()
    );

    // Register Ctrl+Shift+Enter to execute selection
    editorInstance.addCommand(
      monacoInstance.KeyMod.CtrlCmd | monacoInstance.KeyMod.Shift | monacoInstance.KeyCode.Enter,
      () => executeCurrentQuery(true)
    );

    // Register SQL autocomplete provider
    registerSQLCompletionProvider(monacoInstance);
  };
  setTimeout(checkEditor, 300);
}

let completionProviderDisposable: any = null;

function registerSQLCompletionProvider(monacoInstance: any) {
  // Dispose previous registration to avoid duplicates
  if (completionProviderDisposable) {
    completionProviderDisposable.dispose();
  }

  const sqlKeywords = [
    "SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "BETWEEN", "LIKE", "ILIKE",
    "IS", "NULL", "AS", "ON", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER", "FULL", "CROSS",
    "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING", "LIMIT", "OFFSET",
    "INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
    "CREATE", "TABLE", "INDEX", "VIEW", "DROP", "ALTER", "ADD", "COLUMN",
    "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK", "DEFAULT",
    "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX",
    "CASE", "WHEN", "THEN", "ELSE", "END",
    "UNION", "ALL", "INTERSECT", "EXCEPT",
    "EXISTS", "ANY", "SOME",
    "COALESCE", "NULLIF", "CAST",
    "BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION",
    "EXPLAIN", "ANALYZE", "VACUUM", "TRUNCATE",
    "WITH", "RECURSIVE", "RETURNING",
  ];

  const pgFunctions = [
    "now()", "current_timestamp", "current_date", "current_time",
    "coalesce()", "nullif()", "greatest()", "least()",
    "array_agg()", "string_agg()", "json_agg()", "jsonb_agg()",
    "json_build_object()", "jsonb_build_object()",
    "to_char()", "to_date()", "to_timestamp()", "to_number()",
    "length()", "lower()", "upper()", "trim()", "substring()",
    "replace()", "concat()", "split_part()", "regexp_replace()",
    "extract()", "date_trunc()", "age()", "interval",
    "generate_series()", "row_number()", "rank()", "dense_rank()",
    "lag()", "lead()", "first_value()", "last_value()",
  ];

  const mysqlFunctions = [
    "NOW()", "CURDATE()", "CURTIME()", "CURRENT_TIMESTAMP()",
    "IFNULL()", "COALESCE()", "NULLIF()", "IF()",
    "GROUP_CONCAT()", "JSON_OBJECT()", "JSON_ARRAY()",
    "DATE_FORMAT()", "STR_TO_DATE()", "DATEDIFF()",
    "CONCAT()", "CONCAT_WS()", "LENGTH()", "CHAR_LENGTH()",
    "LOWER()", "UPPER()", "TRIM()", "SUBSTRING()", "REPLACE()",
    "CAST()", "CONVERT()", "UUID()",
  ];

  const isPostgres = props.databaseType === "1" || props.databaseType === "POSTGRESQL";
  const builtinFunctions = isPostgres ? pgFunctions : mysqlFunctions;

  completionProviderDisposable = monacoInstance.languages.registerCompletionItemProvider("sql", {
    triggerCharacters: [".", " ", "("],
    provideCompletionItems(model: any, position: any) {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      // Check if we're after a dot (table.column)
      const textBeforeCursor = model.getValueInRange({
        startLineNumber: position.lineNumber,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      });

      const dotMatch = textBeforeCursor.match(/(\w+)\.\s*$/);
      if (dotMatch) {
        const tableName = dotMatch[1];
        const table = tables.value.find(
          (t) => t.name.toLowerCase() === tableName.toLowerCase()
        );
        if (table) {
          return {
            suggestions: table.columns.map((col) => ({
              label: col.name,
              kind: monacoInstance.languages.CompletionItemKind.Field,
              detail: `${col.dataType}${col.isPrimaryKey ? " (PK)" : ""}${col.isNullable ? "" : " NOT NULL"}`,
              insertText: col.name,
              range,
            })),
          };
        }
      }

      const suggestions: any[] = [];

      // SQL keywords
      for (const kw of sqlKeywords) {
        suggestions.push({
          label: kw,
          kind: monacoInstance.languages.CompletionItemKind.Keyword,
          insertText: kw,
          range,
        });
      }

      // Table names
      for (const t of tables.value) {
        suggestions.push({
          label: t.name,
          kind: monacoInstance.languages.CompletionItemKind.Struct,
          detail: `Table (${Number(t.rowCount)} rows)`,
          documentation: t.columns.map((c) => `${c.name}: ${c.dataType}`).join("\n"),
          insertText: t.name,
          range,
        });
      }

      // Views
      for (const v of views.value) {
        suggestions.push({
          label: v.name,
          kind: monacoInstance.languages.CompletionItemKind.Interface,
          detail: "View",
          insertText: v.name,
          range,
        });
      }

      // Schema functions
      for (const f of functions.value) {
        suggestions.push({
          label: f.name,
          kind: monacoInstance.languages.CompletionItemKind.Function,
          detail: `→ ${f.returnType}`,
          insertText: f.name + "()",
          range,
        });
      }

      // Built-in functions
      for (const fn of builtinFunctions) {
        suggestions.push({
          label: fn,
          kind: monacoInstance.languages.CompletionItemKind.Function,
          detail: "Built-in",
          insertText: fn,
          range,
        });
      }

      return { suggestions };
    },
  });
}

// Close history on click outside
function onClickOutside(_e: MouseEvent) {
  if (showHistory.value) {
    showHistory.value = false;
  }
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
});

// Re-register autocomplete when schema loads
watch(tables, () => {
  if (editorRef.value?.monaco?.()) {
    registerSQLCompletionProvider(editorRef.value.monaco());
  }
});
</script>
