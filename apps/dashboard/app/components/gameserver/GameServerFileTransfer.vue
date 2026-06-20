<template>
  <OuiStack gap="lg">
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
            <OuiStack gap="xs" class="min-w-0">
              <OuiText as="h2" size="sm" weight="semibold">SFTP Access</OuiText>
              <OuiText size="sm" color="tertiary">
                Transfer large files, folders, mods, plugins, and worlds with
                any SFTP client.
              </OuiText>
            </OuiStack>
            <OuiButton
              variant="ghost"
              size="sm"
              :loading="loading"
              @click="loadCredentials"
            >
              <ArrowPathIcon
                class="h-3.5 w-3.5"
                :class="{ 'animate-spin': loading }"
              />
              Refresh
            </OuiButton>
          </OuiFlex>

          <ErrorAlert
            v-if="error"
            :error="error"
            title="Failed to load SFTP credentials"
          />

          <div
            v-if="connection"
            class="grid grid-cols-1 md:grid-cols-3 gap-px bg-border-default rounded-lg overflow-hidden border border-border-default"
          >
            <div class="bg-surface-base px-4 py-3">
              <UiCopyField label="Host" :value="connection.host" />
            </div>
            <div class="bg-surface-base px-4 py-3">
              <UiCopyField label="Port" :value="String(connection.port)" />
            </div>
            <div class="bg-surface-base px-4 py-3">
              <UiCopyField
                label="Username"
                :value="connection.username"
                break-all
              />
            </div>
          </div>

          <UiCodeBlock
            v-if="connection?.command"
            label="SFTP command"
            :value="connection.command"
            break-all
          />

          <OuiText size="xs" color="tertiary">
            Passwords are shown once when a credential is created. Store the
            password before closing this page.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard
      v-if="createdCredential"
      variant="outline"
      class="border-success/30 bg-success/5"
    >
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex align="start" gap="sm">
            <CheckCircleIcon class="mt-0.5 h-4 w-4 shrink-0 text-success" />
            <OuiStack gap="xs" class="min-w-0">
              <OuiText size="sm" weight="semibold">Credential created</OuiText>
              <OuiText size="sm" color="tertiary">
                Copy the password now. It will not be shown again.
              </OuiText>
            </OuiStack>
          </OuiFlex>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <UiCopyField
              label="Username"
              :value="createdCredential.connection.username"
              variant="field"
              break-all
            />
            <UiCopyField
              label="Password"
              :value="createdCredential.password"
              variant="field"
              break-all
            />
          </div>
          <UiCodeBlock
            label="SFTP command"
            :value="createdCredential.connection.command"
            break-all
          />
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard variant="outline">
      <OuiCardBody>
        <form @submit.prevent="createCredential">
          <OuiStack gap="md">
            <OuiText as="h3" size="sm" weight="semibold"
              >Create Credential</OuiText
            >

            <div class="grid grid-cols-1 md:grid-cols-[1fr_180px] gap-4">
              <OuiInput
                v-model="form.name"
                label="Name"
                placeholder="Local SFTP client"
                :disabled="creating"
                required
              />
              <OuiSelect
                v-model="form.expiresIn"
                label="Expires"
                :items="expirationOptions"
                :disabled="creating"
              />
            </div>

            <OuiFlex gap="lg" align="center" wrap="wrap">
              <OuiCheckbox
                v-model="form.read"
                label="Read files"
                :disabled="creating || !form.write"
              />
              <OuiCheckbox
                v-model="form.write"
                label="Write files"
                :disabled="creating || !form.read"
              />
            </OuiFlex>

            <OuiFlex justify="end">
              <OuiButton type="submit" color="primary" :loading="creating">
                <KeyIcon class="h-4 w-4" />
                Create SFTP Credential
              </OuiButton>
            </OuiFlex>
          </OuiStack>
        </form>
      </OuiCardBody>
    </OuiCard>

    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
            <OuiText as="h3" size="sm" weight="semibold"
              >Active Credentials</OuiText
            >
            <OuiBadge size="xs" variant="secondary">{{
              credentials.length
            }}</OuiBadge>
          </OuiFlex>

          <div
            v-if="loading && credentials.length === 0"
            class="py-8 text-center"
          >
            <OuiSpinner />
          </div>

          <OuiStack v-else-if="credentials.length > 0" gap="sm">
            <div
              v-for="credential in credentials"
              :key="credential.id"
              class="flex items-start justify-between gap-4 rounded-lg border border-border-default px-4 py-3"
            >
              <OuiStack gap="xs" class="min-w-0">
                <OuiFlex align="center" gap="sm" wrap="wrap">
                  <OuiText size="sm" weight="medium" truncate>{{
                    credential.name
                  }}</OuiText>
                  <OuiBadge
                    v-for="scope in credential.scopes"
                    :key="scope"
                    size="xs"
                    variant="secondary"
                  >
                    {{ scope }}
                  </OuiBadge>
                </OuiFlex>
                <OuiText size="xs" color="tertiary">
                  Created
                  <OuiRelativeTime
                    :value="
                      credential.createdAt
                        ? date(credential.createdAt)
                        : undefined
                    "
                    :style="'short'"
                  />
                  <template v-if="credential.lastUsedAt">
                    · Last used
                    <OuiRelativeTime
                      :value="date(credential.lastUsedAt)"
                      :style="'short'"
                    />
                  </template>
                  <template v-if="credential.expiresAt">
                    · Expires
                    <OuiRelativeTime
                      :value="date(credential.expiresAt)"
                      :style="'short'"
                    />
                  </template>
                </OuiText>
              </OuiStack>
              <OuiButton
                variant="ghost"
                color="danger"
                size="sm"
                :loading="revokingId === credential.id"
                @click="revokeCredential(credential.id)"
              >
                <TrashIcon class="h-3.5 w-3.5" />
                Revoke
              </OuiButton>
            </div>
          </OuiStack>

          <OuiText v-else size="sm" color="tertiary" class="py-8 text-center">
            No active SFTP credentials.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import {
  ArrowPathIcon,
  CheckCircleIcon,
  KeyIcon,
  TrashIcon,
} from "@heroicons/vue/24/outline";
import type {
  GameServerFileTransferConnectionInfo,
  GameServerFileTransferCredential,
} from "@obiente/proto";
import { ConnectError } from "@connectrpc/connect";
import { GameServerService } from "@obiente/proto";
import { date, timestamp } from "@obiente/proto/utils";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import ErrorAlert from "~/components/ErrorAlert.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

