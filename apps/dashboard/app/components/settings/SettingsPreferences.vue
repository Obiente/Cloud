<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <OuiText as="h2" size="lg" weight="semibold">Preferences</OuiText>

      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText size="sm" color="secondary">
              User preferences and display settings
            </OuiText>

            <!-- Environment Variables View Mode -->
            <OuiFlex justify="between" align="center">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium"
                  >Environment Variables View</OuiText
                >
                <OuiText size="xs" color="secondary">
                  Choose how to view environment variables by default
                </OuiText>
              </OuiStack>
              <OuiRadioGroup
                v-model="envVarsViewMode"
                :options="[
                  { label: 'List View', value: 'list' },
                  { label: 'File View', value: 'file' },
                ]"
              />
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { usePreferencesStore } from "~/stores/preferences";
  import OuiRadioGroup from "~/components/oui/RadioGroup.vue";

  const preferencesStore = usePreferencesStore();

  const envVarsViewMode = computed({
    get: () => preferencesStore.envVarsViewMode,
    set: (value) => preferencesStore.setEnvVarsViewMode(value),
  });
</script>
