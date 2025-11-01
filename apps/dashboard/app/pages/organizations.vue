<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  import {
    OrganizationService,
    AdminService,
    type OrganizationMember,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { CheckIcon } from "@heroicons/vue/24/outline";

  const name = ref("");
  const slug = ref("");
  const inviteEmail = ref("");
  const inviteRole = ref("");
  const error = ref("");

  const auth = useAuth();
  const orgClient = useConnectClient(OrganizationService);
  const adminClient = useConnectClient(AdminService);
  const route = useRoute();

  // Store the target org from query params before any other logic
  const targetOrgId =
    route.query.organizationId && typeof route.query.organizationId === "string"
      ? route.query.organizationId
      : null;

  // Check for organizationId in query params (from superadmin navigation)
  if (targetOrgId) {
    auth.switchOrganization(targetOrgId);
  }

  const organizations = computed(() => auth.organizations || []);
  const selectedOrg = computed({
    get: () => auth.currentOrganizationId,
    set: (id: string) => {
      if (id) auth.switchOrganization(id);
    },
  });

  const activeTab = ref("members");
  const transferDialogOpen = ref(false);
  const transferCandidate = ref<OrganizationMember | null>(null);

  // Check for tab in query params
  if (route.query.tab && typeof route.query.tab === "string") {
    activeTab.value = route.query.tab;
  }

  if (!organizations.value.length && auth.isAuthenticated) {
    // Only show user's own organizations in the select, even for superadmins
    const res = await orgClient.listOrganizations({ onlyMine: true });
    auth.setOrganizations(res.organizations || []);
    // Ensure target org is set after organizations are loaded
    if (targetOrgId) {
      auth.switchOrganization(targetOrgId);
    }
  }

  const { data: membersData, refresh: refreshMembers } = await useAsyncData(
    () =>
      selectedOrg.value
        ? `org-members-${selectedOrg.value}`
        : "org-members-none",
    async () => {
      if (!selectedOrg.value) return [] as OrganizationMember[];
      const res = await orgClient.listMembers({
        organizationId: selectedOrg.value,
      });
      return res.members || [];
    },
    { watch: [selectedOrg], server: true }
  );
  const members = computed(() => membersData.value || []);

  const DEFAULT_INVITE_ROLE = "member";
  const OWNER_TRANSFER_FALLBACK_ROLE = "admin";
  const SYSTEM_ROLES = [
    { value: "owner", label: "Owner", disabled: true },
    { value: "admin", label: "Admin", disabled: false },
    { value: "member", label: "Member", disabled: false },
    { value: "viewer", label: "Viewer", disabled: false },
  ];

  const defaultRoleItems = computed(() =>
    SYSTEM_ROLES.map((role) => ({
      label: `${role.label} (system)`,
      value: role.value,
      disabled: role.disabled ?? false,
    }))
  );

  const { data: roleCatalogData, refresh: refreshRoleCatalog } =
    await useAsyncData(
      () =>
        selectedOrg.value
          ? `org-role-catalog-${selectedOrg.value}`
          : "org-role-catalog-none",
      async () => {
        if (!selectedOrg.value) return [] as { id: string; name: string }[];
        const res = await adminClient.listRoles({
          organizationId: selectedOrg.value,
        });
        return (res.roles || []).map((r) => ({ id: r.id, name: r.name }));
      },
      { watch: [selectedOrg], server: true }
    );

  const roleDisplayMap = computed(() => {
    const map = new Map<string, string>();
    SYSTEM_ROLES.forEach((role) => map.set(role.value, role.label));
    (roleCatalogData.value || []).forEach((role) => {
      map.set(role.id, role.name);
    });
    return map;
  });

  const customRoleItems = computed(() =>
    (roleCatalogData.value || []).map((r) => ({
      label: `${r.name} (custom)`,
      value: r.id,
      disabled: false,
    }))
  );

  const roleItems = computed(() => {
    const items = [...defaultRoleItems.value, ...customRoleItems.value];
    const order = new Map(SYSTEM_ROLES.map((role, idx) => [role.value, idx]));
    return items.sort((a, b) => {
      const aIdx = order.has(a.value) ? order.get(a.value)! : order.size + 1;
      const bIdx = order.has(b.value) ? order.get(b.value)! : order.size + 1;
      if (aIdx !== bIdx) return aIdx - bIdx;
      return a.label.localeCompare(b.label);
    });
  });

  const selectableRoleItems = computed(() =>
    roleItems.value.filter((item) => !item.disabled)
  );

  const currentUserIdentifiers = computed(() => {
    const identifiers = new Set<string>();
    const sessionUser: any = auth.user || null;
    if (!sessionUser) {
      return identifiers;
    }
    [sessionUser.id, sessionUser.sub, sessionUser.userId].forEach((id) => {
      if (id) {
        identifiers.add(String(id));
      }
    });
    return identifiers;
  });

  const currentMemberRecord = computed(
    () =>
      members.value.find((member) => {
        const memberUserId = member.user?.id;
        if (!memberUserId) return false;
        return currentUserIdentifiers.value.has(memberUserId);
      }) || null
  );

  const currentUserIsOwner = computed(
    () => currentMemberRecord.value?.role === "owner"
  );

  watch(
    [selectedOrg, roleItems],
    () => {
      inviteEmail.value = "";
      const exists = selectableRoleItems.value.find(
        (item) => item.value === inviteRole.value
      );
      if (!exists) {
        const preferred = selectableRoleItems.value.find(
          (item) => item.value === DEFAULT_INVITE_ROLE
        );
        inviteRole.value =
          (preferred || selectableRoleItems.value[0])?.value ?? "";
      }
    },
    { immediate: true }
  );

  async function syncOrganizations() {
    if (!auth.isAuthenticated) return;
    const res = await orgClient.listOrganizations({});
    auth.setOrganizations(res.organizations || []);
  }

  async function createOrg() {
    error.value = "";
    try {
      const res = await orgClient.createOrganization({
        name: name.value,
        slug: slug.value || undefined,
      });
      await syncOrganizations();
      if (res.organization?.id) {
        await auth.switchOrganization(res.organization.id);
        await refreshMembers();
        await refreshRoleCatalog();
      }
      auth.notifyOrganizationsUpdated();
      name.value = "";
      slug.value = "";
    } catch (e: any) {
      error.value = e?.message || "Error creating organization";
    }
  }

  async function invite() {
    if (!selectedOrg.value || !inviteEmail.value || !inviteRole.value) return;
    await orgClient.inviteMember({
      organizationId: selectedOrg.value,
      email: inviteEmail.value,
      role: inviteRole.value,
    });
    inviteEmail.value = "";
    await refreshMembers();
  }

  async function setRole(memberId: string, role: string) {
    if (!selectedOrg.value) return;
    await orgClient.updateMember({
      organizationId: selectedOrg.value,
      memberId,
      role,
    });
    await refreshMembers();
  }

  function roleLabel(role: string) {
    return roleDisplayMap.value.get(role) || role;
  }

  function openTransferDialog(member: OrganizationMember) {
    transferCandidate.value = member;
    transferDialogOpen.value = true;
  }

  async function confirmTransferOwnership() {
    if (!selectedOrg.value || !transferCandidate.value) return;
    await orgClient.transferOwnership({
      organizationId: selectedOrg.value,
      newOwnerMemberId: transferCandidate.value.id,
      fallbackRole: OWNER_TRANSFER_FALLBACK_ROLE,
    });
    transferDialogOpen.value = false;
    transferCandidate.value = null;
    await refreshAll();
  }

  async function remove(memberId: string) {
    await orgClient.removeMember({
      organizationId: selectedOrg.value,
      memberId,
    });
    await refreshMembers();
  }

  async function refreshAll() {
    await syncOrganizations();
    await Promise.all([refreshMembers(), refreshRoleCatalog()]);
  }

  const memberStats = computed(() => ({
    total: members.value.length,
    owners: members.value.filter((m) => m.role === "owner").length,
    pending: members.value.filter((m) => m.status !== "active").length,
  }));

  const tabs = [
    { id: "members", label: "Members" },
    { id: "roles", label: "Roles" },
    { id: "billing", label: "Billing" },
  ];

  const inviteDisabled = computed(
    () => !selectedOrg.value || !inviteEmail.value || !inviteRole.value
  );

  const transferDialogSummary = computed(() => {
    if (!transferCandidate.value) return "";
    const target = transferCandidate.value;
    const label =
      target.user?.name ||
      target.user?.email ||
      target.user?.preferredUsername ||
      target.user?.id ||
      "this member";
    return `Ownership will move to ${label}. You will become ${OWNER_TRANSFER_FALLBACK_ROLE}.`;
  });
</script>

<template>
  <OuiStack gap="lg">
    <OuiGrid cols="1" colsLg="3" gap="lg">
      <OuiCard class="col-span-2">
        <OuiCardHeader>
          <OuiFlex align="center" justify="between">
            <OuiStack gap="xs">
              <OuiText size="xl" weight="semibold">Organizations</OuiText>
              <OuiText color="muted">Create and manage your teams.</OuiText>
            </OuiStack>
            <OuiButton variant="ghost" @click="refreshAll">Refresh</OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiGrid cols="1" colsLg="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Select Organization</OuiText>
                <OuiSelect
                  v-model="selectedOrg"
                  placeholder="Choose organization"
                  :items="
                    organizations.map((o) => ({
                      label: o.name ?? o.slug ?? o.id,
                      value: o.id,
                    }))
                  "
                />
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Your Role</OuiText>
                <OuiBadge
                  v-if="currentMemberRecord"
                  tone="solid"
                  variant="primary"
                >
                  {{ roleLabel(currentMemberRecord.role) }}
                </OuiBadge>
                <OuiText v-else color="muted" size="sm">
                  You are not a member of this organization.
                </OuiText>
              </OuiStack>
            </OuiGrid>
            <div class="border border-border-muted/40 rounded-xl" />
            <OuiStack gap="md" as="form" @submit.prevent="createOrg">
              <OuiText size="sm" weight="medium">Create Organization</OuiText>
              <OuiGrid cols="1" colsLg="2" gap="md">
                <OuiInput
                  v-model="name"
                  label="Name"
                  placeholder="Acme Inc"
                  required
                />
                <OuiInput v-model="slug" label="Slug" placeholder="acme" />
              </OuiGrid>
              <OuiFlex gap="sm">
                <OuiButton type="submit">Create</OuiButton>
                <OuiText v-if="error" color="danger">{{ error }}</OuiText>
              </OuiFlex>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardHeader>
          <OuiText size="lg" weight="semibold">Member Stats</OuiText>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="sm">
              <OuiText size="2xl" weight="semibold">{{
                memberStats.total
              }}</OuiText>
              <OuiText color="muted" size="sm">Total members</OuiText>
            </OuiStack>
            <OuiStack gap="sm">
              <OuiText size="lg" weight="medium">Owners</OuiText>
              <OuiProgress
                :value="memberStats.owners"
                :max="Math.max(memberStats.total, 1)"
              />
              <OuiText color="muted" size="sm"
                >{{ memberStats.owners }} owners</OuiText
              >
            </OuiStack>
            <OuiStack gap="sm">
              <OuiText size="lg" weight="medium">Pending Invites</OuiText>
              <OuiText color="muted">{{ memberStats.pending }}</OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <OuiTabs v-model="activeTab" :tabs="tabs" />

    <OuiCard>
      <OuiCardBody>
        <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
          <template #members>
            <OuiFlex align="center" justify="between" class="mb-4">
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">Members</OuiText>
                <OuiText color="muted" size="sm">
                  Manage member roles and ownership.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <div class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead>
                  <tr class="text-text-muted uppercase text-xs tracking-wide">
                    <th class="px-4 py-2">Member</th>
                    <th class="px-4 py-2">Email</th>
                    <th class="px-4 py-2">Role</th>
                    <th class="px-4 py-2">Status</th>
                    <th class="px-4 py-2 w-64">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="member in members"
                    :key="member.id"
                    class="border-t border-border-muted/40"
                  >
                    <td class="px-4 py-3">
                      <OuiFlex gap="sm" align="center">
                        <OuiAvatar
                          :name="
                            member.user?.name ||
                            member.user?.email ||
                            member.user?.id
                          "
                          :src="member.user?.avatarUrl"
                        />
                        <div>
                          <OuiText weight="medium">
                            {{
                              member.user?.name ||
                              member.user?.email ||
                              member.user?.id
                            }}
                          </OuiText>
                          <OuiText color="muted" size="xs">
                            {{
                              member.user?.preferredUsername || member.user?.id
                            }}
                          </OuiText>
                        </div>
                      </OuiFlex>
                    </td>
                    <td class="px-4 py-3 text-text-secondary">
                      {{ member.user?.email || "—" }}
                    </td>
                    <td class="px-4 py-3">
                      <OuiSelect
                        :model-value="member.role"
                        :items="roleItems"
                        :disabled="
                          member.role === 'owner' && currentUserIsOwner
                        "
                        @update:model-value="(r) => setRole(member.id, r as string)"
                      />
                    </td>
                    <td class="px-4 py-3">
                      <OuiBadge
                        :tone="member.status === 'active' ? 'solid' : 'soft'"
                        :variant="
                          member.status === 'active' ? 'success' : 'secondary'
                        "
                      >
                        {{ member.status }}
                      </OuiBadge>
                    </td>
                    <td class="px-4 py-3">
                      <OuiFlex gap="sm">
                        <OuiButton
                          v-if="
                            currentUserIsOwner &&
                            member.status === 'active' &&
                            member.role !== 'owner'
                          "
                          size="sm"
                          variant="ghost"
                          @click="openTransferDialog(member)"
                        >
                          Transfer Ownership
                        </OuiButton>
                        <OuiButton
                          v-if="currentUserIsOwner && member.role !== 'owner'"
                          size="sm"
                          variant="ghost"
                          color="danger"
                          @click="remove(member.id)"
                        >
                          {{ member.status === 'invited' ? 'Uninvite' : 'Remove' }}
                        </OuiButton>
                      </OuiFlex>
                    </td>
                  </tr>
                  <tr v-if="!members.length">
                    <td
                      colspan="5"
                      class="px-4 py-6 text-center text-text-muted"
                    >
                      No members yet. Invite someone to get started.
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <OuiStack gap="md" class="mt-6">
              <OuiText size="md" weight="semibold">Invite Member</OuiText>
              <OuiGrid cols="1" colsLg="3" gap="md">
                <OuiInput
                  label="Email"
                  v-model="inviteEmail"
                  placeholder="user@example.com"
                />
                <OuiSelect
                  label="Role"
                  v-model="inviteRole"
                  :items="roleItems"
                />
                <OuiFlex align="end">
                  <OuiButton @click="invite" :disabled="inviteDisabled">
                    Invite
                  </OuiButton>
                </OuiFlex>
              </OuiGrid>
            </OuiStack>
          </template>

          <template #roles>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Roles</OuiText>
              <OuiText color="muted" size="sm">
                System roles appear first. Any custom roles you create follow
                below.
              </OuiText>
              <OuiGrid cols="1" colsLg="2" gap="md">
                <OuiCard v-for="item in roleItems" :key="item.value">
                  <OuiCardBody>
                    <OuiStack gap="xs">
                      <OuiFlex align="center" justify="between">
                        <OuiText weight="medium">{{ item.label }}</OuiText>
                        <CheckIcon
                          v-if="item.disabled"
                          class="h-4 w-4 text-text-muted"
                        />
                      </OuiFlex>
                      <OuiText color="muted" size="sm">
                        {{
                          item.disabled ? "Reserved system role" : "Assignable"
                        }}
                      </OuiText>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </OuiGrid>
            </OuiStack>
          </template>

          <template #billing>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Billing Overview</OuiText>
              <OuiText color="muted" size="sm">
                Billing analytics coming soon.
              </OuiText>
              <OuiCard>
                <OuiCardBody>
                  <OuiSkeleton height="120px" />
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
          </template>
        </OuiTabs>
      </OuiCardBody>
    </OuiCard>

    <OuiDialog v-model:open="transferDialogOpen" title="Transfer Ownership">
      <p class="text-sm text-text-muted">
        {{ transferDialogSummary }}
      </p>
      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="transferDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton
            color="primary"
            @click="confirmTransferOwnership"
            :disabled="!transferCandidate"
          >
            Confirm transfer
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiStack>
</template>
