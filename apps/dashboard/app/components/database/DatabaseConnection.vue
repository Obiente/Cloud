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
            <!-- Quick Connection Info -->
            <OuiGrid cols="1" cols-md="2" gap="md">
              <OuiStack gap="sm">
                <OuiFlex justify="between" align="center">
                  <OuiText weight="semibold" size="sm">Host</OuiText>
                  <OuiButton
                    variant="ghost"
                    size="xs"
                    @click="copyToClipboard(connectionInfo.host, 'Host')"
                  >
                    Copy
                  </OuiButton>
                </OuiFlex>
                <OuiCode :code="connectionInfo.host" />
              </OuiStack>

              <OuiStack gap="sm">
                <OuiFlex justify="between" align="center">
                  <OuiText weight="semibold" size="sm">Port</OuiText>
                  <OuiButton
                    variant="ghost"
                    size="xs"
                    @click="copyToClipboard(String(connectionInfo.port), 'Port')"
                  >
                    Copy
                  </OuiButton>
                </OuiFlex>
                <OuiCode :code="String(connectionInfo.port)" />
              </OuiStack>

              <OuiStack gap="sm">
                <OuiFlex justify="between" align="center">
                  <OuiText weight="semibold" size="sm">Database Name</OuiText>
                  <OuiButton
                    variant="ghost"
                    size="xs"
                    @click="copyToClipboard(connectionInfo.databaseName, 'Database name')"
                  >
                    Copy
                  </OuiButton>
                </OuiFlex>
                <OuiCode :code="connectionInfo.databaseName" />
              </OuiStack>

              <OuiStack gap="sm">
                <OuiFlex justify="between" align="center">
                  <OuiText weight="semibold" size="sm">Username</OuiText>
                  <OuiButton
                    variant="ghost"
                    size="xs"
                    @click="copyToClipboard(connectionInfo.username, 'Username')"
                  >
                    Copy
                  </OuiButton>
                </OuiFlex>
                <OuiCode :code="connectionInfo.username" />
              </OuiStack>
            </OuiGrid>

            <!-- Password Section -->
            <OuiStack gap="sm">
              <OuiFlex justify="between" align="center">
                <OuiText weight="semibold">Password</OuiText>
                <OuiButton variant="outline" size="sm" @click="handleResetPassword">
                  Reset Password
                </OuiButton>
              </OuiFlex>
              <OuiAlert color="warning">
                <OuiText size="sm">
                  Password is only shown once during database creation or after a password reset.
                  If you've lost your password, use the "Reset Password" button above.
                </OuiText>
              </OuiAlert>
            </OuiStack>

            <!-- Connection Strings -->
            <OuiStack gap="sm">
              <OuiText weight="semibold">Connection Strings</OuiText>
              <OuiStack gap="md">
                <OuiStack gap="xs" v-if="connectionInfo.postgresqlUrl">
                  <OuiFlex justify="between" align="center">
                    <OuiText size="sm" color="secondary">PostgreSQL</OuiText>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="copyToClipboard(connectionInfo.postgresqlUrl, 'PostgreSQL connection string')"
                    >
                      Copy
                    </OuiButton>
                  </OuiFlex>
                  <OuiCode :code="connectionInfo.postgresqlUrl" class="text-xs break-all font-mono" />
                </OuiStack>

                <OuiStack gap="xs" v-if="connectionInfo.mysqlUrl">
                  <OuiFlex justify="between" align="center">
                    <OuiText size="sm" color="secondary">MySQL</OuiText>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="copyToClipboard(connectionInfo.mysqlUrl, 'MySQL connection string')"
                    >
                      Copy
                    </OuiButton>
                  </OuiFlex>
                  <OuiCode :code="connectionInfo.mysqlUrl" class="text-xs break-all font-mono" />
                </OuiStack>

                <OuiStack gap="xs" v-if="connectionInfo.mongodbUrl">
                  <OuiFlex justify="between" align="center">
                    <OuiText size="sm" color="secondary">MongoDB</OuiText>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="copyToClipboard(connectionInfo.mongodbUrl, 'MongoDB connection string')"
                    >
                      Copy
                    </OuiButton>
                  </OuiFlex>
                  <OuiCode :code="connectionInfo.mongodbUrl" class="text-xs break-all font-mono" />
                </OuiStack>

                <OuiStack gap="xs" v-if="connectionInfo.redisUrl">
                  <OuiFlex justify="between" align="center">
                    <OuiText size="sm" color="secondary">Redis</OuiText>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="copyToClipboard(connectionInfo.redisUrl, 'Redis connection string')"
                    >
                      Copy
                    </OuiButton>
                  </OuiFlex>
                  <OuiCode :code="connectionInfo.redisUrl" class="text-xs break-all font-mono" />
                </OuiStack>
              </OuiStack>
            </OuiStack>

            <!-- Connection Instructions -->
            <OuiAlert color="info">
              <OuiStack gap="sm">
                <OuiText size="sm" weight="semibold">Connection Instructions</OuiText>
                <OuiText size="sm" class="whitespace-pre-line">{{ connectionInfo.connectionInstructions }}</OuiText>
              </OuiStack>
            </OuiAlert>

            <!-- SSL Info -->
            <OuiAlert v-if="connectionInfo.sslRequired" color="default">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="semibold">SSL/TLS Required</OuiText>
                <OuiText size="sm">This database requires encrypted connections for security.</OuiText>
              </OuiStack>
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

async function copyToClipboard(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(`${label} copied to clipboard`);
  } catch (err) {
    toast.error("Failed to copy to clipboard");
  }
}

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
  } catch (err: unknown) {
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
  } catch (err: unknown) {
    toast.error("Failed to reset password", (err as Error).message);
  }
}

onMounted(() => {
  loadConnectionInfo();
});
</script>
