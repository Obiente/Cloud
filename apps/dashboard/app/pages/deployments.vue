<template>
  <div class="container flex flex-col col-auto columns-4 ">
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-3xl font-bold text-primary">Deployments</h1>
        <p class="text-secondary mt-2">Manage your web application deployments</p>
      </div>
      <OuiButton variant="primary" @click="showCreateDialog = true">
        <PlusIcon class="w-4 h-4 mr-2" />
        New Deployment
      </OuiButton>
    </div>

    <!-- Filters and Search -->
    <OuiCard class="mb-6">
      <OuiCardBody>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <OuiInput v-model="searchQuery" placeholder="Search deployments..." clearable>
            <template #prefix>
              <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
            </template>
          </OuiInput>

          <OuiSelect v-model="statusFilter" :items="statusFilterOptions" placeholder="All Status" />

          <OuiCombobox
            v-model="environmentFilter"
            :options="environmentOptions"
            placeholder="All Environments"
            clearable
          />
        </div>
      </OuiCardBody>
    </OuiCard>

    <!-- Deployments Grid -->
    <div v-if="filteredDeployments.length === 0" class="text-center py-16">
      <RocketLaunchIcon class="mx-auto h-16 w-16 text-secondary mb-4" />
      <h3 class="text-xl font-medium text-primary mb-2">No deployments found</h3>
      <p class="text-secondary mb-6">
        {{
          searchQuery || statusFilter || environmentFilter
            ? 'Try adjusting your filters to see more results.'
            : 'Get started by creating your first deployment.'
        }}
      </p>
      <OuiButton variant="primary" @click="showCreateDialog = true">
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
        class="group relative overflow-hidden border border-transparent transition-all duration-300 ease-out hover:-translate-y-1 hover:border-primary/20 hover:shadow-2xl focus-within:border-primary/40 focus-within:ring-2 focus-within:ring-primary/20"
        :class="getStatusMeta(deployment.status).ring"
      >
        <span
          class="pointer-events-none absolute inset-x-0 -top-12 h-32 rounded-b-[44%] bg-gradient-to-br opacity-70 transition-all duration-500 group-hover:opacity-100"
          :class="getStatusMeta(deployment.status).gradient"
        />
        <span
          class="pointer-events-none absolute -inset-px rounded-2xl opacity-0 blur-3xl transition-all duration-700 group-hover:opacity-100"
          :class="getStatusMeta(deployment.status).halo"
        />

        <div class="relative flex h-full flex-col">
          <OuiCardHeader class="pb-6">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1 space-y-2">
                <div
                  class="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-wide"
                  :class="getStatusMeta(deployment.status).text"
                >
                  <component
                    :is="getStatusMeta(deployment.status).icon"
                    class="h-4 w-4"
                    :class="getStatusMeta(deployment.status).iconClass"
                  />
                  <span>{{ getStatusMeta(deployment.status).label }}</span>
                </div>

                <div class="flex items-center gap-2">
                  <h3 class="oui-card-title truncate text-lg font-semibold">
                    {{ deployment.name }}
                  </h3>
                  <span v-if="deployment.environment === 'production'" class="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-600">
                    Live
                  </span>
                </div>

                <p class="flex items-center gap-2 text-sm text-secondary">
                  <span class="inline-flex items-center gap-1 rounded-full border border-muted bg-muted px-2.5 py-1 text-xs font-medium text-primary transition-colors group-hover:border-primary">
                    <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5 opacity-70" />
                    {{ deployment.domain }}
                  </span>
                </p>
              </div>

              <OuiBadge
                :variant="getStatusMeta(deployment.status).badge"
                class="rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-wide"
              >
                <span class="flex items-center gap-2">
                  <span class="h-2 w-2 rounded-full" :class="getStatusMeta(deployment.status).dot" />
                  {{ deployment.status }}
                </span>
              </OuiBadge>
            </div>
          </OuiCardHeader>

          <OuiCardBody class="flex-1 space-y-6">
            <div class="grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
              <div class="flex items-center gap-2 text-secondary">
                <CodeBracketIcon class="h-4 w-4 text-primary" />
                <span class="truncate font-medium text-primary">
                  {{ deployment.framework }}
                </span>
              </div>
              <div class="flex items-center gap-2 text-secondary sm:justify-end">
                <CalendarIcon class="h-4 w-4 text-primary" />
                <span class="text-xs uppercase tracking-wide">Last deployed</span>
                <span class="text-sm font-medium text-primary">
                  {{ formatDate(deployment.lastDeployedAt) }}
                </span>
              </div>
            </div>

            <div class="flex flex-wrap gap-2">
              <span
                class="inline-flex items-center gap-2 rounded-full bg-muted px-3 py-1 text-xs font-semibold uppercase tracking-wide text-secondary"
              >
                <CpuChipIcon class="h-4 w-4 text-primary" />
                {{ formatEnvironment(deployment.environment) }}
              </span>

              <span
                v-if="deployment.repositoryUrl"
                class="inline-flex items-center gap-2 rounded-full border border-muted px-3 py-1 text-xs font-medium text-secondary transition-colors hover:border-primary hover:text-primary"
              >
                <span class="h-2 w-2 rounded-full bg-primary" />
                {{ cleanRepositoryName(deployment.repositoryUrl) }}
              </span>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="rounded-xl border border-muted bg-muted p-4 shadow-inner">
                <p class="text-xs font-semibold uppercase tracking-wide text-secondary">Build Time</p>
                <p class="mt-1 text-2xl font-semibold text-primary">{{ deployment.buildTime }}s</p>
              </div>
              <div class="rounded-xl border border-muted bg-muted p-4 shadow-inner">
                <p class="text-xs font-semibold uppercase tracking-wide text-secondary">Bundle Size</p>
                <p class="mt-1 text-2xl font-semibold text-primary">{{ deployment.size }}</p>
              </div>
            </div>

            <div
              v-if="deployment.status === 'BUILDING'"
              class="space-y-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-amber-700 dark:border-amber-500/40 dark:bg-amber-500/10 dark:text-amber-200"
            >
              <div class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide">
                <Cog6ToothIcon class="h-4 w-4 animate-spin" />
                <span>Deployment in progress</span>
              </div>
              <div class="h-1.5 w-full overflow-hidden rounded-full bg-amber-200/70 dark:bg-amber-500/30">
                <div class="h-full w-full animate-pulse rounded-full bg-amber-500/80 dark:bg-amber-300/70" />
              </div>
            </div>
          </OuiCardBody>

          <OuiCardFooter class="mt-auto border-t border-muted pt-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <OuiButton
                  v-if="deployment.status === 'RUNNING'"
                  variant="ghost"
                  size="sm"
                  @click="stopDeployment(deployment.id)"
                  title="Stop deployment"
                  :aria-label="`Stop deployment ${deployment.name}`"
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
                >
                  <PlayIcon class="h-4 w-4" />
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="redeployApp(deployment.id)"
                  title="Redeploy deployment"
                  :aria-label="`Redeploy deployment ${deployment.name}`"
                >
                  <ArrowPathIcon class="h-4 w-4" />
                </OuiButton>
              </div>

              <div class="flex items-center gap-2">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="openUrl(deployment.domain)"
                  title="Open deployed site"
                  :aria-label="`Open ${deployment.domain} in a new tab`"
                >
                  <ArrowTopRightOnSquareIcon class="h-4 w-4" />
                </OuiButton>

                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="viewDeployment(deployment.id)"
                  title="View deployment details"
                  :aria-label="`View ${deployment.name} details`"
                >
                  <EyeIcon class="h-4 w-4" />
                </OuiButton>
              </div>
            </div>
          </OuiCardFooter>
        </div>
      </OuiCard>
    </div>

    <!-- Create Deployment Dialog -->
    <OuiDialog
      v-model:open="showCreateDialog"
      title="Create New Deployment"
      description="Deploy your application to Obiente Cloud"
    >
      <form @submit.prevent="createDeployment" class="space-y-4">
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
        <OuiButton variant="ghost" @click="showCreateDialog = false"> Cancel </OuiButton>
        <OuiButton variant="primary" @click="createDeployment">
          <RocketLaunchIcon class="w-4 h-4 mr-2" />
          Deploy Now
        </OuiButton>
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
} from '@heroicons/vue/24/outline';

