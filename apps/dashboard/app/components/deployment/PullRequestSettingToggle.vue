<template>
  <OuiCard variant="outline" class="bg-surface-subtle/40">
    <OuiCardBody>
      <OuiFlex justify="between" align="center" gap="md">
        <OuiStack gap="xs" class="min-w-0">
          <OuiText size="sm" weight="semibold">{{ title }}</OuiText>
          <OuiText size="xs" color="tertiary">{{ description }}</OuiText>
        </OuiStack>
        <OuiSwitch
          :model-value="modelValue"
          size="md"
          @update:model-value="$emit('update:modelValue', $event)"
        />
      </OuiFlex>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import OuiSwitch from "~/components/oui/Switch.vue";
defineProps<{ modelValue: boolean; title: string; description: string }>();
defineEmits<{ "update:modelValue": [value: boolean] }>();
</script>
