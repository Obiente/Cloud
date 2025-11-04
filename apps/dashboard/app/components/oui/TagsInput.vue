<template>
  <Field.Root
    :invalid="!!props.error || props.invalid"
    :required="props.required"
    :disabled="props.disabled"
    :read-only="props.readonly"
    class="oui-field space-y-1 w-full"
  >
    <Field.Label
      v-if="props.label"
      class="block text-sm font-medium text-primary"
    >
      {{ props.label }}
      <Field.RequiredIndicator v-if="props.required" class="text-danger">
        *
      </Field.RequiredIndicator>
    </Field.Label>

    <TagsInput.Root
      :model-value="modelValue"
      :disabled="props.disabled"
      :read-only="props.readonly"
      :invalid="!!props.error || props.invalid"
      :required="props.required"
      :validate="effectiveValidate"
      :max="props.max"
      :blur-behavior="props.blurBehavior"
      :editable="props.editable"
      :delimiter="props.delimiter"
      :add-on-paste="props.addOnPaste"
      @update:model-value="handleValueChange"
    >
      <TagsInput.Context v-slot="tagsInput">
        <TagsInput.Control
          :class="[
            'flex flex-wrap gap-2 p-2 border rounded-xl min-h-[2.5rem] transition-colors',
            'bg-surface-base',
            props.error || props.invalid
              ? 'border-danger'
              : 'border-border-default',
            {
              'opacity-50 cursor-not-allowed': props.disabled,
              'cursor-default': props.readonly,
            },
          ]"
        >
          <TagsInput.Item
            v-for="(value, index) in tagsInput.value"
            :key="index"
            :index="index"
            :value="value"
          >
            <TagsInput.ItemPreview>
              <OuiBadge
                variant="outline"
                size="sm"
                class="gap-1 flex items-center"
              >
                <TagsInput.ItemText>{{ value }}</TagsInput.ItemText>
                <TagsInput.ItemDeleteTrigger
                  v-if="!props.readonly && !props.disabled"
                  class="ml-1 hover:text-danger transition-colors cursor-pointer inline-flex items-center"
                  type="button"
                  aria-label="Remove tag"
                >
                  <XMarkIcon class="h-3 w-3" />
                </TagsInput.ItemDeleteTrigger>
              </OuiBadge>
            </TagsInput.ItemPreview>
            <TagsInput.ItemInput />
          </TagsInput.Item>
          <TagsInput.Input
            :placeholder="tagsInput.value.length === 0 ? props.placeholder : ''"
            :class="[
              'flex-1 rounded-sm min-w-[120px] border-0 outline-none bg-transparent text-sm',
              'placeholder:text-text-secondary',
              'focus:outline-none',
            ]"
          />
        </TagsInput.Control>
      </TagsInput.Context>
      <TagsInput.HiddenInput :name="props.name" />
    </TagsInput.Root>

    <Field.ErrorText v-if="props.error" class="text-sm text-danger">
      {{ props.error }}
    </Field.ErrorText>
    <Field.HelperText
      v-else-if="props.helperText"
      class="text-sm text-secondary"
    >
      {{ props.helperText }}
    </Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { TagsInput } from "@ark-ui/vue/tags-input";
  import { Field } from "@ark-ui/vue/field";
  import { XMarkIcon } from "@heroicons/vue/24/outline";
  import OuiBadge from "./Badge.vue";

  interface Props {
    /**
     * Label for the tags input
     */
    label?: string;

    /**
     * Placeholder text for the input
     */
    placeholder?: string;

    /**
     * Helper text displayed below the input
     */
    helperText?: string;

    /**
     * Error message displayed below the input
     */
    error?: string;

    /**
     * Whether the tags input is required
     */
    required?: boolean;

    /**
     * Whether the tags input is disabled
     */
    disabled?: boolean;

    /**
     * Whether the tags input is read-only
     */
    readonly?: boolean;

    /**
     * Whether the tags input is invalid
     */
    invalid?: boolean;

    /**
     * Maximum number of tags allowed
     */
    max?: number;

    /**
     * Validation function to determine if a tag can be added
     */
    validate?: (details: { inputValue: string; value: string[] }) => boolean;

    /**
     * Behavior when input is blurred
     * - "add": add the input value as a new tag
     * - "clear": clear the input value
     */
    blurBehavior?: "add" | "clear";

    /**
     * Whether tags can be edited after creation
     */
    editable?: boolean;

    /**
     * Delimiter character(s) for splitting tags when pasting
     */
    delimiter?: string | RegExp;

    /**
     * Whether to add tags when pasting values
     */
    addOnPaste?: boolean;

    /**
     * Name attribute for form submission
     */
    name?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    placeholder: "Add tag...",
    blurBehavior: "add",
    editable: false,
    addOnPaste: false,
    delimiter: ",",
  });

  // Default validation: prevent duplicates and empty tags
  const defaultValidate = (details: {
    inputValue: string;
    value: string[];
  }) => {
    const trimmed = details.inputValue.trim();
    if (!trimmed) return false;
    return !details.value.includes(trimmed);
  };

  const effectiveValidate = computed(() => {
    return props.validate || defaultValidate;
  });

  const emit = defineEmits<{
    "update:modelValue": [value: string[]];
  }>();

  const modelValue = defineModel<string[]>({
    default: () => [],
  });

  const handleValueChange = (value: string[]) => {
    modelValue.value = value;
    emit("update:modelValue", value);
  };
</script>
