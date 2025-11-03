<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h4" size="sm" weight="semibold">GitHub Repository</OuiText>
      <OuiButton variant="ghost" size="sm" @click="refreshRepos" :disabled="isLoading">
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
        Refresh
      </OuiButton>
    </OuiFlex>

    <!-- Account/Integration Selector -->
    <OuiSelect
      v-if="isLoadingIntegrations"
      :items="[]"
      label="GitHub Account"
      placeholder="Loading accounts..."
      disabled
    />
    <OuiSelect
      v-else-if="availableIntegrations.length > 1"
      v-model="selectedIntegrationId"
      :items="integrationOptions"
      label="GitHub Account"
      placeholder="Select GitHub account..."
      @update:model-value="handleIntegrationChange"
    />
    <OuiText
      v-else-if="availableIntegrations.length === 0"
      size="xs"
      color="secondary"
    >
      No GitHub accounts available. Please connect a GitHub account in Settings.
    </OuiText>
    <OuiText
      v-else-if="availableIntegrations.length === 1"
      size="xs"
      color="secondary"
    >
      Using account: {{ availableIntegrations[0]?.username }} 
      {{ availableIntegrations[0]?.isUser ? '(Personal)' : `(${availableIntegrations[0]?.obienteOrgName || 'Organization'})` }}
    </OuiText>

    <OuiCombobox
      v-model="selectedRepo"
      :options="repoOptions"
      label="Repository"
      placeholder="Search repositories..."
      :disabled="isLoading"
    />

    <OuiCombobox
      v-if="selectedRepo"
      v-model="selectedBranch"
      :options="branchOptions"
      label="Branch"
      placeholder="Search branches..."
      :disabled="!selectedRepo || isLoadingBranches"
    />

    <OuiButton
      v-if="selectedRepo && selectedBranch"
      variant="ghost"
      size="sm"
      @click="loadComposeFile"
      :disabled="isLoadingFile"
    >
      {{ isLoadingFile ? "Loading..." : "Load docker-compose.yml" }}
    </OuiButton>

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";

interface Props {
  modelValue?: string; // Selected repo full name
  branch?: string; // Selected branch
  organizationId?: string; // Organization ID for GitHub token
}

