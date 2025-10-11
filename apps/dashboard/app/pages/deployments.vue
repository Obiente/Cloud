<template>
  <div class="min-h-screen">
    <div class="mb-8">
      <div class="flex items-start justify-between gap-6">
        <div class="flex-1">
          <div class="flex items-center gap-3 mb-2">
            <div class="p-2.5 rounded-xl bg-primary/10 ring-1 ring-primary/20">
              <RocketLaunchIcon class="w-6 h-6 text-primary" />
            </div>
            <div>
              <h1 class="text-3xl font-bold text-primary tracking-tight">
                Deployments
              </h1>
            </div>
          </div>
          <p class="text-secondary text-base ml-14">
            Manage and monitor your web application deployments
          </p>
        </div>
        <OuiButton 
          variant="primary" 
          @click="showCreateDialog = true"
          class="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
        >
          <PlusIcon class="w-4 h-4 mr-2" />
          New Deployment
        </OuiButton>
      </div>
    </div>

    <OuiCard variant="raised" class="mb-8 backdrop-blur-sm">
      <OuiCardBody>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
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
        </div>
      </OuiCardBody>
    </OuiCard>

    <div v-if="filteredDeployments.length === 0" class="text-center py-20">
      <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-surface-muted/50 ring-1 ring-border-muted mb-6">
        <RocketLaunchIcon class="h-10 w-10 text-secondary" />
      </div>
      <h3 class="text-xl font-semibold text-primary mb-2">
        No deployments found
      </h3>
      <p class="text-secondary max-w-md mx-auto mb-8">
        {{
          searchQuery || statusFilter || environmentFilter
            ? "Try adjusting your filters to see more results."
            : "Get started by creating your first deployment."
        }}
      </p>
      <OuiButton 
        variant="primary" 
        @click="showCreateDialog = true"
        class="shadow-lg shadow-primary/20"
      >
        <PlusIcon class="w-4 h-4 mr-2" />
        Create Your First Deployment
      </OuiButton>
    </div>

    <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3">
      <OuiCard
        v-for="deployment in filteredDeployments"
        :key="deployment.id"
        variant="raised"
        hoverable
        :data-status="deployment.status"
        class="group relative overflow-hidden transition-all duration-300 hover:shadow-2xl"
        :class="[
          getStatusMeta(deployment.status).cardClass,
          'before:absolute before:inset-0 before:rounded-lg before:p-[1px] before:-z-10',
          'before:bg-gradient-to-br before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300',
          getStatusMeta(deployment.status).beforeGradient
        ]"
      >
        <div 
          class="absolute top-0 left-0 right-0 h-1"
          :class="getStatusMeta(deployment.status).barClass"
        />

        <div class="relative flex h-full flex-col">
          <OuiCardHeader>
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1 space-y-3">
                <div class="flex items-center gap-2">
                  <OuiBadge
                    :variant="getStatusMeta(deployment.status).badge"
                    class="inline-flex items-center px-2.5 py-1 text-xs font-semibold uppercase tracking-wide rounded-full"
                    :class="deployment.status === 'BUILDING' ? 'gap-1.5' : 'gap-1'"
                  >
                    <span
                      class="inline-flex h-1.5 w-1.5 rounded-full"
                      :class="[
                        getStatusMeta(deployment.status).dotClass,
                        deployment.status === 'RUNNING' || deployment.status === 'BUILDING' ? 'animate-pulse' : ''
                      ]"
                    />
                    {{ deployment.status }}
                  </OuiBadge>
                  
                  <span
                    v-if="deployment.environment === 'production'"
                    class="inline-flex items-center gap-1 rounded-full bg-success/10 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wide text-success ring-1 ring-success/20"
                  >
                    <BoltIcon class="h-3 w-3" />
                    Live
                  </span>
                </div>

                <div>
                  <h3 class="oui-card-title text-xl font-bold text-primary mb-1.5 truncate group-hover:text-primary/90 transition-colors">
                    {{ deployment.name }}
                  </h3>
                  
                  <a
                    :href="`https://${deployment.domain}`"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center gap-1.5 text-sm text-secondary hover:text-primary transition-colors group/link"
                    @click.stop
                  >
                    <span class="truncate max-w-[200px]">{{ deployment.domain }}</span>
                    <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5 opacity-0 group-hover/link:opacity-100 transition-opacity" />
                  </a>
                </div>
              </div>
            </div>
          </OuiCardHeader>

          <OuiCardBody class="flex-1 space-y-5">
            <div class="flex items-center justify-between text-sm">
              <div class="flex items-center gap-2">
                <div class="p-1.5 rounded-lg bg-surface-muted/50 ring-1 ring-border-muted">
                  <CodeBracketIcon class="h-4 w-4 text-primary" />
                </div>
                <span class="font-medium text-primary">
                  {{ deployment.framework }}
                </span>
              </div>
              <div class="flex items-center gap-1.5 text-xs text-secondary">
                <CalendarIcon class="h-3.5 w-3.5" />
                <span>{{ formatRelativeTime(deployment.lastDeployedAt) }}</span>
              </div>
            </div>

            <div 
              v-if="deployment.repositoryUrl"
              class="flex items-center gap-2 px-3 py-2 rounded-lg bg-surface-muted/30 ring-1 ring-border-muted"
            >
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <svg class="h-4 w-4 text-secondary flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                </svg>
                <span class="text-xs text-secondary truncate font-mono">
                  {{ cleanRepositoryName(deployment.repositoryUrl) }}
                </span>
              </div>
            </div>

            <div class="flex items-center gap-2">
              <span
                class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wide"
                :class="getEnvironmentClass(deployment.environment)"
              >
                <CpuChipIcon class="h-3.5 w-3.5" />
                {{ formatEnvironment(deployment.environment) }}
              </span>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="group/stat relative overflow-hidden rounded-xl bg-surface-muted/40 p-4 ring-1 ring-border-muted backdrop-blur-sm transition-all hover:bg-surface-muted/60 hover:ring-border-default">
                <div class="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity" />
                <div class="relative">
                  <p class="text-[10px] font-bold uppercase tracking-wider text-secondary mb-1.5">
                    Build Time
                  </p>
                  <p class="text-2xl font-bold text-primary">
                    {{ deployment.buildTime }}<span class="text-sm text-secondary ml-0.5">s</span>
                  </p>
                </div>
              </div>
              <div class="group/stat relative overflow-hidden rounded-xl bg-surface-muted/40 p-4 ring-1 ring-border-muted backdrop-blur-sm transition-all hover:bg-surface-muted/60 hover:ring-border-default">
                <div class="absolute inset-0 bg-gradient-to-br from-secondary/5 to-transparent opacity-0 group-hover/stat:opacity-100 transition-opacity" />
                <div class="relative">
                  <p class="text-[10px] font-bold uppercase tracking-wider text-secondary mb-1.5">
                    Bundle Size
                  </p>
                  <p class="text-2xl font-bold text-primary">
                    {{ deployment.size }}
                  </p>
                </div>
              </div>
            </div>

            <div
              v-if="deployment.status === 'BUILDING'"
              class="space-y-2.5 rounded-xl border p-4 backdrop-blur-sm"
              :class="getStatusMeta(deployment.status).progressClass"
            >
              <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider">
                <Cog6ToothIcon class="h-4 w-4 animate-spin" />
                <span>Building deployment</span>
              </div>
              <div class="relative h-2 w-full overflow-hidden rounded-full bg-warning/20">
                <div class="absolute inset-0 flex">
                  <div class="h-full w-1/3 animate-pulse rounded-full bg-gradient-to-r from-transparent via-warning to-transparent" 
                       style="animation: shimmer 2s infinite; background-size: 200% 100%;" />
                </div>
              </div>
            </div>
          </OuiCardBody>

          <OuiCardFooter class="mt-auto">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-1">
                <OuiButton
                  v-if="deployment.status === 'RUNNING'"
                  variant="ghost"
                  size="sm"
                  @click="stopDeployment(deployment.id)"
                  title="Stop deployment"
                  :aria-label="`Stop deployment ${deployment.name}`"
                  class="hover:bg-danger/10 hover:text-danger transition-colors"
                >
                  <StopIcon class="h-4 w-4" />
                </OuiButton>

                <OuiButton
                  v-if="deployment.status === 'STOPPED'"
                  variant="ghost"
                  size="sm"
                  @click="startDeployment(deployment.id)"
                  title="Start deployment"
                  :aria-label="`Start deployment ${deployment.name}`"
                  class="hover:bg-success/10 hover:text-success transition-colors"
                >
                  <PlayIcon class="h-4 w-4" />
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="redeployApp(deployment.id)"
                  title="Redeploy"
                  :aria-label="`Redeploy deployment ${deployment.name}`"
                  class="hover:bg-primary/10 hover:text-primary transition-colors"
                >
                  <ArrowPathIcon class="h-4 w-4" />
                </OuiButton>
              </div>

              <div class="flex items-center gap-1">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="openUrl(deployment.domain)"
                  title="Open site"
                  :aria-label="`Open ${deployment.domain} in a new tab`"
                  class="hover:bg-info/10 hover:text-info transition-colors"
                >
                  <ArrowTopRightOnSquareIcon class="h-4 w-4" />
                </OuiButton>

                <OuiButton
                  variant="primary"
                  size="sm"
                  @click="viewDeployment(deployment.id)"
                  title="View details"
                  :aria-label="`View ${deployment.name} details`"
                  class="shadow-sm hover:shadow-md transition-shadow"
                >
                  <EyeIcon class="h-4 w-4 mr-1.5" />
                  <span class="text-xs font-semibold">Details</span>
                </OuiButton>
              </div>
            </div>
          </OuiCardFooter>
        </div>
      </OuiCard>
    </div>

    <OuiDialog
      v-model:open="showCreateDialog"
      title="Create New Deployment"
      description="Deploy your application to Obiente Cloud with automated builds and deployments"
    >
      <form @submit.prevent="createDeployment" class="space-y-5">
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
      </form>

      <template #footer>
        <div class="flex items-center justify-end gap-3">
          <OuiButton variant="ghost" @click="showCreateDialog = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="primary" 
            @click="createDeployment"
            class="shadow-lg shadow-primary/20"
          >
            <RocketLaunchIcon class="w-4 h-4 mr-2" />
            Deploy Now
          </OuiButton>
        </div>
      </template>
    </OuiDialog>
  </div>
