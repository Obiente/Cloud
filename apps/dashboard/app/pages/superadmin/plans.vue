<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Plans</OuiText>
        <OuiText color="muted">Manage resource limitation plans for organizations.</OuiText>
      </OuiStack>
      <OuiButton @click="openCreateDialog">
        <PlusIcon class="h-4 w-4" />
        Create Plan
      </OuiButton>
    </OuiFlex>

    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="isLoading ? 'Loading plans…' : 'No plans found. Create your first plan to get started.'"
        >
          <template #cell-name="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ value }}</div>
              <div v-if="row.description" class="text-xs text-text-muted mt-0.5">{{ row.description }}</div>
            </div>
          </template>
          <template #cell-resources="{ row }">
            <OuiStack gap="xs">
              <div class="text-sm">
                <span class="text-text-secondary">CPU:</span>
                <span class="font-mono ml-1">{{ row.cpuCores || 0 }} cores</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Memory:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.memoryBytes || 0)) }}</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Deployments:</span>
                <span class="font-mono ml-1">{{ row.deploymentsMax || 0 }} max</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Bandwidth:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.bandwidthBytesMonth || 0)) }}/mo</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Storage:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.storageBytes || 0)) }}</span>
              </div>
            </OuiStack>
          </template>
          <template #cell-minimumPaymentCents="{ value }">
            <span class="font-mono">
              <OuiCurrency :value="typeof value === 'bigint' ? Number(value) : (Number(value) || 0)" />
            </span>
          </template>
          <template #cell-monthlyFreeCreditsCents="{ value }">
            <span class="font-mono">
              <OuiCurrency :value="typeof value === 'bigint' ? Number(value) : (Number(value) || 0)" />
            </span>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="sm" justify="end">
              <OuiButton size="xs" variant="ghost" @click="openEditDialog(row)">
                Edit
              </OuiButton>
              <OuiButton size="xs" variant="ghost" color="danger" @click="openDeleteDialog(row)">
                Delete
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Create/Edit Plan Dialog -->
    <OuiDialog v-model:open="planDialogOpen" :title="editingPlan ? 'Edit Plan' : 'Create Plan'">
      <OuiStack gap="lg">
        <OuiStack gap="md">
          <OuiField label="Plan Name" required>
            <OuiInput v-model="planForm.name" placeholder="e.g., Starter, Pro, Enterprise" />
          </OuiField>
          
          <OuiField label="Description">
            <OuiTextarea v-model="planForm.description" placeholder="Optional description of the plan" :rows="3" />
          </OuiField>

          <OuiField label="Minimum Payment (USD)" required>
            <OuiInput
              v-model="planForm.minimumPaymentCents"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
            />
            <OuiText size="xs" color="muted" class="mt-1">
              Organizations automatically upgrade to this plan when they pay this amount or more (in cents).
            </OuiText>
          </OuiField>

          <OuiField label="Monthly Free Credits (USD)" required>
            <OuiInput
              v-model="planForm.monthlyFreeCreditsCents"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
            />
            <OuiText size="xs" color="muted" class="mt-1">
              Monthly free credits (in cents) automatically granted to organizations on this plan. Set to 0 for no free credits.
            </OuiText>
          </OuiField>
        </OuiStack>

        <OuiDivider />

        <OuiStack gap="md">
          <OuiText size="sm" weight="medium">Resource Limits</OuiText>
          
          <OuiField label="CPU Cores" required>
            <OuiInput v-model="planForm.cpuCores" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">0 = unlimited</OuiText>
          </OuiField>

          <OuiField label="Memory (bytes)" required>
            <OuiInput v-model="planForm.memoryBytes" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">0 = unlimited</OuiText>
          </OuiField>

          <OuiField label="Max Deployments" required>
            <OuiInput v-model="planForm.deploymentsMax" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">0 = unlimited</OuiText>
          </OuiField>

          <OuiField label="Bandwidth per Month (bytes)" required>
            <OuiInput v-model="planForm.bandwidthBytesMonth" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">0 = unlimited</OuiText>
          </OuiField>

          <OuiField label="Storage (bytes)" required>
            <OuiInput v-model="planForm.storageBytes" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">0 = unlimited</OuiText>
          </OuiField>
        </OuiStack>
      </OuiStack>

      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="closePlanDialog">Cancel</OuiButton>
          <OuiButton @click="savePlan" :disabled="isSaving">
            {{ isSaving ? 'Saving...' : (editingPlan ? 'Update' : 'Create') }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Delete Confirmation Dialog -->
    <OuiDialog v-model:open="deleteDialogOpen" title="Delete Plan">
      <OuiStack gap="md">
        <OuiText>
          Are you sure you want to delete the plan <strong>{{ planToDelete?.name }}</strong>?
        </OuiText>
        <OuiText size="sm" color="danger">
          This action cannot be undone. If any organizations are using this plan, deletion will be blocked.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="deleteDialogOpen = false">Cancel</OuiButton>
          <OuiButton color="danger" @click="confirmDelete" :disabled="isDeleting">
            {{ isDeleting ? 'Deleting...' : 'Delete' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { PlusIcon } from "@heroicons/vue/24/outline";
import { computed, ref, onMounted } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService } from "@obiente/proto";
import { useToast } from "~/composables/useToast";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const { toast } = useToast();
const client = useConnectClient(SuperadminService);

const plans = ref<any[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);
const isDeleting = ref(false);
const planDialogOpen = ref(false);
const deleteDialogOpen = ref(false);
const editingPlan = ref<any>(null);
const planToDelete = ref<any>(null);

const planForm = ref({
  name: "",
  description: "",
  cpuCores: "",
  memoryBytes: "",
  deploymentsMax: "",
  bandwidthBytesMonth: "",
  storageBytes: "",
  minimumPaymentCents: "",
  monthlyFreeCreditsCents: "",
});

const tableColumns = computed(() => [
  { key: "name", label: "Plan Name", defaultWidth: 200, minWidth: 150 },
  { key: "resources", label: "Resource Limits", defaultWidth: 300, minWidth: 250 },
  { key: "minimumPaymentCents", label: "Min Payment", defaultWidth: 150, minWidth: 120 },
  { key: "monthlyFreeCreditsCents", label: "Monthly Free Credits", defaultWidth: 180, minWidth: 150 },
  { key: "actions", label: "Actions", defaultWidth: 150, minWidth: 120, resizable: false },
]);

const tableRows = computed(() => plans.value);

const { formatBytes, formatCurrency } = useUtils();

const fetchPlans = async () => {
  isLoading.value = true;
  try {
    const response = await client.listPlans({});
    plans.value = response.plans || [];
  } catch (error: any) {
    toast.error(`Failed to load plans: ${error?.message || "Unknown error"}`);
  } finally {
    isLoading.value = false;
  }
};

const openCreateDialog = () => {
  editingPlan.value = null;
  planForm.value = {
    name: "",
    description: "",
    cpuCores: "",
    memoryBytes: "",
    deploymentsMax: "",
    bandwidthBytesMonth: "",
    storageBytes: "",
    minimumPaymentCents: "",
    monthlyFreeCreditsCents: "",
  };
  planDialogOpen.value = true;
};

const openEditDialog = (plan: any) => {
  editingPlan.value = plan;
  planForm.value = {
    name: plan.name || "",
    description: plan.description || "",
    cpuCores: plan.cpuCores != null ? String(plan.cpuCores) : "",
    memoryBytes: plan.memoryBytes != null ? String(plan.memoryBytes) : "",
    deploymentsMax: plan.deploymentsMax != null ? String(plan.deploymentsMax) : "",
    bandwidthBytesMonth: plan.bandwidthBytesMonth != null ? String(plan.bandwidthBytesMonth) : "",
    storageBytes: plan.storageBytes != null ? String(plan.storageBytes) : "",
    minimumPaymentCents: plan.minimumPaymentCents != null ? String(Number(plan.minimumPaymentCents) / 100) : "", // Convert to dollars for input
    monthlyFreeCreditsCents: plan.monthlyFreeCreditsCents != null ? String(Number(plan.monthlyFreeCreditsCents) / 100) : "", // Convert to dollars for input
  };
  planDialogOpen.value = true;
};

const closePlanDialog = () => {
  planDialogOpen.value = false;
  editingPlan.value = null;
};

const savePlan = async () => {
  if (!planForm.value.name) {
    toast.error("Plan name is required");
    return;
  }

  isSaving.value = true;
  try {
    // Convert string inputs to numbers
    const cpuCores = Number(planForm.value.cpuCores) || 0;
    const memoryBytes = Number(planForm.value.memoryBytes) || 0;
    const deploymentsMax = Number(planForm.value.deploymentsMax) || 0;
    const bandwidthBytesMonth = Number(planForm.value.bandwidthBytesMonth) || 0;
    const storageBytes = Number(planForm.value.storageBytes) || 0;
    const minimumPaymentCents = Math.round((Number(planForm.value.minimumPaymentCents) || 0) * 100);
    const monthlyFreeCreditsCents = Math.round((Number(planForm.value.monthlyFreeCreditsCents) || 0) * 100);

    if (editingPlan.value) {
      const updateRequest: any = {
        id: editingPlan.value.id,
      };
      if (planForm.value.name) updateRequest.name = planForm.value.name;
      if (planForm.value.description !== undefined) updateRequest.description = planForm.value.description;
      if (cpuCores !== undefined) updateRequest.cpuCores = cpuCores;
      if (memoryBytes !== undefined) updateRequest.memoryBytes = BigInt(memoryBytes);
      if (deploymentsMax !== undefined) updateRequest.deploymentsMax = deploymentsMax;
      if (bandwidthBytesMonth !== undefined) updateRequest.bandwidthBytesMonth = BigInt(bandwidthBytesMonth);
      if (storageBytes !== undefined) updateRequest.storageBytes = BigInt(storageBytes);
      if (minimumPaymentCents !== undefined) updateRequest.minimumPaymentCents = BigInt(minimumPaymentCents);
      if (monthlyFreeCreditsCents !== undefined) updateRequest.monthlyFreeCreditsCents = BigInt(monthlyFreeCreditsCents);

      await client.updatePlan(updateRequest);
      toast.success("Plan updated successfully");
    } else {
      await client.createPlan({
        name: planForm.value.name,
        description: planForm.value.description || "",
        cpuCores: cpuCores,
        memoryBytes: BigInt(memoryBytes),
        deploymentsMax: deploymentsMax,
        bandwidthBytesMonth: BigInt(bandwidthBytesMonth),
        storageBytes: BigInt(storageBytes),
        minimumPaymentCents: BigInt(minimumPaymentCents),
        monthlyFreeCreditsCents: BigInt(monthlyFreeCreditsCents),
      });
      toast.success("Plan created successfully");
    }
    closePlanDialog();
    await fetchPlans();
  } catch (error: any) {
    toast.error(`Failed to save plan: ${error?.message || "Unknown error"}`);
  } finally {
    isSaving.value = false;
  }
};

const openDeleteDialog = (plan: any) => {
  planToDelete.value = plan;
  deleteDialogOpen.value = true;
};

const confirmDelete = async () => {
  if (!planToDelete.value) return;

  isDeleting.value = true;
  try {
    await client.deletePlan({ id: planToDelete.value.id });
    toast.success("Plan deleted successfully");
    deleteDialogOpen.value = false;
    planToDelete.value = null;
    await fetchPlans();
  } catch (error: any) {
    toast.error(`Failed to delete plan: ${error?.message || "Unknown error"}`);
  } finally {
    isDeleting.value = false;
  }
};

onMounted(() => {
  fetchPlans();
});
</script>

