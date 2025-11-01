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
      v-if="availableIntegrations.length > 1"
      v-model="selectedIntegrationId"
      :items="integrationOptions"
      label="GitHub Account"
      placeholder="Select GitHub account..."
      @update:model-value="handleIntegrationChange"
    />

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
import { ref, computed, watch, onMounted } from "vue";
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
        emit("update:integrationId", firstIntegration.id);
      }
    }
  } catch (err: any) {
    console.error("Failed to load GitHub integrations:", err);
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
  isLoading.value = true;
  error.value = "";
  try {
    const res = await client.listGitHubRepos({
      organizationId: props.organizationId || "",
      integrationId: selectedIntegrationId.value || undefined,
      page: 1,
      perPage: 100,
    });
    repos.value = res.repos || [];
    if (repos.value.length === 0) {
      error.value = "No repositories found. Please connect your GitHub account or organization.";
    }
  } catch (err: any) {
    console.error("Failed to load repos:", err);
    error.value = "Failed to load GitHub repositories. Please ensure your GitHub account or organization is connected.";
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
    return;
  }

  emit("update:modelValue", repoFullName);
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

// Watch selectedIntegrationId changes and emit to parent
watch(selectedIntegrationId, (newId) => {
  emit("update:integrationId", newId || "");
});

// Watch selectedRepo changes to fetch branches when repository changes
// This ensures branches are fetched whenever the repo changes via combobox selection
watch(selectedRepo, async (newRepo, oldRepo) => {
  // Don't fetch if this is just the initial value from props
  if (oldRepo === undefined && newRepo === props.modelValue) {
    // Still need to fetch branches for initial value
    if (newRepo) {
      await handleRepoChange(newRepo);
    }
    return;
  }

  // Fetch branches if repo changed
  if (newRepo && newRepo !== oldRepo) {
    await handleRepoChange(newRepo);
  } else if (!newRepo && oldRepo) {
    // Clear branches when repo is cleared
    branches.value = [];
    selectedBranch.value = "";
    emit("update:branch", "");
  }
}, { immediate: true });

watch(() => props.organizationId, () => {
  // Reload integrations when organization changes
  loadAvailableIntegrations();
});

onMounted(async () => {
  await loadAvailableIntegrations();
  await refreshRepos();
  // Watcher with immediate: true will handle fetching branches for initial value
});
</script>