</template>

<script setup lang="ts">
import {
  RocketLaunchIcon,
  StopIcon,
  PlayIcon,
  ArrowPathIcon,
  EyeIcon,
  PlusIcon,
  MagnifyingGlassIcon,
  CodeBracketIcon,
  CalendarIcon,
  CpuChipIcon,
  ArrowTopRightOnSquareIcon,
  BoltIcon,
  PauseCircleIcon,
  Cog6ToothIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";

// Reactive state
const searchQuery = ref("");
const statusFilter = ref("");
const environmentFilter = ref("");
const showCreateDialog = ref(false);

// New deployment form
const newDeployment = ref({
  name: "",
  repositoryUrl: "",
  framework: "",
  environment: "",
});

// Filter options
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
    gradient: "from-success/40 via-success/10 to-transparent",
    halo: "bg-success/20",
    ring: "ring-1 ring-success/30",
    text: "text-success",
    icon: BoltIcon,
    iconClass: "text-success",
    label: "Running smoothly",
    dot: "bg-success",
    dotClass: "bg-success",
    barClass: "bg-gradient-to-r from-success to-success/70",
    cardClass: "hover:ring-1 hover:ring-success/30",
    beforeGradient: "before:from-success/20 before:to-success/10",
    progressClass: "border-success/30 bg-success/10 text-success",
  },
  STOPPED: {
    badge: "secondary",
    gradient: "from-secondary/30 via-secondary/10 to-transparent",
    halo: "bg-secondary/20",
    ring: "ring-1 ring-secondary/20",
    text: "text-secondary",
    icon: PauseCircleIcon,
    iconClass: "text-secondary",
    label: "Deployment paused",
    dot: "bg-secondary",
    dotClass: "bg-secondary",
    barClass: "bg-gradient-to-r from-secondary to-secondary/60",
    cardClass: "hover:ring-1 hover:ring-secondary/30",
    beforeGradient: "before:from-secondary/20 before:to-secondary/10",
    progressClass: "border-secondary/30 bg-secondary/10 text-secondary",
  },
  BUILDING: {
    badge: "warning",
    gradient: "from-warning/40 via-warning/20 to-transparent",
    halo: "bg-warning/20",
    ring: "ring-1 ring-warning/30",
    text: "text-warning",
    icon: Cog6ToothIcon,
    iconClass: "text-warning",
    label: "Building new release",
    dot: "bg-warning",
    dotClass: "bg-warning",
    barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
    cardClass: "hover:ring-1 hover:ring-warning/30",
    beforeGradient: "before:from-warning/20 before:to-warning/10",
    progressClass: "border-warning/30 bg-warning/10 text-warning",
  },
  FAILED: {
    badge: "danger",
    gradient: "from-danger/40 via-danger/20 to-transparent",
    halo: "bg-danger/20",
    ring: "ring-1 ring-danger/30",
    text: "text-danger",
    icon: ExclamationTriangleIcon,
    iconClass: "text-danger",
    label: "Deployment failed",
    dot: "bg-danger",
    dotClass: "bg-danger",
    barClass: "bg-gradient-to-r from-danger to-danger/60",
    cardClass: "hover:ring-1 hover:ring-danger/30",
    beforeGradient: "before:from-danger/20 before:to-danger/10",
    progressClass: "border-danger/30 bg-danger/10 text-danger",
  },
  DEFAULT: {
    badge: "secondary",
    gradient: "from-surface-muted/50 via-surface-muted/20 to-transparent",
    halo: "bg-surface-muted/30",
    ring: "ring-1 ring-border-muted",
    text: "text-secondary",
    icon: InformationCircleIcon,
    iconClass: "text-secondary",
    label: "Status unknown",
    dot: "bg-secondary",
    dotClass: "bg-secondary",
    barClass: "bg-gradient-to-r from-surface-muted to-surface-muted/70",
    cardClass: "hover:ring-1 hover:ring-border-muted",
    beforeGradient: "before:from-surface-muted/40 before:to-surface-muted/10",
    progressClass: "border-border-muted bg-surface-muted/40 text-secondary",
  },
} as const;

