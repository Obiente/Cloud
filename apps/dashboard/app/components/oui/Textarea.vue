<template>
  <Field.Root
    :invalid="!!props.error"
    :required="props.required"
    :disabled="props.disabled"
    :read-only="props.readonly"
    class="oui-field space-y-1 w-full"
  >
    <Field.Label v-if="props.label" class="block text-sm font-medium text-primary">
      {{ props.label }}
      <Field.RequiredIndicator v-if="props.required" class="text-danger">
        *
      </Field.RequiredIndicator>
    </Field.Label>

    <div class="relative w-full">
      <textarea
        :placeholder="props.placeholder"
        :class="textareaClasses"
        :rows="props.rows"
        :disabled="props.disabled"
        :readonly="props.readonly"
        v-model="modelValue"
        @blur="handleBlur"
        @focus="handleFocus"
        v-bind="$attrs"
      />
    </div>

    <Field.ErrorText v-if="props.error" class="text-sm text-danger">
      {{ props.error }}
    </Field.ErrorText>

    <Field.HelperText v-else-if="props.helperText" class="text-sm text-secondary">
      {{ props.helperText }}
    </Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Field } from "@ark-ui/vue/field";

interface Props {
  label?: string;
  placeholder?: string;
  helperText?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  rows?: number;
  size?: "sm" | "md" | "lg";
}

const props = withDefaults(defineProps<Props>(), {
  size: "md",
  rows: 4,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  blur: [event: FocusEvent];
  focus: [event: FocusEvent];
}>();

const modelValue = defineModel<string>({ default: "" });

const textareaClasses = computed(() => {
  const base = "oui-textarea";
  const sizeClass =
    props.size === "sm"
      ? "oui-textarea-sm"
      : props.size === "lg"
      ? "oui-textarea-lg"
      : "";
  return `${base} ${sizeClass}`.trim();
});

const handleBlur = (event: FocusEvent) => {
  emit("blur", event);
};

const handleFocus = (event: FocusEvent) => {
  emit("focus", event);
};

defineOptions({
  inheritAttrs: false,
});
</script>

