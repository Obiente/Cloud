<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <!-- Stats Grid -->
      <OuiGrid cols="1" :cols-md="2" gap="md">
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Domain</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <Icon name="uil:globe" class="h-4 w-4 text-secondary" />
            <OuiText size="sm" weight="medium">{{ deployment.domain }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Framework</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <CodeBracketIcon class="h-4 w-4 text-primary" />
            <OuiText size="sm" weight="medium">{{
              getTypeLabel((deployment as any).type)
            }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Environment</OuiText
          >
          <OuiFlex align="center" gap="sm" class="mt-1">
            <CpuChipIcon class="h-4 w-4 text-secondary" />
            <OuiText size="sm" weight="medium">{{
              getEnvironmentLabel(deployment.environment)
            }}</OuiText>
          </OuiFlex>
        </OuiBox>
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30"
        >
          <OuiText
            size="xs"
            color="secondary"
            transform="uppercase"
            weight="bold"
            >Build Time</OuiText
          >
          <OuiText size="lg" weight="bold">{{ deployment.buildTime }}s</OuiText>
        </OuiBox>
      </OuiGrid>

      <!-- Deployment Settings -->
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

            <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
            <OuiText v-if="saveSuccess" size="xs" color="success">Settings saved successfully!</OuiText>

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
                      @update:branch="(branch) => { config.branch = branch; markDirty(); }"
                      @update:integrationId="handleIntegrationIdChange"
                      @compose-loaded="handleComposeFromGitHub"
                    />
                    
                    <!-- Branch field is handled by GitHubRepoPicker when connected -->
                    
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
                @update:model-value="markDirty"
              />
                <!-- Branch field for manual repository -->
                <OuiInput
                  v-model="config.branch"
                  label="Branch"
                  placeholder="main"
                  @update:model-value="markDirty"
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
              />
              <!-- Branch field - different based on repository source -->
              <template v-if="repositorySource === 'manual'">
              <OuiInput
                v-model="config.branch"
                label="Branch"
                placeholder="main"
                @update:model-value="markDirty"
              />
              </template>
              <!-- GitHub branch is handled by GitHubRepoPicker component -->
            </OuiGrid>
            
            <!-- Install and Build Commands - only for strategies that need them -->
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
                @update:model-value="markDirty"
              />
              <OuiInput
                v-model="config.buildCommand"
                :label="buildCommandLabel"
                :placeholder="buildCommandPlaceholder"
                @update:model-value="markDirty"
              />
            </OuiGrid>
            
            <!-- Dockerfile path input (only for DOCKERFILE strategy) -->
            <OuiGrid cols="1" v-if="buildStrategy === BuildStrategy.DOCKERFILE">
              <OuiInput
                v-model="config.dockerfilePath"
                label="Dockerfile Path"
                placeholder="Dockerfile (default: ./Dockerfile)"
                helper-text="Path to Dockerfile relative to repository root (e.g., 'Dockerfile', 'backend/Dockerfile', 'docker/Dockerfile.prod')"
                @update:model-value="markDirty"
              />
            </OuiGrid>
            
            <!-- Compose file path input (for PLAIN_COMPOSE and COMPOSE_REPO strategies) -->
            <OuiGrid cols="1" v-if="buildStrategy === BuildStrategy.PLAIN_COMPOSE || buildStrategy === BuildStrategy.COMPOSE_REPO">
              <OuiInput
                v-model="config.composeFilePath"
                label="Compose File Path"
                placeholder="docker-compose.yml (auto-detected if empty)"
                helper-text="Path to compose file relative to repository root (e.g., 'docker-compose.yml', 'compose/production.yml'). Leave empty to auto-detect."
                @update:model-value="markDirty"
              />
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watchEffect, computed, watch, onMounted } from "vue";
import { CodeBracketIcon, CpuChipIcon, LinkIcon } from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import { DeploymentType, Environment as EnvEnum, BuildStrategy } from "@obiente/proto";
import { useDeploymentActions } from "~/composables/useDeploymentActions";
import { useRoute, useRouter } from "vue-router";
import { useOrganizationsStore } from "~/stores/organizations";
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
const organizationId = computed(() => orgsStore.currentOrgId || "");
const isConfigDirty = ref(false);
const isSaving = ref(false);
const error = ref("");
const saveSuccess = ref(false);
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

// Watch buildStrategy changes to mark dirty
watch(buildStrategy, (newVal, oldVal) => {
  // Only mark dirty if value actually changed
  if (newVal !== oldVal) {
    markDirty();
  }
});

// Initialize from deployment
watchEffect(() => {
  if (props.deployment) {
    // Set repositoryUrl - check for both repositoryUrl and repository_url (proto conversion)
    const repoUrl = props.deployment.repositoryUrl || (props.deployment as any).repository_url || "";
    config.repositoryUrl = repoUrl;
    
    // Preserve empty strings for branch (don't default to "main" as empty string is falsy in JS)
    config.branch = props.deployment.branch !== undefined && props.deployment.branch !== null 
      ? props.deployment.branch 
      : "main";
    // Use deployment build strategy or default
    buildStrategy.value = props.deployment.buildStrategy != null 
      ? props.deployment.buildStrategy
      : BuildStrategy.BUILD_STRATEGY_UNSPECIFIED;
    config.installCommand = props.deployment.installCommand ?? "";
    config.buildCommand = props.deployment.buildCommand ?? "";
    config.dockerfilePath = props.deployment.dockerfilePath ?? "";
    config.composeFilePath = props.deployment.composeFilePath ?? "";
    // Initialize GitHub integration ID
    githubIntegrationId.value = props.deployment.githubIntegrationId ?? "";
    isConfigDirty.value = false;
    saveSuccess.value = false;
    error.value = "";
    
    // Determine repository source and extract GitHub repo if applicable
    // Only update if we have a repository URL, otherwise keep existing source
    if (config.repositoryUrl && config.repositoryUrl.includes("github.com")) {
      repositorySource.value = "github";
      const match = config.repositoryUrl.match(/github\.com\/([^\/]+\/[^\/]+)/);
      if (match && match[1]) {
        selectedGitHubRepo.value = match[1].replace(/\.git$/, "");
      }
    } else if (config.repositoryUrl) {
      repositorySource.value = "manual";
    }
    // If no repository URL, keep the existing repositorySource value (don't reset)
  }
});

const handleGitHubRepoSelected = (repoFullName: string) => {
  if (repoFullName) {
    config.repositoryUrl = `https://github.com/${repoFullName}`;
    markDirty();
  }
};

const handleIntegrationIdChange = (id: string) => {
  githubIntegrationId.value = id;
  markDirty();
};

watch(selectedGitHubRepo, (repo) => {
  if (repo && repositorySource.value === "github") {
    config.repositoryUrl = `https://github.com/${repo}`;
    markDirty();
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
  markDirty();
});

const handleComposeFromGitHub = (composeContent: string) => {
  // Emit event to parent to update compose tab
  console.log("Compose loaded from GitHub:", composeContent.length, "bytes");
};

const navigateToGitHubSettings = () => {
  router.push("/settings?tab=integrations&provider=github");
};

// Check GitHub connection status (user or org)
const checkGitHubConnection = async () => {
  try {
    const { useConnectClient } = await import("~/lib/connect-client");
    const { AuthService, ListGitHubIntegrationsRequestSchema } = await import("@obiente/proto");
    const { create } = await import("@bufbuild/protobuf");
    
    const client = useConnectClient(AuthService);
    const request = create(ListGitHubIntegrationsRequestSchema, {});
    const response = await client.listGitHubIntegrations(request);
    
    // Check if user has GitHub connection or current org has GitHub connection
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

// Watch organizationId and check GitHub connection when it changes
watch(organizationId, () => {
  checkGitHubConnection();
}, { immediate: true });

onMounted(() => {
  checkGitHubConnection();
});

const markDirty = () => {
  isConfigDirty.value = true;
  saveSuccess.value = false;
  error.value = "";
};

const buildStrategyOptions = [
  { label: "Auto-detect", value: BuildStrategy.BUILD_STRATEGY_UNSPECIFIED },
  { label: "Railpacks", value: BuildStrategy.RAILPACKS },
  { label: "Nixpacks", value: BuildStrategy.NIXPACKS },
  { label: "Dockerfile", value: BuildStrategy.DOCKERFILE },
  { label: "Plain Compose", value: BuildStrategy.PLAIN_COMPOSE },
  { label: "Compose from Repository", value: BuildStrategy.COMPOSE_REPO },
  { label: "Static Site", value: BuildStrategy.STATIC_SITE },
];

// Determine if install/build commands should be shown
const showInstallBuildCommands = computed(() => {
  return buildStrategy.value !== BuildStrategy.PLAIN_COMPOSE && buildStrategy.value !== BuildStrategy.COMPOSE_REPO 
    && buildStrategy.value !== BuildStrategy.STATIC_SITE
    && buildStrategy.value !== BuildStrategy.DOCKERFILE;
});

// Get command labels and placeholders based on deployment type
const installCommandLabel = computed(() => {
  const type = (props.deployment as any)?.type || DeploymentType.DOCKER;
  switch (type) {
    case DeploymentType.NODE:
      return "Install Command";
    case DeploymentType.PYTHON:
      return "Install Command";
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
  const type = (props.deployment as any)?.type || DeploymentType.DOCKER;
  switch (type) {
    case DeploymentType.NODE:
      return "Build Command";
    case DeploymentType.PYTHON:
      return "Build Command";
    case DeploymentType.RUBY:
      return "Build Command";
    default:
      return "Build Command";
  }
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

async function saveConfig() {
  if (isSaving.value) return;
  
  isSaving.value = true;
  error.value = "";
  saveSuccess.value = false;

  try {
    await deploymentActions.updateDeployment(String(route.params.id), {
      // Send empty strings explicitly (not undefined) to properly update/clear fields
      repositoryUrl: config.repositoryUrl !== undefined && config.repositoryUrl !== null
        ? (config.repositoryUrl || "")
        : undefined,
      branch: config.branch !== undefined && config.branch !== null
        ? config.branch  // Allow empty strings
        : undefined,
      // Send build strategy (number enum)
      buildStrategy: buildStrategy.value !== BuildStrategy.BUILD_STRATEGY_UNSPECIFIED
        ? buildStrategy.value
        : undefined,
      buildCommand: config.buildCommand || undefined,
      installCommand: config.installCommand || undefined,
      dockerfilePath: config.dockerfilePath || undefined,
      composeFilePath: config.composeFilePath || undefined,
      githubIntegrationId: githubIntegrationId.value || undefined,
    });
    
    // Refresh deployment data
    await refreshNuxtData(`deployment-${route.params.id}`);
    
    isConfigDirty.value = false;
    saveSuccess.value = true;
    
    // Hide success message after 3 seconds
    setTimeout(() => {
      saveSuccess.value = false;
    }, 3000);
  } catch (err: any) {
    console.error("Failed to save config:", err);
    error.value = err.message || "Failed to save settings. Please try again.";
  } finally {
    isSaving.value = false;
  }
}
</script>