interface Emits {
  (e: "update:modelValue", value: string): void;
  (e: "update:branch", value: string): void;
  (e: "update:integrationId", value: string): void;
  (e: "composeLoaded", content: string): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const client = useConnectClient(DeploymentService);
const selectedRepo = ref(props.modelValue || "");
const selectedBranch = ref(props.branch || "");
const selectedIntegrationId = ref<string>("");
const repos = ref<any[]>([]);
const branches = ref<any[]>([]);
const availableIntegrations = ref<Array<{
  id: string;
  username: string;
  isUser: boolean;
  obienteOrgId?: string;
  obienteOrgName?: string;
}>>([]);
const isLoading = ref(false);
const isLoadingBranches = ref(false);
const isLoadingFile = ref(false);
const isLoadingIntegrations = ref(false);
const error = ref("");

const integrationOptions = computed(() =>
  availableIntegrations.value.map((i) => ({
    label: i.isUser
      ? `Personal (${i.username})`
      : `${i.obienteOrgName || i.obienteOrgId} (${i.username})`,
    value: i.id,
  }))
);

const repoOptions = computed(() =>
  repos.value.map((r) => ({
    label: r.fullName || r.name || r.id || "",
    value: r.fullName || r.name || r.id || "",
  }))
);

const branchOptions = computed(() =>
  branches.value.map((b) => ({
    label: b.name + (b.isDefault ? " (default)" : ""),
    value: b.name,
  }))
);

const loadAvailableIntegrations = async () => {
  isLoadingIntegrations.value = true;
  error.value = "";
  try {
    const res = await client.listAvailableGitHubIntegrations({
      organizationId: props.organizationId || "",
    });
    availableIntegrations.value = res.integrations.map((i) => ({
      id: i.id,
      username: i.username,
      isUser: i.isUser,
      obienteOrgId: i.obienteOrgId || undefined,
      obienteOrgName: i.obienteOrgName || undefined,
    }));

    // Auto-select first integration if available and none selected
    if (availableIntegrations.value.length > 0 && !selectedIntegrationId.value) {
      const firstIntegration = availableIntegrations.value[0];
      if (firstIntegration) {
        selectedIntegrationId.value = firstIntegration.id;
        // Always emit integration ID when it's set (critical for private repos)
        emit("update:integrationId", firstIntegration.id);
      }
    } else if (selectedIntegrationId.value) {
      // If we already have a selected integration ID, ensure it's emitted
      // This ensures the parent component always has the integration ID when picker loads
      emit("update:integrationId", selectedIntegrationId.value);
    }

    // Show error if no integrations found
    if (availableIntegrations.value.length === 0) {
      error.value = "No GitHub accounts connected. Please connect a GitHub account in Settings > Integrations.";
    }
  } catch (err: any) {
    console.error("Failed to load GitHub integrations:", err);
    error.value = "Failed to load GitHub accounts. Please ensure your GitHub account is connected in Settings.";
    availableIntegrations.value = [];
  } finally {
    isLoadingIntegrations.value = false;
  }
};

const handleIntegrationChange = () => {
  // Clear repos and branches when integration changes
  repos.value = [];
  branches.value = [];
  selectedRepo.value = "";
  selectedBranch.value = "";
  emit("update:modelValue", "");
  emit("update:branch", "");
  emit("update:integrationId", selectedIntegrationId.value);
  // Refresh repos with new integration
  refreshRepos();
};

const refreshRepos = async () => {
  if (!selectedIntegrationId.value && availableIntegrations.value.length > 0) {
    // Wait for integration to be selected
    return;
  }
  
  isLoading.value = true;
  error.value = "";
  try {
    if (availableIntegrations.value.length === 0) {
      error.value = "No GitHub accounts available. Please connect a GitHub account in Settings > Integrations.";
      isLoading.value = false;
      return;
    }

    const res = await client.listGitHubRepos({
      organizationId: props.organizationId || "",
      integrationId: selectedIntegrationId.value || undefined,
      page: 1,
      perPage: 100,
    });
    repos.value = res.repos || [];
    if (repos.value.length === 0) {
      const integration = availableIntegrations.value.find(i => i.id === selectedIntegrationId.value);
      const accountName = integration?.username || "this account";
      error.value = `No repositories found for ${accountName}. Make sure the account has access to repositories.`;
    }
  } catch (err: any) {
    console.error("Failed to load repos:", err);
    const integration = availableIntegrations.value.find(i => i.id === selectedIntegrationId.value);
    const accountName = integration?.username || "the selected account";
    error.value = `Failed to load repositories for ${accountName}. ${err.message || "Please ensure the GitHub account is properly connected."}`;
    repos.value = [];
  } finally {
    isLoading.value = false;
  }
};

const handleRepoChange = async (repoFullName: string | null | undefined) => {
  if (!repoFullName) {
    selectedBranch.value = "";
    branches.value = [];
    emit("update:modelValue", "");
    emit("update:branch", "");
    // Emit empty integration ID when repo is cleared
    emit("update:integrationId", "");
    return;
  }

  emit("update:modelValue", repoFullName);
  // Always emit integration ID when repo is selected (essential for private repos)
  if (selectedIntegrationId.value) {
    emit("update:integrationId", selectedIntegrationId.value);
  }
  selectedBranch.value = "";
  branches.value = [];

  isLoadingBranches.value = true;
  error.value = "";
  try {
    const res = await client.getGitHubBranches({
      organizationId: props.organizationId || "",
      integrationId: selectedIntegrationId.value || undefined,
      repoFullName,
    });
    branches.value = res.branches || [];
    if (branches.value.length > 0) {
      const defaultBranch = branches.value.find((b) => b.isDefault) || branches.value[0];
      selectedBranch.value = defaultBranch.name;
      emit("update:branch", selectedBranch.value);
    } else {
      error.value = "No branches found for this repository.";
    }
  } catch (err: any) {
    console.error("Failed to load branches:", err);
    error.value = `Failed to load branches for this repository: ${err.message || err}`;
    branches.value = [];
  } finally {
    isLoadingBranches.value = false;
  }
};

const handleBranchChange = (branch: string) => {
  emit("update:branch", branch);
};

const loadComposeFile = async () => {
  if (!selectedRepo.value || !selectedBranch.value) return;

  isLoadingFile.value = true;
  error.value = "";
  try {
    const res = await client.getGitHubFile({
      organizationId: props.organizationId || "",
      integrationId: selectedIntegrationId.value || undefined,
      repoFullName: selectedRepo.value,
      branch: selectedBranch.value,
      path: "docker-compose.yml",
    });
    emit("composeLoaded", res.content || "");
  } catch (err: any) {
    console.error("Failed to load compose file:", err);
    error.value = "docker-compose.yml not found in this repository/branch.";
  } finally {
    isLoadingFile.value = false;
  }
};

watch(() => props.modelValue, (newVal) => {
  if (newVal !== selectedRepo.value) {
    selectedRepo.value = newVal || "";
  }
});

watch(() => props.branch, (newVal) => {
  if (newVal !== selectedBranch.value) {
    selectedBranch.value = newVal || "";
  }
});

// Watch selectedBranch changes and emit to parent
watch(selectedBranch, (newBranch) => {
  emit("update:branch", newBranch || "");
});

  // Watch selectedRepo changes to fetch branches when repository changes
// This ensures branches are fetched whenever the repo changes via combobox selection
watch(selectedRepo, async (newRepo, oldRepo) => {
  // Don't fetch if this is just the initial value from props
  if (oldRepo === undefined && newRepo === props.modelValue) {
    // Still need to fetch branches for initial value
    if (newRepo) {
      await handleRepoChange(newRepo);
      // Ensure integration ID is emitted when repo is set initially
      if (selectedIntegrationId.value) {
        emit("update:integrationId", selectedIntegrationId.value);
      }
    }
    return;
  }

  // Fetch branches if repo changed
  if (newRepo && newRepo !== oldRepo) {
    await handleRepoChange(newRepo);
    // Emit integration ID when repo is selected (ensure it's always set)
    if (selectedIntegrationId.value) {
      emit("update:integrationId", selectedIntegrationId.value);
    }
  } else if (!newRepo && oldRepo) {
    // Clear branches when repo is cleared
    branches.value = [];
    selectedBranch.value = "";
    emit("update:branch", "");
    emit("update:integrationId", "");
  }
}, { immediate: true });

watch(() => props.organizationId, async () => {
  // Reload integrations when organization changes
  await loadAvailableIntegrations();
  // Repos will be refreshed when integration changes
});

watch(selectedIntegrationId, async (newId, oldId) => {
  // Emit integration ID change to parent
  emit("update:integrationId", newId || "");
  // Refresh repos when integration changes
  if (newId && newId !== oldId) {
    await refreshRepos();
  }
}, { immediate: true });

onMounted(async () => {
  await loadAvailableIntegrations();
  // Wait a tick for integration to be selected if auto-selected
  await nextTick();
  await refreshRepos();
  // Watcher with immediate: true will handle fetching branches for initial value
});
</script>
