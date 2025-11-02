<template>
  <div class="p-6">
    <OuiStack gap="xl">
      <!-- General Settings -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiFlex justify="between" align="center">
              <OuiText as="h3" size="md" weight="semibold">General Settings</OuiText>
              <OuiButton
                @click="saveGeneralSettings"
                :disabled="!isGeneralDirty || isSaving"
                size="sm"
                variant="solid"
              >
                {{ isSaving ? "Saving..." : "Save Changes" }}
              </OuiButton>
            </OuiFlex>

            <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
            <OuiText v-if="saveSuccess" size="xs" color="success">Settings saved successfully!</OuiText>

            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiSelect
                v-model="localEnvironment"
                :items="environmentOptions"
                label="Environment"
                @update:model-value="markGeneralDirty"
              />

              <div>
                <OuiText size="sm" weight="medium" class="mb-2 block">Groups/Labels</OuiText>
                <div class="flex flex-wrap gap-2 mb-2 p-2 border border-border-default rounded-lg min-h-[2.5rem]">
                  <OuiBadge
                    v-for="(group, idx) in localGroups"
                    :key="idx"
                    variant="outline"
                    size="sm"
                    class="gap-1"
                  >
                    {{ group }}
                    <button
                      @click="removeGroup(idx)"
                      class="ml-1 hover:text-danger"
                      type="button"
                    >
                      <span class="sr-only">Remove</span>
                      ×
                    </button>
                  </OuiBadge>
                  <input
                    v-model="newGroupInput"
                    @keydown.enter.prevent="addGroup"
                    @blur="addGroup"
                    type="text"
                    placeholder="Add group..."
                    class="flex-1 min-w-[120px] border-0 outline-none bg-transparent text-sm"
                  />
                </div>
                <OuiText size="xs" color="secondary">
                  Press Enter or click outside to add a group/label
                </OuiText>
              </div>
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Deployment Configuration -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiFlex justify="between" align="center">
              <OuiText as="h3" size="md" weight="semibold">Deployment Settings</OuiText>
              <OuiButton
                @click="saveConfig"
                :disabled="!isConfigDirty || isSaving"
                size="sm"
                variant="solid"
              >
                {{ isSaving ? "Saving..." : "Save Changes" }}
              </OuiButton>
            </OuiFlex>

            <OuiText v-if="configError" size="xs" color="danger">{{ configError }}</OuiText>
            <OuiText v-if="configSuccess" size="xs" color="success">Settings saved successfully!</OuiText>

            <!-- Connected Repository Display -->
            <OuiCard 
              v-if="config.repositoryUrl && config.repositoryUrl.trim()" 
              variant="outline" 
              class="border-success/20 bg-success/5"
            >
              <OuiCardBody>
                <OuiStack gap="sm">
                  <OuiFlex align="center" gap="sm">
                    <Icon 
                      name="uil:check-circle" 
                      class="h-5 w-5 text-success shrink-0" 
                    />
                    <OuiText size="sm" weight="semibold" color="success">
                      Repository Connected
                    </OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm" class="pl-7">
                    <Icon
                      name="uil:github"
                      class="h-4 w-4 text-secondary shrink-0"
                    />
                    <OuiText 
                      size="sm" 
                      class="font-mono text-secondary"
                      truncate
                    >
                      {{ config.repositoryUrl }}
                    </OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm" class="pl-7" v-if="config.branch">
                    <Icon
                      name="uil:code-branch"
                      class="h-4 w-4 text-secondary shrink-0"
                    />
                    <OuiText size="sm" color="secondary">
                      Branch: <span class="font-mono">{{ config.branch }}</span>
                    </OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Repository Source Selection -->
            <OuiStack gap="md">
              <OuiText as="h4" size="sm" weight="semibold">Repository Source</OuiText>
              
              <OuiRadioGroup 
                v-model="repositorySource"
                :options="[
                  { label: 'Connect from GitHub', value: 'github' },
                  { label: 'Enter URL manually', value: 'manual' }
                ]"
                @update:model-value="markConfigDirty"
              />

              <!-- GitHub Connection Card -->
              <OuiCard v-if="repositorySource === 'github'" variant="outline" class="border-default">
                <OuiCardBody>
                  <OuiStack gap="md">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" weight="medium">GitHub Integration</OuiText>
                      <OuiButton
                        variant="ghost"
                        size="sm"
                        @click="navigateToGitHubSettings"
                      >
                        <LinkIcon class="h-4 w-4 mr-1" />
                        Connect Account
                      </OuiButton>
                    </OuiFlex>
                    
                    <GitHubRepoPicker
                      v-if="isGitHubConnected"
                      :model-value="selectedGitHubRepo"
                      :branch="config.branch"
                      :organization-id="organizationId"
                      @update:model-value="handleGitHubRepoSelected"
                      @update:branch="(branch) => { config.branch = branch; markConfigDirty(); }"
                      @update:integrationId="handleIntegrationIdChange"
                      @compose-loaded="handleComposeFromGitHub"
                    />
                    
                    <OuiText v-else size="xs" color="secondary">
                      Connect your GitHub account to select repositories directly.
                    </OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>

              <!-- Manual URL Input -->
              <OuiStack v-if="repositorySource === 'manual'" gap="md">
                <OuiInput
                  v-model="config.repositoryUrl"
                  label="Repository URL"
                  placeholder="https://github.com/org/repo or https://gitlab.com/org/repo"
                  @update:model-value="markConfigDirty"
                />
                <OuiInput
                  v-model="config.branch"
                  label="Branch"
                  placeholder="main"
                  @update:model-value="markConfigDirty"
                />
              </OuiStack>
            </OuiStack>

            <!-- Build Strategy -->
            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiSelect
                v-model="buildStrategy"
                :items="buildStrategyOptions"
                label="Build Strategy"
                placeholder="Select build strategy"
                @update:model-value="markConfigDirty"
              />
            </OuiGrid>
            
            <!-- Install and Build Commands -->
            <OuiGrid 
              v-if="showInstallBuildCommands"
              cols="1" 
              :cols-md="2" 
              gap="md"
            >
              <OuiInput
                v-model="config.installCommand"
                :label="installCommandLabel"
                :placeholder="installCommandPlaceholder"
                @update:model-value="markConfigDirty"
              />
              <OuiInput
                v-model="config.buildCommand"
                :label="buildCommandLabel"
                :placeholder="buildCommandPlaceholder"
                @update:model-value="markConfigDirty"
              />
            </OuiGrid>
            
            <!-- Dockerfile path input -->
            <OuiGrid cols="1" v-if="buildStrategy === BuildStrategy.DOCKERFILE">
              <OuiInput
                v-model="config.dockerfilePath"
                label="Dockerfile Path"
                placeholder="Dockerfile (default: ./Dockerfile)"
                helper-text="Path to Dockerfile relative to repository root (e.g., 'Dockerfile', 'backend/Dockerfile', 'docker/Dockerfile.prod')"
                @update:model-value="markConfigDirty"
              />
            </OuiGrid>
            
            <!-- Compose file path input -->
            <OuiGrid cols="1" v-if="buildStrategy === BuildStrategy.PLAIN_COMPOSE || buildStrategy === BuildStrategy.COMPOSE_REPO">
              <OuiInput
                v-model="config.composeFilePath"
                label="Compose File Path"
                placeholder="docker-compose.yml (auto-detected if empty)"
                helper-text="Path to compose file relative to repository root (e.g., 'docker-compose.yml', 'compose/production.yml'). Leave empty to auto-detect."
                @update:model-value="markConfigDirty"
              />
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Danger Zone -->
      <OuiCard variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText as="h3" size="lg" weight="semibold" color="danger">
                Danger Zone
              </OuiText>
              <OuiText size="sm" color="secondary">
                Irreversible and destructive actions
              </OuiText>
            </OuiStack>
            <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
              <OuiStack gap="xs" class="flex-1 min-w-0">
                <OuiText size="sm" weight="medium" color="primary">
                  Delete Deployment
                </OuiText>
                <OuiText size="xs" color="secondary">
                  Once you delete a deployment, there is no going back. This
                  will permanently remove the deployment and all associated
                  data.
                </OuiText>
              </OuiStack>
              <OuiButton
                variant="outline"
                color="danger"
                size="sm"
                @click="handleDelete"
                class="gap-2 shrink-0"
              >
                <TrashIcon class="h-4 w-4" />
                <OuiText as="span" size="xs" weight="medium">Delete Deployment</OuiText>
              </OuiButton>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watchEffect, computed, watch, onMounted } from "vue";
