<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h4" size="sm" weight="semibold">GitHub Repository</OuiText>
      <OuiButton variant="ghost" size="sm" @click="refreshRepos" :disabled="isLoading">
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
        Refresh
      </OuiButton>
    </OuiFlex>

    <OuiSelect
      v-model="selectedRepo"
      :items="repoOptions"
      label="Repository"
      placeholder="Select a repository"
      @update:model-value="handleRepoChange"
    />

    <OuiSelect
      v-if="selectedRepo"
      v-model="selectedBranch"
      :items="branchOptions"
      label="Branch"
      placeholder="Select a branch"
      @update:model-value="handleBranchChange"
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
}

interface Emits {
  (e: "update:modelValue", value: string): void;
  (e: "update:branch", value: string): void;
  (e: "composeLoaded", content: string): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const client = useConnectClient(DeploymentService);
const selectedRepo = ref(props.modelValue || "");
const selectedBranch = ref(props.branch || "");
const repos = ref<any[]>([]);
const branches = ref<any[]>([]);
const isLoading = ref(false);
const isLoadingBranches = ref(false);
const isLoadingFile = ref(false);
const error = ref("");

const repoOptions = computed(() =>
  repos.value.map((r) => ({
    label: r.fullName || r.name,
    value: r.fullName || r.name,
  }))
);

const branchOptions = computed(() =>
  branches.value.map((b) => ({
    label: b.name + (b.isDefault ? " (default)" : ""),
    value: b.name,
  }))
);

const refreshRepos = async () => {
  isLoading.value = true;
  error.value = "";
  try {
    const res = await client.listGitHubRepos({ page: 1, perPage: 100 });
    repos.value = res.repos || [];
    if (repos.value.length === 0) {
      error.value = "No repositories found. Please connect your GitHub account.";
    }
  } catch (err: any) {
    console.error("Failed to load repos:", err);
    error.value = "Failed to load GitHub repositories. Please ensure your GitHub account is connected.";
    repos.value = [];
  } finally {
    isLoading.value = false;
  }
};

const handleRepoChange = async (repoFullName: string) => {
  emit("update:modelValue", repoFullName);
  selectedBranch.value = "";
  branches.value = [];
  
  if (!repoFullName) return;
  
  isLoadingBranches.value = true;
  error.value = "";
  try {
    const res = await client.getGitHubBranches({ repoFullName });
    branches.value = res.branches || [];
    if (branches.value.length > 0) {
      const defaultBranch = branches.value.find((b) => b.isDefault) || branches.value[0];
      selectedBranch.value = defaultBranch.name;
      emit("update:branch", selectedBranch.value);
    }
  } catch (err: any) {
    console.error("Failed to load branches:", err);
    error.value = "Failed to load branches for this repository.";
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

onMounted(() => {
  refreshRepos();
});
</script>

