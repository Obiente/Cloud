<template>
  <OuiCard variant="outline" class="border-default">
    <OuiCardBody>
      <OuiStack gap="lg">
        <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
          <OuiStack gap="xs" class="min-w-0">
            <OuiText as="h4" size="sm" weight="semibold"
              >Pull request environments</OuiText
            >
            <OuiText size="xs" color="tertiary">
              Build a disposable environment for each pull request and report it
              in GitHub.
            </OuiText>
          </OuiStack>
          <OuiFlex align="center" gap="sm">
            <OuiText size="xs" color="tertiary">{{
              form.enabled ? "On" : "Off"
            }}</OuiText>
            <OuiSwitch v-model="form.enabled" size="md" />
          </OuiFlex>
        </OuiFlex>

        <OuiText v-if="loading" size="xs" color="tertiary"
          >Loading settings...</OuiText
        >
        <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
        <OuiText v-if="saved" size="xs" color="success"
          >Pull request settings saved.</OuiText
        >

        <template v-if="!loading">
          <OuiCard variant="outline" class="bg-surface-subtle/40">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold">GitHub App access</OuiText>
                <OuiText size="xs" color="tertiary">
                  Native deployments, checks, and the maintained PR comment
                  require Pull requests: read, Deployments: write, Checks:
                  write, and Issues: write. Subscribe the app to pull request
                  events.
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
            <OuiTagsInput
              v-model="form.baseBranches"
              label="Target branches"
              placeholder="main"
            />
            <OuiInput
              v-model="form.domainTemplate"
              label="Preview hostname template"
              placeholder="pr-{pr}-{deployment}"
            />
            <OuiTagsInput
              v-model="form.includePaths"
              label="Only deploy when paths match"
              placeholder="apps/web/**"
            />
            <OuiTagsInput
              v-model="form.excludePaths"
              label="Ignored paths"
              placeholder="docs/**"
            />
            <OuiInput
              v-model="maxActiveInput"
              type="number"
              min="1"
              max="50"
              label="Maximum active previews"
            />
            <OuiInput
              v-model="ttlInput"
              type="number"
              min="1"
              max="720"
              label="Lifetime in hours"
            />
            <OuiInput
              v-model="restoredTtlInput"
              type="number"
              min="1"
              max="72"
              label="Restored preview lifetime in hours"
            />
            <OuiSelect
              v-model="form.forkPolicy"
              :items="forkPolicyOptions"
              label="Fork pull requests"
            />
          </OuiGrid>

          <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
            <SettingToggle
              v-model="form.deployDrafts"
              title="Deploy drafts"
              description="Start previews before a pull request is marked ready."
            />
            <SettingToggle
              v-model="form.redeployOnPush"
              title="Redeploy updates"
              description="Build each new head revision."
            />
            <SettingToggle
              v-model="form.cleanupOnClose"
              title="Remove on close"
              description="Destroy the runtime when the pull request closes."
            />
            <SettingToggle
              v-model="form.requireApproval"
              title="Require approval"
              description="Hold every pull request revision for a maintainer."
            />
            <SettingToggle
              v-model="form.approvalCoversUpdates"
              title="Approval covers later pushes"
              description="Keep approval when the head SHA changes. Off is safer."
            />
            <SettingToggle
              v-model="form.commentEnabled"
              title="PR comment"
              description="Create one comment and update it in place."
            />
            <SettingToggle
              v-model="form.deploymentStatusEnabled"
              title="GitHub deployment"
              description="Show a transient environment and preview URL in GitHub."
            />
            <SettingToggle
              v-model="form.checkRunEnabled"
              title="GitHub check"
              description="Show build and approval state with the normal PR checks."
            />
          </OuiGrid>

          <OuiCard variant="outline" class="border-warning/30">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="semibold"
                    >Values available to trusted previews</OuiText
                  >
                  <OuiText size="xs" color="tertiary">
                    Nothing is copied by default. Add only values that pull
                    request code may read. Fork previews never receive these
                    values or persistent volumes.
                  </OuiText>
                </OuiStack>
                <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                  <OuiTagsInput
                    v-model="form.environmentVariableNames"
                    label="Environment variable names"
                    placeholder="PUBLIC_API_URL"
                  />
                  <OuiTagsInput
                    v-model="form.buildArgumentNames"
                    label="Build argument names"
                    placeholder="PUBLIC_BUILD_MODE"
                  />
                </OuiGrid>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiFlex justify="end" gap="sm">
            <OuiButton
              variant="ghost"
              size="sm"
              :disabled="saving"
              @click="load"
              >Reset</OuiButton
            >
            <OuiButton
              variant="solid"
              size="sm"
              :disabled="saving"
              @click="save"
            >
              {{ saving ? "Saving..." : "Save pull request settings" }}
            </OuiButton>
          </OuiFlex>

          <OuiStack gap="sm">
            <OuiFlex justify="between" align="center">
              <OuiText as="h5" size="sm" weight="semibold"
                >Pull requests</OuiText
              >
              <OuiButton
                variant="ghost"
                size="sm"
                :disabled="refreshing"
                @click="loadRuns"
              >
                {{ refreshing ? "Refreshing..." : "Refresh" }}
              </OuiButton>
            </OuiFlex>
            <OuiText v-if="runs.length === 0" size="xs" color="tertiary"
              >No pull request environments yet.</OuiText
            >
            <OuiCard v-for="run in runs" :key="run.id" variant="outline">
              <OuiCardBody>
                <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
                  <OuiStack gap="xs" class="min-w-0">
                    <OuiText size="sm" weight="semibold"
                      >PR #{{ run.pullRequestNumber }} ·
                      {{ statusLabel(run.status) }}</OuiText
                    >
                    <OuiText size="xs" color="tertiary" class="font-mono"
                      >{{ run.headRef }} ·
                      {{ run.headSha.slice(0, 12) }}</OuiText
                    >
                    <OuiText v-if="run.fromFork" size="xs" color="warning"
                      >Fork · isolated from deployment values and
                      volumes</OuiText
                    >
                    <OuiText v-if="run.merged" size="xs" color="tertiary"
                      >Merged pull request</OuiText
                    >
                    <OuiText v-if="run.error" size="xs" color="danger">{{
                      run.error
                    }}</OuiText>
                  </OuiStack>
                  <OuiFlex gap="sm" align="center" wrap="wrap">
                    <a
                      v-if="run.stateUrl || run.environmentUrl"
                      :href="run.stateUrl || run.environmentUrl"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-xs text-link hover:underline"
                      >{{
                        run.status ===
                        PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_RUNNING
                          ? "Open preview"
                          : "View preview status"
                      }}</a
                    >
                    <OuiButton
                      v-if="
                        run.status ===
                        PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL
                      "
                      size="sm"
                      variant="solid"
                      :disabled="actingRunId === run.id"
                      @click="approve(run.id)"
                      >Approve revision</OuiButton
                    >
                    <OuiButton
                      v-if="
                        run.status ===
                        PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL
                      "
                      size="sm"
                      variant="ghost"
                      :disabled="actingRunId === run.id"
                      @click="reject(run.id)"
                      >Reject</OuiButton
                    >
                    <OuiButton
                      v-if="canRedeploy(run.status)"
                      size="sm"
                      variant="ghost"
                      :disabled="actingRunId === run.id"
                      @click="redeploy(run.id)"
                      >Redeploy</OuiButton
                    >
                    <OuiButton
                      v-if="canRestore(run)"
                      size="sm"
                      variant="solid"
                      :disabled="actingRunId === run.id"
                      @click="restore(run.id)"
                      >Restore temporarily</OuiButton
                    >
                    <OuiButton
                      v-if="!run.closedAt"
                      size="sm"
                      variant="ghost"
                      :disabled="actingRunId === run.id"
                      @click="remove(run.id)"
                      >Remove</OuiButton
                    >
                  </OuiFlex>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>
        </template>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import {
  DeploymentService,
  PullRequestDeploymentStatus,
  PullRequestForkPolicy,
  type Deployment,
  type PullRequestDeployment,
  type PullRequestDeploymentConfig,
} from "@obiente/proto";
import { computed, onMounted, reactive, ref } from "vue";
import SettingToggle from "~/components/deployment/PullRequestSettingToggle.vue";
import OuiSwitch from "~/components/oui/Switch.vue";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationsStore } from "~/stores/organizations";

