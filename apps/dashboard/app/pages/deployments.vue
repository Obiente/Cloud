<template>
  <OuiContainer size="7xl" py="xl" class="min-h-screen">
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
          <OuiText color="secondary" size="base">
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

      <OuiCard
        variant="raised"
        class="backdrop-blur-sm border border-border-muted/60"
      >
        <OuiCardBody>
          <OuiGrid cols="1" cols-md="3" gap="md">
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
          class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-surface-muted/50 ring-1 ring-border-muted"
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
                searchQuery || statusFilter || environmentFilter
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
          variant="raised"
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
                </OuiStack>
                <OuiFlex gap="sm" justify="end">
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
                      {{ deployment.framework }}
                    </OuiText>
                  </OuiFlex>
                  <OuiFlex
                    align="center"
                    gap="xs"
                    class="text-xs text-secondary"
                  >
                    <CalendarIcon class="h-3.5 w-3.5" />
                    <span>{{
                      formatRelativeTime(deployment.lastDeployedAt)
                    }}</span>
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
                        class="h-4 w-4 text-secondary flex-shrink-0"
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
                    class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wide ml-auto"
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
                    class="group/stat relative overflow-hidden bg-surface-muted/40 ring-1 ring-border-muted backdrop-blur-sm transition-all hover:bg-surface-muted/60 hover:ring-border-default"
                  >
                    <div
                      class="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity"
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
                        {{ deployment.buildTime }}
                        <OuiText
                          as="span"
                          size="base"
                          color="secondary"
                          weight="semibold"
                        >
                          s
                        </OuiText>
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
                      class="absolute inset-0 bg-gradient-to-br from-secondary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity"
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
                        {{ deployment.size }}
                      </OuiText>
                    </OuiStack>
                  </OuiBox>
                </OuiGrid>

                <OuiBox
                  v-if="deployment.status === 'BUILDING'"
                  p="md"
                  rounded="xl"
                  class="border backdrop-blur-sm"
                  :class="getStatusMeta(deployment.status).progressClass"
                >
                  <OuiStack gap="sm">
                    <OuiFlex
                      align="center"
                      gap="sm"
                      class="text-xs font-bold uppercase tracking-wider"
                    >
                      <Cog6ToothIcon class="h-4 w-4 animate-spin" />
                      <span>Building deployment</span>
                    </OuiFlex>
                    <div
                      class="relative h-2 w-full overflow-hidden rounded-full bg-warning/20"
                    >
                      <div class="absolute inset-0 flex">
                        <div
                          class="h-full w-1/3 animate-pulse rounded-full bg-gradient-to-r from-transparent via-warning to-transparent"
                          style="
                            animation: shimmer 2s infinite;
                            background-size: 200% 100%;
                          "
                        />
                      </div>
                    </div>
                  </OuiStack>
                </OuiBox>
              </OuiStack>
            </OuiCardBody>

            <OuiCardFooter class="mt-auto">
              <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
                <OuiButton
                  v-if="deployment.status === 'RUNNING'"
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
                  v-if="deployment.status === 'STOPPED'"
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
                  title="Redeploy"
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
                  @click="viewDeployment(deployment.id)"
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
            <OuiInput
              v-model="newDeployment.name"
              label="Project Name"
              placeholder="my-awesome-app"
              required
            />

            <OuiInput
              v-model="newDeployment.repositoryUrl"
              label="Repository URL"
              placeholder="https://github.com/username/repo"
              required
            />

            <OuiSelect
              v-model="newDeployment.framework"
              :items="frameworkOptions"
              label="Framework"
              placeholder="Select framework"
            />

            <OuiSelect
              v-model="newDeployment.environment"
              :items="environmentOptions"
              label="Environment"
              placeholder="Select environment"
            />
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
} from "@heroicons/vue/24/outline";

const searchQuery = ref("");
const statusFilter = ref("");
const environmentFilter = ref("");
const showCreateDialog = ref(false);

const newDeployment = ref({
  name: "",
  repositoryUrl: "",
  framework: "",
  environment: "",
});

const statusFilterOptions = [
  { label: "All Status", value: "" },
  { label: "Running", value: "RUNNING" },
  { label: "Stopped", value: "STOPPED" },
  { label: "Building", value: "BUILDING" },
  { label: "Failed", value: "FAILED" },
];

const environmentOptions = [
  { label: "Production", value: "production" },
  { label: "Staging", value: "staging" },
  { label: "Development", value: "development" },
];

