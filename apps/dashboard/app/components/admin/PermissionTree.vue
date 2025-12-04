<template>
  <OuiStack gap="xs" class="permission-tree">
    <OuiFlex justify="between" align="center">
      <OuiText size="sm" weight="semibold">Permissions</OuiText>
      <OuiButton
        v-if="showSelectAll"
        variant="ghost"
        size="xs"
        @click="toggleSelectAll"
      >
        {{ allSelected ? "Deselect All" : "Select All" }}
      </OuiButton>
    </OuiFlex>

    <OuiStack gap="xs">
      <OuiFlex v-if="permissionTree.length === 0" justify="center" py="sm">
        <OuiText size="xs" color="secondary">No permissions found</OuiText>
      </OuiFlex>
      <PermissionTreeNode
        v-for="node in permissionTree"
        :key="node.path"
        :node="node"
        :selected-permissions="selectedPermissions"
        :expanded-paths="expandedPaths"
        @toggle-permission="togglePermission"
        @toggle-node="toggleNode"
        @expand-node="expandNode"
      />
    </OuiStack>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import PermissionTreeNode from "./PermissionTreeNode.vue";

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
  isWildcard: boolean; // true if this is a wildcard node (e.g., "organization.members.*")
}

interface Props {
  permissions: Permission[];
  modelValue: string[];
  showSelectAll?: boolean;
  defaultExpanded?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showSelectAll: true,
  defaultExpanded: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
}>();

const selectedPermissions = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});

const expandedPaths = ref<Set<string>>(new Set());

// Build hierarchical tree from flat permissions
const permissionTree = computed<PermissionNode[]>(() => {
  if (!props.permissions || props.permissions.length === 0) {
    return [];
  }

  // Build a tree structure from permissions
  const tree = new Map<string, PermissionNode>();

  for (const perm of props.permissions) {
    const parts = perm.id.split(".").filter((p) => p);
    let currentPath = "";
    let parentNode: PermissionNode | null = null;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      if (!part) continue;
      const isLast = i === parts.length - 1;
      const isWildcard = part === "*";
      
      currentPath = currentPath ? `${currentPath}.${part}` : part;
      
      if (!tree.has(currentPath)) {
        const node: PermissionNode = {
          path: currentPath,
          name: isWildcard ? "*" : formatName(part),
          description: isLast ? perm.description : undefined,
          children: [],
          permissions: [],
          isWildcard,
        };
        tree.set(currentPath, node);
        
        if (parentNode) {
          parentNode.children.push(node);
        }
      }
      
      const node = tree.get(currentPath)!;
      if (isLast) {
        node.permissions.push(perm);
        if (!node.description && perm.description) {
          node.description = perm.description;
        }
      }
      
      parentNode = node;
    }
  }

  // Get root nodes (top-level resource types)
  const rootNodes: PermissionNode[] = [];
  const rootResourceTypes = new Set<string>();
  
  for (const perm of props.permissions) {
    const parts = perm.id.split(".").filter((p) => p);
    if (parts.length > 0 && parts[0]) {
      rootResourceTypes.add(parts[0]);
    }
  }

  for (const resourceType of rootResourceTypes) {
    const node = tree.get(resourceType);
    if (node) {
      rootNodes.push(node);
    }
  }

  // Sort nodes recursively: collapsible nodes first, then leaf nodes
  function sortNode(node: PermissionNode) {
    node.children.sort((a, b) => {
      const aHasChildren = a.children.length > 0;
      const bHasChildren = b.children.length > 0;
      
      // Collapsible nodes come first
      if (aHasChildren && !bHasChildren) return -1;
      if (!aHasChildren && bHasChildren) return 1;
      
      // Wildcards come last within same type
      if (a.isWildcard && !b.isWildcard) return 1;
      if (!a.isWildcard && b.isWildcard) return -1;
      
      return a.name.localeCompare(b.name);
    });
    node.children.forEach(sortNode);
  }

  rootNodes.forEach(sortNode);
  
  // Sort root nodes: collapsible nodes (with children) first, then leaf nodes
  rootNodes.sort((a, b) => {
    const aHasChildren = a.children.length > 0;
    const bHasChildren = b.children.length > 0;
    
    // Collapsible nodes come first
    if (aHasChildren && !bHasChildren) return -1;
    if (!aHasChildren && bHasChildren) return 1;
    
    // Within same type, sort alphabetically
    return a.name.localeCompare(b.name);
  });

  return rootNodes;
});