const props = defineProps<{ deployment: Deployment }>();
const orgsStore = useOrganizationsStore();
const client = useConnectClient(DeploymentService);
const loading = ref(true);
const saving = ref(false);
const refreshing = ref(false);
const error = ref("");
const saved = ref(false);
const runs = ref<PullRequestDeployment[]>([]);
const actingRunId = ref("");

const form = reactive({
  enabled: false,
  baseBranches: [] as string[],
  includePaths: [] as string[],
  excludePaths: [] as string[],
  deployDrafts: false,
  redeployOnPush: true,
  cleanupOnClose: true,
  commentEnabled: true,
  deploymentStatusEnabled: true,
  checkRunEnabled: true,
  domainTemplate: "pr-{pr}-{deployment}",
  maxActivePreviews: 5,
  ttlHours: 72,
  restoredPreviewTtlHours: 4,
  forkPolicy: PullRequestForkPolicy.PULL_REQUEST_FORK_DENY,
  environmentVariableNames: [] as string[],
  buildArgumentNames: [] as string[],
  requireApproval: false,
  approvalCoversUpdates: false,
});

const maxActiveInput = computed({
  get: () => String(form.maxActivePreviews),
  set: (value: string) => {
    form.maxActivePreviews = Number(value);
  },
});
const ttlInput = computed({
  get: () => String(form.ttlHours),
  set: (value: string) => {
    form.ttlHours = Number(value);
  },
});
const restoredTtlInput = computed({
  get: () => String(form.restoredPreviewTtlHours),
  set: (value: string) => {
    form.restoredPreviewTtlHours = Number(value);
  },
});
const forkPolicyOptions = [
  {
    label: "Do not deploy forks",
    value: PullRequestForkPolicy.PULL_REQUEST_FORK_DENY,
  },
  {
    label: "Require maintainer approval",
    value: PullRequestForkPolicy.PULL_REQUEST_FORK_REQUIRE_APPROVAL,
  },
  {
    label: "Deploy automatically, isolated",
    value: PullRequestForkPolicy.PULL_REQUEST_FORK_ISOLATED,
  },
];

