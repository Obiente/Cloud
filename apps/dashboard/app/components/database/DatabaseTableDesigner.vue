<template>
  <OuiDialog v-model:open="open" title="Create Table" size="xl" @close="close">
    <OuiStack gap="lg" style="max-height: 70vh; overflow-y: auto">
      <!-- Table name -->
      <OuiFormField label="Table Name" required :error="errors.tableName">
        <OuiInput
          v-model="tableName"
          placeholder="users"
        />
      </OuiFormField>

      <!-- Columns -->
      <div>
        <OuiFlex justify="between" align="center" style="margin-bottom: 0.75rem">
          <OuiText weight="semibold">Columns</OuiText>
          <OuiButton variant="ghost" size="sm" @click="addColumn">
            <PlusIcon style="width: 1rem; height: 1rem" />
            Add Column
          </OuiButton>
        </OuiFlex>

        <OuiText v-if="errors.columns" color="danger" size="xs" style="margin-bottom: 0.5rem">
          {{ errors.columns }}
        </OuiText>

        <!-- Column Cards -->
        <div style="display: flex; flex-direction: column; gap: 0.5rem">
          <div
            v-for="(col, idx) in columns"
            :key="col.__idx"
            style="border: 1px solid var(--oui-border-default); border-radius: 0.5rem; background: var(--oui-surface-base)"
          >
            <!-- Column Header (collapsed view) -->
            <div
              style="display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1rem; cursor: pointer"
              @click="toggleColumn(col.__idx)"
            >
              <ChevronRightIcon
                :style="{
                  width: '1rem',
                  height: '1rem',
                  color: 'var(--oui-text-secondary)',
                  transition: 'transform 0.15s ease',
                  transform: expandedColumns.has(col.__idx) ? 'rotate(90deg)' : 'rotate(0deg)'
                }"
              />
              <OuiText size="sm" weight="medium" style="flex: 1; min-width: 0">
                <span v-if="col.name">{{ col.name }}</span>
                <span v-else style="color: var(--oui-text-secondary); font-style: italic">Unnamed column</span>
              </OuiText>
              <OuiBadge v-if="col.isPrimaryKey" color="primary" size="xs">PK</OuiBadge>
              <OuiBadge v-if="col.foreignKey" color="tertiary" size="xs">FK</OuiBadge>
              <OuiText size="xs" color="tertiary" style="font-family: monospace">{{ col.dataType }}</OuiText>
              <OuiButton
                variant="ghost"
                size="sm"
                color="danger"
                :disabled="columns.length <= 1"
                @click.stop="removeColumn(idx)"
                style="padding: 0.25rem"
              >
                <TrashIcon style="width: 0.875rem; height: 0.875rem" />
              </OuiButton>
            </div>

            <!-- Column Details (expanded view) -->
            <Transition name="expand">
              <div
                v-if="expandedColumns.has(col.__idx)"
                style="padding: 0 1rem 1rem 1rem; border-top: 1px solid var(--oui-border-default)"
              >
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; padding-top: 0.75rem">
                  <!-- Name -->
                  <OuiFormField label="Column Name" size="sm">
                    <OuiInput
                      v-model="col.name"
                      placeholder="column_name"
                      size="sm"
                    />
                  </OuiFormField>

                  <!-- Data Type -->
                  <OuiFormField label="Data Type" size="sm">
                    <OuiSelect
                      v-model="col.dataType"
                      :items="dataTypeOptions"
                      size="sm"
                    />
                  </OuiFormField>

                  <!-- Default Value -->
                  <OuiFormField label="Default Value" size="sm">
                    <OuiInput
                      v-model="col.defaultValue"
                      placeholder="NULL"
                      size="sm"
                    />
                  </OuiFormField>

                  <!-- Options -->
                  <div style="display: flex; flex-direction: column; gap: 0.5rem; padding-top: 1.25rem">
                    <OuiFlex align="center" gap="sm">
                      <OuiCheckbox
                        v-model="col.isPrimaryKey"
                        @update:model-value="onPrimaryKeyChange(col)"
                      />
                      <OuiText size="sm">Primary Key</OuiText>
                    </OuiFlex>
                    <OuiFlex align="center" gap="sm">
                      <OuiCheckbox
                        v-model="col.isNullable"
                        :disabled="col.isPrimaryKey"
                      />
                      <OuiText size="sm">Nullable</OuiText>
                    </OuiFlex>
                    <OuiFlex align="center" gap="sm">
                      <OuiCheckbox v-model="col.isUnique" />
                      <OuiText size="sm">Unique</OuiText>
                    </OuiFlex>
                  </div>
                </div>

                <!-- Foreign Key Section -->
                <div style="margin-top: 0.75rem; padding-top: 0.75rem; border-top: 1px dashed var(--oui-border-default)">
                  <OuiFlex align="center" gap="sm" style="margin-bottom: 0.5rem">
                    <OuiCheckbox
                      :model-value="!!col.foreignKey"
                      @update:model-value="toggleForeignKey(col, $event)"
                    />
                    <OuiText size="sm" weight="medium">Foreign Key Reference</OuiText>
                  </OuiFlex>

                  <div v-if="col.foreignKey" style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; padding-left: 1.5rem">
                    <OuiFormField label="Reference Table" size="sm">
                      <OuiSelect
                        v-model="col.foreignKey.toTable"
                        :items="availableTablesOptions"
                        size="sm"
                        placeholder="Select table..."
                        @update:model-value="onFkTableChange(col)"
                      />
                    </OuiFormField>

                    <OuiFormField label="Reference Column" size="sm">
                      <OuiSelect
                        v-model="col.foreignKey.toColumn"
                        :items="getFkColumnOptions(col.foreignKey.toTable)"
                        size="sm"
                        placeholder="Select column..."
                        :disabled="!col.foreignKey.toTable"
                      />
                    </OuiFormField>

                    <OuiFormField label="On Delete" size="sm">
                      <OuiSelect
                        v-model="col.foreignKey.onDelete"
                        :items="fkActionOptions"
                        size="sm"
                      />
                    </OuiFormField>

                    <OuiFormField label="On Update" size="sm">
                      <OuiSelect
                        v-model="col.foreignKey.onUpdate"
                        :items="fkActionOptions"
                        size="sm"
                      />
                    </OuiFormField>
                  </div>
                </div>
              </div>
            </Transition>
          </div>
        </div>

        <!-- Empty state -->
        <div
          v-if="columns.length === 0"
          style="border: 1px dashed var(--oui-border-default); border-radius: 0.5rem; padding: 2rem; text-align: center"
        >
          <OuiText color="tertiary" size="sm">Add at least one column</OuiText>
        </div>
      </div>

      <!-- Preview DDL -->
      <div>
        <OuiFlex align="center" gap="sm" style="cursor: pointer; margin-bottom: 0.5rem" @click="showDdlPreview = !showDdlPreview">
          <ChevronRightIcon
            :style="{
              width: '1rem',
              height: '1rem',
              transition: 'transform 0.15s ease',
              transform: showDdlPreview ? 'rotate(90deg)' : 'rotate(0deg)'
            }"
          />
          <OuiText weight="semibold">Preview DDL</OuiText>
          <OuiButton variant="ghost" size="sm" @click.stop="copyDDL" style="margin-left: auto">
            <ClipboardDocumentIcon style="width: 1rem; height: 1rem" />
            Copy
          </OuiButton>
        </OuiFlex>
        <Transition name="expand">
          <pre
            v-if="showDdlPreview"
            style="font-size: 0.75rem; font-family: monospace; background: var(--oui-surface-overlay); border: 1px solid var(--oui-border-default); border-radius: 0.5rem; padding: 1rem; overflow-x: auto; max-height: 200px; white-space: pre-wrap"
          >{{ generatedDDL }}</pre>
        </Transition>
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
import { ref, computed, watch, toRef } from "vue";
import { PlusIcon, TrashIcon, ClipboardDocumentIcon, ChevronRightIcon } from "@heroicons/vue/24/outline";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { useDatabaseSchema } from "~/composables/useDatabaseSchema";

