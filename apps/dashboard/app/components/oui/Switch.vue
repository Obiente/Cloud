<template>
  <Field.Root
    :invalid="!!error"
    :required="required"
    :disabled="disabled"
    class="oui-field space-y-1"
  >
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </Field.Label>

    <Switch.Root v-model:checked="modelValue" :disabled="disabled">
      <Switch.Control :class="trackClasses">
        <Switch.Thumb :class="thumbClasses" />
      </Switch.Control>
      <Switch.Label class="sr-only">{{ label || "Switch" }}</Switch.Label>
      <Switch.HiddenInput />
    </Switch.Root>

    <Field.ErrorText v-if="error" class="text-sm text-danger">{{
      error
    }}</Field.ErrorText>
    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">{{
      helperText
    }}</Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { Field } from "@ark-ui/vue/field";
  import { Switch } from "@ark-ui/vue/switch";

  interface Props {
    label?: string;
    helperText?: string;
    error?: string;
    required?: boolean;
    disabled?: boolean;
    size?: "sm" | "md" | "lg";
  }

  const props = withDefaults(defineProps<Props>(), {
    required: false,
    disabled: false,
    size: "md",
  });

  const modelValue = defineModel<boolean>({ default: false });

  const trackClasses = computed(() => [
    "relative inline-flex items-center rounded-full border oui-input",
    props.error ? "oui-input-error" : "oui-input-base",
    props.disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
    props.size === "sm"
      ? "h-5 w-9"
      : props.size === "lg"
      ? "h-7 w-14"
      : "h-6 w-11",
    modelValue.value ? "bg-surface-muted" : "bg-surface-muted/50",
  ]);

  const thumbClasses = computed(() => {
    const size =
      props.size === "sm"
        ? "h-4 w-4"
        : props.size === "lg"
        ? "h-6 w-6"
        : "h-5 w-5";
    const translate =
      props.size === "sm"
        ? modelValue.value
          ? "translate-x-4"
          : "translate-x-0"
        : props.size === "lg"
        ? modelValue.value
          ? "translate-x-7"
          : "translate-x-0"
        : modelValue.value
        ? "translate-x-5"
        : "translate-x-0";
    return [
      "absolute bg-surface-base rounded-full shadow transition-transform",
      "right-1/2",
      size,
      translate,
    ];
  });

  defineOptions({ inheritAttrs: false });
</script>