// Mock data for now
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

const getStatusMeta = (status: string) =>
  STATUS_META[status as keyof typeof STATUS_META] ?? STATUS_META.DEFAULT;

const formatEnvironment = (environment: string) =>
  environment
    ? environment.charAt(0).toUpperCase() + environment.slice(1)
    : "Unknown";

const getEnvironmentClass = (environment: string) => {
  const classes = {
    production: "bg-success/10 text-success ring-1 ring-success/20",
    staging: "bg-warning/10 text-warning ring-1 ring-warning/20",
    development: "bg-info/10 text-info ring-1 ring-info/20",
  };
  return classes[environment as keyof typeof classes] || "bg-surface-muted text-secondary ring-1 ring-border-muted";
};

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

  // Apply search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter(
      (deployment) =>
        deployment.name.toLowerCase().includes(query) ||
        deployment.domain.toLowerCase().includes(query) ||
        deployment.framework.toLowerCase().includes(query)
    );
  }

  // Apply status filter
  if (statusFilter.value) {
    filtered = filtered.filter(
      (deployment) => deployment.status === statusFilter.value
    );
  }

  // Apply environment filter
  if (environmentFilter.value) {
    filtered = filtered.filter(
      (deployment) => deployment.environment === environmentFilter.value
    );
  }

  return filtered;
});

// Actions
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
    // Simulate transition to running after a delay
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
    // Simulate transition to running after a delay
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
  // TODO: Implement actual deployment creation
  console.log("Creating deployment:", newDeployment.value);

  // Reset form and close dialog
  newDeployment.value = {
    name: "",
    repositoryUrl: "",
    framework: "",
    environment: "",
  };
  showCreateDialog.value = false;
};

const formatDate = (date: Date) => {
  return new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
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
