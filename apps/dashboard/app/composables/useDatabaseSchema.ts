import { ref, type Ref } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";

export interface SchemaTable {
  name: string;
  schema: string;
  type: string;
  rowCount: bigint;
  sizeBytes: bigint;
  columns: SchemaColumn[];
  indexes: SchemaIndex[];
  foreignKeys: SchemaForeignKey[];
}

export interface SchemaColumn {
  name: string;
  dataType: string;
  isNullable: boolean;
  defaultValue?: string;
  isPrimaryKey: boolean;
  isUnique: boolean;
  comment?: string;
  ordinalPosition: number;
}

export interface SchemaIndex {
  name: string;
  isUnique: boolean;
  isPrimary: boolean;
  columnNames: string[];
  type?: string;
}

export interface SchemaForeignKey {
  name: string;
  fromTable: string;
  fromColumns: string[];
  toTable: string;
  toColumns: string[];
  onDelete?: string;
  onUpdate?: string;
}

export interface SchemaView {
  name: string;
  schema: string;
  definition: string;
}

export interface SchemaFunction {
  name: string;
  schema: string;
  returnType: string;
}

// Cache by database ID
const schemaCache = new Map<
  string,
  {
    tables: SchemaTable[];
    views: SchemaView[];
    functions: SchemaFunction[];
    fetchedAt: number;
  }
>();

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

