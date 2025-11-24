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
            View audit logs for your organization. Accessible to organization Admins and Owners.
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Audit Logs Component (scoped to selected organization) -->
      <AuditLogs :organization-id="organizationId" />
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import AuditLogs from "~/components/audit/AuditLogs.vue";
import { useOrganizationId } from "~/composables/useOrganizationId";

// Page meta - require auth (API will check for admin/superadmin role)
definePageMeta({
  layout: "default",
  middleware: "auth",
});

// Get organizationId using SSR-compatible composable
const organizationId = useOrganizationId();
</script>

