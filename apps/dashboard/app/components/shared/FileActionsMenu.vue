<template>
  <OuiMenu v-if="currentNode">
    <template #trigger>
      <OuiButton
        variant="ghost"
        size="sm"
        :class="buttonClass"
        :disabled="disabled"
      >
        <span v-if="showLabel" class="hidden sm:inline">{{ label }}</span>
        <span v-else>{{ label }}</span>
        <EllipsisVerticalIcon v-if="showIcon" class="h-4 w-4 sm:hidden" />
      </OuiButton>
    </template>
    <template #default>
      <!-- Refresh action -->
      <OuiMenuItem
        v-if="showRefresh"
        value="refresh"
        @select="handleRefresh"
      >
        <ArrowPathIcon class="h-4 w-4 mr-2" />
        Refresh
      </OuiMenuItem>

      <!-- Rename action -->
      <OuiMenuItem
        v-if="showRename"
        value="rename"
        @select="handleRename"
      >
        <PencilIcon class="h-4 w-4 mr-2" />
        Rename
      </OuiMenuItem>

      <!-- Copy path action -->
      <OuiMenuItem
        v-if="showCopyPath"
        value="copy-path"
        @select="handleCopyPath"
      >
        <DocumentDuplicateIcon class="h-4 w-4 mr-2" />
        Copy Path
      </OuiMenuItem>

      <!-- Download action (if file) -->
      <OuiMenuItem
        v-if="showDownload && currentNode?.type === 'file'"
        value="download"
        @select="handleDownload"
      >
        <DocumentArrowDownIcon class="h-4 w-4 mr-2" />
        Download
      </OuiMenuItem>

      <!-- Custom menu items slot -->
      <slot name="items" :current-node="currentNode" />

      <!-- Separator before destructive actions -->
      <OuiMenuSeparator v-if="showDelete || showCustomActions" />

      <!-- Delete action -->
      <OuiMenuItem
        v-if="showDelete"
        value="delete"
        @select="handleDelete"
        class="text-danger"
      >
        <TrashIcon class="h-4 w-4 mr-2" />
        Delete
      </OuiMenuItem>

      <!-- Custom destructive actions slot -->
      <slot name="destructive" :current-node="currentNode" />
    </template>
  </OuiMenu>
</template>

<script setup lang="ts">
import {
  ArrowPathIcon,
  DocumentArrowDownIcon,
  DocumentDuplicateIcon,
  EllipsisVerticalIcon,
  PencilIcon,
  TrashIcon,
} from "@heroicons/vue/24/outline";
import type { ExplorerNode } from "~/components/shared/fileExplorerTypes";

interface Props {
  currentNode: ExplorerNode | null;
  label?: string;
  showLabel?: boolean;
  showIcon?: boolean;
  buttonClass?: string;
  disabled?: boolean;
  // Action visibility toggles
  showRefresh?: boolean;
  showRename?: boolean;
  showCopyPath?: boolean;
  showDownload?: boolean;
  showDelete?: boolean;
  showCustomActions?: boolean;
}

interface Emits {
  (e: "refresh", node: ExplorerNode): void;
  (e: "rename", node: ExplorerNode): void;
  (e: "copy-path", node: ExplorerNode): void;
  (e: "download", node: ExplorerNode): void;
  (e: "delete", node: ExplorerNode): void;
}

const props = withDefaults(defineProps<Props>(), {
  label: "More",
  showLabel: true,
  showIcon: true,
  buttonClass: "flex-1 sm:flex-initial",
  disabled: false,
  showRefresh: true,
  showRename: true,
  showCopyPath: false,
  showDownload: false,
  showDelete: true,
  showCustomActions: false,
});

const emit = defineEmits<Emits>();

function handleRefresh() {
  if (props.currentNode) {
    emit("refresh", props.currentNode);
  }
}

function handleRename() {
  if (props.currentNode) {
    emit("rename", props.currentNode);
  }
}

function handleCopyPath() {
  if (props.currentNode) {
    emit("copy-path", props.currentNode);
  }
}

function handleDownload() {
  if (props.currentNode) {
    emit("download", props.currentNode);
  }
}

function handleDelete() {
  if (props.currentNode) {
    emit("delete", props.currentNode);
  }
}
</script>

