<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">
      Organization Quotas (0 = unlimited)
    </OuiText>
    <OuiCard>
      <OuiCardBody>
        <form @submit.prevent="save">
          <OuiGrid cols="1" colsMd="2" gap="md">
            <OuiSelect
              label="Organization"
              v-model="selectedOrg"
              :items="orgItems"
            />
            <OuiInput
              label="Deployments Max"
              v-model="form.deployments_max_override"
              type="number"
            />

            <OuiInput
              label="CPU Cores"
              v-model="form.cpu_cores_override"
              type="number"
            />

            <OuiInput
              label="Memory Bytes"
              v-model="form.memory_bytes_override"
              type="number"
            />

            <OuiInput
              label="Bandwidth Bytes / Month"
              v-model="form.bandwidth_bytes_month_override"
              type="number"
            />

            <OuiInput
              label="Storage Bytes"
              v-model="form.storage_bytes_override"
              type="number"
            />
          </OuiGrid>
          <OuiFlex class="mt-4" justify="start" gap="md">
            <OuiButton type="submit">Save</OuiButton>
            <OuiText v-if="message" color="success">{{ message }}</OuiText>
            <OuiText v-if="error" color="danger">{{ error }}</OuiText>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";
import { OrganizationService, AdminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({ layout: "admin", middleware: "auth" });

const form = reactive<{ [k: string]: string }>({
  deployments_max_override: "0",
  cpu_cores_override: "0",
  memory_bytes_override: "0",
  bandwidth_bytes_month_override: "0",
  storage_bytes_override: "0",
});
const message = ref("");
const error = ref("");

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const orgClient = useConnectClient(OrganizationService);
const adminClient = useConnectClient(AdminService);

if (!orgs.value.length) {
  try {
    const res = await orgClient.listOrganizations({});
    orgStore.setOrganizations(res.organizations || []);
  } catch (e) {
    console.error("Failed to load organizations", e);
  }
}

const orgItems = computed(() =>
  (orgs.value || []).map((o) => ({
    label: o.name ?? o.id,
    value: o.id,
  }))
);
const selectedOrg = computed({
  get: () => currentOrgId.value || "",
  set: (id: string) => {
    if (id) orgStore.switchOrganization(id);
  },
});

async function save() {
  message.value = "";
  error.value = "";
  try {
    await adminClient.upsertOrgQuota({
      organizationId: selectedOrg.value,
      deploymentsMaxOverride: Number(form.deployments_max_override) || 0,
      cpuCoresOverride: Number(form.cpu_cores_override) || 0,
      memoryBytesOverride: BigInt(form.memory_bytes_override || "0"),
      bandwidthBytesMonthOverride: BigInt(
        form.bandwidth_bytes_month_override || "0"
      ),
      storageBytesOverride: BigInt(form.storage_bytes_override || "0"),
    });
    message.value = "Saved";
  } catch (e: any) {
    error.value = e?.message || "Error";
  }
}
</script>
