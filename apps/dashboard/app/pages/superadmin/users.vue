<template>
  <OuiContainer size="full">
    <OuiStack gap="2xl">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Users</OuiText>
        <OuiText color="muted"
          >View and manage all users in the system.</OuiText
        >
      </OuiStack>

      <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
        <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <OuiStack gap="xs">
              <OuiText tag="h2" size="xl" weight="bold">All Users</OuiText>
              <OuiText color="muted" size="sm">
                {{ pagination.total }} total users
              </OuiText>
            </OuiStack>
            <OuiContainer size="sm" class="w-64">
              <OuiInput
                v-model="search"
                type="search"
                placeholder="Search users…"
                clearable
                size="sm"
                @update:model-value="handleSearch"
              />
            </OuiContainer>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody class="p-0">
          <OuiTable
            :columns="columns"
            :rows="tableRows"
            :empty-text="loading ? 'Loading users...' : 'No users found.'"
            row-class="hover:bg-surface-subtle/50"
            @row-click="(row) => viewUser(row.id)"
          >
            <template #cell-user="{ row }">
              <OuiFlex gap="sm" align="center">
              <OuiAvatar
                :name="row.name || row.email || row.id"
                :src="row.avatarUrl"
              />
                <div>
                  <OuiText weight="medium">
                    {{ row.name || row.email || row.id }}
                  </OuiText>
                  <OuiText
                    v-if="row.id"
                    color="muted"
                    size="xs"
                    class="font-mono"
                  >
                    {{ row.id }}
                  </OuiText>
                </div>
              </OuiFlex>
            </template>
            <template #cell-email="{ value }">
              {{ value || "—" }}
            </template>
            <template #cell-username="{ value }">
              {{ value || "—" }}
            </template>
            <template #cell-roles="{ row }">
              <OuiFlex gap="xs" wrap="wrap">
                <OuiBadge
                  v-for="role in row.roles"
                  :key="role"
                  variant="primary"
                  tone="soft"
                  size="sm"
                >
                  {{ role }}
                </OuiBadge>
                <OuiText v-if="!row.roles?.length" color="muted" size="sm">
                  —
                </OuiText>
              </OuiFlex>
            </template>
            <template #cell-actions="{ row }">
              <OuiButton
                size="sm"
                variant="ghost"
                @click.stop="viewUser(row.id)"
              >
                View
              </OuiButton>
            </template>
          </OuiTable>

          <OuiFlex
            v-if="pagination.totalPages > 1"
            align="center"
            justify="between"
            class="px-6 py-4 border-t border-border-muted"
          >
            <OuiText color="muted" size="sm">
              Page {{ pagination.page }} of {{ pagination.totalPages }}
            </OuiText>
            <OuiFlex gap="sm">
              <OuiButton
                variant="ghost"
                size="sm"
                :disabled="pagination.page <= 1"
                @click="goToPage(pagination.page - 1)"
              >
                Previous
              </OuiButton>
              <OuiButton
                variant="ghost"
                size="sm"
                :disabled="pagination.page >= pagination.totalPages"
                @click="goToPage(pagination.page + 1)"
              >
                Next
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const client = useConnectClient(SuperadminService);
const router = useRouter();

const users = ref<any[]>([]);
const pagination = ref({
  page: 1,
  perPage: 50,
  total: 0,
  totalPages: 0,
});
const search = ref("");
const loading = ref(false);
let searchTimeout: NodeJS.Timeout | null = null;

const columns = [
  { key: "user", label: "User", defaultWidth: 250, minWidth: 200 },
  { key: "email", label: "Email", defaultWidth: 200, minWidth: 150 },
  { key: "username", label: "Username", defaultWidth: 150, minWidth: 120 },
  { key: "roles", label: "Roles", defaultWidth: 150, minWidth: 100 },
  { key: "organizations", label: "Organizations", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80, resizable: false },
];

const tableRows = computed(() => users.value);

async function loadUsers() {
  loading.value = true;
  try {
    const response = await client.listUsers({
      page: pagination.value.page,
      perPage: pagination.value.perPage,
      search: search.value || undefined,
    });
    users.value = response.users || [];
    pagination.value = {
      page: response.pagination?.page || 1,
      perPage: response.pagination?.perPage || 50,
      total: response.pagination?.total || 0,
      totalPages: response.pagination?.totalPages || 0,
    };
  } catch (error: any) {
    console.error("Failed to load users:", error);
    const { toast } = useToast();
    toast.error(error?.message || "Failed to load users");
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  if (searchTimeout) {
    clearTimeout(searchTimeout);
  }
  searchTimeout = setTimeout(() => {
    pagination.value.page = 1;
    loadUsers();
  }, 300);
}

function goToPage(page: number) {
  pagination.value.page = page;
  loadUsers();
}

function viewUser(userId: string) {
  router.push(`/superadmin/users/${userId}`);
}

await loadUsers();
</script>

