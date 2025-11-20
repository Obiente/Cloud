<template>
  {{ formattedValue }}
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatBytes } from '~/utils/common'

type UnitDisplay = 'narrow' | 'short' | 'long'
type ByteUnit = 'bit' | 'byte'
type ByteBase = 'binary' | 'decimal'

interface Props {
  value: number | bigint | string | null | undefined
  unit?: ByteUnit
  unitDisplay?: UnitDisplay
  locale?: string | string[]
  base?: ByteBase
}

const props = withDefaults(defineProps<Props>(), {
  unit: 'byte' as ByteUnit,
  unitDisplay: 'short' as UnitDisplay,
  locale: undefined,
  base: 'binary' as ByteBase,
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

const formattedValue = computed(() => {
  // Convert bits to bytes if needed
  const bytes = props.unit === 'bit' ? byteValue.value / 8 : byteValue.value
  return formatBytes(bytes, props.base)
})
</script>

