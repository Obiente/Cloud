<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText tag="h1" size="3xl" weight="extrabold">VPS Management</OuiText>
      <OuiText color="muted">
        Manage VPS instances, size catalog, and public IP addresses across every organization.
      </OuiText>
    </OuiStack>

    <OuiTabs :tabs="tabs" v-model="activeTab" />
  </OuiStack>
</template>

<script setup lang="ts">
import { ServerIcon, CubeIcon, GlobeAltIcon, WifiIcon } from "@heroicons/vue/24/outline";
import { ref } from "vue";
import OuiTabs from "~/components/oui/Tabs.vue";
import VPSInstancesTab from "~/components/superadmin/VPSInstancesTab.vue";
import VPSSizesTab from "~/components/superadmin/VPSSizesTab.vue";
import VPSPublicIPsTab from "~/components/superadmin/VPSPublicIPsTab.vue";
import VPSLeasesTab from "~/components/superadmin/VPSLeasesTab.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const activeTab = ref("instances");

const tabs = [
  {
    id: "instances",
    label: "Instances",
    icon: ServerIcon,
    component: VPSInstancesTab,
  },
  {
    id: "sizes",
    label: "Sizes",
    icon: CubeIcon,
    component: VPSSizesTab,
  },
  {
    id: "public-ips",
    label: "Public IPs",
    icon: GlobeAltIcon,
    component: VPSPublicIPsTab,
  },
  {
    id: "dhcp-leases",
    label: "DHCP Leases",
    icon: WifiIcon,
    component: VPSLeasesTab,
  },
];
</script>
