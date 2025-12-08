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

      <!-- Editor Preferences -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Editor Preferences</OuiText>
            <OuiText size="xs" color="secondary">
              Customize the file editor appearance and behavior
            </OuiText>

            <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
              <!-- Word Wrap -->
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Word Wrap</OuiText>
                <OuiRadioGroup
                  v-model="editorWordWrap"
                  :options="[
                    { label: 'Off', value: 'off' },
                    { label: 'On', value: 'on' },
                    { label: 'Word Wrap Column', value: 'wordWrapColumn' },
                    { label: 'Bounded', value: 'bounded' },
                  ]"
                />
              </OuiStack>

              <!-- Tab Size -->
              <OuiInput
                v-model.number="editorTabSize"
                type="number"
                label="Tab Size"
                :min="1"
                :max="8"
                helper-text="Number of spaces per tab"
              />

              <!-- Font Size -->
              <OuiInput
                v-model.number="editorFontSize"
                type="number"
                label="Font Size"
                :min="10"
                :max="24"
                helper-text="Editor font size in pixels"
              />

              <!-- Line Numbers -->
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Line Numbers</OuiText>
                <OuiRadioGroup
                  v-model="editorLineNumbers"
                  :options="[
                    { label: 'On', value: 'on' },
                    { label: 'Off', value: 'off' },
                    { label: 'Relative', value: 'relative' },
                    { label: 'Interval', value: 'interval' },
                  ]"
                />
              </OuiStack>

              <!-- Insert Spaces -->
              <OuiFlex justify="between" align="center">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Insert Spaces</OuiText>
                  <OuiText size="xs" color="secondary">
                    Use spaces instead of tabs
                  </OuiText>
                </OuiStack>
                <OuiSwitch v-model="editorInsertSpaces" />
              </OuiFlex>

              <!-- Minimap -->
              <OuiFlex justify="between" align="center">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Minimap</OuiText>
                  <OuiText size="xs" color="secondary">
                    Show code minimap on the right
                  </OuiText>
                </OuiStack>
                <OuiSwitch v-model="editorMinimap" />
              </OuiFlex>

              <!-- Render Whitespace -->
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Render Whitespace</OuiText>
                <OuiRadioGroup
                  v-model="editorRenderWhitespace"
                  :options="[
                    { label: 'None', value: 'none' },
                    { label: 'Boundary', value: 'boundary' },
                    { label: 'Selection', value: 'selection' },
                    { label: 'Trailing', value: 'trailing' },
                    { label: 'All', value: 'all' },
                  ]"
                />
              </OuiStack>
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Notification Preferences -->
      <SettingsNotificationPreferences />
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { usePreferencesStore } from "~/stores/preferences";
  import OuiRadioGroup from "~/components/oui/RadioGroup.vue";
  import OuiInput from "~/components/oui/Input.vue";
  import OuiSwitch from "~/components/oui/Switch.vue";
  import OuiGrid from "~/components/oui/Grid.vue";
  import SettingsNotificationPreferences from "./SettingsNotificationPreferences.vue";

  const preferencesStore = usePreferencesStore();

  const envVarsViewMode = computed({
    get: () => preferencesStore.envVarsViewMode,
    set: (value) => preferencesStore.setEnvVarsViewMode(value),
  });

  const editorPreferences = computed(() => preferencesStore.editorPreferences);

  const editorWordWrap = computed({
    get: () => editorPreferences.value.wordWrap,
    set: (value) => preferencesStore.setEditorPreference("wordWrap", value),
  });

  const editorTabSize = computed({
    get: () => String(editorPreferences.value.tabSize),
    set: (value) => preferencesStore.setEditorPreference("tabSize", Number(value) || 2),
  });

  const editorFontSize = computed({
    get: () => String(editorPreferences.value.fontSize),
    set: (value) => preferencesStore.setEditorPreference("fontSize", Number(value) || 14),
  });

  const editorLineNumbers = computed({
    get: () => editorPreferences.value.lineNumbers,
    set: (value) => preferencesStore.setEditorPreference("lineNumbers", value),
  });

  const editorInsertSpaces = computed({
    get: () => editorPreferences.value.insertSpaces,
    set: (value) => preferencesStore.setEditorPreference("insertSpaces", value),
  });

  const editorMinimap = computed({
    get: () => editorPreferences.value.minimap,
    set: (value) => preferencesStore.setEditorPreference("minimap", value),
  });

  const editorRenderWhitespace = computed({
    get: () => editorPreferences.value.renderWhitespace,
    set: (value) => preferencesStore.setEditorPreference("renderWhitespace", value),
  });
</script>
