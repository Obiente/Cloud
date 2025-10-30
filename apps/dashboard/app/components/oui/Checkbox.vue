<template>
  <Field.Root :invalid="!!error" :required="required" :disabled="disabled" class="oui-field space-y-1 w-full">
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </Field.Label>

    <Checkbox.Root v-model:checked="modelValue" :disabled="disabled">
      <div class="inline-flex items-center gap-2 select-none">
        <Checkbox.Control
          :class="[
            'oui-input oui-input-sm inline-flex items-center justify-center rounded border',
            error ? 'oui-input-error' : 'oui-input-base',
            disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
            'h-4 w-4'
          ]"
        >
          <Checkbox.Indicator>
            <svg viewBox="0 0 24 24" class="h-3.5 w-3.5 text-primary">
              <path fill="currentColor" d="M9 16.2 4.8 12l-1.4 1.4L9 19 21 7l-1.4-1.4z" />
            </svg>
          </Checkbox.Indicator>
        </Checkbox.Control>
        <Checkbox.Label class="text-sm text-text-primary cursor-pointer">
          <slot />
        </Checkbox.Label>
      </div>
      <Checkbox.HiddenInput />
    </Checkbox.Root>

    <Field.ErrorText v-if="error" class="text-sm text-danger">{{ error }}</Field.ErrorText>
    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">{{ helperText }}</Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
import { Field } from '@ark-ui/vue/field'
import { Checkbox } from '@ark-ui/vue/checkbox'

interface Props {
  label?: string
  helperText?: string
  error?: string
  required?: boolean
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), { required: false, disabled: false })

const modelValue = defineModel<boolean>({ default: false })

defineOptions({ inheritAttrs: false })
</script>
