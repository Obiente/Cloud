import { computed, ref, watch, type Ref } from "vue";
import { OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

/**
 * Minimal organization type that works with both OrgSummary and Organization proto types
 */
type OrgLike = {
  id: string;
  name?: string | null;
  slug?: string | null;
};

/**
 * Composable to enhance organization labels with owner information
 * when there are duplicate organization names.
 */
export function useOrganizationLabels(organizations: Ref<OrgLike[]>) {
  const orgClient = useConnectClient(OrganizationService);
  const ownerCache = ref<Map<string, string>>(new Map());
  const loadingOwners = ref<Set<string>>(new Set());

  // Check for duplicate names
  const hasDuplicateNames = computed(() => {
    const nameCounts = new Map<string, number>();
    organizations.value.forEach((org) => {
      const name = org.name ?? org.slug ?? org.id;
      nameCounts.set(name, (nameCounts.get(name) || 0) + 1);
    });
    return Array.from(nameCounts.values()).some((count) => count > 1);
  });

  // Get organizations with duplicate names
  const duplicateNameOrgs = computed(() => {
    if (!hasDuplicateNames.value) return [];
    
    const nameCounts = new Map<string, OrgLike[]>();
    organizations.value.forEach((org) => {
      const name = org.name ?? org.slug ?? org.id;
      if (!nameCounts.has(name)) {
        nameCounts.set(name, []);
      }
      nameCounts.get(name)!.push(org);
    });
    
    const duplicates: OrgLike[] = [];
    nameCounts.forEach((orgs) => {
      if (orgs.length > 1) {
        duplicates.push(...orgs);
      }
    });
    return duplicates;
  });

  // Fetch owner for an organization
  async function fetchOwner(orgId: string): Promise<string | null> {
    if (ownerCache.value.has(orgId)) {
      return ownerCache.value.get(orgId) || null;
    }

    if (loadingOwners.value.has(orgId)) {
      // Already loading, wait a bit and retry
      await new Promise((resolve) => setTimeout(resolve, 100));
      return ownerCache.value.get(orgId) || null;
    }

    loadingOwners.value.add(orgId);
    try {
      const res = await orgClient.listMembers({ organizationId: orgId });
      const owner = res.members?.find((m) => m.role === "owner");
      if (owner?.user) {
        const ownerName = owner.user.name || owner.user.email || "Unknown";
        ownerCache.value.set(orgId, ownerName);
        return ownerName;
      }
    } catch (error) {
      console.error(`Failed to fetch owner for org ${orgId}:`, error);
    } finally {
      loadingOwners.value.delete(orgId);
    }
    return null;
  }

  // Fetch owners for all duplicate organizations
  async function loadOwnersForDuplicates() {
    if (!hasDuplicateNames.value) return;
    
    const promises = duplicateNameOrgs.value.map((org) => fetchOwner(org.id));
    await Promise.all(promises);
  }

  // Watch for changes in organizations and load owners if needed
  watch(
    () => organizations.value.map((o) => o.id),
    () => {
      ownerCache.value.clear();
      loadingOwners.value.clear();
      if (hasDuplicateNames.value && organizations.value.length > 0) {
        loadOwnersForDuplicates();
      }
    },
    { immediate: true }
  );

  // Generate enhanced labels (always returns strings)
  const organizationLabels = computed(() => {
    return organizations.value.map((org) => {
      const baseName = org.name ?? org.slug ?? org.id ?? "Unknown";
      
      // If there are duplicates and we have owner info, add it
      if (hasDuplicateNames.value) {
        const ownerName = ownerCache.value.get(org.id);
        if (ownerName) {
          return `${baseName} (${ownerName})`;
        }
      }
      
      return baseName;
    });
  });

  // Generate select items with enhanced labels (ensures label is always string)
  const organizationSelectItems = computed(() => {
    return organizations.value.map((org, index) => {
      const label = organizationLabels.value[index];
      return {
        label: label ?? org.name ?? org.slug ?? org.id ?? "Unknown",
        value: org.id,
      };
    });
  });

  return {
    organizationLabels,
    organizationSelectItems,
    hasDuplicateNames,
    loadOwnersForDuplicates,
  };
}

