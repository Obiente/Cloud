<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiCardTitle>Database Browser</OuiCardTitle>
        <OuiCardDescription>
          Browse your database schema with automatic introspection
        </OuiCardDescription>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="center">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search tables..."
              clearable
              class="max-w-xs"
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
              </template>
            </OuiInput>
            <OuiButton @click="loadSchema">
              <ArrowPathIcon class="h-4 w-4" />
              Refresh Schema
            </OuiButton>
          </OuiFlex>

          <!-- Loading State -->
          <OuiStack v-if="loading" align="center" gap="md" class="py-10">
            <OuiSpinner size="lg" />
            <OuiText color="secondary">Loading schema...</OuiText>
          </OuiStack>

          <!-- Error State -->
          <ErrorAlert
            v-else-if="error"
            :error="error"
            title="Failed to load schema"
          />

          <!-- Tables List -->
          <OuiStack v-else-if="tables.length > 0" gap="sm">
            <OuiText as="h3" size="lg" weight="semibold">
              Tables ({{ filteredTables.length }})
            </OuiText>
            <OuiCard
              v-for="table in filteredTables"
              :key="table.name"
              variant="outline"
              class="cursor-pointer hover:ring-2 hover:ring-primary/20"
              @click="selectedTable = table"
            >
              <OuiCardBody>
                <OuiFlex justify="between" align="center">
                  <OuiStack gap="xs">
                    <OuiText weight="semibold">{{ table.name }}</OuiText>
                    <OuiText color="secondary" size="sm">
                      {{ table.rowCount?.toLocaleString() || 0 }} rows
                      <span v-if="table.sizeBytes">
                        · {{ formatBytes(table.sizeBytes) }}
                      </span>
                    </OuiText>
                  </OuiStack>
                  <ChevronRightIcon class="h-5 w-5 text-secondary" />
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>

          <!-- Empty State -->
          <OuiStack v-else align="center" gap="md" class="py-10">
            <OuiText color="secondary">No tables found</OuiText>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Table Details Dialog -->
    <OuiDialog 
      v-model="showTableDialog"
      :title="selectedTable?.name || 'Table Details'"
      description="Table structure and columns"
    >
      <OuiDialogContent size="xl">

        <OuiStack v-if="selectedTable" gap="lg" class="py-4">
          <OuiTable
            :columns="[
              { key: 'name', label: 'Column' },
              { key: 'type', label: 'Type' },
              { key: 'nullable', label: 'Nullable' },
              { key: 'default', label: 'Default' },
              { key: 'primary', label: 'Primary Key' },
            ]"
            :rows="selectedTable.columns.map((col: any) => ({
              name: col.name,
              type: col.dataType,
              nullable: col.isNullable,
              default: col.defaultValue,
              primary: col.isPrimaryKey,
              column: col,
            }))"
          >
            <template #cell-type="{ row }">
              <OuiCode :code="row.type" />
            </template>
            <template #cell-nullable="{ row }">
              <OuiBadge :color="row.nullable ? 'secondary' : 'warning'">
                {{ row.nullable ? "Yes" : "No" }}
              </OuiBadge>
            </template>
            <template #cell-default="{ row }">
              <OuiText v-if="row.default" size="sm">
                {{ row.default }}
              </OuiText>
              <OuiText v-else color="secondary" size="sm">—</OuiText>
            </template>
            <template #cell-primary="{ row }">
              <OuiBadge v-if="row.primary" color="primary">
                Primary Key
              </OuiBadge>
              <OuiText v-else color="secondary" size="sm">—</OuiText>
            </template>
          </OuiTable>
        </OuiStack>

        <OuiDialogFooter>
          <OuiButton @click="showTableDialog = false">Close</OuiButton>
        </OuiDialogFooter>
      </OuiDialogContent>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { MagnifyingGlassIcon, ArrowPathIcon, ChevronRightIcon } from "@heroicons/vue/24/outline";
import { ref, computed, onMounted } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { formatBytes } from "~/utils/common";
import ErrorAlert from "~/components/ErrorAlert.vue";

const props = defineProps<{
  databaseId: string;
  databaseType: string;
}>();

const organizationId = useOrganizationId();
const dbClient = useConnectClient(DatabaseService);

const searchQuery = ref("");
const loading = ref(false);
const error = ref<any>(null);
const tables = ref<any[]>([]);
const selectedTable = ref<any>(null);
const showTableDialog = computed({
  get: () => selectedTable.value !== null,
  set: (val) => {
    if (!val) selectedTable.value = null;
  },
});

const filteredTables = computed(() => {
  if (!searchQuery.value) return tables.value;
  const query = searchQuery.value.toLowerCase();
  return tables.value.filter((table) =>
    table.name?.toLowerCase().includes(query)
  );
});

async function loadSchema() {
  loading.value = true;
  error.value = null;

  try {
    if (!organizationId.value) return;
    const res = await dbClient.getDatabaseSchema({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
    });
    tables.value = res.tables || [];
  } catch (err: any) {
    error.value = err;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  loadSchema();
});
</script>

