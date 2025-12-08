<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiCardTitle>Connection Information</OuiCardTitle>
        <OuiCardDescription>
          Connection details for your database
        </OuiCardDescription>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <!-- Loading State -->
          <OuiStack v-if="loading" align="center" gap="md" class="py-10">
            <OuiSpinner size="lg" />
            <OuiText color="secondary">Loading connection info...</OuiText>
          </OuiStack>

          <!-- Connection Info -->
          <OuiStack v-else-if="connectionInfo" gap="lg">
            <OuiStack gap="sm">
              <OuiText weight="semibold">Host</OuiText>
              <OuiCode :code="connectionInfo.host" />
            </OuiStack>

            <OuiStack gap="sm">
              <OuiText weight="semibold">Port</OuiText>
              <OuiCode :code="String(connectionInfo.port)" />
            </OuiStack>

            <OuiStack gap="sm">
              <OuiText weight="semibold">Database Name</OuiText>
              <OuiCode :code="connectionInfo.databaseName" />
            </OuiStack>

            <OuiStack gap="sm">
              <OuiText weight="semibold">Username</OuiText>
              <OuiCode :code="connectionInfo.username" />
            </OuiStack>

            <OuiStack gap="sm">
              <OuiFlex justify="between" align="center">
                <OuiText weight="semibold">Password</OuiText>
                <OuiButton variant="ghost" size="sm" @click="handleResetPassword">
                  Reset Password
                </OuiButton>
              </OuiFlex>
              <OuiText color="secondary" size="sm">
                Password is only shown once during database creation or password reset.
              </OuiText>
            </OuiStack>

            <OuiStack gap="sm">
              <OuiText weight="semibold">Connection Strings</OuiText>
              <OuiStack gap="xs">
                <OuiStack gap="xs">
                  <OuiText size="sm" color="secondary">PostgreSQL</OuiText>
                  <OuiCode :code="connectionInfo.postgresqlUrl" class="text-xs break-all" />
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" color="secondary">MySQL</OuiText>
                  <OuiCode :code="connectionInfo.mysqlUrl" class="text-xs break-all" />
                </OuiStack>
              </OuiStack>
            </OuiStack>

            <OuiAlert color="info">
              <OuiText size="sm">{{ connectionInfo.connectionInstructions }}</OuiText>
            </OuiAlert>
          </OuiStack>

          <!-- Error State -->
          <ErrorAlert v-else-if="error" :error="error" title="Failed to load connection info" />
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { DatabaseService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import ErrorAlert from "~/components/ErrorAlert.vue";

const props = defineProps<{
  databaseId: string;
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const loading = ref(false);
const connectionInfo = ref<any>(null);
const error = ref<any>(null);

async function loadConnectionInfo() {
  loading.value = true;
  error.value = null;

  try {
    if (!organizationId.value) return;
    const res = await dbClient.getDatabaseConnectionInfo({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
    });
    connectionInfo.value = res.connectionInfo;
  } catch (err: any) {
    error.value = err;
  } finally {
    loading.value = false;
  }
}

async function handleResetPassword() {
  if (!confirm("Are you sure you want to reset the database password? The new password will only be shown once.")) {
    return;
  }

  try {
    if (!organizationId.value) return;
    const res = await dbClient.resetDatabasePassword({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
    });
    toast.success("Password reset successfully");
    toast.info(
      `New password: ${res.newPassword}`,
      "Save this password - it won't be shown again!"
    );

    loadConnectionInfo();
  } catch (err: any) {
    toast.error("Failed to reset password", err.message);
  }
}

onMounted(() => {
  loadConnectionInfo();
});
</script>

