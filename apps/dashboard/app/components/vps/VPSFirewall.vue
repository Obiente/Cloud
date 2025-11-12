<template>
  <OuiStack gap="lg">
    <!-- Firewall Options -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiText as="h2" class="oui-card-title">Firewall Settings</OuiText>
          <OuiButton
            v-if="!editingOptions"
            variant="outline"
            size="sm"
            @click="editingOptions = true"
            class="gap-2"
          >
            <PencilIcon class="h-4 w-4" />
            Edit Settings
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack v-if="!editingOptions" gap="sm">
          <OuiFlex justify="between">
            <OuiText color="secondary">Firewall Enabled</OuiText>
            <OuiBadge :color="firewallOptions?.enable ? 'success' : 'secondary'">
              {{ firewallOptions?.enable ? "Enabled" : "Disabled" }}
            </OuiBadge>
          </OuiFlex>
          <OuiFlex justify="between" v-if="firewallOptions?.policyIn">
            <OuiText color="secondary">Default Inbound Policy</OuiText>
            <OuiText weight="medium">{{ firewallOptions.policyIn }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" v-if="firewallOptions?.policyOut">
            <OuiText color="secondary">Default Outbound Policy</OuiText>
            <OuiText weight="medium">{{ firewallOptions.policyOut }}</OuiText>
          </OuiFlex>
        </OuiStack>

        <!-- Edit Options Form -->
        <OuiStack v-else gap="md">
          <OuiCheckbox v-model="optionsForm.enable" label="Enable Firewall" />
          <OuiSelect
            v-model="optionsForm.policyIn"
            label="Default Inbound Policy"
            :items="policyOptions"
            placeholder="Select policy"
          />
          <OuiSelect
            v-model="optionsForm.policyOut"
            label="Default Outbound Policy"
            :items="policyOptions"
            placeholder="Select policy"
          />
          <OuiFlex gap="sm">
            <OuiButton @click="saveOptions" :disabled="savingOptions" class="gap-2">
              <CheckIcon class="h-4 w-4" />
              Save
            </OuiButton>
            <OuiButton variant="outline" @click="cancelEditOptions">
              Cancel
            </OuiButton>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Firewall Rules -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiText as="h2" class="oui-card-title">Firewall Rules</OuiText>
          <OuiButton @click="openCreateDialog" class="gap-2" size="sm">
            <PlusIcon class="h-4 w-4" />
            Add Rule
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <!-- Loading State -->
        <div v-if="loadingRules" class="text-center py-8">
          <OuiSpinner size="lg" />
          <OuiText color="secondary" class="mt-4">Loading firewall rules...</OuiText>
        </div>

        <!-- Error State -->
        <div v-else-if="rulesError" class="text-center py-8">
          <ExclamationCircleIcon class="h-12 w-12 text-danger mx-auto mb-4" />
          <OuiText color="danger" class="mb-2">{{ rulesError }}</OuiText>
          <OuiButton variant="outline" @click="loadRules" class="gap-2">
            <ArrowPathIcon class="h-4 w-4" />
            Retry
          </OuiButton>
        </div>

        <!-- Empty State -->
        <div v-else-if="rules.length === 0" class="text-center py-8">
          <ShieldExclamationIcon class="h-12 w-12 text-secondary mx-auto mb-4" />
          <OuiText color="secondary" class="mb-4">No firewall rules configured</OuiText>
          <OuiButton @click="openCreateDialog" variant="outline" class="gap-2">
            <PlusIcon class="h-4 w-4" />
            Add First Rule
          </OuiButton>
        </div>

        <!-- Rules Table -->
        <OuiTable
          v-else
          :columns="tableColumns"
          :rows="tableRows"
          empty-text="No firewall rules configured"
        >
          <template #cell-action="{ value }">
            <OuiBadge
              :color="
                value === FirewallAction.ACCEPT
                  ? 'success'
                  : value === FirewallAction.REJECT
                    ? 'warning'
                    : 'danger'
              "
              size="sm"
            >
              {{ actionLabel(value) }}
            </OuiBadge>
          </template>
          <template #cell-direction="{ value }">
            <OuiBadge color="secondary" size="sm">
              {{ value === FirewallDirection.IN ? "In" : "Out" }}
            </OuiBadge>
          </template>
          <template #cell-source="{ value }">
            <OuiText size="sm" class="font-mono">{{ value || "—" }}</OuiText>
          </template>
          <template #cell-dest="{ value }">
            <OuiText size="sm" class="font-mono">{{ value || "—" }}</OuiText>
          </template>
          <template #cell-protocol="{ value }">
            <OuiText size="sm">{{ protocolLabel(value) }}</OuiText>
          </template>
          <template #cell-port="{ row }">
            <OuiText size="sm" class="font-mono">
              {{ row.dport || row.sport || "—" }}
            </OuiText>
          </template>
          <template #cell-iface="{ value }">
            <OuiText size="sm">{{ value || "—" }}</OuiText>
          </template>
          <template #cell-comment="{ value }">
            <OuiText size="sm" color="secondary">{{ value || "—" }}</OuiText>
          </template>
          <template #cell-enable="{ value }">
            <OuiBadge :color="value ? 'success' : 'secondary'" size="sm">
              {{ value ? "Enabled" : "Disabled" }}
            </OuiBadge>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="xs">
              <OuiButton
                variant="ghost"
                size="sm"
                @click="openEditDialog(row)"
                class="gap-1"
              >
                <PencilIcon class="h-3 w-3" />
              </OuiButton>
              <OuiButton
                variant="ghost"
                size="sm"
                color="danger"
                @click="handleDeleteRule(row)"
                class="gap-1"
              >
                <TrashIcon class="h-3 w-3" />
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Create/Edit Rule Dialog -->
    <OuiDialog
      :open="dialogOpen"
      :title="editingRule ? 'Edit Firewall Rule' : 'Create Firewall Rule'"
      description="Configure firewall rule to control network traffic for this VPS instance."
      @update:open="dialogOpen = $event"
    >
      <OuiDialogContent size="lg">

        <OuiStack gap="md" class="py-4">
          <OuiCheckbox v-model="ruleForm.enable" label="Enable Rule" />

          <OuiSelect
            v-model="ruleForm.action"
            label="Action"
            :items="actionOptions"
            required
          />

          <OuiSelect
            v-model="ruleForm.type"
            label="Direction"
            :items="directionOptions"
            required
          />

          <OuiInput
            v-model="ruleForm.source"
            label="Source IP/CIDR"
            placeholder="e.g., 192.168.1.0/24 or leave empty for any"
            type="text"
          />

          <OuiInput
            v-model="ruleForm.dest"
            label="Destination IP/CIDR"
            placeholder="e.g., 10.0.0.1 or leave empty for any"
            type="text"
          />

          <OuiSelect
            v-model="ruleForm.protocol"
            label="Protocol"
            :items="protocolOptions"
          />

          <OuiInput
            v-if="ruleForm.protocol === FirewallProtocol.TCP || ruleForm.protocol === FirewallProtocol.UDP"
            v-model="ruleForm.dport"
            label="Destination Port(s)"
            placeholder="e.g., 80, 443, or 1000:2000"
            type="text"
          />

          <OuiInput
            v-if="ruleForm.protocol === FirewallProtocol.TCP || ruleForm.protocol === FirewallProtocol.UDP"
            v-model="ruleForm.sport"
            label="Source Port(s)"
            placeholder="e.g., 8080 or leave empty"
            type="text"
          />

          <OuiInput
            v-model="ruleForm.iface"
            label="Network Interface"
            placeholder="e.g., vmbr0 or leave empty for any"
            type="text"
          />

          <OuiTextarea
            v-model="ruleForm.comment"
            label="Comment"
            placeholder="Optional description for this rule"
            :rows="2"
          />

          <OuiCheckbox v-model="ruleForm.log" label="Enable Logging" />
        </OuiStack>

        <template #footer>
          <OuiFlex justify="end" gap="sm">
            <OuiButton variant="outline" @click="closeDialog">Cancel</OuiButton>
            <OuiButton @click="saveRule" :disabled="savingRule" class="gap-2">
              <CheckIcon class="h-4 w-4" />
              {{ editingRule ? "Update" : "Create" }} Rule
            </OuiButton>
          </OuiFlex>
        </template>
      </OuiDialogContent>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import {
  ArrowPathIcon,
  CheckIcon,
  ExclamationCircleIcon,
  PencilIcon,
  PlusIcon,
  ShieldExclamationIcon,
  TrashIcon,
} from "@heroicons/vue/24/outline";
import {
  VPSService,
  FirewallAction,
  FirewallDirection,
  FirewallProtocol,
  FirewallRuleSchema,
  type FirewallRule,
  type FirewallOptions,
} from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";
import OuiSpinner from "~/components/oui/Spinner.vue";

interface Props {
  vpsId: string;
  organizationId: string;
}

const props = defineProps<Props>();

const { toast } = useToast();
const { showConfirm } = useDialog();
const client = useConnectClient(VPSService);

const loadingRules = ref(false);
const rulesError = ref<string | null>(null);
const rules = ref<FirewallRule[]>([]);
const firewallOptions = ref<FirewallOptions | null>(null);
const editingOptions = ref(false);
const savingOptions = ref(false);
const savingRule = ref(false);
const dialogOpen = ref(false);
const editingRule = ref<FirewallRule | null>(null);

const optionsForm = ref({
  enable: false,
  policyIn: "ACCEPT" as string,
  policyOut: "ACCEPT" as string,
});

const ruleForm = ref({
  enable: true,
  action: FirewallAction.ACCEPT,
  type: FirewallDirection.IN,
  source: "",
  dest: "",
  protocol: FirewallProtocol.ALL,
  dport: "",
  sport: "",
  iface: "",
  comment: "",
  log: false,
});

const actionOptions = [
  { label: "ACCEPT", value: FirewallAction.ACCEPT },
  { label: "REJECT", value: FirewallAction.REJECT },
  { label: "DROP", value: FirewallAction.DROP },
];

const directionOptions = [
  { label: "Incoming", value: FirewallDirection.IN },
  { label: "Outgoing", value: FirewallDirection.OUT },
];

const protocolOptions = [
  { label: "All", value: FirewallProtocol.ALL },
  { label: "TCP", value: FirewallProtocol.TCP },
  { label: "UDP", value: FirewallProtocol.UDP },
  { label: "ICMP", value: FirewallProtocol.ICMP },
  { label: "ICMPv6", value: FirewallProtocol.ICMPV6 },
];

const policyOptions = [
  { label: "ACCEPT", value: "ACCEPT" },
  { label: "DROP", value: "DROP" },
  { label: "REJECT", value: "REJECT" },
];

const tableColumns = computed(() => [
  { key: "pos", label: "Position", defaultWidth: 80 },
  { key: "action", label: "Action", defaultWidth: 100 },
  { key: "direction", label: "Direction", defaultWidth: 100 },
  { key: "source", label: "Source", defaultWidth: 150 },
  { key: "dest", label: "Destination", defaultWidth: 150 },
  { key: "protocol", label: "Protocol", defaultWidth: 100 },
  { key: "port", label: "Port", defaultWidth: 120 },
  { key: "iface", label: "Interface", defaultWidth: 100 },
  { key: "comment", label: "Comment", defaultWidth: 200 },
  { key: "enable", label: "Status", defaultWidth: 100 },
  { key: "actions", label: "Actions", defaultWidth: 120 },
]);

const tableRows = computed(() => {
  return rules.value.map((rule) => ({
    ...rule,
    direction: rule.type,
    port: rule.dport || rule.sport || "—",
  }));
});

function actionLabel(action: FirewallAction): string {
  switch (action) {
    case FirewallAction.ACCEPT:
      return "ACCEPT";
    case FirewallAction.REJECT:
      return "REJECT";
    case FirewallAction.DROP:
      return "DROP";
    default:
      return "UNKNOWN";
  }
}

function protocolLabel(protocol?: FirewallProtocol): string {
  if (!protocol) return "—";
  switch (protocol) {
    case FirewallProtocol.TCP:
      return "TCP";
    case FirewallProtocol.UDP:
      return "UDP";
    case FirewallProtocol.ICMP:
      return "ICMP";
    case FirewallProtocol.ICMPV6:
      return "ICMPv6";
    case FirewallProtocol.ALL:
      return "All";
    default:
      return "—";
  }
}

async function loadRules() {
  loadingRules.value = true;
  rulesError.value = null;
  try {
    const res = await client.listFirewallRules({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });
    rules.value = res.rules || [];
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to load firewall rules";
    rulesError.value = message;
    toast.error("Failed to load firewall rules", message);
  } finally {
    loadingRules.value = false;
  }
}

async function loadOptions() {
  try {
    const res = await client.getFirewallOptions({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });
    firewallOptions.value = res.options || null;
    if (res.options) {
      optionsForm.value.enable = res.options.enable;
      optionsForm.value.policyIn = res.options.policyIn || "ACCEPT";
      optionsForm.value.policyOut = res.options.policyOut || "ACCEPT";
    }
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to load firewall options";
    toast.error("Failed to load firewall options", message);
  }
}

async function saveOptions() {
  savingOptions.value = true;
  try {
    await client.updateFirewallOptions({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      options: {
        enable: optionsForm.value.enable,
        policyIn: optionsForm.value.policyIn,
        policyOut: optionsForm.value.policyOut,
      },
    });
    toast.success("Firewall settings updated", "Firewall options have been saved.");
    editingOptions.value = false;
    await loadOptions();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to update firewall options";
    toast.error("Failed to update firewall options", message);
  } finally {
    savingOptions.value = false;
  }
}

function cancelEditOptions() {
  editingOptions.value = false;
  if (firewallOptions.value) {
    optionsForm.value.enable = firewallOptions.value.enable;
    optionsForm.value.policyIn = firewallOptions.value.policyIn || "ACCEPT";
    optionsForm.value.policyOut = firewallOptions.value.policyOut || "ACCEPT";
  }
}

function openCreateDialog() {
  editingRule.value = null;
  resetRuleForm();
  dialogOpen.value = true;
}

function openEditDialog(rule: FirewallRule) {
  editingRule.value = rule;
  ruleForm.value = {
    enable: rule.enable,
    action: rule.action,
    type: rule.type,
    source: rule.source || "",
    dest: rule.dest || "",
    protocol: rule.protocol || FirewallProtocol.ALL,
    dport: rule.dport || "",
    sport: rule.sport || "",
    iface: rule.iface || "",
    comment: rule.comment || "",
    log: rule.log || false,
  };
  dialogOpen.value = true;
}

function closeDialog() {
  dialogOpen.value = false;
  editingRule.value = null;
  resetRuleForm();
}

function resetRuleForm() {
  ruleForm.value = {
    enable: true,
    action: FirewallAction.ACCEPT,
    type: FirewallDirection.IN,
    source: "",
    dest: "",
    protocol: FirewallProtocol.ALL,
    dport: "",
    sport: "",
    iface: "",
    comment: "",
    log: false,
  };
}

async function saveRule() {
  savingRule.value = true;
  try {
    const rule = create(FirewallRuleSchema, {
      pos: editingRule.value?.pos || 0,
      enable: ruleForm.value.enable,
      action: ruleForm.value.action,
      type: ruleForm.value.type,
      source: ruleForm.value.source || undefined,
      dest: ruleForm.value.dest || undefined,
      protocol: ruleForm.value.protocol,
      dport: ruleForm.value.dport || undefined,
      sport: ruleForm.value.sport || undefined,
      iface: ruleForm.value.iface || undefined,
      comment: ruleForm.value.comment || undefined,
      log: ruleForm.value.log || undefined,
    });

    if (editingRule.value) {
      await client.updateFirewallRule({
        organizationId: props.organizationId,
        vpsId: props.vpsId,
        rulePos: editingRule.value.pos,
        rule,
      });
      toast.success("Firewall rule updated", "The firewall rule has been updated.");
    } else {
      await client.createFirewallRule({
        organizationId: props.organizationId,
        vpsId: props.vpsId,
        rule,
      });
      toast.success("Firewall rule created", "The firewall rule has been created.");
    }

    closeDialog();
    await loadRules();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to save firewall rule";
    toast.error("Failed to save firewall rule", message);
  } finally {
    savingRule.value = false;
  }
}

async function handleDeleteRule(rule: FirewallRule) {
  const confirmed = await showConfirm({
    title: "Delete Firewall Rule",
    message: `Are you sure you want to delete this firewall rule? This action cannot be undone.`,
    confirmLabel: "Delete",
    variant: "danger",
  });
  if (!confirmed) return;

  try {
    await client.deleteFirewallRule({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      rulePos: rule.pos,
    });
    toast.success("Firewall rule deleted", "The firewall rule has been deleted.");
    await loadRules();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Failed to delete firewall rule";
    toast.error("Failed to delete firewall rule", message);
  }
}

onMounted(() => {
  loadRules();
  loadOptions();
});
</script>

