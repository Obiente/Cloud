<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  import {
    OrganizationService,
    AdminService,
    type OrganizationMember,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";

  const name = ref("");
  const slug = ref("");
  const inviteEmail = ref("");
  const inviteRole = ref("");
  const error = ref("");

  const auth = useAuth();
  const orgClient = useConnectClient(OrganizationService);
  const adminClient = useConnectClient(AdminService);

  const organizations = computed(() => auth.organizations || []);
  const selectedOrg = computed({
    get: () => auth.currentOrganizationId,
    set: (id: string) => {
      if (id) auth.switchOrganization(id);
    },
  });

  if (!organizations.value.length && auth.isAuthenticated) {
    const res = await orgClient.listOrganizations({});
    auth.setOrganizations(res.organizations || []);
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
        const roles = res.roles || [];
        if (!inviteRole.value && roles.length) {
          inviteRole.value = roles[0]?.id ?? "";
        }
        return roles.map((r) => ({ id: r.id, name: r.name }));
      },
      { watch: [selectedOrg], server: true }
    );
  const roleItems = computed(() =>
    (roleCatalogData.value || []).map((r) => ({ label: r.name, value: r.id }))
  );

  watch(selectedOrg, () => {
    inviteEmail.value = "";
    inviteRole.value = roleItems.value[0]?.value ?? "";
  });

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
    await orgClient.updateMember({
      organizationId: selectedOrg.value,
      memberId,
      role,
    });
    await refreshMembers();
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
</script>

<template>
  <div class="space-y-6">
    <OuiCard>
      <OuiCardBody>
        <OuiStack gap="xs">
          <OuiText size="lg" weight="semibold">Create Organization</OuiText>
          <OuiText size="sm">No fixed plans. You pay for what you use.</OuiText>
        </OuiStack>
        <form class="mt-4" @submit.prevent="createOrg">
          <OuiGrid cols="1" colsMd="2" gap="md">
            <div>
              <OuiText size="sm" weight="medium">Name</OuiText>
              <OuiInput v-model="name" placeholder="Acme Inc" required />
            </div>
            <div>
              <OuiText size="sm" weight="medium">Slug (optional)</OuiText>
              <OuiInput v-model="slug" placeholder="acme" />
            </div>
          </OuiGrid>
          <OuiFlex class="mt-4" gap="md">
            <OuiButton type="submit">Create</OuiButton>
            <OuiText v-if="error" color="danger">{{ error }}</OuiText>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiFlex align="center" justify="between" class="mb-4">
          <OuiText size="lg" weight="semibold">Organizations</OuiText>
          <OuiButton variant="ghost" @click="refreshAll">Refresh</OuiButton>
        </OuiFlex>
        <OuiGrid cols="1" colsMd="2" gap="md">
          <div>
            <OuiText size="sm" weight="medium">Select Organization</OuiText>
            <OuiSelect
              v-model="selectedOrg"
              :items="
                organizations.map((o) => ({
                  label: o.name ?? o.slug ?? o.id,
                  value: o.id,
                }))
              "
            />
          </div>
        </OuiGrid>

        <OuiStack gap="md">
          <OuiText size="md" weight="medium">Members</OuiText>
          <OuiGrid cols="1" colsMd="3" gap="md">
            <OuiCard v-for="m in members" :key="m.id">
              <OuiCardBody>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">{{
                    m.user?.name || m.user?.email || m.user?.id
                  }}</OuiText>
                  <OuiText size="xs">Role: {{ m.role }}</OuiText>
                  <OuiFlex gap="sm">
                    <OuiSelect
                      :model-value="m.role"
                      :items="roleItems"
                      @update:model-value="(r) => setRole(m.id, r as string)"
                    />
                    <OuiButton
                      variant="ghost"
                      color="danger"
                      @click="remove(m.id)"
                      >Remove</OuiButton
                    >
                  </OuiFlex>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiGrid>
        </OuiStack>

        <div class="mt-6">
          <OuiText size="md" weight="medium">Invite Member</OuiText>
          <OuiGrid cols="1" colsMd="3" gap="md" class="mt-2">
            <OuiInput
              label="Email"
              v-model="inviteEmail"
              placeholder="user@example.com"
            />
            <OuiSelect label="Role" v-model="inviteRole" :items="roleItems" />
            <div class="flex items-end">
              <OuiButton
                @click="invite"
                :disabled="!selectedOrg || !inviteEmail || !inviteRole"
                >Invite</OuiButton
              >
            </div>
          </OuiGrid>
        </div>
      </OuiCardBody>
    </OuiCard>
  </div>
</template>