export function useDatabaseSchema(databaseId: Ref<string>) {
  const organizationId = useOrganizationId();
  const dbClient = useConnectClient(DatabaseService);

  const tables = ref<SchemaTable[]>([]);
  const views = ref<SchemaView[]>([]);
  const functions = ref<SchemaFunction[]>([]);
  const loading = ref(false);
  const error = ref<any>(null);

  async function fetchSchema(force = false) {
    const id = databaseId.value;
    if (!id || !organizationId.value) return;

    // Check cache
    if (!force) {
      const cached = schemaCache.get(id);
      if (cached && Date.now() - cached.fetchedAt < CACHE_TTL) {
        tables.value = cached.tables;
        views.value = cached.views;
        functions.value = cached.functions;
        return;
      }
    }

    loading.value = true;
    error.value = null;

    try {
      const res = await dbClient.getDatabaseSchema({
        organizationId: organizationId.value,
        databaseId: id,
      });

      const mappedTables: SchemaTable[] = (res.tables || []).map((t: any) => ({
        name: t.name || "",
        schema: t.schema || "",
        type: t.type || "table",
        rowCount: t.rowCount ?? 0n,
        sizeBytes: t.sizeBytes ?? 0n,
        columns: (t.columns || []).map((c: any) => ({
          name: c.name || "",
          dataType: c.dataType || "",
          isNullable: c.isNullable ?? false,
          defaultValue: c.defaultValue,
          isPrimaryKey: c.isPrimaryKey ?? false,
          isUnique: c.isUnique ?? false,
          comment: c.comment,
          ordinalPosition: c.ordinalPosition ?? 0,
        })),
        indexes: (t.indexes || []).map((i: any) => ({
          name: i.name || "",
          isUnique: i.isUnique ?? false,
          isPrimary: i.isPrimary ?? false,
          columnNames: i.columnNames || [],
          type: i.type,
        })),
        foreignKeys: (t.foreignKeys || []).map((fk: any) => ({
          name: fk.name || "",
          fromTable: fk.fromTable || "",
          fromColumns: fk.fromColumns || [],
          toTable: fk.toTable || "",
          toColumns: fk.toColumns || [],
          onDelete: fk.onDelete,
          onUpdate: fk.onUpdate,
        })),
      }));

      const mappedViews: SchemaView[] = (res.views || []).map((v: any) => ({
        name: v.name || "",
        schema: v.schema || "",
        definition: v.definition || "",
      }));

      const mappedFunctions: SchemaFunction[] = (res.functions || []).map(
        (f: any) => ({
          name: f.name || "",
          schema: f.schema || "",
          returnType: f.returnType || "",
        })
      );

      tables.value = mappedTables;
      views.value = mappedViews;
      functions.value = mappedFunctions;

      schemaCache.set(id, {
        tables: mappedTables,
        views: mappedViews,
        functions: mappedFunctions,
        fetchedAt: Date.now(),
      });
    } catch (err: any) {
      error.value = err;
    } finally {
      loading.value = false;
    }
  }

  function refresh() {
    return fetchSchema(true);
  }

  // DDL operations
  async function createTable(request: {
    tableName: string;
    columns: Array<{
      name: string;
      dataType: string;
      isNullable?: boolean;
      defaultValue?: string;
      isUnique?: boolean;
      autoIncrement?: boolean;
    }>;
    primaryKey?: { columnNames: string[]; name?: string };
    foreignKeys?: Array<{
      name: string;
      fromColumns: string[];
      toTable: string;
      toColumns: string[];
      onDelete?: string;
      onUpdate?: string;
    }>;
    comment?: string;
  }) {
    if (!organizationId.value || !databaseId.value) return;

    await dbClient.createTable({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName: request.tableName,
      columns: request.columns,
      primaryKey: request.primaryKey,
      foreignKeys: request.foreignKeys,
      comment: request.comment,
    });
    await refresh();
  }

  async function alterTable(
    tableName: string,
    operations: Array<{
      addColumn?: {
        column: {
          name: string;
          dataType: string;
          isNullable?: boolean;
          defaultValue?: string;
          isUnique?: boolean;
        };
        afterColumn?: string;
      };
      dropColumn?: { columnName: string; cascade?: boolean };
      modifyColumn?: {
        columnName: string;
        newDataType?: string;
        isNullable?: boolean;
        defaultValue?: string;
        dropDefault?: boolean;
      };
      renameColumn?: { oldName: string; newName: string };
      addForeignKey?: {
        foreignKey: {
          name: string;
          fromColumns: string[];
          toTable: string;
          toColumns: string[];
          onDelete?: string;
          onUpdate?: string;
        };
      };
      dropForeignKey?: { constraintName: string };
      addUnique?: { name: string; columnNames: string[] };
      dropConstraint?: { constraintName: string };
    }>
  ) {
    if (!organizationId.value || !databaseId.value) return;

    // Ensure each operation has $typeName for protobuf compatibility
    const typedOperations = operations.map(op => ({
      $typeName: "obiente.cloud.databases.v1.AlterTableOperation" as const,
      ...op
    }));
    await dbClient.alterTable({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName,
      operations,
    });
    await refresh();
  }

  async function dropTable(tableName: string, cascade = false) {
    if (!organizationId.value || !databaseId.value) return;

    await dbClient.dropTable({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName,
      cascade,
      ifExists: true,
    });
    await refresh();
  }

  async function renameTable(oldName: string, newName: string) {
    if (!organizationId.value || !databaseId.value) return;

    await dbClient.renameTable({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      oldName,
      newName,
    });
    await refresh();
  }

  async function truncateTable(tableName: string, cascade = false) {
    if (!organizationId.value || !databaseId.value) return;

    const res = await dbClient.truncateTable({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName,
      cascade,
    });
    await refresh();
    return res.rowsDeleted;
  }

  async function createIndex(
    tableName: string,
    index: { name: string; columnNames: string[]; isUnique?: boolean; type?: string },
    options?: { ifNotExists?: boolean; concurrently?: boolean }
  ) {
    if (!organizationId.value || !databaseId.value) return;

    await dbClient.createIndex({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName,
      index,
      ifNotExists: options?.ifNotExists ?? false,
      concurrently: options?.concurrently ?? false,
    });
    await refresh();
  }

  async function dropIndex(
    indexName: string,
    options?: { cascade?: boolean; ifExists?: boolean; concurrently?: boolean }
  ) {
    if (!organizationId.value || !databaseId.value) return;

    await dbClient.dropIndex({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      indexName,
      cascade: options?.cascade ?? false,
      ifExists: options?.ifExists ?? true,
      concurrently: options?.concurrently ?? false,
    });
    await refresh();
  }

  async function getTableDDL(tableName: string): Promise<string> {
    if (!organizationId.value || !databaseId.value) return "";

    const res = await dbClient.getTableDDL({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      tableName,
    });
    return res.ddl;
  }

  return {
    tables,
    views,
    functions,
    loading,
    error,
    fetchSchema,
    refresh,
    // DDL operations
    createTable,
    alterTable,
    dropTable,
    renameTable,
    truncateTable,
    createIndex,
    dropIndex,
    getTableDDL,
  };
}
