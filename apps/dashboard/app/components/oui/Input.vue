<template>
  <Field.Root
    :id="fieldId"
    :invalid="!!error"
    :required="required"
    :disabled="disabled"
    :read-only="$props.readonly"
    class="oui-field space-y-1"
  >
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
      <Field.RequiredIndicator v-if="required" class="text-danger">
        *
      </Field.RequiredIndicator>
    </Field.Label>

    <div class="relative">
      <Field.Input
        :type="type"
        :placeholder="placeholder"
        :class="[
          inputClasses,
          {
            'pl-10': $slots.prefix,
            'pr-10': $slots.suffix || clearable,
          },
        ]"
        v-model="modelValue"
        @blur="handleBlur"
        @focus="handleFocus"
        v-bind="$attrs"
      />

      <div
        v-if="$slots.prefix"
        class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none"
      >
        <slot name="prefix" />
      </div>

      <div
        v-if="$slots.suffix || clearable"
        class="absolute inset-y-0 right-0 flex items-center pr-3"
      >
        <button
          v-if="clearable && modelValue"
          type="button"
          @click="handleClear"
          class="text-text-secondary hover:text-primary transition-colors"
        >
          <XMarkIcon class="h-4 w-4" />
        </button>
        <div v-else class="pointer-events-none">
          <slot name="suffix" />
        </div>
      </div>
    </div>

    <Field.ErrorText v-if="error" class="text-sm text-danger">
      {{ error }}
    </Field.ErrorText>

    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">
      {{ helperText }}
    </Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
import { XMarkIcon } from "@heroicons/vue/24/outline";
import { Field } from "@ark-ui/vue/field";
import type { InputHTMLAttributes } from "vue";

interface Props extends /* @vue-ignore */ InputHTMLAttributes {
  modelValue?: string;
  label?: string;
  placeholder?: string;
  helperText?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  clearable?: boolean;
  size?: "sm" | "md" | "lg";
}

const props = withDefaults(defineProps<Props>(), {
  type: "text",
  clearable: false,
  size: "md",
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  blur: [event: FocusEvent];
  focus: [event: FocusEvent];
}>();

const fieldId = computed(
  () => `field-${Math.random().toString(36).substr(2, 9)}`
);

const modelValue = computed({
  get: () => props.modelValue,
  set: (value: string) => emit("update:modelValue", value),
});

const inputClasses = computed(() => [
  "oui-input",
  `oui-input-${props.size}`,
  props.error ? "oui-input-error" : "oui-input-base",
]);

const handleBlur = (event: FocusEvent) => {
  emit("blur", event);
};

const handleFocus = (event: FocusEvent) => {
  emit("focus", event);
};

const handleClear = () => {
  emit("update:modelValue", "");
};

defineOptions({
  inheritAttrs: false,
});
</script>