import { LinkIcon, TrashIcon } from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import { DeploymentType, Environment as EnvEnum, BuildStrategy } from "@obiente/proto";
import { useDeploymentActions } from "~/composables/useDeploymentActions";
import { useRoute, useRouter } from "vue-router";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";
import GitHubRepoPicker from "./GitHubRepoPicker.vue";
import OuiRadioGroup from "~/components/oui/RadioGroup.vue";

interface Props {
  deployment: Deployment;
}

const props = defineProps<Props>();
const route = useRoute();
const router = useRouter();
const deploymentActions = useDeploymentActions();
const orgsStore = useOrganizationsStore();
const { showAlert, showConfirm } = useDialog();
const organizationId = computed(() => orgsStore.currentOrgId || "");

// General settings (environment, groups)
const localEnvironment = ref<string>("");
const localGroups = ref<string[]>([]);
const newGroupInput = ref<string>("");
const isGeneralDirty = ref(false);
const isSaving = ref(false);
const error = ref("");
const saveSuccess = ref(false);

// Deployment config
const isConfigDirty = ref(false);
const configError = ref("");
const configSuccess = ref(false);
const repositorySource = ref<"github" | "manual">("manual");
const selectedGitHubRepo = ref("");
const githubIntegrationId = ref<string>("");
const isGitHubConnected = ref(false);
const buildStrategy = ref<BuildStrategy>(BuildStrategy.BUILD_STRATEGY_UNSPECIFIED);

