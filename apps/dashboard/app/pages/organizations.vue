<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  import {
    OrganizationService,
    AdminService,
    type OrganizationMember,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import {
    CheckIcon,
    PlusIcon,
    CreditCardIcon,
    PencilIcon,
    ArrowDownTrayIcon,
  } from "@heroicons/vue/24/outline";

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
  
  async function addCredits() {
    if (!selectedOrg.value || !addCreditsAmount.value) return;
    const amount = parseFloat(addCreditsAmount.value);
    if (isNaN(amount) || amount <= 0) {
      error.value = "Please enter a valid positive amount";
      return;
    }
    
    addCreditsLoading.value = true;
    error.value = "";
    try {
      // Note: Proto files need to be generated
      await (orgClient as any).addCredits({
        organizationId: selectedOrg.value,
        amountCents: Math.round(amount * 100), // Convert dollars to cents
        note: addCreditsNote.value || undefined,
      });
      addCreditsDialogOpen.value = false;
      addCreditsAmount.value = "";
      addCreditsNote.value = "";
      await syncOrganizations(); // Refresh organizations to get updated credits
    } catch (err: any) {
      error.value = err.message || "Failed to add credits";
    } finally {
      addCreditsLoading.value = false;
    }
  }

  const memberStats = computed(() => {
    const activeMembers = members.value.filter((m) => m.status === "active");
    const pendingInvites = members.value.filter((m) => m.status === "invited");
    return {
      total: activeMembers.length,
      owners: activeMembers.filter((m) => m.role === "owner").length,
      admins: activeMembers.filter((m) => m.role === "admin").length,
      members: activeMembers.filter((m) => m.role === "member").length,
      viewers: activeMembers.filter((m) => m.role === "viewer").length,
      pending: pendingInvites.length,
    };
  });

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

  const currentMonth = computed(() => {
    const now = new Date();
    return now.toLocaleString("default", { month: "long", year: "numeric" });
  });

  // Fetch usage data
  const { data: usageData, refresh: refreshUsage } = await useAsyncData(
    () =>
      selectedOrg.value
        ? `org-usage-${selectedOrg.value}`
        : "org-usage-none",
    async () => {
      if (!selectedOrg.value) return null;
      try {
        // Connect RPC returns the response message directly
        const res = await orgClient.getUsage({
          organizationId: selectedOrg.value,
        });
        // Connect returns the message directly (not wrapped in .msg)
        // The response is GetUsageResponse with properties: organizationId, month, current, estimatedMonthly, quota
        return res;
      } catch (err) {
        console.error("Failed to fetch usage:", err);
        // Return null to show loading state, but log the error
        return null;
      }
    },
    { watch: [selectedOrg], server: false }
  );

  const usage = computed(() => usageData.value);
  
  // Get current organization object to access credits
  const currentOrganization = computed(() => {
    if (!selectedOrg.value) return null;
    return organizations.value.find((o) => o.id === selectedOrg.value) || null;
  });
  
  const creditsBalance = computed(() => {
    const credits = currentOrganization.value?.credits;
    if (credits === undefined || credits === null) return 0;
    // Handle both bigint (from proto) and number types
    return typeof credits === 'bigint' ? Number(credits) : credits;
  });
  
  const addCreditsDialogOpen = ref(false);
  const addCreditsAmount = ref("");
  const addCreditsNote = ref("");
  const addCreditsLoading = ref(false);
  
  // Format helper functions
  const formatBytes = (bytes: number | bigint) => {
    const b = Number(bytes);
    if (b === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return `${(b / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatBytesToGB = (bytes: number | bigint) => {
    const b = Number(bytes);
    if (b === 0) return "0.00";
    return (b / (1024 * 1024 * 1024)).toFixed(2);
  };

  const formatMemoryByteSecondsToGB = (byteSeconds: number | bigint) => {
    const bs = Number(byteSeconds);
    if (bs === 0 || !Number.isFinite(bs)) return "0.00";
    // Convert byte-seconds to GB-hours, then show as GB (for average memory usage)
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
    const secondsInMonth = Math.max(1, Math.floor((now.getTime() - monthStart.getTime()) / 1000));
    const avgBytes = bs / secondsInMonth;
    return formatBytesToGB(avgBytes);
  };

  const formatCoreSecondsToHours = (coreSeconds: number | bigint) => {
    const s = Number(coreSeconds);
    if (!Number.isFinite(s) || s === 0) return "0.00";
    return (s / 3600).toFixed(2);
  };

  const formatCurrency = (cents: number | bigint) => {
    const c = Number(cents);
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
    }).format(c / 100);
  };

  const getUsagePercentage = (
    current: number | bigint,
    quota: number | bigint
  ) => {
    const c = Number(current);
    const q = Number(quota);
    if (q === 0 || !Number.isFinite(q)) return 0; // Unlimited quota
    if (!Number.isFinite(c) || c === 0) return 0;
    return Math.min(100, Math.max(0, Math.round((c / q) * 100)));
  };

  const getUsageBadgeVariant = (percentage: number) => {
    if (percentage >= 90) return "danger";
    if (percentage >= 75) return "warning";
    return "success";
  };

  const billingHistory = [
    {
      id: "1",
      number: "#INV-2024-001",
      date: "Jan 1, 2024",
      amount: "$28.50",
      status: "Paid",
    },
    {
      id: "2",
      number: "#INV-2023-012",
      date: "Dec 1, 2023",
      amount: "$31.75",
      status: "Paid",
    },
    {
      id: "3",
      number: "#INV-2023-011",
      date: "Nov 1, 2023",
      amount: "$24.20",
      status: "Paid",
    },
  ];
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
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText size="2xl" weight="semibold">{{
                memberStats.total
              }}</OuiText>
              <OuiText color="muted" size="sm">Active members</OuiText>
            </OuiStack>
            <OuiGrid cols="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.owners }}</OuiText>
                <OuiText color="muted" size="sm">Owners</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.admins }}</OuiText>
                <OuiText color="muted" size="sm">Admins</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.members }}</OuiText>
                <OuiText color="muted" size="sm">Members</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.viewers }}</OuiText>
                <OuiText color="muted" size="sm">Viewers</OuiText>
              </OuiStack>
            </OuiGrid>
            <OuiStack gap="xs">
              <OuiText size="lg" weight="semibold">{{ memberStats.pending }}</OuiText>
              <OuiText color="muted" size="sm">Pending invites</OuiText>
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
            <OuiStack gap="xl">
              <template v-if="!usage || !usage.current">
                <OuiStack gap="md" align="center" class="py-12">
                  <OuiText color="muted">Loading usage data...</OuiText>
                  <OuiSkeleton width="100%" height="200px" />
                </OuiStack>
              </template>
              <template v-else>
                <!-- Current Usage -->
                <OuiStack gap="lg">
                  <OuiText size="2xl" weight="bold">Current Usage</OuiText>
                  <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="lg">
                    <!-- vCPU Usage -->
                    <OuiCard>
                      <OuiCardBody>
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="start">
                            <OuiStack gap="xs">
                              <OuiText size="sm" color="muted">vCPU Hours</OuiText>
                              <OuiText size="2xl" weight="bold">
                                {{ formatCoreSecondsToHours(usage.current.cpuCoreSeconds) }}
                              </OuiText>
                            </OuiStack>
                            <OuiBadge 
                              :variant="getUsageBadgeVariant(
                                getUsagePercentage(
                                  usage.current.cpuCoreSeconds,
                                  usage.quota?.cpuCoreSecondsMonthly || 0
                                )
                              )"
                            >
                              Active
                            </OuiBadge>
                          </OuiFlex>
                          <OuiProgress 
                            :value="getUsagePercentage(
                              usage.current.cpuCoreSeconds,
                              usage.quota?.cpuCoreSecondsMonthly || 0
                            )" 
                            :max="100" 
                          />
                          <OuiText size="sm" color="muted">
                            <template v-if="Number(usage.quota?.cpuCoreSecondsMonthly || 0) === 0">
                              Unlimited allocation
                            </template>
                            <template v-else>
                              {{ getUsagePercentage(
                                usage.current.cpuCoreSeconds,
                                usage.quota?.cpuCoreSecondsMonthly || 0
                              ) }}% of monthly allocation
                              ({{ formatCoreSecondsToHours(usage.quota?.cpuCoreSecondsMonthly || 0) }} hours)
                            </template>
                          </OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>

                    <!-- Memory Usage -->
                    <OuiCard>
                      <OuiCardBody>
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="start">
                            <OuiStack gap="xs">
                              <OuiText size="sm" color="muted">Memory (GB avg)</OuiText>
                              <OuiText size="2xl" weight="bold">
                                {{ formatMemoryByteSecondsToGB(usage.current.memoryByteSeconds) }}
                              </OuiText>
                            </OuiStack>
                            <OuiBadge 
                              :variant="getUsageBadgeVariant(
                                getUsagePercentage(
                                  usage.current.memoryByteSeconds,
                                  usage.quota?.memoryByteSecondsMonthly || 0
                                )
                              )"
                            >
                              <template v-if="getUsagePercentage(
                                usage.current.memoryByteSeconds,
                                usage.quota?.memoryByteSecondsMonthly || 0
                              ) >= 90">High</template>
                              <template v-else-if="getUsagePercentage(
                                usage.current.memoryByteSeconds,
                                usage.quota?.memoryByteSecondsMonthly || 0
                              ) >= 75">Warning</template>
                              <template v-else>Normal</template>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiProgress 
                            :value="getUsagePercentage(
                              usage.current.memoryByteSeconds,
                              usage.quota?.memoryByteSecondsMonthly || 0
                            )" 
                            :max="100" 
                          />
                          <OuiText size="sm" color="muted">
                            <template v-if="Number(usage.quota?.memoryByteSecondsMonthly || 0) === 0">
                              Unlimited allocation
                            </template>
                            <template v-else>
                              {{ getUsagePercentage(
                                usage.current.memoryByteSeconds,
                                usage.quota?.memoryByteSecondsMonthly || 0
                              ) }}% of monthly allocation
                              ({{ formatMemoryByteSecondsToGB(usage.quota?.memoryByteSecondsMonthly || 0) }} GB)
                            </template>
                          </OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>

                    <!-- Storage Usage -->
                    <OuiCard>
                      <OuiCardBody>
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="start">
                            <OuiStack gap="xs">
                              <OuiText size="sm" color="muted">Storage (GB)</OuiText>
                              <OuiText size="2xl" weight="bold">
                                {{ formatBytesToGB(usage.current.storageBytes) }}
                              </OuiText>
                            </OuiStack>
                            <OuiBadge 
                              :variant="getUsageBadgeVariant(
                                getUsagePercentage(
                                  usage.current.storageBytes,
                                  usage.quota?.storageBytes || 0
                                )
                              )"
                            >
                              <template v-if="getUsagePercentage(
                                usage.current.storageBytes,
                                usage.quota?.storageBytes || 0
                              ) >= 90">High</template>
                              <template v-else-if="getUsagePercentage(
                                usage.current.storageBytes,
                                usage.quota?.storageBytes || 0
                              ) >= 75">Warning</template>
                              <template v-else>Normal</template>
                            </OuiBadge>
                          </OuiFlex>
                          <OuiProgress 
                            :value="getUsagePercentage(
                              usage.current.storageBytes,
                              usage.quota?.storageBytes || 0
                            )" 
                            :max="100" 
                          />
                          <OuiText size="sm" color="muted">
                            <template v-if="Number(usage.quota?.storageBytes || 0) === 0">
                              Unlimited allocation
                            </template>
                            <template v-else>
                              {{ getUsagePercentage(
                                usage.current.storageBytes,
                                usage.quota?.storageBytes || 0
                              ) }}% of monthly allocation
                              ({{ formatBytesToGB(usage.quota?.storageBytes || 0) }} GB)
                            </template>
                          </OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                  </OuiGrid>

                  <!-- Credits Balance -->
                  <OuiCard variant="outline">
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">Credits Balance</OuiText>
                            <OuiText size="3xl" weight="bold">
                              {{ formatCurrency(creditsBalance) }}
                            </OuiText>
                            <OuiText size="sm" color="muted">
                              Available credits for your organization
                            </OuiText>
                          </OuiStack>
                          <OuiButton 
                            variant="solid" 
                            size="sm" 
                            @click="addCreditsDialogOpen = true"
                            v-if="currentUserIsOwner"
                          >
                            <PlusIcon class="h-4 w-4 mr-2" />
                            Add Credits
                          </OuiButton>
                        </OuiFlex>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Current Month Summary -->
                  <OuiCard variant="outline">
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <OuiText size="xl" weight="semibold">Current Month Estimate</OuiText>
                        <OuiFlex justify="between" align="center">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">{{ currentMonth }}</OuiText>
                            <OuiText size="3xl" weight="bold">
                              {{ usage.estimatedMonthly?.estimatedCostCents 
                                ? formatCurrency(usage.estimatedMonthly.estimatedCostCents)
                                : '$0.00' }}
                            </OuiText>
                            <OuiText size="sm" color="muted">
                              Based on current usage patterns
                            </OuiText>
                          </OuiStack>
                          <OuiButton variant="outline" size="sm" @click="refreshUsage">
                            Refresh
                          </OuiButton>
                        </OuiFlex>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiStack>
              </template>

              <!-- Payment Methods -->
              <OuiStack gap="lg">
                <OuiFlex justify="between" align="center">
                  <OuiText size="2xl" weight="bold">Payment Methods</OuiText>
                  <OuiButton variant="solid" size="sm">
                    <PlusIcon class="h-4 w-4 mr-2" />
                    Add Payment Method
                  </OuiButton>
                </OuiFlex>

                <OuiGrid cols="1" cols-lg="2" gap="lg">
                  <!-- Primary Payment Method -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="sm">
                            <OuiFlex align="center" gap="sm">
                              <CreditCardIcon class="h-6 w-6 text-accent-primary" />
                              <OuiText size="lg" weight="semibold">•••• •••• •••• 4242</OuiText>
                              <OuiBadge variant="success">Primary</OuiBadge>
                            </OuiFlex>
                            <OuiText size="sm" color="muted">Visa • Expires 12/26</OuiText>
                          </OuiStack>
                          <OuiButton variant="ghost" size="sm">
                            <PencilIcon class="h-4 w-4" />
                          </OuiButton>
                        </OuiFlex>
                        <OuiFlex gap="sm">
                          <OuiButton variant="outline" size="sm">Update</OuiButton>
                          <OuiButton variant="ghost" size="sm">Remove</OuiButton>
                        </OuiFlex>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Backup Payment Method -->
                  <OuiCard variant="outline" class="border-dashed">
                    <OuiCardBody>
                      <OuiStack gap="lg" align="center">
                        <OuiStack gap="sm" align="center">
                          <CreditCardIcon class="h-12 w-12 text-muted" />
                          <OuiText size="lg" weight="semibold">Add Backup Payment</OuiText>
                          <OuiText size="sm" color="muted">
                            Ensure uninterrupted service with a backup payment method
                          </OuiText>
                        </OuiStack>
                        <OuiButton variant="outline" size="sm">Add Card</OuiButton>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiGrid>
              </OuiStack>

              <!-- Billing History -->
              <OuiStack gap="lg">
                <OuiFlex justify="between" align="center">
                  <OuiText size="2xl" weight="bold">Billing History</OuiText>
                  <OuiButton variant="outline" size="sm">
                    <ArrowDownTrayIcon class="h-4 w-4 mr-2" />
                    Download All
                  </OuiButton>
                </OuiFlex>

                <OuiCard>
                  <OuiCardBody class="p-0">
                    <OuiStack>
                      <!-- Table Header -->
                      <OuiBox p="md" borderBottom="1" borderColor="muted">
                        <OuiGrid cols="5" gap="md">
                          <OuiText size="sm" weight="medium" color="muted">Invoice</OuiText>
                          <OuiText size="sm" weight="medium" color="muted">Date</OuiText>
                          <OuiText size="sm" weight="medium" color="muted">Amount</OuiText>
                          <OuiText size="sm" weight="medium" color="muted">Status</OuiText>
                          <OuiText size="sm" weight="medium" color="muted">Actions</OuiText>
                        </OuiGrid>
                      </OuiBox>

                      <!-- Table Rows -->
                      <OuiStack>
                        <OuiBox
                          v-for="invoice in billingHistory"
                          :key="invoice.id"
                          p="md"
                          borderBottom="1"
                          borderColor="muted"
                        >
                          <OuiGrid cols="5" gap="md" align="center">
                            <OuiText size="sm" weight="medium">{{ invoice.number }}</OuiText>
                            <OuiText size="sm" color="muted">{{ invoice.date }}</OuiText>
                            <OuiText size="sm" weight="medium">{{ invoice.amount }}</OuiText>
                            <OuiBadge variant="success">{{ invoice.status }}</OuiBadge>
                            <OuiFlex gap="sm">
                              <OuiButton variant="ghost" size="xs">View</OuiButton>
                              <OuiButton variant="ghost" size="xs">Download</OuiButton>
                            </OuiFlex>
                          </OuiGrid>
                        </OuiBox>
                      </OuiStack>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </OuiStack>
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

    <!-- Add Credits Dialog -->
    <OuiDialog v-model:open="addCreditsDialogOpen" title="Add Credits">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Add credits to your organization balance. Credits are stored in cents ($0.01 units).
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>
        
        <OuiStack gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Amount (USD)</OuiText>
            <OuiInput
              v-model="addCreditsAmount"
              type="number"
              step="0.01"
              min="0.01"
              placeholder="0.00"
            />
          </OuiStack>
          
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Note (Optional)</OuiText>
            <OuiInput
              v-model="addCreditsNote"
              type="text"
              placeholder="Reason for adding credits"
            />
          </OuiStack>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="addCreditsDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="addCredits"
            :disabled="addCreditsLoading || !addCreditsAmount || parseFloat(addCreditsAmount) <= 0"
          >
            {{ addCreditsLoading ? "Adding..." : "Add Credits" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>