interface ForeignKeyDef {
  toTable: string;
  toColumn: string;
  onDelete: string;
  onUpdate: string;
}

interface ColumnDef {
  __idx: number;
  name: string;
  dataType: string;
  isNullable: boolean;
  isPrimaryKey: boolean;
  isUnique: boolean;
  defaultValue: string;
  autoIncrement: boolean;
  foreignKey: ForeignKeyDef | null;
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

// Get existing tables for FK references
const { tables: existingTables, fetchSchema } = useDatabaseSchema(toRef(props, "databaseId"));

let idxCounter = 2;

const tableName = ref("");
const columns = ref<ColumnDef[]>([
  { __idx: 0, name: "id", dataType: "serial", isNullable: false, isPrimaryKey: true, isUnique: false, defaultValue: "", autoIncrement: true, foreignKey: null },
  { __idx: 1, name: "created_at", dataType: "timestamptz", isNullable: false, isPrimaryKey: false, isUnique: false, defaultValue: "now()", autoIncrement: false, foreignKey: null },
]);
const creating = ref(false);
const errors = ref<{ tableName?: string; columns?: string; foreignKeys?: string }>({});
const expandedColumns = ref(new Set<number>([0, 1]));
const showDdlPreview = ref(true);

const isPostgres = computed(() =>
  props.databaseType === "1" || props.databaseType === "POSTGRESQL"
);

// FK action options
const fkActionOptions = [
  { label: "NO ACTION", value: "NO ACTION" },
  { label: "CASCADE", value: "CASCADE" },
  { label: "SET NULL", value: "SET NULL" },
  { label: "SET DEFAULT", value: "SET DEFAULT" },
  { label: "RESTRICT", value: "RESTRICT" },
];

// Available tables for FK reference
const availableTablesOptions = computed(() => {
  return existingTables.value.map(t => ({
    label: t.name,
    value: t.name,
  }));
});

// Get columns for a specific table (for FK column selection)
function getFkColumnOptions(tableName: string) {
  if (!tableName) return [];
  const table = existingTables.value.find(t => t.name === tableName);
  if (!table) return [];
  // Prefer PK columns first
  return table.columns
    .sort((a, b) => (b.isPrimaryKey ? 1 : 0) - (a.isPrimaryKey ? 1 : 0))
    .map(c => ({
      label: `${c.name} (${c.dataType})${c.isPrimaryKey ? ' PK' : ''}`,
      value: c.name,
    }));
}

function toggleColumn(idx: number) {
  if (expandedColumns.value.has(idx)) {
    expandedColumns.value.delete(idx);
  } else {
    expandedColumns.value.add(idx);
  }
}

function toggleForeignKey(col: ColumnDef, enabled: boolean) {
  if (enabled) {
    col.foreignKey = {
      toTable: "",
      toColumn: "",
      onDelete: "NO ACTION",
      onUpdate: "NO ACTION",
    };
  } else {
    col.foreignKey = null;
  }
}

function onFkTableChange(col: ColumnDef) {
  if (col.foreignKey) {
    col.foreignKey.toColumn = "";
    // Auto-select PK column if available
    const table = existingTables.value.find(t => t.name === col.foreignKey!.toTable);
    if (table) {
      const pkCol = table.columns.find(c => c.isPrimaryKey);
      if (pkCol) {
        col.foreignKey.toColumn = pkCol.name;
        // Match the data type
        col.dataType = pkCol.dataType;
      }
    }
  }
}

// Data type options for OuiSelect
const dataTypeOptions = computed(() => {
  if (isPostgres.value) {
    return [
      { label: "serial (auto-increment)", value: "serial" },
      { label: "bigserial (auto-increment)", value: "bigserial" },
      { label: "integer", value: "integer" },
      { label: "bigint", value: "bigint" },
      { label: "smallint", value: "smallint" },
      { label: "decimal", value: "decimal" },
      { label: "numeric", value: "numeric" },
      { label: "real", value: "real" },
      { label: "double precision", value: "double precision" },
      { label: "varchar(255)", value: "varchar(255)" },
      { label: "varchar(50)", value: "varchar(50)" },
      { label: "varchar(100)", value: "varchar(100)" },
      { label: "text", value: "text" },
      { label: "char(1)", value: "char(1)" },
      { label: "uuid", value: "uuid" },
      { label: "timestamp", value: "timestamp" },
      { label: "timestamptz", value: "timestamptz" },
      { label: "date", value: "date" },
      { label: "time", value: "time" },
      { label: "interval", value: "interval" },
      { label: "boolean", value: "boolean" },
      { label: "jsonb", value: "jsonb" },
      { label: "json", value: "json" },
      { label: "bytea", value: "bytea" },
      { label: "inet", value: "inet" },
      { label: "cidr", value: "cidr" },
    ];
  }
  return [
    { label: "INT AUTO_INCREMENT", value: "INT AUTO_INCREMENT" },
    { label: "BIGINT AUTO_INCREMENT", value: "BIGINT AUTO_INCREMENT" },
    { label: "INT", value: "INT" },
    { label: "BIGINT", value: "BIGINT" },
    { label: "SMALLINT", value: "SMALLINT" },
    { label: "DECIMAL", value: "DECIMAL" },
    { label: "FLOAT", value: "FLOAT" },
    { label: "DOUBLE", value: "DOUBLE" },
    { label: "VARCHAR(255)", value: "VARCHAR(255)" },
    { label: "VARCHAR(50)", value: "VARCHAR(50)" },
    { label: "VARCHAR(100)", value: "VARCHAR(100)" },
    { label: "TEXT", value: "TEXT" },
    { label: "CHAR(1)", value: "CHAR(1)" },
    { label: "DATETIME", value: "DATETIME" },
    { label: "TIMESTAMP", value: "TIMESTAMP" },
    { label: "DATE", value: "DATE" },
    { label: "TIME", value: "TIME" },
    { label: "BOOLEAN", value: "BOOLEAN" },
    { label: "JSON", value: "JSON" },
    { label: "BLOB", value: "BLOB" },
  ];
});

// Generate DDL preview
const generatedDDL = computed(() => {
  if (!tableName.value) return "-- Enter a table name";

  const lines: string[] = [];
  const quote = isPostgres.value ? '"' : '`';
  lines.push(`CREATE TABLE ${quote}${tableName.value}${quote} (`);

  const colDefs: string[] = [];
  const pkColumns: string[] = [];
  const fkDefs: string[] = [];

  for (const col of columns.value) {
    if (!col.name) continue;

    let def = `  ${quote}${col.name}${quote} ${col.dataType}`;

    if (!col.isNullable) def += " NOT NULL";
    if (col.defaultValue) def += ` DEFAULT ${col.defaultValue}`;
    if (col.isUnique && !col.isPrimaryKey) def += " UNIQUE";

    colDefs.push(def);

    if (col.isPrimaryKey) {
      pkColumns.push(col.name);
    }

    if (col.foreignKey && col.foreignKey.toTable && col.foreignKey.toColumn) {
      const fkName = `fk_${tableName.value}_${col.name}`;
      fkDefs.push(
        `  CONSTRAINT ${quote}${fkName}${quote} FOREIGN KEY (${quote}${col.name}${quote}) ` +
        `REFERENCES ${quote}${col.foreignKey.toTable}${quote}(${quote}${col.foreignKey.toColumn}${quote}) ` +
        `ON DELETE ${col.foreignKey.onDelete} ON UPDATE ${col.foreignKey.onUpdate}`
      );
    }
  }

  lines.push(colDefs.join(",\n"));

  if (pkColumns.length > 0) {
    lines.push(`  ,PRIMARY KEY (${pkColumns.map((c) => `${quote}${c}${quote}`).join(", ")})`);
  }

  for (const fk of fkDefs) {
    lines.push("  ," + fk.trim());
  }

  lines.push(");");

  return lines.join("\n");
});

function addColumn() {
  const newIdx = idxCounter++;
  columns.value.push({
    __idx: newIdx,
    name: "",
    dataType: "varchar(255)",
    isNullable: true,
    isPrimaryKey: false,
    isUnique: false,
    defaultValue: "",
    autoIncrement: false,
    foreignKey: null,
  });
  expandedColumns.value.add(newIdx);
}

function removeColumn(idx: number) {
  if (columns.value.length <= 1) return;
  const col = columns.value[idx];
  if (col) {
    expandedColumns.value.delete(col.__idx);
  }
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

  // Validate foreign keys
  for (const col of validColumns) {
    if (col.foreignKey) {
      if (!col.foreignKey.toTable || !col.foreignKey.toColumn) {
        errors.value.columns = `Foreign key on "${col.name}" is incomplete`;
        break;
      }
    }
  }

  return Object.keys(errors.value).length === 0;
}

async function createTable() {
  if (!validate() || !organizationId.value) return;

  creating.value = true;

  try {
    const pkColumns = columns.value.filter((c) => c.isPrimaryKey && c.name).map((c) => c.name);

    // Build foreign keys array
    const foreignKeys = columns.value
      .filter((c) => c.name.trim() && c.foreignKey && c.foreignKey.toTable && c.foreignKey.toColumn)
      .map((c) => ({
        name: `fk_${tableName.value}_${c.name}`,
        fromColumns: [c.name],
        toTable: c.foreignKey!.toTable,
        toColumns: [c.foreignKey!.toColumn],
        onDelete: c.foreignKey!.onDelete,
        onUpdate: c.foreignKey!.onUpdate,
      }));

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
      foreignKeys: foreignKeys.length > 0 ? foreignKeys : undefined,
    });

    toast.success(`Table "${tableName.value}" created`);
    emit("created");
    close();
  } catch (err: unknown) {
    toast.error("Failed to create table", (err as Error).message);
  } finally {
    creating.value = false;
  }
}

function close() {
  open.value = false;
  // Reset form
  tableName.value = "";
  idxCounter = 2;
  columns.value = [
    { __idx: 0, name: "id", dataType: "serial", isNullable: false, isPrimaryKey: true, isUnique: false, defaultValue: "", autoIncrement: true, foreignKey: null },
    { __idx: 1, name: "created_at", dataType: "timestamptz", isNullable: false, isPrimaryKey: false, isUnique: false, defaultValue: "now()", autoIncrement: false, foreignKey: null },
  ];
  expandedColumns.value = new Set([0, 1]);
  errors.value = {};
}

// Load schema when dialog opens
watch(open, (isOpen) => {
  if (isOpen) {
    fetchSchema();
  } else {
    close();
  }
});
</script>

<style scoped>
.expand-enter-active,
.expand-leave-active {
  transition: all 0.15s ease;
  overflow: hidden;
}
.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0 !important;
  padding-bottom: 0 !important;
}
.expand-enter-to,
.expand-leave-from {
  max-height: 500px;
}
</style>
