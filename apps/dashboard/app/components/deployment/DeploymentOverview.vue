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
                      @update:model-value="handleGitHubRepoSelected"
                      @update:branch="(branch) => { config.branch = branch; markDirty(); }"
                      @compose-loaded="handleComposeFromGitHub"
                    />
                    
                    <OuiText v-else size="xs" color="secondary">
                      Connect your GitHub account to select repositories directly.
                    </OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>

              <!-- Manual URL Input -->
              <OuiInput
                v-if="repositorySource === 'manual'"
                v-model="config.repositoryUrl"
                label="Repository URL"
                placeholder="https://github.com/org/repo or https://gitlab.com/org/repo"
                @update:model-value="markDirty"
              />
            </OuiStack>

            <!-- Branch and Commands -->
            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiInput
                v-model="config.branch"
                label="Branch"
                placeholder="main"
                @update:model-value="markDirty"
              />
              <OuiSelect
                v-model="config.runtime"
                :items="runtimeOptions"
                label="Runtime"
                placeholder="Select runtime"
                @update:model-value="markDirty"
                disabled
              />
            </OuiGrid>
            
            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiInput
                v-model="config.installCommand"
                label="Install Command"
                placeholder="pnpm install"
                @update:model-value="markDirty"
              />
              <OuiInput
                v-model="config.buildCommand"
                label="Build Command"
                placeholder="pnpm build"
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
import { ref, reactive, watchEffect, computed, watch } from "vue";
import { CodeBracketIcon, CpuChipIcon, LinkIcon } from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import { DeploymentType, Environment as EnvEnum } from "@obiente/proto";
import { useDeploymentActions } from "~/composables/useDeploymentActions";
import { useRoute, useRouter } from "vue-router";
import GitHubRepoPicker from "./GitHubRepoPicker.vue";
import OuiRadioGroup from "~/components/oui/RadioGroup.vue";

interface Props {
  deployment: Deployment;
}

const props = defineProps<Props>();
const route = useRoute();
const router = useRouter();
const deploymentActions = useDeploymentActions();
const isConfigDirty = ref(false);
const isSaving = ref(false);
const error = ref("");
const saveSuccess = ref(false);
const repositorySource = ref<"github" | "manual">("manual");
const selectedGitHubRepo = ref("");
const isGitHubConnected = ref(false); // TODO: Check from store/API

const config = reactive({
  repositoryUrl: "",
  branch: "main",
  runtime: "node",
  installCommand: "",
  buildCommand: "",
});

// Initialize from deployment
watchEffect(() => {
  if (props.deployment) {
    config.repositoryUrl = props.deployment.repositoryUrl ?? "";
    config.branch = props.deployment.branch ?? "main";
    config.installCommand = props.deployment.installCommand ?? "";
    config.buildCommand = props.deployment.buildCommand ?? "";
    isConfigDirty.value = false;
    saveSuccess.value = false;
    error.value = "";
    
    // Determine repository source and extract GitHub repo if applicable
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
    markDirty();
  }
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

const markDirty = () => {
  isConfigDirty.value = true;
  saveSuccess.value = false;
  error.value = "";
};

const runtimeOptions = [
  { label: "Node.js", value: "node" },
  { label: "Go", value: "go" },
  { label: "Docker", value: "docker" },
  { label: "Static", value: "static" },
];

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
      repositoryUrl: config.repositoryUrl || undefined,
      branch: config.branch,
      buildCommand: config.buildCommand || undefined,
      installCommand: config.installCommand || undefined,
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