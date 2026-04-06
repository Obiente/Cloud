<template>
  <OuiStack gap="md">
    <!-- Loading State -->
    <OuiCard v-if="loading" variant="outline">
      <OuiCardBody>
        <OuiStack align="center" gap="md" class="py-10">
          <OuiSpinner size="lg" />
          <OuiText size="sm" color="tertiary">Loading connection info...</OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Error State -->
    <ErrorAlert v-else-if="error" :error="error" title="Failed to load connection info" />

    <!-- Connection Info -->
    <template v-else-if="connectionInfo">
      <!-- Connection Details Grid -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Connection Details</OuiText>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-px bg-border-default rounded-lg overflow-hidden border border-border-default">
              <div class="bg-surface-base px-4 py-3">
                <UiCopyField label="Host" :value="connectionInfo.host" />
              </div>
              <div class="bg-surface-base px-4 py-3">
                <UiCopyField label="Port" :value="String(connectionInfo.port)" />
              </div>
              <div class="bg-surface-base px-4 py-3">
                <UiCopyField label="Database Name" :value="connectionInfo.databaseName" />
              </div>
              <div class="bg-surface-base px-4 py-3">
                <UiCopyField label="Username" :value="connectionInfo.username" />
              </div>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Password -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="semibold">Password</OuiText>
              <OuiText size="xs" color="tertiary">
                Only shown during creation or after reset.
              </OuiText>
            </OuiStack>
            <OuiButton variant="outline" size="sm" @click="handleResetPassword">
              Reset Password
            </OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Connection Strings -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Connection Strings</OuiText>

            <OuiStack gap="sm">
              <UiCodeBlock
                v-for="cs in connectionStrings"
                :key="cs.label"
                :label="cs.label"
                :value="cs.value"
                break-all
              />
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Connection Instructions -->
      <OuiCard v-if="connectionInfo.connectionInstructions" variant="outline" status="info">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">Connection Instructions</OuiText>
            <OuiText size="xs" color="tertiary" class="whitespace-pre-line">{{ connectionInfo.connectionInstructions }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- SSL -->
      <OuiCard v-if="connectionInfo.sslRequired" variant="outline" status="success">
        <OuiCardBody>
          <OuiFlex align="center" gap="sm">
            <ShieldCheckIcon class="h-4 w-4 text-success shrink-0" />
            <OuiText size="sm">SSL/TLS connections required</OuiText>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>
    </template>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ShieldCheckIcon } from "@heroicons/vue/24/outline";
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

const connectionStrings = computed(() => {
  if (!connectionInfo.value) return [];
  const strings: { label: string; value: string }[] = [];
  if (connectionInfo.value.postgresqlUrl) strings.push({ label: 'PostgreSQL', value: connectionInfo.value.postgresqlUrl });
  if (connectionInfo.value.mysqlUrl) strings.push({ label: 'MySQL', value: connectionInfo.value.mysqlUrl });
  if (connectionInfo.value.mongodbUrl) strings.push({ label: 'MongoDB', value: connectionInfo.value.mongodbUrl });
  if (connectionInfo.value.redisUrl) strings.push({ label: 'Redis', value: connectionInfo.value.redisUrl });
  return strings;
});

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