const frameworkOptions = [
  { label: "Next.js", value: "nextjs" },
  { label: "Nuxt.js", value: "nuxtjs" },
  { label: "React (Vite)", value: "react-vite" },
  { label: "Vue.js (Vite)", value: "vue-vite" },
  { label: "Static HTML", value: "static" },
  { label: "Node.js", value: "nodejs" },
];

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
    badge: "success",
    label: "Unknown",
    description: "Status information is unavailable.",
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

type StatusKey = keyof typeof STATUS_META;

const getStatusMeta = (status: string) =>
  STATUS_META[status as StatusKey] ?? STATUS_META.DEFAULT;

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

const getEnvironmentMeta = (environment: string) =>
  ENVIRONMENT_META[environment as EnvironmentKey] ?? ENVIRONMENT_META.DEFAULT;

const deployments = ref([
  {
    id: "1",
    name: "My Portfolio",
    domain: "portfolio.obiente.cloud",
    repositoryUrl: "https://github.com/user/portfolio",
    status: "RUNNING",
    lastDeployedAt: new Date("2024-01-15T10:30:00Z"),
    framework: "Next.js",
    environment: "production",
    buildTime: 45,
    size: "2.1MB",
  },
  {
    id: "2",
    name: "E-commerce Site",
    domain: "shop.obiente.cloud",
    repositoryUrl: "https://github.com/user/ecommerce",
    status: "BUILDING",
    lastDeployedAt: new Date("2024-01-14T14:20:00Z"),
    framework: "Nuxt.js",
    environment: "production",
    buildTime: 67,
    size: "3.4MB",
  },
  {
    id: "3",
    name: "Blog",
    domain: "blog.obiente.cloud",
    repositoryUrl: "https://github.com/user/blog",
    status: "STOPPED",
    lastDeployedAt: new Date("2024-01-13T09:15:00Z"),
    framework: "Static HTML",
    environment: "staging",
    buildTime: 12,
    size: "850KB",
  },
  {
    id: "4",
    name: "Dashboard App",
    domain: "dashboard.obiente.cloud",
    repositoryUrl: "https://github.com/user/dashboard",
    status: "RUNNING",
    lastDeployedAt: new Date("2024-01-16T08:45:00Z"),
    framework: "Vue.js (Vite)",
    environment: "development",
    buildTime: 32,
    size: "1.8MB",
  },
]);

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

const formatRelativeTime = (date: Date) => {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;

  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
  }).format(date);
};

const filteredDeployments = computed(() => {
  let filtered = deployments.value;

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter(
      (deployment) =>
        deployment.name.toLowerCase().includes(query) ||
        deployment.domain.toLowerCase().includes(query) ||
        deployment.framework.toLowerCase().includes(query)
    );
  }

  if (statusFilter.value) {
    filtered = filtered.filter(
      (deployment) => deployment.status === statusFilter.value
    );
  }

  if (environmentFilter.value) {
    filtered = filtered.filter(
      (deployment) => deployment.environment === environmentFilter.value
    );
  }

  return filtered;
});

const stopDeployment = (id: string) => {
  const deployment = deployments.value.find((d) => d.id === id);
  if (deployment) {
    deployment.status = "STOPPED";
  }
};

const startDeployment = (id: string) => {
  const deployment = deployments.value.find((d) => d.id === id);
  if (deployment) {
    deployment.status = "BUILDING";
    setTimeout(() => {
      const dep = deployments.value.find((d) => d.id === id);
      if (dep) dep.status = "RUNNING";
    }, 2000);
  }
};

const redeployApp = (id: string) => {
  const deployment = deployments.value.find((d) => d.id === id);
  if (deployment) {
    deployment.status = "BUILDING";
    deployment.lastDeployedAt = new Date();
    setTimeout(() => {
      const dep = deployments.value.find((d) => d.id === id);
      if (dep) dep.status = "RUNNING";
    }, 3000);
  }
};

const viewDeployment = (id: string) => {
  navigateTo(`/deployments/${id}`);
};

const openUrl = (domain: string) => {
  window.open(`https://${domain}`, "_blank");
};

const createDeployment = () => {
  console.log("Creating deployment:", newDeployment.value);

  newDeployment.value = {
    name: "",
    repositoryUrl: "",
    framework: "",
    environment: "",
  };
  showCreateDialog.value = false;
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
