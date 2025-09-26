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

    <div v-else class="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
      <OuiCard
        v-for="deployment in filteredDeployments"
        :key="deployment.id"
        variant="raised"
        hoverable
        class="group"
      >
        <OuiCardHeader>
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <h3 class="oui-card-title truncate">{{ deployment.name }}</h3>
              <p class="oui-card-description truncate">{{ deployment.domain }}</p>
            </div>
            <OuiBadge :variant="getStatusVariant(deployment.status)">
              {{ deployment.status }}
            </OuiBadge>
          </div>
        </OuiCardHeader>

        <OuiCardBody class="space-y-3">
          <div class="flex items-center text-sm text-secondary">
            <CodeBracketIcon class="h-4 w-4 mr-2" />
            <span class="truncate">{{ deployment.framework }}</span>
          </div>

          <div class="flex items-center text-sm text-secondary">
            <CalendarIcon class="h-4 w-4 mr-2" />
            <span>{{ formatDate(deployment.lastDeployedAt) }}</span>
          </div>

          <div class="flex items-center text-sm text-secondary">
            <CpuChipIcon class="h-4 w-4 mr-2" />
            <span>{{ deployment.environment }}</span>
          </div>

          <!-- Quick Stats -->
          <div class="grid grid-cols-2 gap-4 pt-2 border-t border-muted">
            <div class="text-center">
              <div class="text-lg font-semibold text-primary">{{ deployment.buildTime }}s</div>
              <div class="text-xs text-secondary">Build Time</div>
            </div>
            <div class="text-center">
              <div class="text-lg font-semibold text-primary">{{ deployment.size }}</div>
              <div class="text-xs text-secondary">Bundle Size</div>
            </div>
          </div>
        </OuiCardBody>

        <OuiCardFooter>
          <div class="flex items-center justify-between">
            <div class="flex space-x-2">
              <OuiButton
                v-if="deployment.status === 'RUNNING'"
                variant="ghost"
                size="sm"
                @click="stopDeployment(deployment.id)"
                title="Stop deployment"
              >
                <StopIcon class="w-4 h-4" />
              </OuiButton>

              <OuiButton
                v-if="deployment.status === 'STOPPED'"
                variant="ghost"
                size="sm"
                @click="startDeployment(deployment.id)"
                title="Start deployment"
              >
                <PlayIcon class="w-4 h-4" />
              </OuiButton>

              <OuiButton
                variant="ghost"
                size="sm"
                @click="redeployApp(deployment.id)"
                title="Redeploy"
              >
                <ArrowPathIcon class="w-4 h-4" />
              </OuiButton>
            </div>

            <div class="flex space-x-2">
              <OuiButton
                variant="ghost"
                size="sm"
                @click="openUrl(deployment.domain)"
                title="Open site"
              >
                <ArrowTopRightOnSquareIcon class="w-4 h-4" />
              </OuiButton>

              <OuiButton
                variant="ghost"
                size="sm"
                @click="viewDeployment(deployment.id)"
                title="View details"
              >
                <EyeIcon class="w-4 h-4" />
              </OuiButton>
            </div>
          </div>
        </OuiCardFooter>
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

// Status badge variant helper
const getStatusVariant = (status: string) => {
  switch (status) {
    case 'RUNNING':
      return 'success';
    case 'STOPPED':
      return 'secondary';
    case 'BUILDING':
      return 'warning';
    case 'FAILED':
      return 'danger';
    default:
      return 'secondary';
  }
};

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