const props = defineProps<{
  gameServerId: string;
}>();

const client = useConnectClient(GameServerService);
const { toast } = useToast();

const loading = ref(false);
const creating = ref(false);
const revokingId = ref<string | null>(null);
const error = ref<Error | ConnectError | null>(null);
const credentials = ref<GameServerFileTransferCredential[]>([]);
const connection = ref<GameServerFileTransferConnectionInfo | null>(null);
const createdCredential = ref<{
  password: string;
  connection: GameServerFileTransferConnectionInfo;
} | null>(null);

const form = reactive({
  name: "SFTP access",
  expiresIn: "30d",
  read: true,
  write: true,
});

const expirationOptions = [
  { label: "7 days", value: "7d" },
  { label: "30 days", value: "30d" },
  { label: "90 days", value: "90d" },
  { label: "Never", value: "never" },
];

const selectedScopes = computed(() => {
  const scopes: string[] = [];
  if (form.read) scopes.push("read");
  if (form.write) scopes.push("write");
  return scopes;
});

async function loadCredentials() {
  loading.value = true;
  error.value = null;
  try {
    const res = await client.listGameServerFileTransferCredentials({
      gameServerId: props.gameServerId,
    });
    credentials.value = res.credentials ?? [];
    connection.value = res.connection ?? null;
  } catch (err) {
    error.value = normalizeError(err);
  } finally {
    loading.value = false;
  }
}

async function createCredential() {
  if (selectedScopes.value.length === 0) {
    toast.error("Select at least one permission");
    return;
  }

  creating.value = true;
  error.value = null;
  try {
    const res = await client.createGameServerFileTransferCredential({
      gameServerId: props.gameServerId,
      name: form.name.trim() || "SFTP access",
      scopes: selectedScopes.value,
      expiresAt: expiresAtValue(),
    });

    if (res.connection && res.password) {
      createdCredential.value = {
        password: res.password,
        connection: res.connection,
      };
    }
    toast.success("SFTP credential created");
    await loadCredentials();
  } catch (err) {
    error.value = normalizeError(err);
    toast.error("Failed to create SFTP credential");
  } finally {
    creating.value = false;
  }
}

async function revokeCredential(credentialId: string) {
  revokingId.value = credentialId;
  error.value = null;
  try {
    await client.revokeGameServerFileTransferCredential({
      gameServerId: props.gameServerId,
      credentialId,
    });
    if (createdCredential.value) {
      createdCredential.value = null;
    }
    toast.success("SFTP credential revoked");
    await loadCredentials();
  } catch (err) {
    error.value = normalizeError(err);
    toast.error("Failed to revoke SFTP credential");
  } finally {
    revokingId.value = null;
  }
}

function expiresAtValue() {
  if (form.expiresIn === "never") {
    return undefined;
  }
  const days = Number(form.expiresIn.replace("d", ""));
  if (!Number.isFinite(days) || days <= 0) {
    return undefined;
  }
  const expiresAt = new Date();
  expiresAt.setDate(expiresAt.getDate() + days);
  return timestamp(expiresAt);
}

function normalizeError(err: unknown): Error | ConnectError {
  if (err instanceof Error || err instanceof ConnectError) {
    return err;
  }
  return new Error(String(err));
}

onMounted(loadCredentials);
</script>
