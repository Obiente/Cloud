<template>
  <div class="p-6">
    <OuiStack gap="lg">
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

      <!-- GitHub Integration -->
      <OuiCard v-if="!config.repositoryUrl" variant="subtle">
        <OuiCardBody>
          <GitHubRepoPicker
            v-model="selectedGitHubRepo"
            v-model:branch="selectedGitHubBranch"
            @compose-loaded="handleComposeFromGitHub"
          />
        </OuiCardBody>
      </OuiCard>

      <!-- Configuration -->
      <OuiStack gap="md">
        <OuiText as="h3" size="md" weight="semibold">Configuration</OuiText>
        <OuiInput
          v-model="config.repositoryUrl"
          label="Repository URL"
          placeholder="https://github.com/org/repo or select from GitHub"
          @update:model-value="markDirty"
        />
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
        <OuiFlex justify="end">
          <OuiButton
            @click="saveConfig"
            :disabled="!isConfigDirty"
            size="sm"
          >
            Save Changes
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watchEffect, watch } from "vue";
import { CodeBracketIcon, CpuChipIcon } from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import { DeploymentType, Environment as EnvEnum } from "@obiente/proto";
import { useDeploymentActions } from "~/composables/useDeploymentActions";
import { useRoute } from "vue-router";
import GitHubRepoPicker from "~/components/deployments/GitHubRepoPicker.vue";

interface Props {
  deployment: Deployment;
}

const props = defineProps<Props>();
const route = useRoute();
const deploymentActions = useDeploymentActions();
const isConfigDirty = ref(false);
const selectedGitHubRepo = ref("");
const selectedGitHubBranch = ref("");

const config = reactive({
  repositoryUrl: "",
  branch: "main",
  runtime: "node",
  installCommand: "",
  buildCommand: "",
});

watchEffect(() => {
  if (props.deployment) {
    config.repositoryUrl = props.deployment.repositoryUrl ?? "";
    config.branch = props.deployment.branch ?? "main";
    config.installCommand = props.deployment.installCommand ?? "";
    config.buildCommand = props.deployment.buildCommand ?? "";
    isConfigDirty.value = false;
    
    // Extract GitHub repo from URL if present
    if (config.repositoryUrl && config.repositoryUrl.includes("github.com")) {
      const match = config.repositoryUrl.match(/github\.com\/([^\/]+\/[^\/]+)/);
      if (match) {
        selectedGitHubRepo.value = match[1].replace(/\.git$/, "");
      }
    }
  }
});

watch(selectedGitHubRepo, (repo) => {
  if (repo) {
    config.repositoryUrl = `https://github.com/${repo}`;
    markDirty();
  }
});

watch(selectedGitHubBranch, (branch) => {
  if (branch) {
    config.branch = branch;
    markDirty();
  }
});

const handleComposeFromGitHub = (composeContent: string) => {
  // Emit event to parent to update compose tab
  // This could trigger a notification or auto-switch to compose tab
  console.log("Compose loaded from GitHub:", composeContent.length, "bytes");
};

const markDirty = () => {
  isConfigDirty.value = true;
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
  try {
    await deploymentActions.updateDeployment(String(route.params.id), {
      branch: config.branch,
      buildCommand: config.buildCommand,
      installCommand: config.installCommand,
    });
    await refreshNuxtData(`deployment-${route.params.id}`);
    isConfigDirty.value = false;
  } catch (error) {
    console.error("Failed to save config:", error);
  }
}
</script>

