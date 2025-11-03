<template>
  <Format.Byte
    :value="byteValue"
    :unit="unit"
    :unit-display="unitDisplay"
    :locale="locale"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Format } from '@ark-ui/vue/format'

type UnitDisplay = 'narrow' | 'short' | 'long'
type ByteUnit = 'bit' | 'byte'

interface Props {
  value: number | bigint | string | null | undefined
  unit?: ByteUnit
  unitDisplay?: UnitDisplay
  locale?: string | string[]
}

const props = withDefaults(defineProps<Props>(), {
  unit: 'byte' as ByteUnit,
  unitDisplay: 'short' as UnitDisplay,
  locale: undefined,
})

const byteValue = computed<number>(() => {
  if (!props.value) return 0
  if (typeof props.value === 'number') return props.value
  if (typeof props.value === 'bigint') return Number(props.value)
  if (typeof props.value === 'string') {
    const parsed = Number.parseInt(props.value, 10)
    return Number.isNaN(parsed) ? 0 : parsed
  }
  return 0
})
</script>