// Reactive state
const searchQuery = ref('');
const statusFilter = ref('');
const environmentFilter = ref('');
const showCreateDialog = ref(false);

// New deployment form
const newDeployment = ref({
  name: '',
  repositoryUrl: '',
  framework: '',
  environment: '',
});

// Filter options
const statusFilterOptions = [
  { label: 'All Status', value: '' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Stopped', value: 'STOPPED' },
  { label: 'Building', value: 'BUILDING' },
  { label: 'Failed', value: 'FAILED' },
];

const environmentOptions = [
  { label: 'Production', value: 'production' },
  { label: 'Staging', value: 'staging' },
  { label: 'Development', value: 'development' },
];

const frameworkOptions = [
  { label: 'Next.js', value: 'nextjs' },
  { label: 'Nuxt.js', value: 'nuxtjs' },
  { label: 'React (Vite)', value: 'react-vite' },
  { label: 'Vue.js (Vite)', value: 'vue-vite' },
  { label: 'Static HTML', value: 'static' },
  { label: 'Node.js', value: 'nodejs' },
];

const STATUS_META = {
  RUNNING: {
    badge: 'success',
    gradient: 'from-emerald-400/30 via-emerald-400/10 to-transparent',
    halo: 'bg-emerald-400/20',
    ring: 'ring-1 ring-emerald-500/20',
    text: 'text-emerald-600 dark:text-emerald-300',
    icon: BoltIcon,
    iconClass: 'text-emerald-500 dark:text-emerald-300',
    label: 'Running smoothly',
    dot: 'bg-emerald-400',
  },
  STOPPED: {
    badge: 'secondary',
    gradient: 'from-slate-400/20 via-slate-400/10 to-transparent',
    halo: 'bg-slate-400/15',
    ring: 'ring-1 ring-slate-400/20',
    text: 'text-slate-600 dark:text-slate-300',
    icon: PauseCircleIcon,
    iconClass: 'text-slate-400 dark:text-slate-300',
    label: 'Deployment paused',
    dot: 'bg-slate-400',
  },
  BUILDING: {
    badge: 'warning',
    gradient: 'from-amber-400/30 via-amber-400/15 to-transparent',
    halo: 'bg-amber-400/15',
    ring: 'ring-1 ring-amber-400/30',
    text: 'text-amber-600 dark:text-amber-300',
    icon: Cog6ToothIcon,
    iconClass: 'text-amber-500 dark:text-amber-300',
    label: 'Building new release',
    dot: 'bg-amber-400',
  },
  FAILED: {
    badge: 'danger',
    gradient: 'from-rose-500/25 via-rose-500/10 to-transparent',
    halo: 'bg-rose-500/20',
    ring: 'ring-1 ring-rose-500/30',
    text: 'text-rose-600 dark:text-rose-300',
    icon: ExclamationTriangleIcon,
    iconClass: 'text-rose-500 dark:text-rose-300',
    label: 'Deployment failed',
    dot: 'bg-rose-500',
  },
  DEFAULT: {
    badge: 'secondary',
    gradient: 'from-slate-400/15 via-slate-400/5 to-transparent',
    halo: 'bg-slate-400/15',
    ring: 'ring-1 ring-slate-400/15',
    text: 'text-slate-600 dark:text-slate-400',
    icon: InformationCircleIcon,
    iconClass: 'text-slate-400 dark:text-slate-300',
    label: 'Status unknown',
    dot: 'bg-slate-400',
  },
} as const;

// Mock data for now
const deployments = ref([
  {
    id: '1',
    name: 'My Portfolio',
    domain: 'portfolio.obiente.cloud',
    repositoryUrl: 'https://github.com/user/portfolio',
    status: 'RUNNING',
    lastDeployedAt: new Date('2024-01-15T10:30:00Z'),
    framework: 'Next.js',
    environment: 'production',
    buildTime: 45,
    size: '2.1MB',
  },
  {
    id: '2',
    name: 'E-commerce Site',
    domain: 'shop.obiente.cloud',
    repositoryUrl: 'https://github.com/user/ecommerce',
    status: 'BUILDING',
    lastDeployedAt: new Date('2024-01-14T14:20:00Z'),
    framework: 'Nuxt.js',
    environment: 'production',
    buildTime: 67,
    size: '3.4MB',
  },
  {
    id: '3',
    name: 'Blog',
    domain: 'blog.obiente.cloud',
    repositoryUrl: 'https://github.com/user/blog',
    status: 'STOPPED',
    lastDeployedAt: new Date('2024-01-13T09:15:00Z'),
    framework: 'Static HTML',
    environment: 'staging',
    buildTime: 12,
    size: '850KB',
  },
  {
    id: '4',
    name: 'Dashboard App',
    domain: 'dashboard.obiente.cloud',
    repositoryUrl: 'https://github.com/user/dashboard',
    status: 'RUNNING',
    lastDeployedAt: new Date('2024-01-16T08:45:00Z'),
    framework: 'Vue.js (Vite)',
    environment: 'development',
    buildTime: 32,
    size: '1.8MB',
  },
]);

const getStatusMeta = (status: string) =>
  STATUS_META[status as keyof typeof STATUS_META] ?? STATUS_META.DEFAULT;

const formatEnvironment = (environment: string) =>
  environment ? environment.charAt(0).toUpperCase() + environment.slice(1) : 'Unknown';

const cleanRepositoryName = (url: string) => {
  if (!url) return '';

  try {
    const parsed = new URL(url);
    const repoPath = parsed.pathname.replace(/\.git$/, '').replace(/^\//, '');
    return repoPath || parsed.hostname;
  } catch (error) {
    return url.replace(/^https?:\/\//, '').replace(/\.git$/, '');
  }
};

const filteredDeployments = computed(() => {
  let filtered = deployments.value;

  // Apply search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter(
      deployment =>
        deployment.name.toLowerCase().includes(query) ||
        deployment.domain.toLowerCase().includes(query) ||
        deployment.framework.toLowerCase().includes(query)
    );
  }

  // Apply status filter
  if (statusFilter.value) {
    filtered = filtered.filter(deployment => deployment.status === statusFilter.value);
  }

  // Apply environment filter
  if (environmentFilter.value) {
    filtered = filtered.filter(deployment => deployment.environment === environmentFilter.value);
  }

  return filtered;
});

// Actions
const stopDeployment = (id: string) => {
  const deployment = deployments.value.find(d => d.id === id);
  if (deployment) {
    deployment.status = 'STOPPED';
  }
};

const startDeployment = (id: string) => {
  const deployment = deployments.value.find(d => d.id === id);
  if (deployment) {
    deployment.status = 'BUILDING';
    // Simulate transition to running after a delay
    setTimeout(() => {
      const dep = deployments.value.find(d => d.id === id);
      if (dep) dep.status = 'RUNNING';
    }, 2000);
  }
};

const redeployApp = (id: string) => {
  const deployment = deployments.value.find(d => d.id === id);
  if (deployment) {
    deployment.status = 'BUILDING';
    deployment.lastDeployedAt = new Date();
    // Simulate transition to running after a delay
    setTimeout(() => {
      const dep = deployments.value.find(d => d.id === id);
      if (dep) dep.status = 'RUNNING';
    }, 3000);
  }
};

const viewDeployment = (id: string) => {
  navigateTo(`/deployments/${id}`);
};

const openUrl = (domain: string) => {
  window.open(`https://${domain}`, '_blank');
};

const createDeployment = () => {
  // TODO: Implement actual deployment creation
  console.log('Creating deployment:', newDeployment.value);

  // Reset form and close dialog
  newDeployment.value = {
    name: '',
    repositoryUrl: '',
    framework: '',
    environment: '',
  };
  showCreateDialog.value = false;
};

const formatDate = (date: Date) => {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
};
</script>
