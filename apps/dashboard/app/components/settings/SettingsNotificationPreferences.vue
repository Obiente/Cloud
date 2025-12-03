<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <OuiStack gap="xs">
        <OuiText as="h2" size="lg" weight="semibold"
          >Notification Preferences</OuiText
        >
        <OuiText size="sm" color="secondary">
          Configure how and when you receive notifications for each type
        </OuiText>
      </OuiStack>

      <div v-if="loading" class="flex items-center justify-center py-8">
        <OuiText size="sm" color="secondary">Loading preferences...</OuiText>
      </div>

      <div v-else-if="error" class="rounded-lg bg-error/10 p-4">
        <OuiText size="sm" color="danger">{{ error }}</OuiText>
      </div>

      <div v-else-if="notificationTypes.length === 0" class="py-8">
        <OuiText size="sm" color="secondary"
          >No notification types available</OuiText
        >
      </div>

      <OuiStack v-else gap="md">
        <OuiCard
          v-for="type in notificationTypes"
          :key="type.type"
          variant="outline"
        >
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="semibold">{{ type.name }}</OuiText>
                <OuiText size="xs" color="secondary">
                  {{ type.description }}
                </OuiText>
              </OuiStack>

              <OuiGrid cols="1" cols-md="2" gap="md">
                <!-- Email Enabled -->
                <OuiFlex justify="between" align="center">
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium">Email Notifications</OuiText>
                    <OuiText size="xs" color="secondary">
                      Receive email notifications for this type
                    </OuiText>
                  </OuiStack>
                  <OuiSwitch
                    :model-value="getPreference(type.type)?.emailEnabled ?? type.defaultEmailEnabled"
                    @update:model-value="
                      updatePreference(type.type, { emailEnabled: $event })
                    "
                  />
                </OuiFlex>

                <!-- In-App Enabled -->
                <OuiFlex justify="between" align="center">
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium">In-App Notifications</OuiText>
                    <OuiText size="xs" color="secondary">
                      Show notifications in the app
                    </OuiText>
                  </OuiStack>
                  <OuiSwitch
                    :model-value="getPreference(type.type)?.inAppEnabled ?? type.defaultInAppEnabled"
                    @update:model-value="
                      updatePreference(type.type, { inAppEnabled: $event })
                    "
                  />
                </OuiFlex>

                <!-- Frequency -->
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Email Frequency</OuiText>
                  <OuiRadioGroup
                    :model-value="
                      String(
                        getPreference(type.type)?.frequency ??
                        NotificationFrequency.IMMEDIATE
                      )
                    "
                    :options="frequencyOptions"
                    @update:model-value="
                      updatePreference(type.type, {
                        frequency: Number($event) as NotificationFrequency,
                      })
                    "
                  />
                </OuiStack>

                <!-- Minimum Severity -->
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Minimum Severity</OuiText>
                  <OuiSelect
                    :model-value="
                      getPreference(type.type)?.minSeverity ??
                      type.defaultMinSeverity
                    "
                    :items="severityOptions"
                    @update:model-value="
                      updatePreference(type.type, { minSeverity: $event })
                    "
                  />
                </OuiStack>
              </OuiGrid>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <OuiFlex v-if="hasChanges" justify="end" gap="md">
          <OuiButton variant="outline" @click="resetChanges">
            Reset
          </OuiButton>
          <OuiButton
            :loading="saving"
            @click="savePreferences"
          >
            Save Preferences
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import {
  NotificationService,
  NotificationType,
  NotificationFrequency,
  NotificationSeverity,
  UpdateNotificationPreferencesRequestSchema,
  NotificationPreferenceSchema,
  type NotificationTypeInfo,
  type NotificationPreference,
} from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import OuiCard from "~/components/oui/Card.vue";
import OuiCardBody from "~/components/oui/CardBody.vue";
import OuiStack from "~/components/oui/Stack.vue";
import OuiText from "~/components/oui/Text.vue";
import OuiFlex from "~/components/oui/Flex.vue";
import OuiGrid from "~/components/oui/Grid.vue";
import OuiSwitch from "~/components/oui/Switch.vue";
import OuiRadioGroup from "~/components/oui/RadioGroup.vue";
import OuiSelect from "~/components/oui/Select.vue";
import OuiButton from "~/components/oui/Button.vue";
import { useToast } from "~/composables/useToast";

const client = useConnectClient(NotificationService);
const { toast } = useToast();

const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);
const notificationTypes = ref<NotificationTypeInfo[]>([]);
const preferences = ref<Map<NotificationType, NotificationPreference>>(
  new Map()
);
const originalPreferences = ref<
  Map<NotificationType, NotificationPreference>
>(new Map());
const pendingChanges = ref<
  Map<NotificationType, Partial<NotificationPreference>>
