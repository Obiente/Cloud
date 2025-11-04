<template>
  <Slider.Root
    :modelValue="modelValue"
    :min="min"
    :max="max"
    :step="step"
    :disabled="disabled"
    @update:modelValue="handleValueChange"
    class="w-full"
  >
    <Slider.Label v-if="label" class="flex items-center justify-between mb-2 text-sm font-medium text-primary">
      {{ label }}
      <Slider.ValueText class="text-sm font-semibold text-accent-primary" />
    </Slider.Label>
    <Slider.Control class="relative w-full">
      <Slider.Track class="relative h-2 w-full rounded-full bg-border-muted">
        <Slider.Range class="absolute h-full rounded-full bg-accent-primary" />
      </Slider.Track>
      <Slider.Thumb
        v-for="(_, index) in modelValue"
        :key="index"
        :index="index"
        class="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 h-4 w-4 rounded-full bg-accent-primary border-2 border-background shadow-md cursor-grab active:cursor-grabbing hover:scale-110 transition-transform focus:outline-none focus:ring-2 focus:ring-accent-primary focus:ring-offset-2 data-[disabled]:opacity-50 data-[disabled]:cursor-not-allowed"
      >
        <Slider.HiddenInput />
      </Slider.Thumb>
    </Slider.Control>
  </Slider.Root>
</template>

<script setup lang="ts">
import { Slider } from "@ark-ui/vue/slider";

interface Props {
  modelValue: number[];
  min?: number;
  max?: number;
  step?: number;
  label?: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  min: 0,
  max: 100,
  step: 1,
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: number[]];
}>();

const handleValueChange = (value: number[]) => {
  emit("update:modelValue", value);
};
</script>