// Initialize expanded paths
watch(
  () => permissionTree.value,
  (tree) => {
    if (props.defaultExpanded && expandedPaths.value.size === 0) {
      // Expand all root nodes by default
      tree.forEach((node) => {
        expandedPaths.value.add(node.path);
      });
    }
  },
  { immediate: true }
);

const allSelected = computed(() => {
  return props.permissions && props.permissions.length > 0 && 
    props.permissions.every((p) => selectedPermissions.value.includes(p.id));
});

function toggleSelectAll() {
  if (allSelected.value) {
    selectedPermissions.value = [];
  } else {
    // Select all non-wildcard permissions
    selectedPermissions.value = props.permissions
      .filter((p) => !p.id.endsWith(".*"))
      .map((p) => p.id);
  }
}

function togglePermission(permissionId: string, checked: boolean) {
  const newSelection = new Set(selectedPermissions.value);
  if (checked) {
    newSelection.add(permissionId);
    
    // Check if all siblings are now selected - if so, add wildcard
    const parts = permissionId.split(".");
    if (parts.length > 1) {
      const parentPath = parts.slice(0, -1).join(".");
      const wildcardPath = parentPath + ".*";
      
      // Find all permissions under this parent
      function findSiblingPermissions(parent: string): string[] {
        return props.permissions
          .filter((p) => {
            const permParts = p.id.split(".");
            const permParent = permParts.slice(0, -1).join(".");
            return permParent === parent && !p.id.endsWith(".*");
          })
          .map((p) => p.id);
      }
      
      const siblings = findSiblingPermissions(parentPath);
      const allSiblingsSelected = siblings.every((sib) => newSelection.has(sib));
      
      if (allSiblingsSelected && siblings.length > 0) {
        // Auto-add wildcard when all children are selected
        newSelection.add(wildcardPath);
      }
    }
  } else {
    newSelection.delete(permissionId);
    // Also remove any wildcard permissions that would grant this
    const wildcardPattern = permissionId.split(".").slice(0, -1).join(".") + ".*";
    newSelection.delete(wildcardPattern);
  }
  selectedPermissions.value = Array.from(newSelection);
}

function toggleNode(nodePath: string, checked: boolean) {
  // Find all permissions under this node path (excluding wildcards)
  function collectPermissions(node: PermissionNode, pathPrefix: string): string[] {
    const perms: string[] = [];
    const fullPath = pathPrefix ? `${pathPrefix}.${node.path}` : node.path;
    
    // Add direct permissions (excluding wildcards)
    perms.push(...node.permissions.filter((p) => !p.id.endsWith(".*")).map((p) => p.id));
    
    // Collect all child permissions
    if (node.children.length > 0 && !node.isWildcard) {
      for (const child of node.children) {
        perms.push(...collectPermissions(child, fullPath));
      }
    }
    
    return perms;
  }

  // Find the node
  function findNode(nodes: PermissionNode[], path: string): PermissionNode | null {
    for (const node of nodes) {
      if (node.path === path) return node;
      const found = findNode(node.children, path);
      if (found) return found;
    }
    return null;
  }

  const node = findNode(permissionTree.value, nodePath);
  if (!node) return;

  const perms = collectPermissions(node, "");
  const newSelection = new Set(selectedPermissions.value);

  const wildcardPath = nodePath + ".*";
  
  if (checked) {
    // Add all child permissions (excluding wildcards)
    perms.forEach((p) => {
      if (!p.endsWith(".*")) {
        newSelection.add(p);
      }
    });
    
    // If all permissions are selected, also add wildcard
    const nonWildcardPerms = perms.filter((p) => !p.endsWith(".*"));
    const allSelected = nonWildcardPerms.length > 0 && 
      nonWildcardPerms.every((p) => newSelection.has(p));
    if (allSelected) {
      newSelection.add(wildcardPath);
    }
  } else {
    // Remove all child permissions (excluding wildcards)
    perms.forEach((p) => {
      if (!p.endsWith(".*")) {
        newSelection.delete(p);
      }
    });
    // Also remove wildcard
    newSelection.delete(wildcardPath);
  }

  selectedPermissions.value = Array.from(newSelection);
}

function expandNode(path: string, expanded: boolean) {
  if (expanded) {
    expandedPaths.value.add(path);
  } else {
    expandedPaths.value.delete(path);
  }
}

function formatName(part: string): string {
  if (!part) return "Unknown";
  // Convert snake_case to Title Case
  return part
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}
</script>

<style scoped>
.permission-tree {
  @apply w-full;
}
</style>
