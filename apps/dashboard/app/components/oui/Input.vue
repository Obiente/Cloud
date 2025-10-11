<template>
  <div class="space-y-1">
    <label
      v-if="label"
      :for="inputId"
      class="block text-sm font-medium text-primary"
    >
      {{ label }}
      <span v-if="required" class="text-danger">*</span>
    </label>

    <div class="relative">
      <input
        :id="inputId"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :required="required"
        :disabled="disabled"
        :readonly="readonly"
        @input="handleInput"
        @blur="handleBlur"
        @focus="handleFocus"
        :class="[
          inputClasses,
          {
            'pl-10': $slots.prefix,
            'pr-10': $slots.suffix || clearable,
          },
        ]"
        v-bind="$attrs"
      />

      <div
        v-if="$slots.prefix"
        class="absolute inset-y-0 left-0 flex items-center pl-3"
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
        <slot v-else name="suffix" />
      </div>
    </div>

    <OuiText v-if="error" size="sm" color="danger">
      {{ error }}
    </OuiText>

    <OuiText v-else-if="helperText" size="sm" color="secondary">
      {{ helperText }}
    </OuiText>
  </div>
</template>

<script setup lang="ts">
import { XMarkIcon } from "@heroicons/vue/24/outline";
import OuiText from "./Text.vue";

interface Props {
  modelValue?: string;
  type?: "text" | "email" | "password" | "search" | "url" | "tel";
  label?: string;
  placeholder?: string;
  helperText?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  readonly?: boolean;
  clearable?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "text",
  clearable: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  blur: [event: FocusEvent];
  focus: [event: FocusEvent];
}>();

const inputId = `input-${Math.random().toString(36).substr(2, 9)}`;

const inputClasses = computed(() => [
  props.error ? "oui-input-error" : "oui-input-base",
]);

const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement;
  emit("update:modelValue", target.value);
};

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
