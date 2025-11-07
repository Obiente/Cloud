<template>
  <OuiContainer size="full" py="xl">
    <OuiStack gap="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary">
            Audit Logs
          </OuiText>
          <OuiText color="secondary">
            View all audit logs across all resources and organizations.
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Audit Logs Component (no filters for resource type/resource id) -->
      <AuditLogs :organization-id="organizationId" />
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import AuditLogs from "~/components/audit/AuditLogs.vue";
import { useOrganizationId } from "~/composables/useOrganizationId";

// Page meta
definePageMeta({
  layout: "default",
  middleware: "auth",
});

// Get organizationId using SSR-compatible composable
const organizationId = useOrganizationId();
</script>

