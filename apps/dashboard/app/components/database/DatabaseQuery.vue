<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiCardTitle>Query Editor</OuiCardTitle>
        <OuiCardDescription>
          Execute SQL queries on your database
        </OuiCardDescription>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiTextarea
            v-model="query"
            placeholder="SELECT * FROM users LIMIT 10;"
            :rows="10"
            class="font-mono text-sm"
          />

          <OuiFlex justify="between" align="center">
            <OuiFlex gap="sm" align="center">
              <OuiText color="secondary" size="sm">
                Max rows:
              </OuiText>
              <OuiInput
                v-model="maxRows"
                type="number"
                size="sm"
                class="w-24"
                :min="1"
                :max="10000"
              />
            </OuiFlex>
            <OuiButton
              color="primary"
              :loading="executing"
              :disabled="!query.trim()"
              @click="executeQuery"
            >
              <PlayIcon class="h-4 w-4" />
              Execute Query
            </OuiButton>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Results -->
    <OuiCard v-if="results">
      <OuiCardHeader>
        <OuiCardTitle>
          Results
          <OuiBadge v-if="results.truncated" color="warning" class="ml-2">
            Truncated
          </OuiBadge>
        </OuiCardTitle>
        <OuiCardDescription>
          {{ results.rowCount }} row(s) in {{ results.executionTimeMs }}ms
        </OuiCardDescription>
      </OuiCardHeader>
      <OuiCardBody>
        <div class="overflow-x-auto">
          <OuiTable
            :columns="results.columns.map((col: any) => ({ key: col.name, label: col.name }))"
            :rows="results.rows.map((row: any, idx: number) => ({
              ...row.cells.reduce((acc: Record<string, any>, cell: any) => {
                acc[cell.columnName] = cell.isNull ? null : cell.value;
                return acc;
              }, {} as Record<string, any>),
              _rowIndex: idx,
            }))"
          >
            <template v-for="col in results.columns" :key="col.name" #[`header-${col.name}`]>
              <div>
                {{ col.name }}
                <OuiText color="secondary" size="xs" class="block">
                  {{ col.dataType }}
                </OuiText>
              </div>
            </template>
            <template v-for="col in results.columns" :key="col.name" #[`cell-${col.name}`]="{ row }">
              <OuiText v-if="row[col.name] === null" color="secondary" size="sm">
                NULL
              </OuiText>
              <OuiText v-else size="sm">
                {{ row[col.name] }}
              </OuiText>
            </template>
          </OuiTable>
        </div>

        <OuiAlert v-if="results.truncated" color="warning" class="mt-4">
          Results were truncated. Only the first {{ maxRows }} rows are shown.
        </OuiAlert>
      </OuiCardBody>
    </OuiCard>

    <!-- Error -->
    <OuiAlert v-if="queryError" color="danger">
      <OuiText weight="semibold">Query Error</OuiText>
      <OuiText size="sm">{{ queryError }}</OuiText>
    </OuiAlert>
  </OuiStack>
</template>

<script setup lang="ts">
import { PlayIcon } from "@heroicons/vue/24/outline";
import { ref } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";

const props = defineProps<{
  databaseId: string;
  databaseType: string;
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const query = ref("");
const maxRows = ref<string>("1000");
const executing = ref(false);
const results = ref<any>(null);
const queryError = ref<string | null>(null);

async function executeQuery() {
  if (!query.value.trim()) {
    return;
  }

  executing.value = true;
  queryError.value = null;
  results.value = null;

  try {
    if (!organizationId.value) return;
    const res = await dbClient.executeQuery({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      query: query.value,
      maxRows: parseInt(maxRows.value) || 1000,
      timeoutSeconds: 30,
    });
    results.value = res;
    toast.success("Query executed successfully");
  } catch (err: any) {
    queryError.value = err.message || "Query execution failed";
    toast.error("Query execution failed", err.message);
  } finally {
    executing.value = false;
  }
}
</script>

