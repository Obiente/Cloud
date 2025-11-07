<template>
  <OuiContainer size="full" py="xl">
    <OuiStack gap="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary">
            Global Audit Logs
          </OuiText>
          <OuiText color="secondary">
            View all audit logs across all resources and organizations. Only accessible to superadmins.
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Audit Logs Component (no organizationId prop = shows ALL audit logs from ALL organizations) -->
      <AuditLogs />
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import AuditLogs from "~/components/audit/AuditLogs.vue";

// Page meta - require superadmin access
definePageMeta({
  layout: "default",
  middleware: ["auth", "superadmin"],
});
</script>