const config = reactive({
  repositoryUrl: "",
  branch: "main",
  installCommand: "",
  buildCommand: "",
  dockerfilePath: "",
  composeFilePath: "",
});

const environmentOptions = [
  { label: "Production", value: String(EnvEnum.PRODUCTION) },
  { label: "Staging", value: String(EnvEnum.STAGING) },
  { label: "Development", value: String(EnvEnum.DEVELOPMENT) },
];

// Initialize from deployment
watchEffect(() => {
  if (props.deployment) {
    // General settings
    localEnvironment.value = String(props.deployment.environment ?? EnvEnum.PRODUCTION);
    const deploymentGroups = (props.deployment as any).groups || (props.deployment as any).group ? [(props.deployment as any).group].filter(Boolean) : [];
    localGroups.value = Array.isArray(deploymentGroups) ? deploymentGroups : [];
    
    // Config settings
    const repoUrl = props.deployment.repositoryUrl || (props.deployment as any).repository_url || "";
    config.repositoryUrl = repoUrl;
    config.branch = props.deployment.branch !== undefined && props.deployment.branch !== null 
      ? props.deployment.branch 
      : "main";
    buildStrategy.value = props.deployment.buildStrategy != null 
      ? props.deployment.buildStrategy
      : BuildStrategy.BUILD_STRATEGY_UNSPECIFIED;
    config.installCommand = props.deployment.installCommand ?? "";
    config.buildCommand = props.deployment.buildCommand ?? "";
    config.dockerfilePath = props.deployment.dockerfilePath ?? "";
    config.composeFilePath = props.deployment.composeFilePath ?? "";
    githubIntegrationId.value = props.deployment.githubIntegrationId ?? "";
    
    // Reset dirty flags
    isGeneralDirty.value = false;
    isConfigDirty.value = false;
    saveSuccess.value = false;
    configSuccess.value = false;
    error.value = "";
    configError.value = "";
    
    // Determine repository source
    if (config.repositoryUrl && config.repositoryUrl.includes("github.com")) {
      repositorySource.value = "github";
      const match = config.repositoryUrl.match(/github\.com\/([^\/]+\/[^\/]+)/);
      if (match && match[1]) {
        selectedGitHubRepo.value = match[1].replace(/\.git$/, "");
      }
    } else if (config.repositoryUrl) {
      repositorySource.value = "manual";
    }
  }
});

