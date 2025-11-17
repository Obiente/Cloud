<template>
  <SuperadminPageLayout
    title="Users"
    description="View and manage all users in the system."
    :columns="columns"
    :rows="tableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="loading ? 'Loading users...' : 'No users match your filters.'"
    :loading="loading"
    :pagination="{
      page: pagination.page,
      totalPages: pagination.totalPages,
      total: pagination.total,
      perPage: pagination.perPage,
    }"
    search-placeholder="Search by name, email, username, ID…"
    @update:search="handleSearchUpdate"
    @filter-change="handleFilterChange"
    @refresh="loadUsers"
    @row-click="(row) => viewUser(row.id)"
    @page-change="goToPage"
  >
            <template #cell-user="{ row }">
              <OuiFlex gap="sm" align="center">
                <OuiAvatar
                  :name="row.name || row.email || row.id"
                  :src="row.avatarUrl"
                />
                <SuperadminResourceCell
                  :name="row.name || row.email"
                  :id="row.id"
                />
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
            <template #cell-organizations="{ row }">
              <OuiFlex gap="xs" wrap="wrap">
                <OuiBadge
                  v-for="org in row.organizations"
                  :key="org.organizationId"
                  variant="secondary"
                  tone="soft"
                  size="sm"
                >
                  {{ org.organizationName || org.organizationId }}
                </OuiBadge>
                <OuiText v-if="!row.organizations?.length" color="muted" size="sm">
                  —
                </OuiText>
              </OuiFlex>
            </template>
            <template #cell-actions="{ row }">
              <SuperadminActionsCell :actions="getUserActions(row)" />
            </template>
  </SuperadminPageLayout>
</template>

<script setup lang="ts">
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminActionsCell, { type Action } from "~/components/superadmin/SuperadminActionsCell.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";

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
const roleFilter = ref<string>("all");
let searchTimeout: NodeJS.Timeout | null = null;

const columns = [
  { key: "user", label: "User", defaultWidth: 250, minWidth: 200 },
  { key: "email", label: "Email", defaultWidth: 200, minWidth: 150 },
  { key: "username", label: "Username", defaultWidth: 150, minWidth: 120 },
  { key: "roles", label: "Roles", defaultWidth: 150, minWidth: 100 },
  { key: "organizations", label: "Organizations", defaultWidth: 200, minWidth: 150 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80, resizable: false },
];

const roleOptions = computed(() => {
  const roles = new Set<string>();
  users.value.forEach((user) => {
    if (user.roles) {
      user.roles.forEach((role: string) => roles.add(role));
    }
  });
  const sortedRoles = Array.from(roles).sort();
  return [
    { label: "All roles", value: "all" },
    ...sortedRoles.map((role) => ({ label: role, value: role })),
  ];
});

const filterConfigs = computed(() => [
  {
    key: "role",
    placeholder: "Role",
    items: roleOptions.value,
  },
] as FilterConfig[]);

const filteredUsers = computed(() => {
  const term = search.value.trim().toLowerCase();
  const role = roleFilter.value;

  return users.value.map((user) => {
    // Fetch organizations for each user
    const userOrgs = user.organizations || [];
    return {
      ...user,
      organizations: userOrgs,
    };
  }).filter((user) => {
    // Role filter
    if (role !== "all") {
      if (!user.roles || !user.roles.includes(role)) {
        return false;
      }
    }

    // Search filter
    if (!term) return true;

    const searchable = [
      user.name,
      user.email,
      user.preferredUsername,
      user.id,
      ...(user.roles || []),
      ...(user.organizations || []).map((org: any) => org.organizationName || org.organizationId),
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    return searchable.includes(term);
  });
});

const tableRows = computed(() => filteredUsers.value);

function handleSearchUpdate(value: string) {
  search.value = value;
  handleSearch();
}

function handleFilterChange(key: string, value: string) {
  if (key === "role") {
    roleFilter.value = value;
  }
}

async function loadUsers() {
  try {
    const response = await client.listUsers({
      page: pagination.value.page,
      perPage: pagination.value.perPage,
      search: search.value || undefined,
    });
    const userList = response.users || [];
    
    // Fetch organizations for each user
    const usersWithOrgs = await Promise.all(
      userList.map(async (user) => {
        try {
          const userDetail = await client.getUser({ userId: user.id });
          return {
            ...user,
            organizations: userDetail.organizations || [],
          };
        } catch (err) {
          console.error(`Failed to load orgs for user ${user.id}:`, err);
          return {
            ...user,
            organizations: [],
          };
        }
      })
    );
    
    users.value = usersWithOrgs;
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
  }
}

// Use client-side fetching for non-blocking navigation
const { pending: loading } = useClientFetch(
  () => `superadmin-users-${pagination.value.page}-${search.value}`,
  loadUsers
);

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

const getUserActions = (row: any): Action[] => {
  return [
    {
      key: "view",
      label: "View",
      onClick: () => viewUser(row.id),
    },
  ];
};

await loadUsers();
</script>

