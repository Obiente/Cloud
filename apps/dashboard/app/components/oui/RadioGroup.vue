<template>
  <Field.Root
    :invalid="!!error"
    :required="required"
    :disabled="disabled"
    class="oui-field w-full space-y-1"
  >
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </Field.Label>

    <RadioGroup.Root v-model="modelValue" :disabled="disabled">
      <div class="flex flex-col gap-2">
        <RadioGroup.Item
          v-for="opt in options"
          :key="opt.value"
          :value="opt.value"
          class="inline-flex items-center gap-2 select-none"
        >
          <RadioGroup.ItemControl
            :class="[
              'oui-input oui-input-sm rounded-full p-0 h-4 w-4 inline-flex items-center justify-center border ',
              error ? 'oui-input-error' : 'oui-input-base',
              opt.value === modelValue ? 'bg-surface-muted' : '',
            ]"
          >
          <!-- FIX: data-[state=checked] is not working -->
            <span
              class="h-2 w-2 rounded-full bg-primary opacity-0 data-[state=checked]:opacity-100 transition-opacity"
            />
          </RadioGroup.ItemControl>
          <RadioGroup.ItemText class="text-sm text-text-primary">{{
            opt.label
          }}</RadioGroup.ItemText>
          <RadioGroup.ItemHiddenInput />
        </RadioGroup.Item>
      </div>
    </RadioGroup.Root>

    <Field.ErrorText v-if="error" class="text-sm text-danger">{{
      error
    }}</Field.ErrorText>
    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">{{
      helperText
    }}</Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
  import { Field } from "@ark-ui/vue/field";
  import { RadioGroup } from "@ark-ui/vue/radio-group";

  interface Option {
    label: string;
    value: string;
  }

  interface Props {
    label?: string;
    helperText?: string;
    error?: string;
    required?: boolean;
    disabled?: boolean;
    options: Option[];
  }

  withDefaults(defineProps<Props>(), { required: false, disabled: false });

  const modelValue = defineModel<string>({ default: "" });

  defineOptions({ inheritAttrs: false });
</script>
