import { computed, watch, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Container {
  containerId: string;
  serviceName?: string;
  status?: string;
}

export function useDeploymentContainerQuery(
  deploymentId: string,
  organizationId?: string
) {
  const route = useRoute();
  const router = useRouter();
  const orgsStore = useOrganizationsStore();
  const client = useConnectClient(DeploymentService);

  const effectiveOrgId = computed(
    () => organizationId || orgsStore.currentOrgId || ""
  );

  const containers = ref<Container[]>([]);
  const isLoading = ref(false);
  const selectedContainerId = ref<string>("");
  const selectedServiceName = ref<string>("");

  // Load containers for this deployment
  const loadContainers = async () => {
    if (!deploymentId || !effectiveOrgId.value) {
      containers.value = [];
      return;
    }

    isLoading.value = true;
    try {
      const res = await (client as any).listDeploymentContainers({
        deploymentId,
        organizationId: effectiveOrgId.value,
      });

      if (res?.containers) {
        containers.value = res.containers.map((c: any) => ({
          containerId: c.containerId,
          serviceName: c.serviceName || undefined,
          status: c.status || "unknown",
        }));

        // Sort: running containers first, then starting/restarting, then stopped/exited, then others
        containers.value.sort((a, b) => {
          const statusA = (a.status || "unknown").toLowerCase();
          const statusB = (b.status || "unknown").toLowerCase();

          const priority = (status: string) => {
            if (status === "running") return 0;
            if (status === "starting" || status === "restarting") return 1;
            if (status === "stopped" || status === "exited") return 2;
            return 3;
          };

          const priorityA = priority(statusA);
          const priorityB = priority(statusB);

          if (priorityA !== priorityB) {
            return priorityA - priorityB;
          }

          const nameA = a.serviceName || a.containerId;
          const nameB = b.serviceName || b.containerId;
          return nameA.localeCompare(nameB);
        });

        // After loading containers, sync with query params
        syncWithQueryParams();
      } else {
        containers.value = [];
      }
    } catch (err) {
      console.error("Failed to load containers:", err);
      containers.value = [];
    } finally {
      isLoading.value = false;
    }
  };

  // Find container by service name or container ID
  const findContainer = (
    serviceName?: string,
    containerId?: string
  ): Container | null => {
    if (!containers.value.length) return null;

    if (serviceName) {
      // Find by service name (prefer running containers)
      const byService = containers.value.filter(
        (c) => c.serviceName === serviceName
      );
      const running = byService.find((c) => c.status === "running");
      if (running) return running;
      if (byService.length > 0 && byService[0]) return byService[0];
    }

    if (containerId) {
      const found = containers.value.find((c) => c.containerId === containerId);
      if (found) return found;
    }

    // Default to first container
    return containers.value[0] || null;
  };

  // Sync selection with query params
  const syncWithQueryParams = () => {
    const serviceParam = route.query.service;
    const containerParam = route.query.container;

    if (typeof serviceParam === "string" && serviceParam) {
      const container = findContainer(serviceParam);
      if (container) {
        selectedServiceName.value = container.serviceName || "";
        selectedContainerId.value = container.serviceName || container.containerId;
        return;
      }
    }

    if (typeof containerParam === "string" && containerParam) {
      const container = findContainer(undefined, containerParam);
      if (container) {
        selectedContainerId.value = container.containerId;
        selectedServiceName.value = container.serviceName || "";
        return;
      }
    }

    // No query params - default to first container
    if (containers.value.length > 0 && containers.value[0]) {
      const first = containers.value[0];
      selectedContainerId.value = first.serviceName || first.containerId;
      selectedServiceName.value = first.serviceName || "";
    } else {
      selectedContainerId.value = "";
      selectedServiceName.value = "";
    }
  };

  // Update query params when selection changes
  const updateQueryParams = (
    serviceName?: string,
    containerId?: string
  ) => {
    const currentQuery = { ...route.query };
    const query: Record<string, string | string[] | undefined> = {};

    // Copy existing query params except service/container
    Object.keys(currentQuery).forEach((key) => {
      if (key !== "service" && key !== "container") {
        const value = currentQuery[key];
        if (value !== null && value !== undefined) {
          query[key] = value as string | string[];
        }
      }
    });

    if (serviceName) {
      query.service = serviceName;
    } else if (containerId) {
      query.container = containerId;
    }
    // If neither provided, we just don't add service/container to query

    router.replace({
      query: Object.keys(query).length > 0 ? query : undefined,
    });
  };

  // Set container selection
  const setContainer = (serviceName?: string, containerId?: string) => {
    const container = findContainer(serviceName, containerId);
    
    if (container) {
      if (container.serviceName) {
        selectedServiceName.value = container.serviceName;
        selectedContainerId.value = container.serviceName;
        updateQueryParams(container.serviceName, undefined);
      } else {
        selectedServiceName.value = "";
        selectedContainerId.value = container.containerId;
        updateQueryParams(undefined, container.containerId);
      }
    } else {
      selectedContainerId.value = "";
      selectedServiceName.value = "";
      updateQueryParams(undefined, undefined);
    }
  };

  // Get current selected container object
  const selectedContainer = computed<Container | null>(() => {
    if (!selectedContainerId.value) {
      return containers.value.length > 0 && containers.value[0] 
        ? containers.value[0] 
        : null;
    }

    const found = containers.value.find(
      (c) =>
        (c.serviceName || c.containerId) === selectedContainerId.value
    );
    
    return found || null;
  });

  // Watch for query param changes (e.g., browser back/forward)
  watch(
    () => route.query.service,
    () => {
      if (containers.value.length > 0) {
        syncWithQueryParams();
      }
    }
  );

  watch(
    () => route.query.container,
    () => {
      if (containers.value.length > 0) {
        syncWithQueryParams();
      }
    }
  );

  // Watch for deployment changes
  watch(
    () => deploymentId,
    async () => {
      selectedContainerId.value = "";
      selectedServiceName.value = "";
      await loadContainers();
    }
  );

  onMounted(() => {
    loadContainers();
  });

  return {
    containers: computed(() => containers.value),
    isLoading: computed(() => isLoading.value),
    selectedContainerId: computed(() => selectedContainerId.value),
    selectedServiceName: computed(() => selectedServiceName.value),
    selectedContainer,
    loadContainers,
    setContainer,
    refresh: loadContainers,
  };
}