const handleGitHubRepoSelected = (repoFullName: string) => {
  if (repoFullName) {
    config.repositoryUrl = `https://github.com/${repoFullName}`;
    markConfigDirty();
  }
};

const handleIntegrationIdChange = (id: string) => {
  githubIntegrationId.value = id;
  markConfigDirty();
};

watch(selectedGitHubRepo, (repo) => {
  if (repo && repositorySource.value === "github") {
    config.repositoryUrl = `https://github.com/${repo}`;
    markConfigDirty();
  }
});

watch(() => repositorySource.value, () => {
  if (repositorySource.value === "manual") {
    selectedGitHubRepo.value = "";
  } else if (repositorySource.value === "github" && config.repositoryUrl) {
    const match = config.repositoryUrl.match(/github\.com\/([^\/]+\/[^\/]+)/);
    if (match && match[1]) {
      selectedGitHubRepo.value = match[1].replace(/\.git$/, "");
    }
  }
  markConfigDirty();
});

const handleComposeFromGitHub = (composeContent: string) => {
  console.log("Compose loaded from GitHub:", composeContent.length, "bytes");
};

const navigateToGitHubSettings = () => {
  router.push("/settings?tab=integrations&provider=github");
};

const checkGitHubConnection = async () => {
  try {
    const { useConnectClient } = await import("~/lib/connect-client");
    const { AuthService, ListGitHubIntegrationsRequestSchema } = await import("@obiente/proto");
    const { create } = await import("@bufbuild/protobuf");
    
    const client = useConnectClient(AuthService);
    const request = create(ListGitHubIntegrationsRequestSchema, {});
    const response = await client.listGitHubIntegrations(request);
    
    const hasUserConnection = response.integrations.some(i => i.isUser === true);
    const hasOrgConnection = organizationId.value && response.integrations.some(
      i => i.isUser === false && i.organizationId === organizationId.value
    );
    
    isGitHubConnected.value = hasUserConnection || (hasOrgConnection === true);
  } catch (err) {
    console.error("Failed to check GitHub connection:", err);
    isGitHubConnected.value = false;
  }
};

watch(organizationId, () => {
  checkGitHubConnection();
}, { immediate: true });

onMounted(() => {
  checkGitHubConnection();
});

const markGeneralDirty = () => {
  isGeneralDirty.value = true;
  saveSuccess.value = false;
  error.value = "";
};

const addGroup = () => {
  const trimmed = newGroupInput.value.trim();
  if (trimmed && !localGroups.value.includes(trimmed)) {
    localGroups.value.push(trimmed);
    newGroupInput.value = "";
    markGeneralDirty();
  }
};

const removeGroup = (index: number) => {
  localGroups.value.splice(index, 1);
  markGeneralDirty();
};

const markConfigDirty = () => {
  isConfigDirty.value = true;
  configSuccess.value = false;
  configError.value = "";
};

watch(buildStrategy, () => {
  markConfigDirty();
});

const buildStrategyOptions = [
  { label: "Auto-detect", value: BuildStrategy.BUILD_STRATEGY_UNSPECIFIED },
  { label: "Railpacks", value: BuildStrategy.RAILPACKS },
  { label: "Nixpacks", value: BuildStrategy.NIXPACKS },
  { label: "Dockerfile", value: BuildStrategy.DOCKERFILE },
  { label: "Plain Compose", value: BuildStrategy.PLAIN_COMPOSE },
  { label: "Compose from Repository", value: BuildStrategy.COMPOSE_REPO },
  { label: "Static Site", value: BuildStrategy.STATIC_SITE },
];

const showInstallBuildCommands = computed(() => {
  return buildStrategy.value !== BuildStrategy.PLAIN_COMPOSE && buildStrategy.value !== BuildStrategy.COMPOSE_REPO 
    && buildStrategy.value !== BuildStrategy.STATIC_SITE
    && buildStrategy.value !== BuildStrategy.DOCKERFILE;
});

