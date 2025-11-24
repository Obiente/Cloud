import { ref, type Ref } from "vue";
import type { ExplorerNode } from "~/components/shared/fileExplorerTypes";

export interface MultiSelectOptions {
  selectedNodes: Ref<Set<string>>;
  lastSelectedIndex: Ref<number | null>;
  visibleNodes: Ref<ExplorerNode[]>;
}

/**
 * Composable for handling multi-select with Ctrl and Shift keys
 * Similar to standard file browser behavior
 */
export function useMultiSelect(options: MultiSelectOptions) {
  const { selectedNodes, lastSelectedIndex, visibleNodes } = options;

  /**
   * Get all visible nodes in a flat array (for range selection)
   */
  function getAllVisibleNodes(nodes: ExplorerNode[]): ExplorerNode[] {
    const result: ExplorerNode[] = [];
    
    function traverse(nodeList: ExplorerNode[]) {
      for (const node of nodeList) {
        if (node.path !== "/") {
          result.push(node);
        }
        if (node.isExpanded && node.children?.length) {
          traverse(node.children);
        }
      }
    }
    
    traverse(nodes);
    return result;
  }

  /**
   * Find the index of a node in the visible nodes array
   */
  function findNodeIndex(nodePath: string): number {
    return visibleNodes.value.findIndex(n => n.path === nodePath);
  }

  /**
   * Handle node click with support for Ctrl and Shift
   */
  function handleNodeClick(
    node: ExplorerNode,
    event: MouseEvent,
    onSelectionChange?: (selectedPaths: string[]) => void
  ) {
    console.log("[useMultiSelect] handleNodeClick called", {
      path: node.path,
      ctrlKey: event.ctrlKey,
      metaKey: event.metaKey,
      shiftKey: event.shiftKey,
      visibleNodesCount: visibleNodes.value.length,
      selectedNodesCount: selectedNodes.value.size,
      lastSelectedIndex: lastSelectedIndex.value,
    });

    // Don't allow selecting root
    if (node.path === "/") {
      console.log("[useMultiSelect] Ignoring root node");
      return;
    }

    let nodeIndex = findNodeIndex(node.path);
    console.log("[useMultiSelect] Node index found:", nodeIndex);
    
    // If node not found, try to add it to visible nodes and find again
    if (nodeIndex === -1) {
      console.log("[useMultiSelect] Node not in visibleNodes, adding it");
      // Add the node to visible nodes if it's not there
      // This can happen if the tree structure changed
      visibleNodes.value.push(node);
      nodeIndex = visibleNodes.value.length - 1;
      console.log("[useMultiSelect] Added node, new index:", nodeIndex);
    }

    if (event.ctrlKey || event.metaKey) {
      console.log("[useMultiSelect] Ctrl/Cmd+Click detected");
      // Ctrl/Cmd+Click: Toggle selection
      if (selectedNodes.value.has(node.path)) {
        console.log("[useMultiSelect] Removing from selection");
        selectedNodes.value.delete(node.path);
      } else {
        console.log("[useMultiSelect] Adding to selection");
        selectedNodes.value.add(node.path);
      }
      lastSelectedIndex.value = nodeIndex;
    } else if (event.shiftKey && lastSelectedIndex.value !== null) {
      console.log("[useMultiSelect] Shift+Click detected, range selection", {
        start: Math.min(lastSelectedIndex.value, nodeIndex),
        end: Math.max(lastSelectedIndex.value, nodeIndex),
      });
      // Shift+Click: Select range
      const start = Math.min(lastSelectedIndex.value, nodeIndex);
      const end = Math.max(lastSelectedIndex.value, nodeIndex);
      
      for (let i = start; i <= end; i++) {
        const rangeNode = visibleNodes.value[i];
        if (rangeNode && rangeNode.path !== "/") {
          selectedNodes.value.add(rangeNode.path);
        }
      }
      // Don't update lastSelectedIndex for range selection
    } else {
      console.log("[useMultiSelect] Normal click, clearing and selecting single node");
      // Normal click: Clear selection and select only this node
      selectedNodes.value.clear();
      selectedNodes.value.add(node.path);
      lastSelectedIndex.value = nodeIndex;
    }

    console.log("[useMultiSelect] Final state", {
      selectedCount: selectedNodes.value.size,
      selectedPaths: Array.from(selectedNodes.value),
      lastSelectedIndex: lastSelectedIndex.value,
    });

    // Notify parent of selection change
    if (onSelectionChange) {
      onSelectionChange(Array.from(selectedNodes.value));
    }
  }

  /**
   * Clear all selections
   */
  function clearSelection() {
    selectedNodes.value.clear();
    lastSelectedIndex.value = null;
  }

  /**
   * Check if a node is selected
   */
  function isSelected(nodePath: string): boolean {
    return selectedNodes.value.has(nodePath);
  }

  /**
   * Get all selected paths
   */
  function getSelectedPaths(): string[] {
    return Array.from(selectedNodes.value);
  }

  /**
   * Select multiple nodes
   */
  function selectNodes(paths: string[]) {
    selectedNodes.value.clear();
    paths.forEach(path => {
      if (path && path !== "/") {
        selectedNodes.value.add(path);
      }
    });
    // Update last selected index to the last selected node
    if (paths.length > 0) {
      const lastPath = paths[paths.length - 1];
      if (lastPath) {
        const index = findNodeIndex(lastPath);
        if (index !== -1) {
          lastSelectedIndex.value = index;
        }
      }
    }
  }

  return {
    handleNodeClick,
    clearSelection,
    isSelected,
    getSelectedPaths,
    selectNodes,
    getAllVisibleNodes,
  };
}

