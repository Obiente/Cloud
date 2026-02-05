<template>
  <OuiDialog v-model:open="open" title="Create Table" size="xl" @close="close">
    <OuiStack gap="lg">
      <!-- Table name -->
      <OuiFormField label="Table Name" required>
        <OuiInput
          v-model="tableName"
          placeholder="users"
          :error="errors.tableName"
        />
      </OuiFormField>

      <!-- Columns -->
      <div>
        <OuiFlex justify="between" align="center" class="mb-2">
          <OuiText weight="semibold">Columns</OuiText>
          <OuiButton variant="ghost" size="sm" @click="addColumn">
            <PlusIcon class="h-4 w-4" />
            Add Column
          </OuiButton>
        </OuiFlex>

        <div class="border border-border-default rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border-default bg-surface-base">
                <th class="text-left py-2 px-3 font-medium w-8"></th>
                <th class="text-left py-2 px-3 font-medium">Name</th>
                <th class="text-left py-2 px-3 font-medium">Type</th>
                <th class="text-left py-2 px-3 font-medium w-20">Nullable</th>
                <th class="text-left py-2 px-3 font-medium w-16">PK</th>
                <th class="text-left py-2 px-3 font-medium">Default</th>
                <th class="w-10"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(col, idx) in columns"
                :key="idx"
                class="border-b border-border-default/50 last:border-0"
              >
                <td class="py-1 px-3 text-secondary text-xs">{{ idx + 1 }}</td>
                <td class="py-1 px-2">
                  <input
                    v-model="col.name"
                    class="w-full bg-transparent border border-border-default rounded px-2 py-1 text-sm focus:outline-none focus:border-primary"
                    placeholder="column_name"
                  />
                </td>
                <td class="py-1 px-2">
                  <select
                    v-model="col.dataType"
                    class="w-full bg-surface-base border border-border-default rounded px-2 py-1 text-sm focus:outline-none focus:border-primary"
                  >
                    <optgroup label="Numeric">
                      <option value="integer">integer</option>
                      <option value="bigint">bigint</option>
                      <option value="smallint">smallint</option>
                      <option value="serial">serial (auto-increment)</option>
                      <option value="bigserial">bigserial (auto-increment)</option>
                      <option value="decimal">decimal</option>
                      <option value="numeric">numeric</option>
                      <option value="real">real</option>
                      <option value="double precision">double precision</option>
                    </optgroup>
                    <optgroup label="String">
                      <option value="varchar(255)">varchar(255)</option>
                      <option value="varchar(50)">varchar(50)</option>
                      <option value="varchar(100)">varchar(100)</option>
                      <option value="text">text</option>
                      <option value="char(1)">char(1)</option>
                      <option value="uuid">uuid</option>
                    </optgroup>
                    <optgroup label="Date/Time">
                      <option value="timestamp">timestamp</option>
                      <option value="timestamptz">timestamptz</option>
                      <option value="date">date</option>
                      <option value="time">time</option>
                      <option value="interval">interval</option>
                    </optgroup>
                    <optgroup label="Other">
                      <option value="boolean">boolean</option>
                      <option value="jsonb">jsonb</option>
                      <option value="json">json</option>
                      <option value="bytea">bytea</option>
                      <option value="inet">inet</option>
                      <option value="cidr">cidr</option>
                    </optgroup>
                  </select>
                </td>
                <td class="py-1 px-3 text-center">
                  <input
                    type="checkbox"
                    v-model="col.isNullable"
                    class="rounded"
                    :disabled="col.isPrimaryKey"
                  />
                </td>
                <td class="py-1 px-3 text-center">
                  <input
                    type="checkbox"
                    v-model="col.isPrimaryKey"
                    class="rounded"
                    @change="onPrimaryKeyChange(col)"
                  />
                </td>
                <td class="py-1 px-2">
                  <input
                    v-model="col.defaultValue"
                    class="w-full bg-transparent border border-border-default rounded px-2 py-1 text-sm focus:outline-none focus:border-primary"
                    placeholder="NULL"
                  />
                </td>
                <td class="py-1 px-2">
                  <button
                    @click="removeColumn(idx)"
                    class="text-secondary hover:text-danger p-1"
                    :disabled="columns.length <= 1"
                  >
                    <TrashIcon class="h-4 w-4" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <OuiText v-if="errors.columns" color="danger" size="xs" class="mt-1">
          {{ errors.columns }}
        </OuiText>
      </div>

      <!-- Preview DDL -->
      <div>
        <OuiFlex justify="between" align="center" class="mb-2">
          <OuiText weight="semibold">Preview DDL</OuiText>
          <OuiButton variant="ghost" size="sm" @click="copyDDL">
            <ClipboardDocumentIcon class="h-4 w-4" />
            Copy
          </OuiButton>
        </OuiFlex>
        <pre
          class="text-xs font-mono bg-surface-base border border-border-default rounded-lg p-4 overflow-x-auto max-h-40 whitespace-pre-wrap"
        >{{ generatedDDL }}</pre>
      </div>
    </OuiStack>

    <template #footer>
      <OuiButton variant="ghost" @click="close">Cancel</OuiButton>
      <OuiButton color="primary" :loading="creating" @click="createTable">
        Create Table
      </OuiButton>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { PlusIcon, TrashIcon, ClipboardDocumentIcon } from "@heroicons/vue/24/outline";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";