const installCommandLabel = computed(() => {
  const type = (props.deployment as any)?.type || DeploymentType.DOCKER;
  switch (type) {
    case DeploymentType.NODE:
    case DeploymentType.PYTHON:
    case DeploymentType.RUBY:
      return "Install Command";
    default:
      return "Install Command";
  }
});

const installCommandPlaceholder = computed(() => {
  const type = (props.deployment as any)?.type || DeploymentType.DOCKER;
  switch (type) {
    case DeploymentType.NODE:
      return "npm install, pnpm install, or yarn install";
    case DeploymentType.PYTHON:
      return "pip install -r requirements.txt";
    case DeploymentType.RUBY:
      return "bundle install";
    case DeploymentType.GO:
      return "go mod download";
    default:
      return "npm install";
  }
});

const buildCommandLabel = computed(() => {
  return "Build Command";
});

const buildCommandPlaceholder = computed(() => {
  const type = (props.deployment as any)?.type || DeploymentType.DOCKER;
  switch (type) {
    case DeploymentType.NODE:
      return "npm run build, pnpm build, or yarn build";
    case DeploymentType.PYTHON:
      return "python setup.py build or no build step";
    case DeploymentType.RUBY:
      return "bundle exec rake assets:precompile";
    case DeploymentType.GO:
      return "go build -o app";
    default:
      return "npm run build";
  }
});

async function saveGeneralSettings() {
  if (isSaving.value) return;
  
  isSaving.value = true;
  error.value = "";
  saveSuccess.value = false;

  try {
    await deploymentActions.updateDeployment(String(route.params.id), {
      environment: Number(localEnvironment.value) as EnvEnum,
      groups: localGroups.value.filter(g => g.trim()),
    });
    
    await refreshNuxtData(`deployment-${route.params.id}`);
    
    isGeneralDirty.value = false;
    saveSuccess.value = true;
    
    setTimeout(() => {
      saveSuccess.value = false;
    }, 3000);
  } catch (err: any) {
    console.error("Failed to save general settings:", err);
    error.value = err.message || "Failed to save settings. Please try again.";
  } finally {
    isSaving.value = false;
  }
}

async function saveConfig() {
  if (isSaving.value) return;
  
  isSaving.value = true;
  configError.value = "";
  configSuccess.value = false;

  try {
    await deploymentActions.updateDeployment(String(route.params.id), {
      repositoryUrl: config.repositoryUrl !== undefined && config.repositoryUrl !== null
        ? (config.repositoryUrl || "")
        : undefined,
      branch: config.branch !== undefined && config.branch !== null
        ? config.branch
        : undefined,
      buildStrategy: buildStrategy.value !== BuildStrategy.BUILD_STRATEGY_UNSPECIFIED
        ? buildStrategy.value
        : undefined,
      buildCommand: config.buildCommand || undefined,
      installCommand: config.installCommand || undefined,
      dockerfilePath: config.dockerfilePath || undefined,
      composeFilePath: config.composeFilePath || undefined,
      githubIntegrationId: githubIntegrationId.value || undefined,
    });
    
    await refreshNuxtData(`deployment-${route.params.id}`);
    
    isConfigDirty.value = false;
    configSuccess.value = true;
    
    setTimeout(() => {
      configSuccess.value = false;
    }, 3000);
  } catch (err: any) {
    console.error("Failed to save config:", err);
    configError.value = err.message || "Failed to save settings. Please try again.";
  } finally {
    isSaving.value = false;
  }
}

const handleDelete = async () => {
  const confirmed = await showConfirm({
    title: "Delete Deployment",
    message: `Are you sure you want to delete "${props.deployment.name}"? This action cannot be undone and will permanently remove the deployment and all associated data.`,
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (confirmed) {
    try {
      await deploymentActions.deleteDeployment(String(route.params.id));
      router.push("/deployments");
    } catch (err: any) {
      await showAlert({
        title: "Failed to Delete",
        message: err.message || "Failed to delete deployment. Please try again.",
      });
    }
  }
};
</script>