>(new Map());

const frequencyOptions = [
  { label: "Immediate", value: String(NotificationFrequency.IMMEDIATE) },
  { label: "Daily Digest", value: String(NotificationFrequency.DAILY) },
  { label: "Weekly Digest", value: String(NotificationFrequency.WEEKLY) },
  { label: "Never", value: String(NotificationFrequency.NEVER) },
];

const severityOptions = [
  { label: "Low", value: NotificationSeverity.LOW },
  { label: "Medium", value: NotificationSeverity.MEDIUM },
  { label: "High", value: NotificationSeverity.HIGH },
  { label: "Critical", value: NotificationSeverity.CRITICAL },
];

const hasChanges = computed(() => pendingChanges.value.size > 0);

function getPreference(
  type: NotificationType
): NotificationPreference | undefined {
  const pending = pendingChanges.value.get(type);
  const existing = preferences.value.get(type);
  if (pending) {
    return {
      ...existing,
      ...pending,
      notificationType: type,
    } as NotificationPreference;
  }
  return existing;
}

function updatePreference(
  type: NotificationType,
  updates: Partial<NotificationPreference>
) {
  const current = getPreference(type);
  const pending = pendingChanges.value.get(type) || {};
  pendingChanges.value.set(type, { ...pending, ...updates });
}

function resetChanges() {
  pendingChanges.value.clear();
}

async function loadNotificationTypes() {
  try {
    const response = await client.getNotificationTypes({});
    notificationTypes.value = response.types || [];
  } catch (err: any) {
    console.error("Failed to load notification types:", err);
    error.value = err.message || "Failed to load notification types";
  }
}

async function loadPreferences() {
  try {
    const response = await client.getNotificationPreferences({});
    const prefsMap = new Map<NotificationType, NotificationPreference>();
    (response.preferences || []).forEach((pref) => {
      if (pref.notificationType !== undefined) {
        prefsMap.set(pref.notificationType, pref);
      }
    });
    preferences.value = prefsMap;
    originalPreferences.value = new Map(prefsMap);
  } catch (err: any) {
    console.error("Failed to load preferences:", err);
    error.value = err.message || "Failed to load preferences";
  }
}

async function savePreferences() {
  if (!hasChanges.value) return;

  saving.value = true;
  error.value = null;

  try {
    // Build preferences array from pending changes
    const prefsToSave: NotificationPreference[] = [];
    
    // Include all existing preferences plus updates
    const allTypes = new Set([
      ...preferences.value.keys(),
      ...pendingChanges.value.keys(),
    ]);

    for (const type of allTypes) {
      const existing = preferences.value.get(type);
      const pending = pendingChanges.value.get(type);
      
      // Get default values from notification type
      const typeInfo = notificationTypes.value.find((t) => t.type === type);
      
      const pref = create(NotificationPreferenceSchema, {
        notificationType: type,
        emailEnabled:
          pending?.emailEnabled ??
          existing?.emailEnabled ??
          typeInfo?.defaultEmailEnabled ??
          false,
        inAppEnabled:
          pending?.inAppEnabled ??
          existing?.inAppEnabled ??
          typeInfo?.defaultInAppEnabled ??
          true,
        frequency:
          pending?.frequency ??
          existing?.frequency ??
          NotificationFrequency.IMMEDIATE,
        minSeverity:
          pending?.minSeverity ??
          existing?.minSeverity ??
          typeInfo?.defaultMinSeverity ??
          NotificationSeverity.LOW,
      });
      
      prefsToSave.push(pref);
    }

    const request = create(UpdateNotificationPreferencesRequestSchema, {
      preferences: prefsToSave,
    });

    const response = await client.updateNotificationPreferences(request);

    // Update local state only on success
    const prefsMap = new Map<NotificationType, NotificationPreference>();
    (response.preferences || []).forEach((pref) => {
      if (pref.notificationType !== undefined) {
        prefsMap.set(pref.notificationType, pref);
      }
    });
    preferences.value = prefsMap;
    originalPreferences.value = new Map(prefsMap);
    pendingChanges.value.clear();

    toast.success(
      "Preferences Saved",
      "Your notification preferences have been saved successfully."
    );
  } catch (err: any) {
    console.error("Failed to save preferences:", err);
    const errorMessage = err.message || "Failed to save preferences";
    error.value = errorMessage;
    // Don't update preferences on error - keep the existing state
    toast.error("Failed to Save Preferences", errorMessage);
  } finally {
    saving.value = false;
  }
}

onMounted(async () => {
  loading.value = true;
  try {
    await Promise.all([loadNotificationTypes(), loadPreferences()]);
  } finally {
    loading.value = false;
  }
});
</script>

