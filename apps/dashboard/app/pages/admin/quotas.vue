<template>
  <OuiStack gap="md">
    <OuiText as="h1" size="xl" weight="semibold">
      Organization Quotas
    </OuiText>
    <OuiText size="sm" color="secondary">Enter 0 to use the plan default. Overrides are capped by the plan.</OuiText>
    <OuiCard>
      <OuiCardBody>
        <form @submit.prevent="save">
          <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
            <OuiSelect
              label="Organization"
              v-model="selectedOrg"
              :items="orgItems"
            />
            <OuiInput
              label="Deployments Max"
              :helper-text="quotaHints.deployments_max_override"
              v-model="form.deployments_max_override"
              type="number" min="0" step="1"
            />

            <OuiInput
              label="CPU Cores"
              :helper-text="quotaHints.cpu_cores_override"
              v-model="form.cpu_cores_override"
              type="number" min="0" step="1"
            />

            <OuiInput
              label="Memory Bytes"
              :helper-text="quotaHints.memory_bytes_override"
              v-model="form.memory_bytes_override"
              type="number" min="0" step="1"
            />

            <OuiInput
              label="Bandwidth Bytes / Month"
              :helper-text="quotaHints.bandwidth_bytes_month_override"
              v-model="form.bandwidth_bytes_month_override"
              type="number" min="0" step="1"
            />

            <OuiInput
              label="Storage Bytes"
              :helper-text="quotaHints.storage_bytes_override"
              v-model="form.storage_bytes_override"
              type="number" min="0" step="1"
            />
          </OuiGrid>
          <OuiFlex mt="md" justify="start" gap="md" align="center">
            <OuiButton type="submit" :disabled="loading || !loaded" :loading="saving">Save</OuiButton>
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
import { formatBytes } from "~/utils/common";
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
const { currentOrgId } = storeToRefs(orgStore);

const orgClient = useConnectClient(OrganizationService);
const adminClient = useConnectClient(AdminService);

// Share the resolved list through the Nuxt payload so the first client render
// cannot race the layout's organization refresh.
const { data: quotaOrganizations } = await useAsyncData("quota-organizations", async () => {
  const res = await orgClient.listOrganizations({});
  return res.organizations;
});


const orgItems = computed(() =>
  (quotaOrganizations.value || []).map((o) => ({
    label: o.name ?? o.id,
    value: o.id,
  }))
);
// Administrative selection is local to this editor. Populating it must not
// mutate the already-rendered account sidebar or switch the tenant workspace.
const selectedOrg = ref(currentOrgId.value || quotaOrganizations.value?.[0]?.id || "");


const loading = ref(false);
const loaded = ref(false);
const saving = ref(false);
const planLimits = ref<Record<string, string>>({});
const quotaHints = computed(() => Object.fromEntries(Object.entries(planLimits.value).map(([key, raw]) => {
 const plan = BigInt(raw);
 const draft = /^\d+$/.test(form[key] || "") ? BigInt(form[key]!) : 0n;
 const effective = draft === 0n ? plan : plan > 0n && draft > plan ? plan : draft;
 const display = (value: bigint) => value === 0n ? "Unlimited" : key.includes("bytes") ? `${formatBytes(Number(value))} (${value} bytes)` : String(value);
 return [key, `Plan: ${display(plan)}. Effective: ${display(effective)}.`];
})));
let quotaRequest = 0;
async function loadQuota(orgId: string) {
  const request = ++quotaRequest;
  loaded.value = false;
  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    if (!orgId) return;
    const response = await adminClient.getOrgQuota({ organizationId: orgId });
    if (request !== quotaRequest) return;
    const quota = response.quota;
    if (!quota) throw new Error("Quota settings could not be loaded.");
    form.deployments_max_override = String(quota.deploymentsMaxOverride);
    form.cpu_cores_override = String(quota.cpuCoresOverride);
    form.memory_bytes_override = String(quota.memoryBytesOverride);
    form.bandwidth_bytes_month_override = String(quota.bandwidthBytesMonthOverride);
    form.storage_bytes_override = String(quota.storageBytesOverride);
    const defaults = response.planDefaults;
    planLimits.value = defaults ? {
      deployments_max_override: String(defaults.deploymentsMax), cpu_cores_override: String(defaults.cpuCores),
      memory_bytes_override: String(defaults.memoryBytes), bandwidth_bytes_month_override: String(defaults.bandwidthBytesMonth), storage_bytes_override: String(defaults.storageBytes),
    } : {};
    loaded.value = true;
  } catch (err) { if (request === quotaRequest) error.value = "Could not load saved quotas. Try selecting the organization again."; }
  finally { if (request === quotaRequest) loading.value = false; }
}
watch(selectedOrg, loadQuota);
onMounted(() => loadQuota(selectedOrg.value));

async function save() {
  if (!loaded.value || loading.value || saving.value) return;
  if (Object.values(form).some(value => !/^\d+$/.test(value))) { error.value = "Enter whole numbers of zero or more."; return; }
  if (Object.entries(form).some(([key,value]) => BigInt(value) > (key.includes("bytes") ? 9223372036854775807n : 2147483647n))) { error.value = "A quota value exceeds the supported range."; return; }
  const savedOrg = selectedOrg.value;
  saving.value = true;
  message.value = "";
  error.value = "";
  try {
    await adminClient.upsertOrgQuota({
      organizationId: savedOrg,
      deploymentsMaxOverride: Number(form.deployments_max_override) || 0,
      cpuCoresOverride: Number(form.cpu_cores_override) || 0,
      memoryBytesOverride: BigInt(form.memory_bytes_override || "0"),
      bandwidthBytesMonthOverride: BigInt(
        form.bandwidth_bytes_month_override || "0"
      ),
      storageBytesOverride: BigInt(form.storage_bytes_override || "0"),
    });
    if (selectedOrg.value === savedOrg) message.value = "Saved";
  } catch (e: unknown) {
    if (selectedOrg.value === savedOrg) error.value = (e as any)?.message || "Could not save quotas.";
  } finally { saving.value = false; }
}
</script>
