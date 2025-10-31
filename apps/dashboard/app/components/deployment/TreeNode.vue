<script setup lang="ts">
import { TreeView } from '@ark-ui/vue/tree-view'
import { FolderIcon, DocumentIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'

interface FileNode {
  id: string
  name: string
  path: string
  isDirectory: boolean
  size?: number
  children?: FileNode[]
}

interface Props {
  node: FileNode
  indexPath: number[]
}

const props = defineProps<Props>()

interface ClickEvent {
  path: string
  isDirectory: boolean
}

const emit = defineEmits<{
  (e: 'click', event: ClickEvent): void
}>()

const handleClick = () => {
  emit('click', {
    path: props.node.path,
    isDirectory: props.node.isDirectory
  })
}
</script>

<template>
  <TreeView.NodeProvider :node="props.node" :indexPath="props.indexPath">
    <TreeView.Branch v-if="node.children || node.isDirectory">
      <TreeView.BranchControl @click.stop="handleClick" class="cursor-pointer hover:bg-surface-hover">
        <TreeView.BranchText class="flex items-center gap-2">
          <FolderIcon class="h-4 w-4 text-primary" />
          <span>{{ node.name }}</span>
        </TreeView.BranchText>
        <TreeView.BranchIndicator class="flex items-center">
          <ChevronRightIcon class="h-4 w-4" />
        </TreeView.BranchIndicator>
      </TreeView.BranchControl>
      <TreeView.BranchContent v-if="node.children">
        <TreeView.BranchIndentGuide />
          <TreeNode
            v-for="(child, index) in node.children"
            :key="child.id"
            :node="child"
            :indexPath="[...props.indexPath, index]"
            @click="$emit('click', $event)"
          />
      </TreeView.BranchContent>
    </TreeView.Branch>
    <TreeView.Item v-else @click.stop="handleClick" class="cursor-pointer hover:bg-surface-hover">
      <TreeView.ItemText class="flex items-center gap-2">
        <DocumentIcon class="h-4 w-4 text-secondary" />
        <span>{{ node.name }}</span>
        <span v-if="node.size" class="ml-auto text-xs text-secondary">
          {{ formatSize(node.size) }}
        </span>
      </TreeView.ItemText>
    </TreeView.Item>
  </TreeView.NodeProvider>
</template>

<script lang="ts">
const formatSize = (bytes: number | bigint) => {
  const size = typeof bytes === 'bigint' ? Number(bytes) : bytes
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}
</script>

