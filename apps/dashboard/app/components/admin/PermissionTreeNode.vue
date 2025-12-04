<template>
  <!-- Leaf node (no children, just permissions) - render flat, not collapsible -->
  <template v-if="!hasChildren && node.permissions.length > 0">
    <OuiFlex
      v-for="permission in node.permissions"
      :key="permission.id"
      align="start"
      gap="xs"
      p="xs"
      class="rounded hover:bg-background-muted transition-colors"
    >
      <OuiFlex class="flex-shrink-0" py="xs">
        <OuiCheckbox
          :model-value="selectedPermissions.includes(permission.id)"
          @update:model-value="(checked) => $emit('toggle-permission', permission.id, checked)"
        >
          <OuiText class="sr-only">{{ permission.id }}</OuiText>
        </OuiCheckbox>
      </OuiFlex>
      <OuiStack gap="xs" class="flex-1 min-w-0">
        <OuiText size="xs" weight="medium">
          {{ formatPermissionName(permission.id) }}
        </OuiText>
        <OuiText size="xs" color="secondary">
          {{ permission.description || "No description available" }}
        </OuiText>
        <OuiText size="xs" color="secondary" class="font-mono">
          {{ permission.id }}
        </OuiText>
      </OuiStack>
    </OuiFlex>
  </template>

  <!-- Node with children (collapsible) -->
  <OuiCollapsible
    v-else-if="hasChildren"
    :model-value="expanded"
    @update:model-value="(open) => $emit('expand-node', node.path, open)"
  >
    <template #trigger>
      <OuiFlex justify="between" align="center" px="sm" py="xs" gap="sm" class="w-full">
        <OuiFlex align="center" gap="xs" class="flex-1 min-w-0">
          <OuiFlex @click.stop align="center" gap="xs" class="flex-shrink-0">
            <OuiCheckbox
              :model-value="wildcardPermission ? selectedPermissions.includes(wildcardPermission.id) : isFullySelected"
              :indeterminate="wildcardPermission ? false : isPartiallySelected"
              @update:model-value="(checked) => {
                if (wildcardPermission) {
                  $emit('toggle-permission', wildcardPermission.id, checked);
                } else {
                  $emit('toggle-node', node.path, checked);
                }
              }"
            >
              <OuiText class="sr-only">{{ node.name }}</OuiText>
            </OuiCheckbox>
          </OuiFlex>
          <OuiText size="sm" :weight="isWildcardNode ? 'medium' : 'semibold'" class="flex-1 min-w-0">
            {{ wildcardPermission ? `All ${node.name.toLowerCase()} permissions` : node.name }}
          </OuiText>
        </OuiFlex>
        <OuiText v-if="!wildcardPermission && (hasChildren || node.permissions.length > 0)" size="xs" color="secondary" class="flex-shrink-0">
          {{ selectionCount }}/{{ totalCount }}
        </OuiText>
      </OuiFlex>
    </template>

    <OuiStack gap="xs" px="sm" pb="xs">
      <!-- Direct permissions (excluding wildcard, which is shown on the trigger) -->
      <OuiFlex
        v-for="permission in node.permissions.filter(p => !p.id.endsWith('.*'))"
        :key="permission.id"
        align="start"
        gap="xs"
        p="xs"
        class="rounded hover:bg-background-muted transition-colors"
      >
        <OuiFlex class="flex-shrink-0" py="xs">
          <OuiCheckbox
            :model-value="selectedPermissions.includes(permission.id)"
            @update:model-value="(checked) => $emit('toggle-permission', permission.id, checked)"
          >
            <OuiText class="sr-only">{{ permission.id }}</OuiText>
          </OuiCheckbox>
        </OuiFlex>
        <OuiStack gap="xs" class="flex-1 min-w-0">
          <OuiText size="xs" weight="medium">
            {{ formatPermissionName(permission.id) }}
          </OuiText>
          <OuiText size="xs" color="secondary">
            {{ permission.description || "No description available" }}
          </OuiText>
          <OuiText size="xs" color="secondary" class="font-mono">
            {{ permission.id }}
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Child nodes (sub-trees) -->
      <PermissionTreeNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :selected-permissions="selectedPermissions"
        :expanded-paths="expandedPaths"
        @toggle-permission="(id, checked) => $emit('toggle-permission', id, checked)"
        @toggle-node="(path, checked) => $emit('toggle-node', path, checked)"
        @expand-node="(path, expanded) => $emit('expand-node', path, expanded)"
      />
    </OuiStack>
  </OuiCollapsible>