interface ColumnDef {
  name: string;
  dataType: string;
  isNullable: boolean;
  isPrimaryKey: boolean;
  isUnique: boolean;
  defaultValue: string;
  autoIncrement: boolean;
}

const props = defineProps<{
  databaseId: string;
  databaseType: string;
}>();

const emit = defineEmits<{
  (e: "created"): void;
}>();

const open = defineModel<boolean>("open", { default: false });

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const tableName = ref("");
const columns = ref<ColumnDef[]>([
  { name: "id", dataType: "serial", isNullable: false, isPrimaryKey: true, isUnique: false, defaultValue: "", autoIncrement: true },
  { name: "created_at", dataType: "timestamptz", isNullable: false, isPrimaryKey: false, isUnique: false, defaultValue: "now()", autoIncrement: false },
]);
const creating = ref(false);
const errors = ref<{ tableName?: string; columns?: string }>({});

const isPostgres = computed(() =>
  props.databaseType === "1" || props.databaseType === "POSTGRESQL"
);

// Generate DDL preview
const generatedDDL = computed(() => {
  if (!tableName.value) return "-- Enter a table name";

  const lines: string[] = [];
  const quote = isPostgres.value ? '"' : '`';
  lines.push(`CREATE TABLE ${quote}${tableName.value}${quote} (`);

  const colDefs: string[] = [];
  const pkColumns: string[] = [];

  for (const col of columns.value) {
    if (!col.name) continue;

    let def = `  ${quote}${col.name}${quote} ${col.dataType}`;

    if (!col.isNullable) def += " NOT NULL";
    if (col.defaultValue) def += ` DEFAULT ${col.defaultValue}`;

    colDefs.push(def);

    if (col.isPrimaryKey) {
      pkColumns.push(col.name);
    }
  }

  lines.push(colDefs.join(",\n"));

  if (pkColumns.length > 0) {
    lines.push(`  ,PRIMARY KEY (${pkColumns.map((c) => `${quote}${c}${quote}`).join(", ")})`);
  }

  lines.push(");");

  return lines.join("\n");
});

function addColumn() {
  columns.value.push({
    name: "",
    dataType: "varchar(255)",
    isNullable: true,
    isPrimaryKey: false,
    isUnique: false,
    defaultValue: "",
    autoIncrement: false,
  });
}

function removeColumn(idx: number) {
  if (columns.value.length <= 1) return;
  columns.value.splice(idx, 1);
}

function onPrimaryKeyChange(col: ColumnDef) {
  if (col.isPrimaryKey) {
    col.isNullable = false;
  }
}

async function copyDDL() {
  try {
    await navigator.clipboard.writeText(generatedDDL.value);
    toast.success("DDL copied to clipboard");
  } catch {
    toast.error("Failed to copy to clipboard");
  }
}

function validate(): boolean {
  errors.value = {};

  if (!tableName.value.trim()) {
    errors.value.tableName = "Table name is required";
  } else if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(tableName.value)) {
    errors.value.tableName = "Invalid table name (use letters, numbers, underscores)";
  }

  const validColumns = columns.value.filter((c) => c.name.trim());
  if (validColumns.length === 0) {
    errors.value.columns = "At least one column is required";
  }

  const colNames = validColumns.map((c) => c.name.toLowerCase());
  const duplicates = colNames.filter((n, i) => colNames.indexOf(n) !== i);
  if (duplicates.length > 0) {
    errors.value.columns = `Duplicate column name: ${duplicates[0]}`;
  }

  return Object.keys(errors.value).length === 0;
}

async function createTable() {
  if (!validate() || !organizationId.value) return;

  creating.value = true;

  try {
    const pkColumns = columns.value.filter((c) => c.isPrimaryKey && c.name).map((c) => c.name);

    await dbClient.createTable({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      tableName: tableName.value,
      columns: columns.value
        .filter((c) => c.name.trim())
        .map((c) => ({
          name: c.name,
          dataType: c.dataType,
          isNullable: c.isNullable,
          defaultValue: c.defaultValue || undefined,
          isUnique: c.isUnique,
          autoIncrement: c.dataType === "serial" || c.dataType === "bigserial",
        })),
      primaryKey: pkColumns.length > 0 ? { columnNames: pkColumns } : undefined,
    });

    toast.success(`Table "${tableName.value}" created`);
    emit("created");
    close();
  } catch (err: any) {
    toast.error("Failed to create table", err.message);
  } finally {
    creating.value = false;
  }
}

function close() {
  open.value = false;
  // Reset form
  tableName.value = "";
  columns.value = [
    { name: "id", dataType: "serial", isNullable: false, isPrimaryKey: true, isUnique: false, defaultValue: "", autoIncrement: true },
    { name: "created_at", dataType: "timestamptz", isNullable: false, isPrimaryKey: false, isUnique: false, defaultValue: "now()", autoIncrement: false },
  ];
  errors.value = {};
}

// Watch for open changes to reset
watch(open, (isOpen) => {
  if (!isOpen) {
    close();
  }
});
</script>
