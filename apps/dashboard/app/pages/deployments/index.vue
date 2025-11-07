<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="sm" class="max-w-xl">
          <OuiFlex align="center" gap="md">
            <OuiBox
              p="sm"
              rounded="xl"
              bg="accent-primary"
              class="bg-primary/10 ring-1 ring-primary/20"
            >
              <RocketLaunchIcon class="w-6 h-6 text-primary" />
            </OuiBox>
            <OuiText as="h1" size="3xl" weight="bold"> Deployments </OuiText>
          </OuiFlex>
          <OuiText color="secondary" size="md">
            Manage and monitor your web application deployments with real-time
            insights.
          </OuiText>
        </OuiStack>

        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium">New Deployment</OuiText>
        </OuiButton>
      </OuiFlex>

      <!-- Show error alert if there was a problem loading deployments -->
      <ErrorAlert
        v-if="listError"
        :error="listError"
        title="Failed to load deployments"
        hint="Please try refreshing the page. If the problem persists, contact support."
      />

      <OuiCard
        variant="default"
        class="backdrop-blur-sm border border-border-muted/60"
      >
        <OuiCardBody>
          <OuiGrid cols="1" cols-md="4" gap="md">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search by name, domain, or framework..."
              clearable
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
              </template>
            </OuiInput>

            <OuiSelect
              v-model="statusFilter"
              :items="statusFilterOptions"
              placeholder="All Status"
            />

            <OuiCombobox
              v-model="environmentFilter"
              :options="environmentOptions"
              placeholder="All Environments"
              clearable
            />

            <OuiCombobox
              v-model="groupFilter"
              :options="groupOptions"
              placeholder="All Groups"
              clearable
            />
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <OuiStack
        v-if="filteredDeployments.length === 0"
        align="center"
        gap="lg"
        class="text-center py-20"
      >
        <OuiBox
          class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
        >
          <RocketLaunchIcon class="h-10 w-10 text-secondary" />
        </OuiBox>
        <OuiStack align="center" gap="sm">
          <OuiText as="h3" size="xl" weight="semibold" color="primary">
            No deployments found
          </OuiText>
          <OuiBox class="max-w-md">
            <OuiText color="secondary">
              {{
                searchQuery || statusFilter || environmentFilter || groupFilter
                  ? "Try adjusting your filters to see more results."
                  : "Get started by creating your first deployment."
              }}
            </OuiText>
          </OuiBox>
        </OuiStack>
        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium"
            >Create Your First Deployment</OuiText
          >
        </OuiButton>
      </OuiStack>

      <OuiGrid v-else cols="1" cols-md="2" :cols-2xl="3" gap="lg">
        <OuiCard
          v-for="deployment in filteredDeployments"
          :key="deployment.id"
          variant="default"
          hoverable
          :data-status="deployment.status"
          :class="[
            'group relative overflow-hidden transition-all duration-300 hover:shadow-2xl',
            getStatusMeta(deployment.status).cardClass,
            getStatusMeta(deployment.status).beforeGradient,
          ]"
        >
          <div
            class="absolute top-0 left-0 right-0 h-1"
            :class="getStatusMeta(deployment.status).barClass"
          />

          <OuiFlex direction="col" h="full" class="relative">
            <OuiCardHeader>
              <OuiFlex justify="between" align="center" gap="lg" wrap="wrap">
                <OuiStack gap="xs" class="min-w-0">
                  <OuiText
                    as="h3"
                    size="xl"
                    weight="semibold"
                    color="primary"
                    truncate
                    class="transition-colors group-hover:text-primary/90"
                  >
                    {{ deployment.name }}
                  </OuiText>
                  <a
                    :href="`https://${deployment.domain}`"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center gap-1.5 text-sm text-secondary hover:text-primary transition-colors group/link"
                    @click.stop
                  >
                    <span class="truncate max-w-[200px]">{{
                      deployment.domain
                    }}</span>
                    <ArrowTopRightOnSquareIcon
                      class="h-3.5 w-3.5 opacity-0 group-hover/link:opacity-100 transition-opacity"
                    />
                  </a>
                  <!-- Show ownership information if we implement it later -->
                  <OuiFlex v-if="false" align="center" gap="xs" class="mt-0.5">
                    <UserIcon class="h-3 w-3 text-secondary" />
                    <OuiText size="xs" color="secondary"
                      >Created by you</OuiText
                    >
                  </OuiFlex>
                </OuiStack>
                <OuiFlex gap="sm" justify="end" wrap="wrap">
                  <OuiBadge :variant="getStatusMeta(deployment.status).badge">
                    <span
                      class="inline-flex h-1.5 w-1.5 rounded-full"
                      :class="[
                        getStatusMeta(deployment.status).dotClass,
                        getStatusMeta(deployment.status).pulseDot
                          ? 'animate-pulse'
                          : '',
                      ]"
                    />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="semibold"
                      transform="uppercase"
                      class="text-[11px]"
                    >
                      {{ getStatusMeta(deployment.status).label }}
                    </OuiText>
                  </OuiBadge>
                  <OuiBadge
                    v-if="deployment.containersTotal && deployment.containersTotal > 0 && (deployment.containersRunning ?? 0) > 0 && (deployment.containersRunning ?? 0) < deployment.containersTotal"
                    :variant="getContainerStatusVariant(deployment.containersRunning ?? 0, deployment.containersTotal ?? 0)"
                  >
                    <OuiText
                      as="span"
                      size="xs"
                      weight="semibold"
                      class="text-[11px]"
                    >
                      {{ deployment.containersRunning ?? 0 }}/{{ deployment.containersTotal }} running
                    </OuiText>
                  </OuiBadge>
                  <OuiBadge
                    v-for="(group, idx) in ((deployment as any).groups || [])"
                    :key="idx"
                    variant="secondary"
                    size="sm"
                  >
                    <OuiText
                      as="span"
                      size="xs"
                      weight="semibold"
                      class="text-[11px]"
                    >
                      {{ group }}
                    </OuiText>
                  </OuiBadge>
                </OuiFlex>
              </OuiFlex>
            </OuiCardHeader>

            <OuiCardBody class="flex-1">
              <OuiStack gap="lg">
                <OuiFlex justify="between" align="center" class="text-sm">
                  <OuiFlex align="center" gap="sm">
                    <OuiBox
                      p="xs"
                      rounded="lg"
                      bg="surface-muted"
                      class="bg-surface-muted/50 ring-1 ring-border-muted"
                    >
                      <CodeBracketIcon class="h-4 w-4 text-primary" />
                    </OuiBox>
                    <OuiText size="sm" weight="medium" color="primary">
                      {{ getTypeLabel(deployment.type) }}
                    </OuiText>
                  </OuiFlex>
                  <OuiFlex
                    align="center"
                    gap="xs"
                    class="text-xs text-secondary"
                  >
                    <CalendarIcon class="h-3.5 w-3.5" />
                    <OuiRelativeTime :value="deployment.lastDeployedAt ? date(deployment.lastDeployedAt) : undefined" :style="'short'" />
                  </OuiFlex>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiBox
                    v-if="deployment.repositoryUrl"
                    p="sm"
                    rounded="lg"
                    w="4xl"
                    bg="surface-muted"
                    class="bg-surface-muted/30 ring-1 ring-border-muted"
                  >
                    <OuiFlex align="center" gap="sm" class="min-w-0">
                      <Icon
                        name="uil:github"
                        class="h-4 w-4 text-secondary shrink-0"
                      />
                      <OuiText
                        size="xs"
                        color="secondary"
                        truncate
                        class="font-mono"
                      >
                        {{ cleanRepositoryName(deployment.repositoryUrl) }}
                      </OuiText>
                    </OuiFlex>
                  </OuiBox>
                  <span
                    class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold uppercase tracking-wide ml-auto"
                    :class="
                      getEnvironmentMeta(deployment.environment).chipClass
                    "
                  >
                    <CpuChipIcon class="h-3.5 w-3.5" />
                    {{ getEnvironmentMeta(deployment.environment).label }}
                  </span>
                </OuiFlex>
                <OuiGrid cols="2" gap="md">
                  <OuiBox
                    p="md"
                    rounded="xl"
                    bg="surface-muted"
                    class="group/stat overflow-hidden bg-surface-muted/40 ring-1 ring-border-muted backdrop-blur-sm transition-all hover:bg-surface-muted/60 hover:ring-border-default"
                  >
                    <div
                      class="absolute inset-0 bg-linear-to-br from-primary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity"
                    />
                    <OuiStack gap="xs" class="relative">
                      <OuiText
                        size="xs"
                        weight="bold"
                        transform="uppercase"
                        color="secondary"
                        class="tracking-wider"
                      >
                        Build Time
                      </OuiText>
                      <OuiText size="2xl" weight="bold" color="primary">
                        {{ formatBuildTime(deployment.buildTime ?? 0) }}
                      </OuiText>
                    </OuiStack>
                  </OuiBox>

                  <OuiBox
                    p="md"
                    rounded="xl"
                    bg="surface-muted"
                    class="group/stat relative overflow-hidden bg-surface-muted/40 ring-1 ring-border-muted backdrop-blur-sm transition-all hover:bg-surface-muted/60 hover:ring-border-default"
                  >
                    <div
                      class="absolute inset-0 bg-linear-to-br from-secondary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity"
                    />
                    <OuiStack gap="xs" class="relative">
                      <OuiText
                        size="xs"
                        weight="bold"
                        transform="uppercase"
                        color="secondary"
                        class="tracking-wider"
                      >
                        Bundle Size
                      </OuiText>
                      <OuiText size="2xl" weight="bold" color="primary">
                        <OuiByte
                          :value="deployment.size ?? 0"
                          unit-display="short"
                        />
                      </OuiText>
                    </OuiStack>
                  </OuiBox>
                </OuiGrid>

                <OuiBox
                  v-if="
                    deployment.status === DeploymentStatus.BUILDING ||
                    deployment.status === DeploymentStatus.DEPLOYING ||
                    (deployment.status === DeploymentStatus.FAILED && progressValues[deployment.id] !== undefined)
                  "
                  p="md"
                  rounded="xl"
                  class="border backdrop-blur-sm"
                  :class="
                    deployment.status === DeploymentStatus.FAILED
                      ? 'border-danger/30 bg-danger/10 text-danger'
                      : getStatusMeta(deployment.status).progressClass
                  "
                >
                  <OuiStack gap="sm">
                    <OuiFlex
                      align="center"
                      gap="sm"
                      class="text-xs font-bold uppercase tracking-wider"
                      :class="
                        deployment.status === DeploymentStatus.FAILED
                          ? 'text-danger'
                          : ''
                      "
                    >
                      <Cog6ToothIcon
                        v-if="deployment.status !== DeploymentStatus.FAILED"
                        class="h-4 w-4 animate-spin"
                      />
                      <ExclamationTriangleIcon
                        v-else
                        class="h-4 w-4 text-danger"
                      />
                      <span
                        :class="
                          deployment.status === DeploymentStatus.FAILED
                            ? 'text-danger'
                            : ''
                        "
                      >
                        {{ progressPhases[deployment.id] || 'Starting deployment...' }}
                      </span>
                    </OuiFlex>
                    <div
                      class="relative h-2 w-full overflow-hidden rounded-full"
                      :class="
                        deployment.status === DeploymentStatus.FAILED
                          ? 'bg-danger/20'
                          : 'bg-warning/20'
                      "
                    >
                      <div
                        v-if="deployment.status === DeploymentStatus.FAILED"
                        class="absolute inset-y-0 left-0 rounded-full bg-danger transition-all duration-300"
                        :style="{ width: '100%' }"
                      />
                      <div
                        v-else
                        class="absolute inset-y-0 left-0 rounded-full bg-warning transition-all duration-300"
                        :style="{ width: `${progressValues[deployment.id] || 0}%` }"
                      />
                    </div>
                  </OuiStack>
                </OuiBox>
              </OuiStack>
            </OuiCardBody>

            <OuiCardFooter class="mt-auto">
              <OuiFlex justify="between" align="center" gap="none">
                <OuiButton
                  v-if="deployment.status === DeploymentStatus.RUNNING"
                  variant="ghost"
                  size="sm"
                  color="danger"
                  @click="stopDeployment(deployment.id)"
                  title="Stop deployment"
                  :aria-label="`Stop deployment ${deployment.name}`"
                  class="gap-2"
                >
                  <StopIcon class="h-4 w-4" />
                  <OuiText as="span" size="xs" weight="medium">Stop</OuiText>
                </OuiButton>

                <OuiButton
                  v-if="deployment.status === DeploymentStatus.STOPPED"
                  variant="ghost"
                  color="success"
                  size="sm"
                  @click="startDeployment(deployment.id)"
                  title="Start deployment"
                  :aria-label="`Start deployment ${deployment.name}`"
                  class="gap-2"
                >
                  <PlayIcon class="h-4 w-4" />
                  <OuiText as="span" size="xs" weight="medium">Start</OuiText>
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  color="warning"
                  size="sm"
                  @click="redeployApp(deployment.id)"
                  title="Redeploy (rebuild and redeploy)"
                  :aria-label="`Redeploy deployment ${deployment.name}`"
                  class="gap-2"
                >
                  <ArrowPathIcon class="h-4 w-4" />
                  <OuiText as="span" size="xs" weight="medium"
                    >Redeploy</OuiText
                  >
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="openUrl(deployment.domain)"
                  title="Open site"
                  :aria-label="`Open ${deployment.domain} in a new tab`"
                  class="gap-2"
                >
                  <ArrowTopRightOnSquareIcon class="h-4 w-4" />
                  <OuiText as="span" size="xs" weight="medium">Open</OuiText>
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  color="secondary"
                  size="sm"
                  :as="NuxtLink"
                  :to="`/deployments/${deployment.id}`"
                  title="View details"
                  :aria-label="`View ${deployment.name} details`"
                  class="gap-2"
                >
                  <EyeIcon class="h-4 w-4" />
                  <OuiText as="span" size="xs" weight="medium">Details</OuiText>
                </OuiButton>
              </OuiFlex>
            </OuiCardFooter>
          </OuiFlex>
        </OuiCard>
      </OuiGrid>

      <OuiDialog
        v-model:open="showCreateDialog"
        title="Create New Deployment"
        description="Deploy your application to Obiente Cloud with automated builds and deployments"
      >
        <form @submit.prevent="createDeployment">
          <OuiStack gap="lg">
            <!-- Error display for permission/authentication issues -->
            <ErrorAlert
              v-if="createError"
              :error="createError"
              title="Unable to create deployment"
              hint="Make sure you're logged in and have permission to create deployments."
            />

            <OuiInput
              v-model="newDeployment.name"
              label="Project Name"
              placeholder="my-awesome-app"
              required
            />

            <OuiSelect
              v-model="newDeployment.environment"
              :items="environmentOptions"
              label="Environment"
              required
            />

            <OuiTagsInput
              v-model="newDeploymentGroups"
              label="Groups/Labels (Optional)"
              placeholder="Add group..."
              helper-text="Press Enter or click outside to add a group/label"
            />

            <OuiText size="xs" color="secondary">
              The deployment type will be automatically detected when you connect your repository. You can configure the repository and other settings after creating the deployment.
            </OuiText>
          </OuiStack>
        </form>
        <template #footer>
          <OuiFlex justify="end" align="center" gap="md">
            <OuiButton variant="ghost" @click="showCreateDialog = false">
              Cancel
            </OuiButton>
            <OuiButton
              color="primary"
              @click="createDeployment"
              class="gap-2 shadow-lg shadow-primary/20"
            >
              <RocketLaunchIcon class="h-4 w-4" />
              <OuiText as="span" size="sm" weight="medium">Deploy Now</OuiText>
            </OuiButton>
          </OuiFlex>
        </template>
      </OuiDialog>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  import { ref, computed, reactive, watch, onMounted, onUnmounted } from "vue";
  import { useRouter } from "vue-router";
  import { NuxtLink } from "#components";
  import {
    ArrowPathIcon,
    ArrowTopRightOnSquareIcon,
    BoltIcon,
    CalendarIcon,
    CodeBracketIcon,
    Cog6ToothIcon,
    CpuChipIcon,
    EyeIcon,
    ExclamationTriangleIcon,
    InformationCircleIcon,
    MagnifyingGlassIcon,
    PauseCircleIcon,
    PlayIcon,
    PlusIcon,
    RocketLaunchIcon,
    StopIcon,
    UserIcon,
  } from "@heroicons/vue/24/outline";

  import {
    DeploymentService,
    type Deployment,
    DeploymentType,
    DeploymentStatus,
    Environment as EnvEnum,
  } from "@obiente/proto";
  import { date, timestamp } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { useDeploymentActions } from "~/composables/useDeploymentActions";
  import ErrorAlert from "~/components/ErrorAlert.vue";
  import GitHubRepoPicker from "~/components/deployment/GitHubRepoPicker.vue";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
  import OuiByte from "~/components/oui/Byte.vue";
  import { useDialog } from "~/composables/useDialog";
  import { useBuildProgress } from "~/composables/useBuildProgress";
  definePageMeta({
    layout: "default",
    middleware: "auth",
  });
  
  const route = useRoute();
  
  // Error handling
  const createError = ref<Error | null>(null);
  const listError = ref<Error | null>(null);
  
  // Organizations
  const orgsStore = useOrganizationsStore();
  
  // Check for organizationId in query params (from superadmin navigation)
  if (route.query.organizationId && typeof route.query.organizationId === "string") {
    orgsStore.switchOrganization(route.query.organizationId);
  }
  
  const effectiveOrgId = computed(() => {
    // Prefer query param if present, otherwise use store
    if (route.query.organizationId && typeof route.query.organizationId === "string") {
      return route.query.organizationId;
    }
    return orgsStore.currentOrgId || "";
  });

  // Deployment service client
  const deploymentClient = useConnectClient(DeploymentService);

  // Filters
  const searchQuery = ref("");
  // Initialize statusFilter from query params
  const statusFilter = ref(
    typeof route.query.status === "string" ? route.query.status : ""
  );
  const environmentFilter = ref("");
  const groupFilter = ref("");
  const showCreateDialog = ref(false);

  const newDeployment = ref({
    name: "",
    environment: String(EnvEnum.PRODUCTION),
  });
  const newDeploymentGroups = ref<string[]>([]);

  const statusFilterOptions = [
    { label: "All Status", value: "" },
    { label: "Running", value: String(DeploymentStatus.RUNNING) },
    { label: "Stopped", value: String(DeploymentStatus.STOPPED) },
    { label: "Building", value: String(DeploymentStatus.BUILDING) },
    { label: "Failed", value: String(DeploymentStatus.FAILED) },
  ];

  const environmentOptions = [
    { label: "Production", value: String(EnvEnum.PRODUCTION) },
    { label: "Staging", value: String(EnvEnum.STAGING) },
    { label: "Development", value: String(EnvEnum.DEVELOPMENT) },
  ];

  // Compute available groups from deployments
  const groupOptions = computed(() => {
    const groups = new Set<string>();
    (deployments.value || []).forEach((deployment: any) => {
      const deploymentGroups = (deployment.groups || []).filter((g: string) => g && g.trim());
      deploymentGroups.forEach((g: string) => groups.add(g.trim()));
      // Also check legacy group field for backward compatibility
      if ((deployment as any).group && (deployment as any).group.trim()) {
        groups.add((deployment as any).group.trim());
      }
    });
    return Array.from(groups)
      .sort()
      .map((group) => ({ label: group, value: group }));
  });


  const STATUS_META = {
    RUNNING: {
      badge: "success",
      label: "Running",
      description: "This deployment is serving traffic.",
      cardClass: "hover:ring-1 hover:ring-success/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-success/20 before:via-success/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-success to-success/70",
      dotClass: "bg-success",
      icon: BoltIcon,
      iconClass: "text-success",
      progressClass: "border-success/30 bg-success/10 text-success",
      pulseDot: true,
    },
    STOPPED: {
      badge: "danger",
      label: "Stopped",
      description: "Deployment is currently paused.",
      cardClass: "hover:ring-1 hover:ring-danger/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-danger to-danger/60",
      dotClass: "bg-danger",
      icon: PauseCircleIcon,
      iconClass: "text-danger",
      progressClass: "border-danger/30 bg-danger/10 text-danger",
      pulseDot: false,
    },
    BUILDING: {
      badge: "warning",
      label: "Building",
      description: "A new build is in progress.",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      dotClass: "bg-warning",
      icon: Cog6ToothIcon,
      iconClass: "text-warning",
      progressClass: "border-warning/30 bg-warning/10 text-warning",
      pulseDot: true,
    },
    FAILED: {
      badge: "danger",
      label: "Failed",
      description: "The last build encountered an error.",
      cardClass: "hover:ring-1 hover:ring-danger/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-danger to-danger/60",
      dotClass: "bg-danger",
      icon: ExclamationTriangleIcon,
      iconClass: "text-danger",
      progressClass: "border-danger/30 bg-danger/10 text-danger",
      pulseDot: false,
    },
    DEFAULT: {
      badge: "primary",
      label: "New deployment",
      description: "This deployment is being created.",
      cardClass: "hover:ring-1 hover:ring-border-muted",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-surface-muted/30 before:via-surface-muted/20 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-surface-muted to-surface-muted/70",
      dotClass: "bg-secondary",
      icon: InformationCircleIcon,
      iconClass: "text-secondary",
      progressClass: "border-border-muted bg-surface-muted/40 text-secondary",
      pulseDot: false,
    },
  } as const;

  const getStatusMeta = (status: number | DeploymentStatus) => {
    switch (status) {
      case DeploymentStatus.RUNNING:
        return STATUS_META.RUNNING;
      case DeploymentStatus.STOPPED:
        return STATUS_META.STOPPED;
      case DeploymentStatus.BUILDING:
      case DeploymentStatus.DEPLOYING:
        return STATUS_META.BUILDING;
      case DeploymentStatus.FAILED:
        return STATUS_META.FAILED;
      default:
        return STATUS_META.DEFAULT;
    }
  };

  const getContainerStatusVariant = (running: number, total: number): "success" | "warning" | "danger" => {
    if (total === 0) return "danger";
    if (running === total) return "success"; // All running
    if (running === 0) return "danger"; // None running
    return "warning"; // Partial (e.g., 2/5 running)
  };

  const ENVIRONMENT_META = {
    production: {
      label: "Production",
      badge: "success",
      chipClass: "bg-success/10 text-success ring-1 ring-success/20",
      highlightIcon: BoltIcon,
      highlightClass: "bg-success/10 text-success ring-1 ring-success/20",
    },
    staging: {
      label: "Staging",
      badge: "warning",
      chipClass: "bg-warning/10 text-warning ring-1 ring-warning/20",
      highlightIcon: null,
      highlightClass: "",
    },
    development: {
      label: "Development",
      badge: "secondary",
      chipClass: "bg-info/10 text-info ring-1 ring-info/20",
      highlightIcon: null,
      highlightClass: "",
    },
    DEFAULT: {
      label: "Environment",
      badge: "secondary",
      chipClass: "bg-surface-muted text-secondary ring-1 ring-border-muted",
      highlightIcon: null,
      highlightClass: "",
    },
  } as const;

  type EnvironmentKey = keyof typeof ENVIRONMENT_META;

  const getEnvironmentMeta = (environment: string | EnvEnum) => {
    // Accept either enum numeric or string key
    if (typeof environment === "number") {
      switch (environment) {
        case EnvEnum.PRODUCTION:
          return ENVIRONMENT_META.production;
        case EnvEnum.STAGING:
          return ENVIRONMENT_META.staging;
        case EnvEnum.DEVELOPMENT:
          return ENVIRONMENT_META.development;
        default:
          return ENVIRONMENT_META.DEFAULT;
      }
    }

    return (
      ENVIRONMENT_META[environment as EnvironmentKey] ??
      ENVIRONMENT_META.DEFAULT
    );
  };

  const getTypeLabel = (t: DeploymentType | number | undefined) => {
    switch (t) {
      case DeploymentType.DOCKER:
        return "Docker";
      case DeploymentType.STATIC:
        return "Static Site";
      case DeploymentType.NODE:
        return "Node.js";
      case DeploymentType.GO:
        return "Go";
      default:
        return "Custom";
    }
  };

  const formatBuildTime = (seconds: number) => {
    if (!seconds || seconds === 0) return "0s";
    if (seconds < 60) {
      return `${seconds}s`;
    }
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
  };

  const getEnvironmentLabel = (env: string | EnvEnum | number) => {
    if (typeof env === "number") {
      switch (env) {
        case EnvEnum.PRODUCTION:
          return "Production";
        case EnvEnum.STAGING:
          return "Staging";
        case EnvEnum.DEVELOPMENT:
          return "Development";
        default:
          return "Environment";
      }
    }

    return String(env);
  };

  // Fetch deployments via Nuxt's useAsyncData so the request runs during SSR and
  // the transport injected by the server plugin is available.
  const client = useConnectClient(DeploymentService);
  const deploymentActions = useDeploymentActions();

  // Get organizationId using SSR-compatible composable
  const organizationId = useOrganizationId();

  const { data: deployments, refresh: refreshDeployments } = await useAsyncData(
    () => `deployments-list-${organizationId.value}`,
    async () => {
      try {
        // Use organizationId from composable (SSR-compatible)
        const response = await client.listDeployments({
          organizationId: organizationId.value || undefined,
        });
        return response.deployments;
      } catch (error) {
        console.error("Failed to list deployments:", error);
        listError.value = error as Error;
        return [];
      }
    }
  );

  // Custom refresh function that preserves existing data during fetch
  const refreshDeploymentsWithoutClearing = async () => {
    try {
      const response = await client.listDeployments({});
      // Only update if we got a successful response
      deployments.value = response.deployments;
      listError.value = null;
    } catch (error) {
      console.error("Failed to refresh deployments:", error);
      // Don't clear the data on error, just log it
      // Don't set listError here to avoid breaking the UI
    }
  };

  // Periodic refresh - faster when deployments are building/deploying
  const hasActiveDeployments = computed(() => {
    return (deployments.value ?? []).some(
      (d) =>
        d.status === DeploymentStatus.BUILDING ||
        d.status === DeploymentStatus.DEPLOYING
    );
  });

  // Use faster refresh (3 seconds) when there are active deployments, slower (30 seconds) otherwise
  const refreshIntervalMs = computed(() => (hasActiveDeployments.value ? 3000 : 30000));

  // Track page visibility using VueUse
  const visibility = useDocumentVisibility();
  const isVisible = computed(() => visibility.value === "visible");

  // Periodic refresh - use ref to store interval ID so we can restart when interval changes
  const refreshIntervalId = ref<ReturnType<typeof setInterval> | null>(null);

  // Function to setup/restart the interval
  const setupRefreshInterval = () => {
    // Clear existing interval if any
    if (refreshIntervalId.value) {
      clearInterval(refreshIntervalId.value);
      refreshIntervalId.value = null;
    }

    // Only setup if page is visible
    if (isVisible.value && !listError.value) {
      refreshIntervalId.value = setInterval(async () => {
        if (isVisible.value && !listError.value) {
          await refreshDeploymentsWithoutClearing();
        }
      }, refreshIntervalMs.value);
    }
  };

  // Watch for interval changes (e.g., when active deployments change)
  watch([refreshIntervalMs, isVisible], () => {
    setupRefreshInterval();
  });

  // Start refreshing when component is mounted
  onMounted(() => {
    setupRefreshInterval();
  });

  // Cleanup on unmount
  onUnmounted(() => {
    if (refreshIntervalId.value) {
      clearInterval(refreshIntervalId.value);
      refreshIntervalId.value = null;
    }
  });

  // Track build progress for each deployment
  const buildProgressMap = new Map<string, ReturnType<typeof useBuildProgress>>();

  // Get or create build progress tracker for a deployment
  const getBuildProgress = (deploymentId: string) => {
    if (!buildProgressMap.has(deploymentId)) {
      const progress = useBuildProgress({
        deploymentId,
        organizationId: effectiveOrgId.value,
      });
      buildProgressMap.set(deploymentId, progress);
    }
    return buildProgressMap.get(deploymentId)!;
  };

  // Create a reactive map of progress values for template access
  const progressValues = reactive<Record<string, number>>({});
  const progressPhases = reactive<Record<string, string>>({});

  // Helper to update reactive progress values
  const updateProgressValues = () => {
    buildProgressMap.forEach((progress, id) => {
      progressValues[id] = progress.progress.value;
      progressPhases[id] = progress.currentPhase.value;
    });
  };

  // Track watchers to avoid creating duplicates
  const progressWatchers = new Map<string, { progressWatcher: () => void; phaseWatcher: () => void }>();

  // Watch for progress changes and update reactive values
  watch(
    () => deployments.value,
    () => {
      deployments.value?.forEach((deployment) => {
        // Skip if watchers already exist for this deployment
        if (progressWatchers.has(deployment.id)) {
          return;
        }

        const progress = getBuildProgress(deployment.id);
        
        // Set up watchers for each deployment's progress
        const progressWatcher = watch(
          () => progress.progress.value,
          (newValue) => {
            progressValues[deployment.id] = newValue;
          },
          { immediate: true }
        );
        
        const phaseWatcher = watch(
          () => progress.currentPhase.value,
          (newValue) => {
            progressPhases[deployment.id] = newValue;
          },
          { immediate: true }
        );

        progressWatchers.set(deployment.id, { progressWatcher, phaseWatcher });
      });

      // Clean up watchers for deployments that no longer exist
      const currentIds = new Set(deployments.value?.map((d) => d.id) || []);
      progressWatchers.forEach((watchers, id) => {
        if (!currentIds.has(id)) {
          watchers.progressWatcher();
          watchers.phaseWatcher();
          progressWatchers.delete(id);
        }
      });
    },
    { immediate: true, deep: true }
  );

  // Sync statusFilter with query params
  const router = useRouter();
  
  // Update query param when statusFilter changes (user changes filter in UI)
  watch(statusFilter, (newStatus) => {
    const currentStatus = route.query.status;
    if (newStatus !== currentStatus) {
      router.replace({
        query: {
          ...route.query,
          status: newStatus || undefined, // Remove query param if empty
        },
      });
    }
  });

  // Update statusFilter when query param changes (browser navigation, shared links)
  watch(
    () => route.query.status,
    (statusParam) => {
      const newStatus = typeof statusParam === "string" ? statusParam : "";
      if (newStatus !== statusFilter.value) {
        statusFilter.value = newStatus;
      }
    }
  );

  // Watch deployments and enable/disable progress tracking based on status
  watch(
    () => deployments.value,
    (newDeployments) => {
      if (!newDeployments) return;

      newDeployments.forEach((deployment) => {
        const isBuilding =
          deployment.status === DeploymentStatus.BUILDING ||
          deployment.status === DeploymentStatus.DEPLOYING;
        const isFailed = deployment.status === DeploymentStatus.FAILED;
        const progress = getBuildProgress(deployment.id);

        if (isBuilding && !progress.isStreaming.value) {
          // Start streaming when deployment enters building state
          progress.startStreaming();
        } else if ((!isBuilding || isFailed) && progress.isStreaming.value) {
          // Stop streaming when deployment is no longer building or has failed
          progress.stopStreaming();
          // If failed, update phase immediately; otherwise reset after delay
          if (isFailed) {
            // Update phase to show failure, but keep progress at current value
            // Also ensure we update reactive values immediately
            const currentProgress = progress.progress.value;
            progressValues[deployment.id] = currentProgress > 0 ? currentProgress : progress.progress.value;
            progressPhases[deployment.id] = progress.currentPhase.value || "Build failed";
          } else {
            // Reset after a delay to allow final progress update
            setTimeout(() => {
              progress.reset();
              progressValues[deployment.id] = 0;
              progressPhases[deployment.id] = "Starting deployment...";
            }, 1000);
          }
        } else if (isFailed && !isBuilding && !progress.isStreaming.value) {
          // If deployment is already failed and we're not streaming, ensure UI is updated
          // This handles the case where status changed to FAILED before we started tracking
          const currentProgress = progress.progress.value;
          if (currentProgress > 0 || progressValues[deployment.id] === undefined) {
            progressValues[deployment.id] = currentProgress > 0 ? currentProgress : 0;
            progressPhases[deployment.id] = progress.currentPhase.value || "Build failed";
          }
        }
      });

      // Clean up progress trackers for deployments that no longer exist
      const currentIds = new Set(newDeployments.map((d) => d.id));
      buildProgressMap.forEach((progress, id) => {
        if (!currentIds.has(id)) {
          progress.stopStreaming();
          buildProgressMap.delete(id);
        }
      });
    },
    { immediate: true, deep: true }
  );
  const cleanRepositoryName = (url: string) => {
    if (!url) return "";

    try {
      const parsed = new URL(url);
      const repoPath = parsed.pathname.replace(/\.git$/, "").replace(/^\//, "");
      return repoPath || parsed.hostname;
    } catch (error) {
      return url.replace(/^https?:\/\//, "").replace(/\.git$/, "");
    }
  };


  const filteredDeployments = computed<Deployment[]>(() => {
    let filtered: Deployment[] = (deployments.value ?? []) as Deployment[];

    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      filtered = filtered.filter((deployment) => {
        const nameMatch = deployment.name.toLowerCase().includes(query);
        const domainMatch = deployment.domain.toLowerCase().includes(query);
        const frameworkLabel = getTypeLabel((deployment as any).type)
          .toLowerCase()
          .includes(query);
        return nameMatch || domainMatch || frameworkLabel;
      });
    }

    if (statusFilter.value) {
      const filterStatus = Number(statusFilter.value);
      filtered = filtered.filter(
        (deployment) => deployment.status === filterStatus
      );
    }

    if (environmentFilter.value) {
      const filterEnv = Number(environmentFilter.value);
      filtered = filtered.filter((deployment) => {
        const deploymentEnv =
          typeof (deployment as any).environment === "number"
            ? (deployment as any).environment
            : EnvEnum.ENVIRONMENT_UNSPECIFIED;
        return deploymentEnv === filterEnv;
      });
    }

    if (groupFilter.value) {
      filtered = filtered.filter((deployment) => {
        const deploymentGroups = (deployment as any).groups || [];
        // Also check legacy group field for backward compatibility
        const legacyGroup = (deployment as any).group;
        if (legacyGroup) {
          deploymentGroups.push(legacyGroup);
        }
        return deploymentGroups.some((g: string) => g && g.trim() === groupFilter.value.trim());
      });
    }

    return filtered;
  });

  const stopDeployment = async (id: string) => {
    await deploymentActions.stopDeployment(id, deployments.value ?? []);
    // Refresh to get latest status from server (preserve data during refresh)
    await refreshDeploymentsWithoutClearing();
  };

  const startDeployment = async (id: string) => {
    await deploymentActions.startDeployment(id, deployments.value ?? []);
    // Refresh to get latest status from server (preserve data during refresh)
    await refreshDeploymentsWithoutClearing();
  };

  const redeployApp = async (id: string) => {
    await deploymentActions.redeployDeployment(id, deployments.value ?? []);
    // Refresh to get latest status from server (preserve data during refresh)
    await refreshDeploymentsWithoutClearing();
  };

  const viewDeployment = (id: string) => {
    const router = useRouter();
    router.push(`/deployments/${id}`);
  };

  const openUrl = (domain: string) => {
    window.open(`https://${domain}`, "_blank");
  };

  const { showAlert } = useDialog();


  const createDeployment = async () => {
    if (!newDeployment.value.name.trim()) {
      await showAlert({
        title: "Validation Error",
        message: "Please enter a project name",
      });
      return;
    }

    // Clear previous errors
    createError.value = null;

    try {
      const deployment = await deploymentActions.createDeployment({
        name: newDeployment.value.name,
        environment: Number(newDeployment.value.environment) as EnvEnum,
        groups: newDeploymentGroups.value.filter(g => g.trim()),
      });

      showCreateDialog.value = false;
      // Reset form for next time
      newDeployment.value = {
        name: "",
        environment: String(EnvEnum.PRODUCTION),
      };
      newDeploymentGroups.value = [];

      // Add to local deployments list if it's not there already
      if (
        deployment &&
        !deployments.value?.find((d) => d.id === deployment.id)
      ) {
        deployments.value = [...(deployments.value || []), deployment];
      }

      // Navigate to the detail page to finish configuration
      if (deployment) {
        const router = useRouter();
        router.push(`/deployments/${deployment.id}`);
      }
    } catch (error) {
      console.error("Failed to create deployment:", error);
      createError.value = error as Error;
    }
  };

  const composeFromGitHub = ref<string | null>(null);

  const handleComposeFromGitHub = (composeContent: string) => {
    // Store compose content for when deployment is created
    composeFromGitHub.value = composeContent;
  };
</script>

<style scoped>
  @keyframes shimmer {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(300%);
    }
  }
</style>