const organizationId = computed(() => orgsStore.currentOrgId || "");

function applyConfig(config: PullRequestDeploymentConfig) {
  Object.assign(form, {
    enabled: config.enabled,
    baseBranches: [...config.baseBranches],
    includePaths: [...config.includePaths],
    excludePaths: [...config.excludePaths],
    deployDrafts: config.deployDrafts,
    redeployOnPush: config.redeployOnPush,
    cleanupOnClose: config.cleanupOnClose,
    commentEnabled: config.commentEnabled,
    deploymentStatusEnabled: config.deploymentStatusEnabled,
    checkRunEnabled: config.checkRunEnabled,
    domainTemplate: config.domainTemplate || "pr-{pr}-{deployment}",
    maxActivePreviews: config.maxActivePreviews || 5,
    ttlHours: config.ttlHours || 72,
    restoredPreviewTtlHours: config.restoredPreviewTtlHours || 4,
    forkPolicy:
      config.forkPolicy || PullRequestForkPolicy.PULL_REQUEST_FORK_DENY,
    environmentVariableNames: [...config.environmentVariableNames],
    buildArgumentNames: [...config.buildArgumentNames],
    requireApproval: config.requireApproval,
    approvalCoversUpdates: config.approvalCoversUpdates,
  });
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const response = await client.getPullRequestDeploymentConfig({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
    });
    if (response.config) applyConfig(response.config);
    await loadRuns();
  } catch (cause: unknown) {
    error.value =
      (cause as Error).message || "Could not load pull request settings.";
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  saved.value = false;
  error.value = "";
  try {
    const response = await client.updatePullRequestDeploymentConfig({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      config: { deploymentId: props.deployment.id, ...form },
    });
    if (response.config) applyConfig(response.config);
    saved.value = true;
    globalThis.setTimeout(() => {
      saved.value = false;
    }, 3000);
  } catch (cause: unknown) {
    error.value =
      (cause as Error).message || "Could not save pull request settings.";
  } finally {
    saving.value = false;
  }
}

async function loadRuns() {
  refreshing.value = true;
  try {
    runs.value = (
      await client.listPullRequestDeployments({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        includeClosed: true,
      })
    ).deployments;
  } catch (cause: unknown) {
    error.value =
      (cause as Error).message || "Could not load pull request environments.";
  } finally {
    refreshing.value = false;
  }
}

async function approve(id: string) {
  await perform(id, () =>
    client.approvePullRequestDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      pullRequestDeploymentId: id,
    })
  );
}
async function reject(id: string) {
  await perform(id, () =>
    client.rejectPullRequestDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      pullRequestDeploymentId: id,
      reason: "Rejected by a maintainer in Obiente.",
    })
  );
}
async function redeploy(id: string) {
  await perform(id, () =>
    client.redeployPullRequestDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      pullRequestDeploymentId: id,
    })
  );
}
async function remove(id: string) {
  await perform(id, () =>
    client.deletePullRequestDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      pullRequestDeploymentId: id,
    })
  );
}
async function restore(id: string) {
  await perform(id, () =>
    client.restorePullRequestDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      pullRequestDeploymentId: id,
    })
  );
}
async function perform(id: string, action: () => Promise<unknown>) {
  error.value = "";
  actingRunId.value = id;
  try {
    await action();
    await loadRuns();
  } catch (cause: unknown) {
    error.value =
      (cause as Error).message || "The pull request environment action failed.";
  } finally {
    actingRunId.value = "";
  }
}

function statusLabel(status: PullRequestDeploymentStatus) {
  return (
    (
      {
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_QUEUED]: "Queued",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_BUILDING]:
          "Building",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_RUNNING]: "Ready",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_FAILED]: "Failed",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_SKIPPED]:
          "Not deployed",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_CLOSED]: "Removed",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_WAITING_APPROVAL]:
          "Waiting for approval",
        [PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_REJECTED]:
          "Rejected",
      } as Record<number, string>
    )[status] || "Pending"
  );
}
function canRedeploy(status: PullRequestDeploymentStatus) {
  return [
    PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_RUNNING,
    PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_FAILED,
  ].includes(status);
}
function canRestore(run: PullRequestDeployment) {
  return Boolean(
    run.merged &&
      run.closedAt &&
      run.approvedAt &&
      (form.approvalCoversUpdates || run.approvedHeadSha === run.headSha) &&
      run.status === PullRequestDeploymentStatus.PULL_REQUEST_DEPLOYMENT_CLOSED
  );
}

onMounted(load);
</script>