</template>

<script setup lang="ts">
import { computed } from "vue";

interface Permission {
  id: string;
  description: string;
  resourceType: string;
}

interface PermissionNode {
  path: string;
  name: string;
  description?: string;
  children: PermissionNode[];
  permissions: Permission[];
  isWildcard: boolean;
}

interface Props {
  node: PermissionNode;
  selectedPermissions: string[];
  expandedPaths: Set<string>;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  "toggle-permission": [permissionId: string, checked: boolean];
  "toggle-node": [nodePath: string, checked: boolean];
  "expand-node": [path: string, expanded: boolean];
}>();

const expanded = computed(() => props.expandedPaths.has(props.node.path));

const hasChildren = computed(() => props.node.children.length > 0);

// Check if this node has a wildcard permission (e.g., "vps.*")
const wildcardPermission = computed(() => {
  return props.node.permissions.find((p) => p.id.endsWith(".*"));
});

// Check if this node IS a wildcard (the node itself represents a wildcard)
const isWildcardNode = computed(() => {
  return props.node.isWildcard || !!wildcardPermission.value;
});

const totalCount = computed(() => {
  // Count direct permissions (excluding wildcards)
  let count = props.node.permissions.filter((p) => !p.id.endsWith(".*")).length;
  
  function countChildren(nodes: PermissionNode[]): number {
    let total = 0;
    for (const child of nodes) {
      // Count non-wildcard permissions
      total += child.permissions.filter((p) => !p.id.endsWith(".*")).length;
      total += countChildren(child.children);
    }
    return total;
  }
  
  return count + countChildren(props.node.children);
});

const selectionCount = computed(() => {
  // If this node has a wildcard and it's selected, count all child permissions
  if (wildcardPermission.value && props.selectedPermissions.includes(wildcardPermission.value.id)) {
    return totalCount.value;
  }
  
  // Count selected direct permissions (excluding wildcards)
  let count = props.node.permissions
    .filter((p) => !p.id.endsWith(".*") && props.selectedPermissions.includes(p.id))
    .length;
  
  function countSelected(nodes: PermissionNode[]): number {
    let total = 0;
    for (const child of nodes) {
      // Check if child has a wildcard that's selected
      const childWildcard = child.permissions.find((p) => p.id.endsWith(".*"));
      if (childWildcard && props.selectedPermissions.includes(childWildcard.id)) {
        // Count all child permissions as selected
        const allChildPerms = child.permissions.filter((p) => !p.id.endsWith(".*"));
        total += allChildPerms.length;
      } else {
        // Count selected non-wildcard permissions
        total += child.permissions
          .filter((p) => !p.id.endsWith(".*") && props.selectedPermissions.includes(p.id))
          .length;
      }
      
      total += countSelected(child.children);
    }
    return total;
  }
  
  return count + countSelected(props.node.children);
});

const isFullySelected = computed(() => {
  return selectionCount.value === totalCount.value && totalCount.value > 0;
});

const isPartiallySelected = computed(() => {
  return selectionCount.value > 0 && selectionCount.value < totalCount.value;
});

function formatPermissionName(permissionId: string): string {
  const parts = permissionId.split(".").filter((p) => p);
  const lastPart = parts[parts.length - 1];
  if (!lastPart) return permissionId;
  // Convert snake_case to Title Case
  return lastPart
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
</script>